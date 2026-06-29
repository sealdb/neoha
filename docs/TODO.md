# NeoHA TODO

> **架构与配置（长期参考）**
>
> - [docs/architecture.md](./architecture.md) — 四层架构、Coordinator/Driver/Reconcile、MySQL/PG 路径
> - [docs/config-design.md](./config-design.md) — 目标配置 schema、校验规则、与 v0.1 迁移
> - `internal/database/driver/` — L4 数据库抽象接口

> **当前阶段**：先把功能层面做到稳定、测试全绿，**暂缓** `internal/election/raft/` 架构重构。
> 架构讨论结论见 [architecture.md](./architecture.md) 与下文「Deferred — Raft 架构重构」。

---

## 当前优先（功能完善）

### P0 — 单元测试全部调通

`make test` / `go test $(go list ./... | grep -v test/integration)` 需无 FAIL。


| 状态  | 包                         | 测试                             | 现象 / 方向                                         |
| --- | ------------------------- | ------------------------------ | ----------------------------------------------- |
| [x] | `internal/database/mysql` | `TestMysqlRPCStatus`           | 已更新 mock 期望                                     |
| [x] | `internal/neohactl`       | `TestCLIRaftCommand_MySQL_MGR` | 已补 MGR mock                                     |
| [ ] | （持续）                      | raft MGR flaky                 | 用 `MockWaitUntil` / handler reset stabilization |


### P0 — 架构脚手架（v0.2，进行中）


| 状态  | 项                                                                    | 位置                                                                             |
| --- | -------------------------------------------------------------------- | ------------------------------------------------------------------------------ |
| [x] | L4 `database/driver` 接口                                              | `internal/database/driver/`                                                    |
| [x] | MySQL `Driver` 实现                                                    | `internal/database/mysql/driver.go`                                            |
| [x] | PG `Driver` stub                                                     | `internal/database/postgresql/driver.go`（v0.5 实装 bootstrap/promote/demote）     |
| [x] | L2 `coordination.Coordinator`                                        | `internal/coordination/`                                                       |
| [x] | Raft adapter                                                         | `internal/coordination/raftadapter/`                                           |
| [x] | L3 `ha.Reconciler` 骨架                                                | `internal/ha/reconcile.go`（apply 仍为 noop）                                      |
| [x] | 配置 `coordination` + `Validate()`                                     | `internal/config/config_validate.go`                                           |
| [x] | v0.3（进行中）：raft 经 dbDriver 调 Promotable/Demote/ChangeToMaster         | `internal/election/raft/dbops.go`                                              |
| [x] | v0.3：Reconciler apply（demote 安全网；promote 默认关闭）                       | `internal/ha/reconcile.go`                                                     |
| [x] | v0.3：server 启动 reconcile loop                                        | `internal/server/server.go`                                                    |
| [x] | v0.3：`ha.delegate_db_apply` MGR 两阶段 promote + 委托                     | `mysql/driver.go`, `ha/reconcile.go`, `raft/leader.go`                         |
| [x] | v0.4：`ha.primary_hooks` + Reconciler 触发（VIP 等）                       | `internal/ha/primary_hook.go`                                                  |
| [x] | 示例配置 YAML 逐行注释                                                       | `configs/examples/*.yaml`, `configs/README.md`                                 |
| [x] | v0.3 收口：Reconciler `ApplyReplica` + IT `delegate_db_apply`           | `internal/ha/reconcile.go`, `test/integration/harness/`                        |
| [x] | v0.3：Raft 在 delegate 模式下移除剩余 MySQL promote 路径                        | `internal/election/raft/leader.go`, `follower.go`                              |
| [x] | v0.5：PG ApplyReplica + ReplicationLagBytes                           | `internal/database/postgresql/`                                                |
| [x] | v0.5：etcd DCS + election 生命周期                                        | `internal/coordination/etcd/`, `internal/election/`                            |
| [x] | PG+etcd 示例配置                                                         | `configs/examples/postgresql/etcd-node1.yaml`                                  |
| [x] | v0.5：pg_rewind 自动化（ApplyReplica）                                     | `internal/database/postgresql/rewind.go`                                       |
| [x] | PG IT harness + ApplyReplica/pg_rewind 测试                            | `test/integration/harness/postgresql.go`                                       |
| [x] | IT 提速：并行 StartAll + 150ms poll + 分段计时                                | `test/integration/harness/backend.go`                                          |
| [x] | v0.5：MGR 多数派丢失（sole survivor force-bootstrap + read-only PRIMARY）    | `internal/election/raft/candidate.go`, `mysql/mysqlbase.go`, `ha/reconcile.go` |
| [x] | v0.5：MGR rejoin 后开放写（Phase 2，`mgrQuorum=2`）                          | `internal/ha/reconcile.go`, `mysql/driver.go`                                  |
| [x] | MGR majority-loss IT（`SoleSurvivorBootstrap` / `RejoinThenWritable`） | `test/integration/mgr_neoha_test.go`                                           |


