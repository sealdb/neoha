# NeoHA 运维手册

> **状态：** v0.2.x 手工运维流程。安装组网见 [deployment.md](./deployment.md)；切换时序见 [ha-failover.md](./ha-failover.md)；配置字段见 [config-design.md](./config-design.md)。
>
> **集成测试不在 CI 中运行**（需本机 MySQL / Xtrabackup / SSH），请在变更后于开发机执行 `make test-integration`。CI 仅跑单元测试与覆盖率（见 [README](../README.md)）。

---

## 1. 日常巡检

建议 **每班次或每日** 对生产集群执行以下检查。三节点 MySQL + 内嵌 Raft 与 PG + etcd 的检查项略有不同。

### 1.1 NeoHA agent

| 检查项 | 命令 / 方式 | 期望 |
|--------|-------------|------|
| 进程存活 | `systemctl status neoha` 或 `ps` | 各节点 running |
| RPC 可达 | `neohactl neoha ping -c <endpoint>` | 成功 |
| Raft 角色 | `neohactl raft status -c <config>` | 恰好 **1** 个 `LEADER`，其余 `FOLLOWER` / `LEARNER` |
| 成员列表 | 查看 `coordination.raft.meta-datadir/peers.json` | 与规划节点一致 |
| 日志 ERROR | `grep -E 'ERROR|PANIC' /var/log/neoha/*.log` | 无持续 ERROR（偶发网络可接受） |
| `tags.nofailover` | 配置 | 备节点按需标记，避免误升主 |

### 1.2 MySQL（semi-sync / MGR）

| 检查项 | SQL / 命令 | 期望 |
|--------|------------|------|
| 实例存活 | `mysqladmin ping` | alive |
| 只读状态 | `SELECT @@read_only, @@super_read_only` | 主：`0`；从：`1` |
| 复制（semi-sync） | `SHOW SLAVE STATUS\G` | `Slave_IO_Running=Yes`，`Slave_SQL_Running=Yes` |
| 半同步 | `SHOW VARIABLES LIKE 'rpl_semi_sync%'` | 主库 plugin ON；从库 ON |
| MGR 成员 | `SELECT MEMBER_HOST, MEMBER_STATE, MEMBER_ROLE FROM performance_schema.replication_group_members` | 多数 `ONLINE`；**1** 个 `PRIMARY` |
| GTID | `SHOW MASTER STATUS` / `@@gtid_executed` | 从库 GTID 不长期落后主库 |
| 连接数 | `Threads_connected` vs `max_connections` | 远离 1040 阈值 |

### 1.3 PostgreSQL + etcd

| 检查项 | 命令 | 期望 |
|--------|------|------|
| PG 存活 | `pg_isready -h <listen>` | accepting connections |
| 复制 | `SELECT pg_is_in_recovery(), pg_last_wal_replay_lsn()` | 主：`f`；从：`t` 且 lag 可控 |
| Slot | `pg_replication_slots` | 从库 slot 活跃、`active=true` |
| etcd 健康 | `etcdctl endpoint health` | 全部 healthy |
| DCS leader | NeoHA 日志 / `neohactl`（若已暴露） | 与当前可写 PG 节点一致 |
| Lease TTL | `coordination.etcd.ttl` | 与 Patroni 风格预期一致（默认 30s） |

### 1.4 主机与网络

- NTP / chrony 偏移 **< 1s**
- 节点间 **`endpoint` 端口**（NeoHA RPC）互通
- 磁盘：`meta-datadir`、MySQL datadir、PG `data_dir` 使用率 **< 80%**
- 若启用 `watchdog`：设备可写、NeoHA 无 watchdog timeout 日志

---

## 2. Failover 演练 SOP

演练前：**备份**、选择 **维护窗口**、通知下游只读连接。建议在 **staging** 先完整走一遍；生产首次演练保留原主不立即重启，便于对比 GTID / 位点。

### 2.1 通用准备

1. 确认三节点 NeoHA + DB 均 healthy（§1 巡检通过）。
2. 记录当前 Raft LEADER、`scope`、各节点 `name` 与 `endpoint`。
3. 记录当前 **可写 primary**（MySQL `read_only=0` 或 MGR `PRIMARY` / PG 非 recovery）。
4. 准备观测：`tail -f neoha.log`、复制状态 SQL、应用写入探针（可选）。

