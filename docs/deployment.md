# NeoHA 部署指南

> **状态：** 手工三节点部署流程（无 Ansible/Helm 一键包）。配置字段详见 [config-design.md](./config-design.md)；切换行为见 [ha-failover.md](./ha-failover.md)。

---

## 1. 部署拓扑概览

| 场景 | 协调层 | 数据库 | 示例配置 | 节点数 |
|------|--------|--------|----------|--------|
| MySQL 半同步 | 内嵌 **Raft**（P2P） | MySQL 8.0 semi-sync | [semisync-node1.yaml](../configs/examples/mysql/semisync-node1.yaml) | 建议 **3**（奇数 quorum） |
| MySQL MGR | 内嵌 **Raft** | MySQL 8.0 Group Replication | [mgr-mysql80-node1.yaml](../configs/examples/mysql/mgr-mysql80-node1.yaml) | **3** |
| PostgreSQL 流复制 | **etcd** DCS | PostgreSQL 14+ | [etcd-node1.yaml](../configs/examples/postgresql/etcd-node1.yaml) | **3** PG + **3** etcd |

**每节点进程：**

```
mysqld / postmaster  +  neoha agent  +  （可选）neohactl 运维
```

NeoHA 通过 **nrpc**（配置项 `endpoint`，如 `10.0.0.1:8080`）与 peer 通信；Raft 成员列表保存在各节点 `coordination.raft.meta-datadir` 下的 **`peers.json`**。

---

## 2. 前置条件

### 2.1 通用

- Linux（生产）或 WSL（开发/IT）
- Go 1.21+（源码编译时）
- 各节点 **时钟同步**（NTP）
- 节点间 **`endpoint` 端口** 互通（NeoHA RPC）
- 同一集群 **`scope` 相同**，**`name` 唯一**

### 2.2 MySQL

| 项 | 要求 |
|----|------|
| 版本 | **8.0**（开发与 IT 基准；5.6/5.7 未正式支持） |
| 半同步 | `rpl_semi_sync_master` / `rpl_semi_sync_slave` 插件 |
| MGR | `group_replication.so`；`group_replication_group_name` 全集群一致 |
| 复制用户 | `repl` 用户（配置 `database.mysql.repl-user` / `repl-passwd`） |
| GTID | 建议开启（选主与 failover 依赖 GTID 比较） |

### 2.3 PostgreSQL + etcd

| 项 | 要求 |
|----|------|
| PostgreSQL | 14+，已 `initdb`，配置 `listen` / `data_dir` |
| 复制 | 复制用户、`pg_hba` 允许 replication；建议 replication slot |
| etcd | 独立集群可达（`coordination.etcd.host` 或 `hosts`） |
| pg_rewind | 可选；`use_pg_rewind: true` 时 failover 后旧主 rewind |

---

## 3. 编译与安装

```bash
git clone https://github.com/sealdb/neoha.git
cd neoha
make build          # 产出 bin/neoha、bin/neohactl
sudo make install   # 默认 PREFIX=/usr/local → /usr/local/sbin/
```

或指定安装前缀：

```bash
make build
sudo make install PREFIX=/opt/neoha
# 二进制在 /opt/neoha/sbin/neoha、neohactl
```

---

## 4. 目录与配置布局（建议）

以三节点 MySQL 为例，**每节点**独立目录：

```
/etc/neoha/
  node1.yaml              # NeoHA 主配置
  config.path             # neohactl 用：内容为 node1.yaml 的绝对路径

/var/lib/neoha/
  meta/                   # coordination.raft.meta-datadir → peers.json
  logs/

/u01/mysql/               # database.mysql.basedir
  data/                   # mysqld datadir（由 defaults-file 指定）
```

`config.path` 示例（`neohactl` 在含此文件的目录下执行）：

```
/etc/neoha/node1.yaml
```

复制示例并修改：

```bash
sudo mkdir -p /etc/neoha /var/lib/neoha/meta
sudo cp configs/examples/mysql/semisync-node1.yaml /etc/neoha/node1.yaml
echo /etc/neoha/node1.yaml | sudo tee /etc/neoha/config.path
```

**每节点必改字段：**

| 字段 | 说明 |
|------|------|
| `name` | 成员名，集群内唯一，如 `db1` / `db2` / `db3` |
| `endpoint` | 本机 NeoHA RPC，如 `10.0.0.1:8080` |
| `coordination.raft.meta-datadir` | 本机 Raft 元数据目录 |
| `database.mysql.port` / `defaults-file` | 本机 mysqld |
| `database.mysql.repl-host` | 本机 IP（供对端 CHANGE MASTER） |