```bash
make test
# 或
go test $(go list ./... | grep -v test/integration)
```

### P0 — 集成测试提速（对标 Xenon ~20s 选主）

**目标**：单次集群启动 + Raft 选出 LEADER（含 MGR）在 **~20s 内**完成；全量 integration suite **~6–8 分钟**（datadir 已存在、`NEOHA_IT_BIN` 预编译）。

**实测（2026-06-29，WSL，datadir 已存在，`NEOHA_IT_BIN`/`NEOHA_IT_CTL_BIN` 预编译）**：


| 测试                                 | 耗时           | 说明                                          |
| ---------------------------------- | ------------ | ------------------------------------------- |
| `TestMySQL3NodeScaffold`           | **~11–27s**  | 仅 3×mysqld，无 NeoHA                          |
| `TestNeoHAMGRWarmSuite`            | **~62–112s** | warm 集群；`FailoverMinority` 子测仅计故障段（~30–78s） |
| `TestNeoHAMGRFailoverMajorityLoss` | **~84–91s**  | Phase1 sole-survivor ~17–50s；Rejoin ~5s     |
| `TestNeoHASemiSyncWarmSuite`       | **~52–69s**  | warm 集群；`FailoverMinority` ~39–51s          |
| `TestNeoHAXtrabackupRebuildMe`     | **~36–56s**  | 仍走 `neohactl` wire + rebuildme              |
| `make test-integration`（全量）        | **~6–7 min** | Makefile 全局超时 **15m**                       |


**结论：慢的主因不是 Raft election timeout 本身**（MGR IT 用 500/1500ms；**semi-sync IT 应为 2s×5≈10s**，见下），而是：

### Semi-sync IT 心跳 / 故障判定（对齐 Xenon）


| 参数                            | 值            | 含义                |
| ----------------------------- | ------------ | ----------------- |
| `heartbeat-timeout`           | **2000 ms**  | 每 2 秒一次心跳         |
| `admit-defeat-hearbeat-count` | **5**        | 连续 5 次心跳失败        |
| **主故障判定**                     | **~10 s**    | 5 × 2 s           |
| `election-timeout`            | **10000 ms** | Follower 侧与上述量级一致 |


已在 `harness/neoha.go` 的 `applySemiSyncRaftIT()` 落地；MGR IT 仍用 `500/1500`（`applyMGRRaftIT()`）。Failover 用例在 **集群已 warm** 后，semi-sync 重新选主应在 **~10–15 s** 内完成。

**其它慢因（已缓解 / 剩余）**：

1. ~~**每个测试独立冷启动整集群**~~ → MGR / semi-sync 已用 **warm fixture** 复用同一集群（`TestNeoHAMGRWarmSuite`、`TestNeoHASemiSyncWarmSuite`）
2. ~~`**StartAll` 串行**~~ → 已并行；poll **150ms**
3. ~~**Harness 多轮 CLI wire**~~ → MGR / semi-sync warm 测已 **预写 `peers.json`**（`skipCLIWire`）；xtrabackup 仍保留 CLI 路径
4. **与 Xenon 对比口径不同**：Xenon ~20s 指 **集群已运行** 下的 failover 段；NeoHA IT 仍把首次 datadir init 算进用例时间
5. ~~**Failover 从零 bootstrap**~~ → warm 子测只计 stop primary → 新 PRIMARY 段

**待办 / 方向**：