### 2.2 MySQL semi-sync —  minority 故障（停主）

**场景：** 单节点 mysqld + NeoHA 同时不可用，期望 **~10–20s** 内 survivors 选出新主（取决于 `heartbeat-timeout` × `admit-defeat-hearbeat-count` + 复制切换）。

| 步骤 | 操作 | 验证 |
|------|------|------|
| 1 | 确认当前 primary（Raft LEADER 对应 mysqld） | `SHOW SLAVE STATUS` 仅在从库有输出 |
| 2 | **停止 primary 的 mysqld**：`systemctl stop mysqld` | 应用写入失败（预期） |
| 3 | **停止 primary 的 NeoHA**：`systemctl stop neoha` | — |
| 4 | 等待 | 存活节点 NeoHA 日志：选举 + demote/promote |
| 5 | 检查新主 | 唯一 `read_only=0`；`SHOW MASTER STATUS` 可写 |
| 6 | 检查从库 | `Master_Host` 指向新主；IO/SQL Running |
| 7 | 恢复旧主 | 先修数据目录 / `CHANGE MASTER` 或 xtrabackup rebuild，再 `start` |

**回切（可选）：** 旧主追平后 `neohactl raft trytoleader`（需无 `tags.nofailover`）或维护窗口内计划切换。

### 2.3 MySQL MGR — minority 故障

**场景：** 停当前 MGR `PRIMARY` 所在 mysqld，MGR 在 survivors 内重选 PRIMARY；NeoHA Raft + Reconciler（`ha.delegate_db_apply: true`）对齐角色。

| 步骤 | 操作 | 验证 |
|------|------|------|
| 1 | `SELECT MEMBER_ROLE, MEMBER_STATE FROM replication_group_members` | 记录 PRIMARY |
| 2 | 停止 PRIMARY 节点 mysqld + NeoHA | — |
| 3 | 等待 | survivors 上 **1** 个 `PRIMARY`，`ONLINE` ≥ 2 |
| 4 | 写探针 | `INSERT` 在新 PRIMARY 成功 |
| 5 | 恢复旧节点 | 以 clone / rebuild / 重新 join MGR 方式；见 [ha-failover.md §6](./ha-failover.md) majority-loss 若曾发生 |

### 2.4 MySQL MGR — majority loss（仅演练环境）

**场景：** 两节点 down，sole survivor 以 **read_only PRIMARY** bootstrap，rejoin 后开放写。生产环境 **极高风险**，仅限隔离环境验证。

参考 [ha-failover.md](./ha-failover.md) Phase 1 / Phase 2 时序；IT 用例 `TestNeoHAMGRFailoverMajorityLoss`。

### 2.5 PostgreSQL + etcd — minority 故障

| 步骤 | 操作 | 验证 |
|------|------|------|
| 1 | 确认 DCS leader 与 PG primary 一致 | `pg_is_in_recovery()` 主库为 `f` |
| 2 | 停止 primary 的 **postmaster + neoha** | — |
| 3 | 等待 etcd lease 过期 + Reconciler promote | survivor 上 `pg_is_in_recovery()=f` |
| 4 | 检查 lag 策略 | `maximum_lag_on_failover` 未阻止 promote（或日志说明原因） |
| 5 | 旧主 rejoin | `use_pg_rewind: true` 时自动 rewind；否则 rebuild |

---

## 3. 生产调参指南

以下参数影响 **故障发现时间（RTO）** 与 **数据安全（RPO）**。调参后需 **滚动重启 NeoHA**（DB 参数另论）。Semi-sync 与 MGR 不要混用同一套 Raft 心跳数值。

### 3.1 Raft / 协调层（`coordination.raft`）

| 参数 | 默认（示例） | 调大效果 | 调小效果 | 建议 |
|------|--------------|----------|----------|------|
| `heartbeat-timeout` | 1000 ms | 降低误报，**延长**故障发现 | 更快发现，易抖动 | semi-sync 生产可 **2000 ms**（对齐 Xenon IT） |
| `admit-defeat-hearbeat-count` | 10 | 更耐网络闪断 | 更快 step-down | semi-sync：**5**（≈10s 判定） |
| `election-timeout` | 3000 ms | Follower 更慢发起选举 | 更激进选举 | 两节点集群适当 **增大** |
| `requesttimeout` | 1000 ms | RPC 更宽容 | 更快失败 | 跨 AZ 可 **2000–3000 ms** |
| `candidate-wait-for-2nodes` | 60000 ms | 两节点更不易脑裂双主 | 两节点 failover 更慢 | 两节点集群 **必调** |