MGR 还需按节点调整 `database.mysql.slave-sysvars` 中的 `group_replication_local_address` 与 `group_replication_group_seeds`（见 [mgr 示例](../configs/examples/mysql/mgr-mysql80-node1.yaml)）。

---

## 5. 启动 NeoHA agent

```bash
neoha -config /etc/neoha/node1.yaml
# 或
neoha -c /etc/neoha/node1.yaml -log STD
```

可选 `-role LEADER|FOLLOWER|IDLE` 仅用于调试；生产由 Raft 自动选举。

**systemd 单元（示例）：**

```ini
[Unit]
Description=NeoHA agent
After=network.target mysqld.service

[Service]
Type=simple
ExecStart=/usr/local/sbin/neoha -config /etc/neoha/node1.yaml -log SYS
Restart=on-failure
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
```

三台机器上分别启动对应配置的 `neoha`（**先起 mysqld，再起 neoha**）。

---

## 6. 集群组网（Raft）

协调层为 **`coordination.provider: raft`** 时，需把各节点注册进 Raft 集群。

### 6.1 推荐：neohactl（与集成测试一致）

在 **每个节点** 的工作目录（含 `config.path`）：

```bash
cd /etc/neoha
neohactl raft enable
```

在 **当前或未来的 Raft LEADER 节点**（通常第一台）添加其余成员 endpoint：

```bash
# endpoint 列表为 NeoHA RPC 地址，逗号分隔，不含本机
neohactl cluster add 10.0.0.2:8080,10.0.0.3:8080
```

等价于 IT harness 中的 `WireNeoHAClusterViaCLI`（见 [test/integration/README.md](../test/integration/README.md)）。

### 6.2 手工：peers.json

也可在 LEADER 的 `meta-datadir/peers.json` 中写入成员列表（格式与运行时一致）；**仅建议在自动化或调试时使用**，日常运维优先 `neohactl cluster add`。

### 6.3 验证

```bash
neohactl neoha ping                    # 本节点 agent 存活
neohactl cluster status                # 集群视图
neohactl cluster raft                  # Raft 角色
neohactl cluster mysql                 # MySQL 复制状态
neohactl cluster gtid                  # GTID 比较（选主相关）
```

---

## 7. 生产推荐：`ha.delegate_db_apply`

MySQL HA **推荐**开启（与 [ha-failover.md](./ha-failover.md) 及当前 IT 一致）：

```yaml
ha:
    reconcile_interval: 3
    delegate_db_apply: true
    primary_hooks:
        on_primary_start: nop   # 可改为 VIP 脚本，见下
        on_primary_stop: nop
```

- **`delegate_db_apply: true`**：Raft 只负责选主；**Reconciler + Driver** 执行 promote/demote/CHANGE MASTER。
- **`false`**：旧路径，Raft handler 内直接改 MySQL（MGR 两阶段 promote 等仍部分耦合在 raft 包）。

---

## 8. 按场景部署要点

### 8.1 MySQL 半同步（3 节点 + Raft）

1. 三台 mysqld：GTID、半同步插件、`repl` 用户。
2. 三台 NeoHA：复制 [semisync-node1.yaml](../configs/examples/mysql/semisync-node1.yaml)，改 `name` / `endpoint` / 端口 / `meta-datadir`。
3. 调优 Raft 心跳（影响故障判定时间，见 [TODO.md](./TODO.md) IT baseline）：

   | 参数 | 生产常见 | 说明 |
   |------|----------|------|
   | `heartbeat-timeout` | 1000–2000 ms | Leader 心跳间隔 |
   | `admit-defeat-hearbeat-count` | 5–10 | 连续失败次数 |
   | `election-timeout` | ≥ 心跳×次数 | Follower 判主 down |

4. `neohactl raft enable` + `cluster add` 组网。
5. 等待 Raft 选出 LEADER，Reconciler 将 LEADER 节点提升为 MySQL 主库。

切换细节：[ha-failover.md § 半同步](./ha-failover.md)。

### 8.2 MySQL MGR（3 节点 + Raft）

1. 三台 mysqld 8.0 + GR 插件；**同一** `group_replication_group_name`。
2. 每节点 `replication-mode: group-replication`；`master-sysvars` / `slave-sysvars` 配置 GR 地址与 seeds（**每节点 local_address 不同**）。
3. 建议 `coordination.raft.purge-binlog-disabled: true`（MGR 场景通常不由 NeoHA purge binlog）。
4. 组网、验证同 §6；`delegate_db_apply: true` 时 promote 走 Driver 两阶段（read-only PRIMARY → 开放写）。

