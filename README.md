[![Github Actions Status](https://github.com/sealdb/neoha/workflows/NeoHA%20Build/badge.svg?event=push)](https://github.com/sealdb/neoha/actions?query=workflow%3A%22NeoHA+Build%22+event%3Apush)
[![Github Actions Status](https://github.com/sealdb/neoha/workflows/NeoHA%20Test/badge.svg?event=push)](https://github.com/sealdb/neoha/actions?query=workflow%3A%22NeoHA+Test%22+event%3Apush)
[![Github Actions Status](https://github.com/sealdb/neoha/workflows/NeoHA%20Coverage/badge.svg)](https://github.com/sealdb/neoha/actions?query=workflow%3A%22NeoHA+Coverage%22)
[![Go Report Card](https://goreportcard.com/badge/github.com/sealdb/neoha)](https://goreportcard.com/report/github.com/sealdb/neoha)
[![codecov](https://codecov.io/gh/sealdb/neoha/branch/main/graph/badge.svg?token=XPLUHW3DU3)](https://codecov.io/gh/sealdb/neoha)

# NeoHA

NeoHA is a template for MySQL and PostgreSQL High Availability with Etcd, Consul, ZooKeeper, or Kubernetes written in Golang, inspired by [zalando/patroni](https://github.com/zalando/patroni) and [radondb/xenon](https://github.com/radondb/xenon).

# Unit Tests

depend:

- watch:
  - On Ubuntu/CentOS: built-in system
  - On MacOS: install command is `brew install watch`


# Todo List

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

- [ ] cli
  - 借鉴xenoncli，不同在于要添加前缀指令，比如 neohacli mysql cluster gtid
- [ ] ctl
  - 借鉴 xenon ctl，也要添加前缀
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
- [ ] 编写集成测试