**故障发现粗算：** `heartbeat-timeout × admit-defeat-hearbeat-count`（Leader 侧）+ 选举 + Driver apply。

### 3.2 MySQL Driver（`database.mysql`）

| 参数 | 说明 | 建议 |
|------|------|------|
| `ping-timeout` | 管理连接 ping 间隔 | 1000 ms；与 Raft 心跳独立 |
| `admit-defeat-ping-count` | ping 失败多少次报 MysqlDead | 2–3；过小易误 failover |
| `failover-on-too-many-connections` | 1040 是否触发 failover | 连接打满场景可 `true` |
| `semi-sync-timeout-for-two-nodes` | 两节点 semi-sync 微秒超时 | 按 RPO 评估 |
| `ha.delegate_db_apply` | Reconciler 负责升/降主 | **MGR 建议 true** |
| `ha.reconcile_interval` | Reconcile tick（秒） | 生产 3；低 RTO 可 **1–2** |

### 3.3 PostgreSQL（`database.postgresql`）

| 参数 | 说明 | 建议 |
|------|------|------|
| `maximum_lag_on_failover` | 允许 promote 的最大字节 lag | 按业务设定；0=不限制 |
| `use_slots` / `primary_slot_name` | 复制 slot | 生产建议 **true** + 命名 slot |
| `use_pg_rewind` | 旧主 rejoin | 需 `bin_dir` + 同集群；生产建议 **true** |
| `synchronous_mode` | 同步提交（语义演进中） | 严格 RPO 时评估开启 |

### 3.4 etcd DCS（`coordination.etcd`）

| 参数 | 默认 | 说明 |
|------|------|------|
| `ttl` | 30 s | Leader lease；越小 failover 越快，etcd 负载越高 |
| `loop_wait` | 10 s | Reconcile 间隔（Patroni 风格） |
| `retry_timeout` | 10 s | DCS 操作重试窗口 |

**注意：** PG failover 策略在 `database.postgresql.*`；勿与 `bootstrap.postgresql.dcs.*` 混淆（后者为 legacy 首次初始化）。

### 3.5 标签与钩子

| 项 | 用途 |
|----|------|
| `tags.nofailover` | 永久禁止升主（哨兵 / 观测节点） |
| `tags.nosync` | PG 非同步副本 |
| `ha.primary_hooks.on_primary_start/stop` | VIP、LB 注册/摘除；生产替换默认 `nop` |

---

## 4. 常见故障处理

| 现象 | 可能原因 | 处理 |
|------|----------|------|
| 双主 / 双写 | 网络分区、两节点未调 `candidate-wait-for-2nodes` | 隔离一方；以 GTID 为准选 survivor；见 [ha-failover.md](./ha-failover.md) |
| Raft 无 LEADER | 多数 NeoHA down 或 peers.json 不一致 | 恢复 quorum；`neohactl cluster add` / 修 peers |
| MGR 全 OFFLINE | 多数 mysqld down | majority-loss 流程；勿在生产强行单节点写 |
| PG promote 被拒绝 | lag > `maximum_lag_on_failover` | 追平从库或临时调参（需变更流程） |
| etcd lease 丢失 | etcd 集群不健康 | 先修 etcd；NeoHA 会 demote |
| `rebuildme` / xtrabackup 失败 | SSH、磁盘、权限 | 见 [deployment.md §7](./deployment.md) 与 IT README |

---

## 5. 相关文档

| 文档 | 内容 |
|------|------|
| [deployment.md](./deployment.md) | 编译、三节点组网、neohactl |
| [ha-failover.md](./ha-failover.md) | semi-sync / MGR 时序图 |
| [config-design.md](./config-design.md) | 全量配置 schema |
| [test/integration/README.md](../test/integration/README.md) | 本地 IT、failover 段 baseline |
| [TODO.md](./TODO.md) | 路线图与已知限制 |