- [x] 用 `t.Log(time.Since(...))` 分段计时（init / mysqld ready / neoha ready / MGR online / failover 段）
- [x] **共享 warm fixture**：`bootstrapWarmNeoHA` + `newWarmMySQLCluster`；failover 为子测试（`mgr_neoha_test.go`, `semisync_neoha_test.go`, `mysql_warm_test.go`）
- [x] 并行 `StartAll` mysqld；poll 间隔 500ms → 150ms
- [x] peers 写入 `peers.json`，MGR / semi-sync warm 跳过 `WireNeoHAClusterViaCLI`（xtrabackup 仍走 CLI）
- [ ] 跨测试复用 datadir（同端口 + 稳定 cluster 名）或 `make it-prep` 一次初始化
- [x] 修复 semi-sync failover（`TestNeoHASemiSyncWarmSuite/FailoverMinority`）
- [x] v0.5：PG Driver 基础（bootstrap / promote / demote / status）
- [x] v0.5：PG ApplyReplica（primary_conninfo + slot）
- [x] v0.5：etcd Coordinator DCS + server wiring
- [x] v0.5：PG ReplicationLagBytes / Promotable lag 校验
- [x] PG+etcd IT：`TestNeoHAPGEtcdFailoverMinority`
- [x] pg_rewind IT：`TestPostgreSQLApplyReplicaPgRewind`
- [x] MGR IT delegate 路径回归（`TestNeoHAMGRWarmSuite/3NodeMGR`）
- [x] MGR majority-loss harness：独立端口（13326–13328 / GR 13381–13383）、动态 `group_replication_group_seeds`、`StartNode` 不随 ctx 取消杀 mysqld
- [ ] 文档：区分「单测端到端」vs「failover 段」baseline（README 已部分更新）
- [x] 文档：MySQL 3 节点 semi-sync / MGR 切换逻辑与时序图 → [ha-failover.md](./ha-failover.md)
- [ ] Makefile 默认传入 `NEOHA_IT_BIN`（避免 IT 隐式编译）

验证命令：

```bash
export NEOHA_IT_BIN=$PWD/bin/neoha NEOHA_IT_CTL_BIN=$PWD/bin/neohactl
/usr/bin/time go test -tags=integration -count=1 -timeout=15m ./test/integration -run TestNeoHAMGRWarmSuite
/usr/bin/time go test -tags=integration -count=1 -timeout=15m ./test/integration -run TestNeoHAMGRFailoverMajorityLoss
make test-integration
```

---

## Deferred — Raft 架构重构（暂缓）

以下为用户确认的讨论结论，**实施顺序在功能完善之后**。

### 原则

- **不合并** Idle / Learner / Invalid 为单一 `passiveRole`（三者语义、API、迁移路径不同，须保留独立 type）
- 最小 diff、保持现有 mock / `setProcess*Handler` 测试能力
- 分 PR：`helpers` → 单 loop（F/C/L）→ MySQL 副作用层 → 可选 etcd 式 step

### 目标形态（三层）

1. **L1 共识**（学 etcd）：单 event loop + `become`* + 六个 `stepXxx`（Follower/Candidate/Leader/Idle/Learner/Invalid **各自独立**）
2. **L2 编排**（学 Patroni `Ha.run_cycle`）：`reconcile()` + `CandidacyEvaluator`（Promotable、GTID、backoff），共识与 MySQL 角色对齐
3. **L3 执行**：`change master` / readonly / semi-sync / MGR / VIP 从 handler 抽到 `mysqlfx` 或等价模块

### 已识别冗余（仅 helper 级可先动）

- RPC 响应构造（`newRaftRPCResponse`）
- `applyEpochView`、replica `stateInit` 公共 MySQL 步骤
- 六个角色的 handler 注入样板（测试 mock 保留）

### 参考对比（不照搬）


| 来源            | 可借鉴                                                    | 不照搬                          |
| ------------- | ------------------------------------------------------ | ---------------------------- |
| **etcd/raft** | 单 struct、`step`/`tick` 函数指针、`Step()` 统一入口              | Learner 作 flag；仅 3 共识态       |
| **Patroni**   | reconcile 单循环、eligibility 再抢主、DCS 不可达 demote、async 副作用 | 外置 DCS leader lock 替代内嵌 Raft |
| **Xenon**     | 无 DCS、P2P Raft、GTID + semi-sync 选主                     | —                            |


### 相关文件

```
internal/election/raft/     # leader/follower/candidate/idle/learner/invalid
internal/election/election.go   # ElectionEtcd 仍为 TODO
internal/election/etcd/etcd.go  # 空 stub
```

---

## 中长期 backlog（原 README Todo 整理）

以下条目来自早期 README 与内部讨论，**优先级低于上文 P0**；已完成项保留作历史对照，避免与 P0 脚手架表重复展开。

### 产品与生态

| 状态 | 项 | 说明 |
|------|-----|------|
| [ ] | 多引擎单机 HA | 统一框架覆盖 MySQL、PostgreSQL；Oracle 等暂不在 v1 范围 |
| [ ] | 云平台能力下沉 | 慢日志分析、binlog/GTID 检索、备份恢复、闪回等原计划在平台层实现的能力，逐步收敛到 NeoHA |

### 功能增强

