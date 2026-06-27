[![NeoHA Build](https://github.com/sealdb/neoha/actions/workflows/neoha-build.yml/badge.svg?branch=main)](https://github.com/sealdb/neoha/actions/workflows/neoha-build.yml)
[![NeoHA Test](https://github.com/sealdb/neoha/actions/workflows/neoha-test.yml/badge.svg?branch=main)](https://github.com/sealdb/neoha/actions/workflows/neoha-test.yml)
[![NeoHA Coverage](https://github.com/sealdb/neoha/actions/workflows/neoha-coverage.yml/badge.svg?branch=main)](https://github.com/sealdb/neoha/actions/workflows/neoha-coverage.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/sealdb/neoha.svg)](https://pkg.go.dev/github.com/sealdb/neoha)
[![codecov](https://codecov.io/gh/sealdb/neoha/branch/main/graph/badge.svg)](https://codecov.io/gh/sealdb/neoha)

# NeoHA

NeoHA is a template for MySQL and PostgreSQL High Availability with Etcd, Consul, ZooKeeper, or Kubernetes written in Golang, inspired by [zalando/patroni](https://github.com/zalando/patroni) and [radondb/xenon](https://github.com/radondb/xenon).

## Authors

Copyright 2022-2026 [The NeoHA Authors](AUTHORS). See [AUTHORS](AUTHORS) for contributor details.

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE).

# Unit Tests

```bash
make test      # unit tests (~5 min; raft package is the slowest)
make coverage  # unit-test coverage (~8–15 min; raft alone ~5–10 min with -covermode=atomic)
```

CI runs `make build`, `make test`, and `make coverage-ci` on push/PR to `main` (see [.github/workflows/](.github/workflows/)).

# Integration Tests

Requires MySQL 8.0 debug build, optional Xtrabackup + SSH for backup tests. See [test/integration/README.md](test/integration/README.md).

```bash
cp test/integration/it.local.yaml.example test/integration/it.local.yaml
make test-integration
```

# Todo List

详细优先级与架构重构备忘见 [docs/TODO.md](docs/TODO.md)（当前：**单元测试全绿**、**集成测试提速 ~20s 选主**，Raft 重构暂缓）。

对于 MySQL 的本地事务：

> 腾讯 TDSQL 对本地事务，如果是 DML，就用工具闪回；如果是 DDL，就只能重建从库。

TODO:

1. 管理所有关系型单机数据库，比如 MySQL、PG、Oracle
2. EverDB 正在使用类似 radon 的分布式中间件来做单个 MySQL 集群的高可用，想用 xenon 做单机 MySQL 的高可用
3. 慢日志分析展示（将 pt-xxxx 分析结果的前 N 条缓存起来 ）
4. 选举条件支持比较 GTID 和 binlog position 两种方式。
5. 主从节点 IO hang 或者 主 hang 从不 hang 的情况
6. ifdown 网卡，短时间内 ifup 拉起来？目前 Plus 是靠外围 deploy 脚本来做（其实没有）
7. 从库主动添加本地事务，没有被标记 INVALID 状态？
8. 删除 MySQL 数据目录的情况，貌似 MySQL 依然可以操作写事务？
9. 整合 mysql 的 driver ：go-mysql-driver、go-mysqlstack，
    pg 的 driver ：https://github.com/jackc/pgx
    其中 PG 的 driver 项目 star 很高，并且其中各个子mod可以独立使用

- 改进点：
  - 功能层面：很多功能原计划做在云平台层，实际上做到该组件中会更好一些
    - 慢日志分析：
    - binlog 分析：比如快速找到某条 GTID 对应的事务信息
    - 备份恢复：
    - 闪回：只能闪回 DML，不能闪回 DDL
  - 代码架构层面：
    - 将 raft 层抽象出来，那么就不仅仅能支持 mysql，凡是具有同样特点的高可用组件，都可以嵌入进去


- 基础代码
  - [ ] 支持 MySQL 5.6、5.7、8.0
  - [ ] 支持 Raft
  - [ ] 支持 Semi-sync、MGR
  - [ ] 半同步模式支持哨兵，即将其中一个节点设为仅具备投票权，不复制数据，也不发起选举
  - [ ] 为了避免切换时旧主产生本地事务，需要注意两点：
    - 内核层或工具层支持 truncate binlog 功能：避免旧主mysqldown导致的脏事务
    - 旧主不要关闭semi_sync_master，而是先 kill 处于 `Waiting for semi-sync ACK from slave`状态的SQL，之后才能关闭半同步、change master to新主、start slave
      - 试了下，kill 不掉；估计需要修改内核代码，以解决这个问题。比如添加 kill force 语法
```sql
mysql> show processlist;
+----+-----------------+-----------------+--------------------+---------+------+--------------------------------------+---------------------+
| Id | User            | Host            | db                 | Command | Time | State                                | Info                |
+----+-----------------+-----------------+--------------------+---------+------+--------------------------------------+---------------------+
|  5 | event_scheduler | localhost       | NULL               | Daemon  | 6865 | Waiting on empty queue               | NULL                |
| 11 | root            | localhost:60716 | performance_schema | Killed  | 6221 | Waiting for semi-sync ACK from slave | create database db4 |
| 12 | root            | localhost:48762 | NULL               | Killed  | 5699 | waiting for handler commit           | create database db5 |
| 13 | root            | localhost:34868 | NULL               | Killed  | 4990 | waiting for handler commit           | create database db6 |
| 14 | root            | localhost:58780 | mysql              | Query   |    0 | init                                 | show processlist    |
| 15 | root            | localhost:52360 | performance_schema | Query   |  225 | Waiting for schema metadata lock     | create database db4 |
| 16 | root            | localhost:48172 | NULL               | Query   |  172 | Waiting for schema metadata lock     | create database db6 |
| 17 | root            | localhost:58266 | NULL               | Query   |    9 | Waiting for schema metadata lock     | create database db5 |
+----+-----------------+-----------------+--------------------+---------+------+--------------------------------------+---------------------+
8 rows in set (0.00 sec)
```

- [ ] 重构测试代码，即将重复代码封装成函数，放在 *_test.go或mock.go文件中
- [ ] 支持 PostgreSQL
- [ ] 支持其他选举工具
  - [ ] 支持 etcd
  - [ ] 支持 Consul
  - [ ] 支持 ZooKeeper
  - [ ] 支持 Kubernetes

- [ ] 支持一键部署

- [ ] 编写文档
  - 部署手册
  - 运维手册
  - 原理
- [x] 编写集成测试（见 [test/integration/README.md](test/integration/README.md)）