切换细节：[ha-failover.md § MGR](./ha-failover.md)（含多数派丢失 sole-survivor）。

### 8.3 PostgreSQL + etcd

1. 部署 **etcd 集群**（与 NeoHA 分离）。
2. 三台 PostgreSQL：流复制、`replication` 用户、slot（可选）。
3. 每节点 NeoHA：`coordination.provider: etcd`，填写 `coordination.etcd.host`（或 `hosts`）、`ttl`。
4. `database.type: postgresql`；`ha.delegate_db_apply: true`（示例已开启）。
5. **无需** `neohactl cluster add`；成员与 leader 由 etcd 注册（Patroni 风格 lease）。

示例：[etcd-node1.yaml](../configs/examples/postgresql/etcd-node1.yaml)。

---

## 9. VIP 与 primary_hooks

历史配置使用 `coordination.raft.leader-start-command` / `leader-stop-command`（或 legacy `election.raft` 同名字段）。**目标**是迁移到：

```yaml
ha:
    primary_hooks:
        on_primary_start: /etc/neoha/vip-up.sh
        on_primary_stop: /etc/neoha/vip-down.sh
```

也可用交互式生成（半同步场景）：

```bash
neohactl init --address=<本机IP> --port=8080 --vip=<VIP> --repluser=repl --replpwd=...
```

脚本需 idempotent；切换时由 Reconciler 在 promote/demote 路径触发（见 `internal/ha/primary_hook.go`）。

---

## 10. 常用运维命令

| 命令 | 用途 |
|------|------|
| `neohactl neoha ping` | 探测本节点 agent |
| `neohactl cluster status` | 集群总览 |
| `neohactl raft trytoleader` | 手动抢主（受 `tags.nofailover` 等约束） |
| `neohactl mysql status` | 本节点 MySQL 状态 JSON |
| `neohactl mysql rebuildme --from=<leader-endpoint> --force` | 从指定主重建从库（xtrabackup 路径） |
| `neohactl cluster add` / `remove` | 扩缩 Raft 成员 |

**标签（`tags`）：**

| Tag | 效果 |
|-----|------|
| `nofailover: true` | 不参与升主 |
| `nosync: true` | PG 非同步副本（MySQL semi-sync 行为 TBD） |

---

## 11. 部署检查清单

- [ ] 三节点 `scope` 一致，`name` / `endpoint` 互不冲突
- [ ] mysqld 已运行且 `repl` 用户可用
- [ ] `coordination.raft.meta-datadir` 可写，`peers.json` 含全部成员
- [ ] `neohactl neoha ping` 在三节点均成功
- [ ] `neohactl cluster raft` 显示唯一 LEADER
- [ ] MySQL：`cluster mysql` / `read_only` 与预期一致；MGR 时 `performance_schema.replication_group_members` 为 ONLINE
- [ ] PG+etcd：`coordination.provider=etcd` 且 etcd 健康；lag 低于 `maximum_lag_on_failover`
- [ ] 生产已设 `ha.delegate_db_apply: true`（MySQL）
- [ ] 防火墙放行 NeoHA RPC、MySQL/PG 端口、MGR `group_replication_local_address`

---

## 12. 故障排查（简要）

| 现象 | 方向 |
|------|------|
| 无法选主 | 检查 `peers.json`、`endpoint` 连通、`cluster raft` |
| LEADER 但 MySQL 只读 | `delegate_db_apply` 与 Reconciler 日志；MGR 是否等待 quorum |
| semi-sync 切换慢 | 增大/减小 `heartbeat-timeout` 与 `admit-defeat-hearbeat-count` |
| `neohactl` 找不到配置 | 当前目录需有 `config.path` 指向 yaml |
| PG failover 失败 | etcd lease、复制 lag、slot、`pg_rewind` 是否安装 |

深入 failover 时序见 [ha-failover.md](./ha-failover.md)。

---

## 13. 相关文档

| 文档 | 内容 |
|------|------|
| [config-design.md](./config-design.md) | 配置 schema、校验、`election.*` 迁移 |
| [architecture.md](./architecture.md) | L1–L5 分层与接口 |
| [ha-failover.md](./ha-failover.md) | MySQL 切换逻辑与时序图 |
| [configs/README.md](../configs/README.md) | 示例 YAML 与字段注释 |
| [test/integration/README.md](../test/integration/README.md) | 集成测试环境（可对照验证部署） |
| [TODO.md](./TODO.md) | 路线图与 IT 耗时 baseline |

---

## 变更记录

| 日期 | 说明 |
|------|------|
| 2026-06-29 | 初版：编译安装、三场景组网、neohactl、delegate_db_apply、检查清单 |