| 状态 | 项 | 说明 |
|------|-----|------|
| [ ] | 慢日志分析 | 集成 pt-query-digest 等工具，缓存 Top N 结果供查询 |
| [ ] | Binlog / GTID 检索 | 按 GTID 快速定位事务与 binlog 位置 |
| [ ] | 闪回 | DML 闪回；DDL 无法闪回时需 rebuild / xtrabackup 等路径（参考 TDSQL：DML 工具闪回，DDL 重建从库） |
| [ ] | 选主位点策略 | 除 GTID 外，可选 binlog file + position 比较 |
| [ ] | 一键部署 | 安装包 / Ansible / Helm 等自动化部署 |

### 复制与 failover 边界条件

| 状态 | 项 | 说明 |
|------|-----|------|
| [ ] | 半同步「哨兵」节点 | 仅 Raft 投票、不复制数据、不发起选举（对应 Learner/投票专用角色设计） |
| [ ] | 从库本地事务 → INVALID | 从库写入本地事务后应进入 INVALID，阻止参与选主；当前行为待完善 |
| [ ] | 主 IO hang / 主 hang 从不 hang | 复制线程或主库 hang 时选主与 demote 语义 |
| [ ] | 网卡 flapping | 短时间 `ifdown`/`ifup` 时的脑裂与恢复；现多依赖外围 deploy，组件内策略 TBD |
| [ ] | datadir 被删 | MySQL 数据目录缺失时仍可能接受写请求；需与 Driver 健康检查联动 |
| [ ] | 旧主半同步脏写 | 切换时旧主在 `Waiting for semi-sync ACK from slave` 状态下可能产生本地事务；见下 |

**旧主半同步切换（待设计 / 内核配合）：**

1. **Truncate binlog**：旧主 mysqld down 或 demote 前截断 binlog，避免恢复后携带脏事务（`purge_binlog` / 内核能力）。
2. **优雅 demote 顺序**：理想顺序为先结束处于 semi-sync ACK 等待的会话，再关 semi-sync、再 `CHANGE MASTER` + `START SLAVE`；实测普通 `KILL` 无法终止如下状态，可能需要内核 `KILL FORCE` 或等价机制：

```sql
-- 示例：Killed 后仍卡在 semi-sync ACK / handler commit
| Id | State                                | Info                |
| 11 | Waiting for semi-sync ACK from slave | create database db4 |
| 12 | waiting for handler commit           | create database db5 |
```

### 基础能力

| 状态 | 项 | 说明 |
|------|-----|------|
| [ ] | MySQL 5.6 / 5.7 | 当前开发与 IT 以 **8.0** 为主 |
| [x] | MySQL 8.0 + Semi-sync / MGR | 已实现；切换逻辑见 [ha-failover.md](./ha-failover.md) |
| [x] | 内嵌 Raft | `internal/election/raft/` |
| [x] | PostgreSQL Driver（v0.5） | bootstrap / promote / demote / ApplyReplica / pg_rewind |
| [x] | etcd Coordinator（v0.5） | DCS MVP + PG IT |
| [ ] | 其他协调后端 | Consul、ZooKeeper、Kubernetes |
| [ ] | SQL 驱动整合 | MySQL：`go-sql-driver` / mysqlstack 评估；PG：[jackc/pgx](https://github.com/jackc/pgx) 模块化接入 |

### 文档

| 状态 | 项 | 说明 |
|------|-----|------|
| [x] | 架构与配置设计 | [architecture.md](./architecture.md)、[config-design.md](./config-design.md) |
| [x] | MySQL HA 切换原理 | [ha-failover.md](./ha-failover.md)（含时序图） |
| [x] | 集成测试说明 | [test/integration/README.md](../test/integration/README.md) |
| [ ] | 部署手册 | 安装、配置、peers 组网 |
| [ ] | 运维手册 | 日常巡检、failover 演练、故障处理 |
| [ ] | 生产调参指南 | heartbeat / election / MGR quorum 与 RPO/RTO 关系 |

### 代码质量

| 状态 | 项 | 说明 |
|------|-----|------|
| [ ] | 测试代码去重 | 公共 setup/assert 收敛到 `*_test.go` / `mock.go` |
| [ ] | Raft 层通用化 | 与 L2 Coordinator 边界清晰后，共识层可复用于其他引擎（见「Deferred — Raft 架构重构」） |

---

## 变更记录


| 日期         | 说明                                                                  |
| ---------- | ------------------------------------------------------------------- |
| 2026-06-28 | 创建本文档；架构重构 deferred；P0 为单元测试 + IT 提速                                |
| 2026-06-28 | 新增 architecture.md、config-design.md、database/driver.go 设计基线         |
| 2026-06-29 | MGR majority-loss + warm IT 落地；更新 IT 耗时 baseline；Makefile IT 超时 15m |
| 2026-06-29 | 添加「中长期 backlog」；链出 ha-failover.md |
