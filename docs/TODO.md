# NeoHA TODO

> **文档地图**（运行时分层见 [architecture.md](./architecture.md) **L1–L5**）
>
> | 文档 | 内容 |
> |------|------|
> | [architecture.md](./architecture.md) | L1 Shell → L5 Manager；Coordinator / Driver / Reconcile |
> | [config-design.md](./config-design.md) | 目标 `coordination.*` schema、校验、与 `election.*` 迁移 |
> | [ha-failover.md](./ha-failover.md) | MySQL 3 节点 semi-sync / MGR 切换与时序图 |
> | [deployment.md](./deployment.md) | 编译安装、三节点组网、neohactl、检查清单 |
> | [test/integration/README.md](../test/integration/README.md) | IT harness、环境、`make test-integration` |
> | `internal/database/driver/` | L4 数据库抽象接口 |

> **当前阶段**：功能稳定、测试全绿优先；**暂缓** `internal/election/raft/` 内聚式大重构（见 [Deferred — Raft 重构](#deferred--raft-重构暂缓)）。

---

## P0 — 进行中

### 质量门禁 — 单元测试

`make test` / `go test $(go list ./... | grep -v test/integration)` 需无 FAIL。

| 状态 | 包 | 测试 / 主题 | 说明 |
|------|-----|-------------|------|
| [x] | `internal/database/mysql` | `TestMysqlRPCStatus` | mock 期望已对齐 |
| [x] | `internal/neohactl` | `TestCLIRaftCommand_MySQL_MGR` | 已补 MGR mock |
| [ ] | `internal/election/raft` | MGR flaky | `MockWaitUntil` / handler reset 稳定化（当前 `make test` 全绿，持续观察） |

```bash
make test
# 或
go test $(go list ./... | grep -v test/integration)
```

### 工程化 — 集成测试

**目标**：warm 集群下 failover 段对标 Xenon **~10–20s**；全量 suite **~6–8 min**（datadir 已存在、`NEOHA_IT_BIN` 预编译）。

| 状态 | 项 | 说明 |
|------|-----|------|
| [x] | 跨测试复用 datadir | 稳定 cluster 名 + 默认保留 workdir；`make test-integration-prep` 预初始化 |
| [x] | Makefile 默认 `NEOHA_IT_BIN` | `make test-integration` / `make test-integration-prep` 依赖 `make build`，默认 `./bin/neoha` |
| [x] | 文档：failover 段 baseline | [test/integration/README.md](../test/integration/README.md) § Failover segment baseline |

**实测 baseline（2026-06-29，WSL，datadir 已存在，bin 预编译）**：

| 测试 | 耗时 | 说明 |
|------|------|------|
| `TestMySQL3NodeScaffold` | ~11–27s | 仅 3×mysqld，无 NeoHA |
| `TestNeoHAMGRWarmSuite` | ~62–112s | warm；`FailoverMinority` 故障段 ~30–78s |
| `TestNeoHAMGRFailoverMajorityLoss` | ~84–91s | sole-survivor ~17–50s；Rejoin ~5s |
| `TestNeoHASemiSyncWarmSuite` | ~52–69s | warm；`FailoverMinority` ~39–51s |
| `TestNeoHAXtrabackupRebuildMe` | ~36–56s | `neohactl` wire + rebuildme |
| `make test-integration`（全量） | ~6–7 min | Makefile 全局超时 **15m** |

Semi-sync IT 心跳（`harness/neoha.go` → `applySemiSyncRaftIT()`，对齐 Xenon）：

| 参数 | 值 | 含义 |
|------|-----|------|
| `heartbeat-timeout` | 2000 ms | 每 2s 心跳 |
| `admit-defeat-hearbeat-count` | 5 | 连续失败次数 |
| **主故障判定** | **~10 s** | 5 × 2s |
| `election-timeout` | 10000 ms | Follower 侧 |

MGR IT 仍用 `500/1500`（`applyMGRRaftIT()`）。慢因主因是冷启动 datadir init，非 election timeout 本身；warm fixture 已缓解冷启动与串行 `StartAll`、CLI wire 等问题。

```bash
export NEOHA_IT_BIN=$PWD/bin/neoha NEOHA_IT_CTL_BIN=$PWD/bin/neohactl
/usr/bin/time go test -tags=integration -count=1 -timeout=15m ./test/integration -run TestNeoHAMGRWarmSuite
/usr/bin/time go test -tags=integration -count=1 -timeout=15m ./test/integration -run TestNeoHAMGRFailoverMajorityLoss
make test-integration
```

---

## 已完成里程碑（v0.1.1 → v0.1.4，合并发布 **v0.2.0**）

按 L2/L3/L4 的摘要如下；版本约定见 [architecture.md §15](./architecture.md#15-implementation-roadmap)。**逐项交付清单（含文件路径）** 见 [§15.2](./architecture.md#152-delivered-items-v011v014)。

### L2 Coordination

| 版本 | 交付 | 位置 |
|------|------|------|
| v0.1.1 | `coordination.Coordinator` 接口 | `internal/coordination/` |
| v0.1.1 | Raft adapter | `internal/coordination/raftadapter/` |
| v0.1.1 | `coordination.*` + `Validate()`（兼容 `election.*`） | `internal/config/config_validate.go` |
| v0.1.4 | etcd DCS + election 生命周期 | `internal/coordination/etcd/`, `internal/election/` |
| v0.1.4 | PG+etcd 示例配置 | `configs/examples/postgresql/etcd-node1.yaml` |

### L3 HA Reconcile

| 版本 | 交付 | 位置 |
|------|------|------|
| v0.1.1 | `ha.Reconciler` 骨架 | `internal/ha/reconcile.go` |
| v0.1.2 | Reconciler apply（demote 安全网） | `internal/ha/reconcile.go` |
| v0.1.2 | server reconcile loop | `internal/server/server.go` |
| v0.1.2 | `ha.delegate_db_apply` MGR 两阶段 promote | `mysql/driver.go`, `ha/reconcile.go`, `raft/leader.go` |
| v0.1.2 | `ApplyReplica` + IT delegate 路径 | `internal/ha/reconcile.go`, `test/integration/harness/` |
| v0.1.3 | `ha.primary_hooks`（VIP 等） | `internal/ha/primary_hook.go` |
| v0.1.4 | MGR 多数派丢失（sole survivor read-only → rejoin 开放写） | `candidate.go`, `mysqlbase.go`, `reconcile.go`, `mysql/driver.go` |

### L4 Database Driver

| 版本 | 交付 | 位置 |
|------|------|------|
| v0.1.1 | `database/driver` 接口 | `internal/database/driver/` |
| v0.1.1–v0.1.2 | MySQL `Driver` + delegate 模式下 Raft 去 MySQL promote | `internal/database/mysql/` |
| v0.1.2 | Raft 经 dbDriver 调 Promotable / Demote / ChangeToMaster | `internal/election/raft/dbops.go` |
| v0.1.4 | PG bootstrap / promote / demote / ApplyReplica / pg_rewind | `internal/database/postgresql/` |
| v0.1.4 | PG ReplicationLagBytes / Promotable lag 校验 | `internal/database/postgresql/lag.go` |

### 测试与文档

| 版本 | 交付 | 位置 |
|------|------|------|
| v0.1.1+ | 示例配置 YAML 注释 | `configs/examples/*.yaml`, `configs/README.md` |
| v0.1.4 | IT 提速：warm fixture、并行 StartAll、150ms poll、分段计时 | `test/integration/harness/backend.go`, `*_warm_test.go` |
| v0.1.4 | PG IT：ApplyReplica / pg_rewind / etcd failover | `harness/postgresql.go`, `pg_*_test.go` |
| v0.1.4 | MGR IT：delegate、majority-loss、独立端口 harness | `test/integration/mgr_neoha_test.go` |
| v0.1.4 | semi-sync failover 修复 | `test/integration/semisync_neoha_test.go` |
| v0.1.4 | MySQL HA 切换文档 | [ha-failover.md](./ha-failover.md) |
| v0.1.4 | 架构 / 配置设计文档 | [architecture.md](./architecture.md), [config-design.md](./config-design.md) |
| v0.1.4 | 部署指南 | [deployment.md](./deployment.md) |

---

## Deferred — Raft 重构（暂缓）

功能完善之后再做。与产品 **L1–L5** 不同，此处指 **Raft 包内部** 三层拆分：

| 层 | 目标 | 参考 |
|----|------|------|
| 共识 | 单 event loop + `become*` + 六角色独立 `stepXxx` | etcd/raft |
| 编排 | `reconcile()` + `CandidacyEvaluator`（Promotable、GTID、backoff） | Patroni `Ha.run_cycle` |
| 执行 | semi-sync / MGR / VIP 等从 handler 抽到 `mysqlfx` 或等价模块 | — |

**原则**

- **不合并** Idle / Learner / Invalid 为单一 passive role
- 最小 diff；保留 mock / `setProcess*Handler` 测试能力
- 分 PR：`helpers` → 单 loop（F/C/L）→ MySQL 副作用层 → 可选 etcd 式 step

**可先动的 helper 级冗余**：RPC 响应构造、`applyEpochView`、replica `stateInit` 公共步骤、六角色 handler 注入样板。

**参考对比**

| 来源 | 可借鉴 | 不照搬 |
|------|--------|--------|
| **etcd/raft** | 单 struct、`step`/`tick`、`Step()` 统一入口 | Learner 作 flag；仅 3 共识态 |
| **Patroni** | reconcile 单循环、eligibility、DCS 不可达 demote | 外置 DCS 替代内嵌 Raft |
| **Xenon** | P2P Raft、GTID + semi-sync/MGR 选主 | — |

**相关文件**：`internal/election/raft/`、`internal/election/election.go`

---

## 中长期 backlog

优先级低于 P0；**仅列未完成项**。已交付能力见上文「已完成里程碑」。

### 产品与生态

| 项 | 说明 |
|-----|------|
| 多引擎扩展 | Oracle 等暂不在 v1；MySQL / PG 为主 |
| 云平台能力下沉 | 慢日志、binlog/GTID 检索、备份恢复、闪回等收敛到 NeoHA |

### 功能增强

| 项 | 说明 |
|-----|------|
| 慢日志分析 | pt-query-digest 等，缓存 Top N |
| Binlog / GTID 检索 | 按 GTID 定位事务与位点 |
| 闪回 | DML 闪回；DDL 需 rebuild（TDSQL 策略：DML 工具闪回，DDL 重建从库） |
| 选主位点策略 | GTID 之外支持 binlog file + position |
| 一键部署 | Ansible / Helm 等 |

### 复制与 failover 边界（MySQL 为主）

| 项 | 说明 |
|-----|------|
| 半同步「哨兵」节点 | 仅投票、不复制、不发起选举（Learner 类角色） |
| 从库本地事务 → INVALID | 写入本地事务后禁止参与选主 |
| 主 IO hang / 主 hang 从不 hang | hang 时选主与 demote 语义 |
| 网卡 flapping | 短 `ifdown`/`ifup` 脑裂与恢复；组件内策略 TBD |
| datadir 被删 | 数据目录缺失时仍可能写；需 Driver 健康检查 |
| 旧主半同步脏写 | `Waiting for semi-sync ACK from slave` 下可能产生本地事务 |

**旧主半同步切换（待设计 / 内核配合）**

1. **Truncate binlog**：demote 前截断，避免脏事务（`purge_binlog` / 内核能力）。
2. **优雅 demote**：先结束 semi-sync ACK 等待会话，再关 semi-sync、`CHANGE MASTER`、`START SLAVE`；普通 `KILL` 无效，或需 `KILL FORCE`：

```sql
-- Killed 后仍卡在 semi-sync ACK / handler commit
| Id | State                                | Info                |
| 11 | Waiting for semi-sync ACK from slave | create database db4 |
| 12 | waiting for handler commit           | create database db5 |
```

### 基础能力

| 项 | 说明 |
|-----|------|
| MySQL 5.6 / 5.7 | 开发与 IT 以 **8.0** 为主 |
| 其他协调后端 | Consul、ZooKeeper、Kubernetes |
| SQL 驱动整合 | MySQL：`go-sql-driver` / mysqlstack；PG：[jackc/pgx](https://github.com/jackc/pgx) |

### 文档

| 项 | 说明 |
|-----|------|
| 运维手册 | 巡检、failover 演练、故障处理（部署见 [deployment.md](./deployment.md)） |
| 生产调参指南 | heartbeat / election / MGR quorum 与 RPO/RTO |

### 代码质量

| 项 | 说明 |
|-----|------|
| 测试代码去重 | setup/assert 收敛到 `*_test.go` / `mock.go` |
| Raft 层通用化 | L2 Coordinator 边界清晰后，共识层复用于其他引擎 |

---

## 变更记录

| 日期 | 说明 |
|------|------|
| 2026-06-28 | 创建本文档；Raft 重构 deferred；P0 为单测 + IT 提速 |
| 2026-06-28 | 新增 architecture.md、config-design.md、driver 设计基线 |
| 2026-06-29 | MGR majority-loss + warm IT；IT baseline；Makefile 超时 15m |
| 2026-06-29 | 整理 backlog；链出 ha-failover.md |
| 2026-06-29 | 重组文档结构：P0 仅未完成项；里程碑按 L2/L3/L4 归档；backlog 去重 |
| 2026-06-29 | 新增 [deployment.md](./deployment.md) |
| 2026-06-29 | 版本号对齐：dev 里程碑 v0.1.1–v0.1.4，PR 合并后打 tag **v0.2.0** |
| 2026-07-01 | 阶段 1：config 对齐 config-design；test-integration-prep + 默认 bin + datadir 复用 |
