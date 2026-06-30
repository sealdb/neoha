# NeoHA Configuration Design

> Target schema for the **full product vision**. Current code still uses `election.*`; see [Migration from today](#migration-from-today).
>
> Architecture context: [architecture.md](./architecture.md)

---

## 1. Design principles

| Principle | Meaning |
|-----------|---------|
| **One product, two engines** | `database.type` selects MySQL or PostgreSQL; the rest of the tree shares the same shell and coordination model. |
| **Provider is orthogonal to engine** | Any supported `coordination.provider` may pair with MySQL or PG (subject to implementation maturity and validation). |
| **Active subtree only** | Only the block matching `coordination.provider` and `database.type` is required; other blocks are ignored or warned. |
| **Bootstrap ≠ runtime** | First-init settings live under `bootstrap.*`; failover/day-2 ops do not re-read bootstrap. |
| **Inspired, not compatible** | Field names may resemble Patroni or Xenon; NeoHA does **not** guarantee drop-in config or API compatibility. |
| **Version selects handler** | `database.mysql.version` / `database.postgresql.version` pick SQL/dialect handlers inside the driver. |

---

## 2. Top-level layout

```yaml
scope: mycluster          # cluster name (Patroni-style identity)
name: node1               # this member name
namespace: /service/      # coordination key prefix when provider uses a KV store

endpoint: 10.0.0.1:8080   # NeoHA RPC (nrpc); required for raft / member registration
peer-address: :6060      # optional HTTP (REST / metrics); TBD final API surface

restapi: { ... }         # L1 shell
ctl: { ... }
log: { ... }
watchdog: { ... }
tags: { ... }

coordination: { ... }    # L2 — target name (today: election)
bootstrap: { ... }       # first init only
database: { ... }        # L4 driver config
manager: { ... }         # optional process/backup ops (Xenon-style); TBD unify with database.*
```

---

## 3. L0 — Cluster identity

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `scope` | string | yes | Logical cluster id. |
| `name` | string | yes | Member name within scope (unique per cluster). |
| `namespace` | string | when provider=etcd/consul/k8s | Prefix for coordination keys. Default `/service/`. |
| `endpoint` | string | yes (raft) | `host:port` for NeoHA P2P RPC. |
| `peer-address` | string | no | HTTP listen address for REST. |

### Tags (`tags`)

Patroni-inspired member tags; reconcile honours them before promoting.

| Tag | Default | Effect |
|-----|---------|--------|
| `nofailover` | false | Node will not become primary. |
| `noloadbalance` | false | Exclude from read pool (when load balancer integration exists). TBD |
| `clonefrom` | false | Prefer as backup clone source. TBD |
| `nosync` | false | PG: non-synchronous replica; MySQL: TBD semi-sync behaviour |

---

## 4. L1 — Shell

### `restapi`

| Key | Description |
|-----|-------------|
| `enable-apis` | Enable HTTP API. |
| `listen` | Bind address. |
| `connect_address` | Address advertised to clients/peers. |
| `certfile` / `keyfile` | TLS. |
| `authentication.username` / `password` | Basic auth. |

### `ctl`

Client TLS settings for `neohactl` → NeoHA RPC.

### `log`

| Key | Values |
|-----|--------|
| `level` | DEBUG, INFO, WARNING, ERROR, PANIC |

### `watchdog`

Linux hardware watchdog integration. TBD: interaction with demote on timeout.

---

## 5. L2 — Coordination (`coordination`)

> **Today:** `election.algorithm` + `election.raft` / `election.etcd`

```yaml
coordination:
  provider: raft    # raft | etcd | consul | kubernetes | zookeeper
  raft: { ... }     # active when provider=raft
  etcd: { ... }     # active when provider=etcd
  consul: { ... }   # TBD
  kubernetes: { ... }  # TBD
  zookeeper: { ... }   # TBD
```

### Validation rules

- Exactly one `provider` must be set and recognised.
- Subtree for **non-selected** providers: optional; if present → log warning in v0.1.1, error in v1.0.
- `provider=raft` requires `endpoint`, `coordination.raft.meta-datadir`, heartbeat/election timers.
- `provider=etcd` requires non-empty `hosts` or `host`; must **not** require raft peers file for membership (members come from DCS).

### `coordination.raft` (Xenon-inspired)

| Key | Unit | Description |
|-----|------|-------------|
| `meta-datadir` | path | Peers, view/epoch metadata (`peers.json`). |
| `heartbeat-timeout` | ms | Leader → followers heartbeat period. |
| `admit-defeat-hearbeat-count` | count | Missed heartbeats before leader steps down. |
| `election-timeout` | ms | Follower election timer base. |
| `purge-binlog-interval` | ms | MySQL primary binlog purge cadence. |
| `purge-binlog-disabled` | bool | Skip purge. |
| `super-idle` | bool | Super-IDLE HA mode (Xenon). |
| `leader-start-command` | shell | Hook after becoming raft leader. |
| `leader-stop-command` | shell | Hook after losing raft leadership. |
| `requesttimeout` | ms | Inter-node RPC timeout. |
| `candidate-wait-for-2nodes` | ms | Two-node cluster candidate backoff. |

**Note:** Raft timers are **coordination** semantics. Do not duplicate them under `database.mysql` except where Xenon historically split “MySQL ping defeat” (see mysql `ping-timeout`) — reconcile layer should document the combined fault-detection window.

### `coordination.etcd` (Patroni-inspired layout)

| Key | Description |
|-----|-------------|
| `host` | Single discovery endpoint. |
| `hosts` | Endpoint list. |
| `use_proxies` | Discovery vs advertised client URLs. |

Additional keys TBD: `ttl`, `loop_wait`, `retry_timeout` — these belong here, **not** under `bootstrap.postgresql`.

### Other providers

| Provider | Status | Notes |
|----------|--------|-------|
| `consul` | TBD | Session lock + KV layout TBD |
| `kubernetes` | TBD | Lease on Endpoints / custom CRD TBD |
| `zookeeper` | TBD | May defer indefinitely |

---

## 6. L4 — Database (`database`)

```yaml
database:
  type: mysql          # mysql | postgresql
  mysql: { ... }       # required when type=mysql
  postgresql: { ... }  # required when type=postgresql
```

### 6.1 MySQL (`database.mysql`)

| Key | Description |
|-----|-------------|
| `version` | `mysql56` \| `mysql57` \| `mysql80` — handler selection |
| `replication-mode` | `semi-sync` \| `mgr` |
| `host` / `port` | Admin connection |
| `admin` / `passwd` | Admin credentials |
| `basedir` / `defaults-file` | Instance paths |
| `repl-host` / `repl-user` / `repl-passwd` | Replication account |
| `repl-gtid-purged` | Bootstrap GTID state |
| `ping-timeout` | ms between admin pings |
| `admit-defeat-ping-count` | Failed pings before `MysqlDead` |
| `failover-on-too-many-connections` | Treat 1040 as fatal for primary |
| `semi-sync-timeout-for-two-nodes` | 2-node semi-sync special case |
| `master-sysvars` / `slave-sysvars` | Semicolon-separated SET on role change |
| `monitor-disabled` | Disable mysqld monitor RPC path |
| `max-open-conns` / `max-idle-conns` | Admin pool |
| `backup` | Xtrabackup/SSH settings (may move to `manager` — TBD) |

#### Mode-specific blocks (target v0.1.2)

```yaml
database:
  mysql:
    replication-mode: mgr
    mgr:
      group-name: ""           # TBD
      group-seeds: []          # TBD
      single-primary-mode: true # TBD
    semi-sync:
      wait-slave-count: 1      # TBD
```

Today these are flat keys; splitting reduces cross-mode confusion.

### 6.2 PostgreSQL (`database.postgresql`)

NeoHA-native PG driver config (Patroni-**inspired** grouping, not Patroni schema).

| Key | Description |
|-----|-------------|
| `version` | `postgresql14`, `postgresql15`, `postgresql16`, … |
| `listen` | Local PG address |
| `connect_address` | Advertised address for replication |
| `data_dir` | PGDATA |
| `bin_dir` / `config_dir` | Binaries and config paths |
| `pgpass` | Optional pgpass file |
| `authentication.replication` | Repl user |
| `authentication.superuser` | Admin user |
| `authentication.rewind` | pg_rewind user |
| `parameters` | Dynamic GUC map applied on promote/reload |
| `pre_promote` | Shell hook before promote |
| `krbsrvname` | Kerberos service name |

#### Failover policy (PG-specific, stays under `database.postgresql`)

| Key | Description |
|-----|-------------|
| `maximum_lag_on_failover` | Max bytes behind primary to allow promote |
| `use_pg_rewind` | Attempt rewind on rejoin |
| `use_slots` | Manage replication slots |
| `synchronous_mode` | Require sync replica for commit (TBD exact semantics) |
| `synchronous_mode_strict` | TBD |
| `primary_slot_name` | Slot on primary when this node is replica |

Do **not** conflate these with `coordination.etcd.ttl`; they are **eligibility** rules for the PG driver/evaluator.

---

## 7. Bootstrap (`bootstrap`)

Used **once** when initializing an empty data directory.

### `bootstrap.postgresql`

| Key | Description |
|-----|-------------|
| `initdb.encoding` / `data_checksums` | initdb options |
| `pg_hba` | Initial HBA lines |
| `post_init` | SQL/shell after init |
| `users` | Additional roles |

`bootstrap.postgresql.dcs` in current `neoha-full.yaml` mixes Patroni DCS defaults with PG — **target:** move runtime failover policy to `database.postgresql.*`; keep bootstrap minimal. TBD exact split.

### `bootstrap.mysql`

TBD: repl user creation, GTID seed, MGR bootstrap SQL. Today empty struct.

---

## 8. Manager (`manager`) — TBD

Optional Xenon-style process management. Current code: `internal/manager/mysqld`.

```yaml
manager:
  mysqld: { ... }      # start/stop/kill mysqld
  postmaster: { ... }  # TBD
  backup: { ... }      # may alias database.mysql.backup
```

**Open:** single `manager` tree vs keeping backup under `database.mysql.backup`.

---

## 9. Example profiles

### MySQL semi-sync + embedded Raft (primary v0.1 path)

See `configs/examples/mysql/semisync-node1.yaml`.

### MySQL MGR + embedded Raft

See `configs/examples/mysql/mgr-mysql80-node1.yaml`.

### PostgreSQL + embedded Raft (target)

```yaml
scope: pgcluster
name: pg1
endpoint: 10.0.0.1:8080

coordination:
  provider: raft
  raft:
    meta-datadir: /var/lib/neoha/raft
    heartbeat-timeout: 2000
    admit-defeat-hearbeat-count: 5
    election-timeout: 10000

database:
  type: postgresql
  postgresql:
    version: postgresql16
    listen: 127.0.0.1:5432
    connect_address: 10.0.0.1:5432
    data_dir: /var/lib/postgresql/16/main
    maximum_lag_on_failover: 1048576
    use_pg_rewind: true
    use_slots: true
    authentication:
      replication: { username: repl, password: repl }
      superuser: { username: postgres, password: "" }
```

### PostgreSQL + etcd (target)

Same as above with `coordination.provider: etcd` and populated `coordination.etcd.hosts`. PG driver/evaluator unchanged.

---

## 10. Config loading & validation pipeline (target)

```
Read file → detect format (yaml/json) → parse → defaults merge
    → semantic validate (provider + type + mode)
    → optional env override (TBD)
    → immutable Config snapshot passed to Server
```

Validation examples:

| Condition | Result |
|-----------|--------|
| `database.type=mysql`, `replication-mode=mgr`, missing MGR plugin assumptions | warn/fail at SetupBootstrap |
| `tags.nofailover=true` + manual `trytoleader` | CLI error |
| `provider=etcd`, empty hosts | fail fast at start |
| `database.type=postgresql`, `provider=raft` | allowed (v0.1.3+) |

---

## 11. Migration from today

| Current (`v0.1`) | Target | Notes |
|------------------|--------|-------|
| `election.algorithm` | `coordination.provider` | Accept alias until v1.0 |
| `election.raft` | `coordination.raft` | Field names unchanged |
| `election.etcd` | `coordination.etcd` | |
| `database.mysql.*` | same | Add nested `mgr` / `semi-sync` later |
| `bootstrap.postgresql.dcs.*` | `database.postgresql.*` (failover policy) | Deprecate dcs nesting |
| `GetMysql()` in raft | `database.Driver` | See `internal/database/driver.go` |

---

## 12. Open questions (config)

- [ ] Final REST API path layout vs `peer-address` and `restapi.listen`
- [ ] Whether `manager` is top-level or nested under `database`
- [ ] Consul/K8s/ZK key namespace conventions
- [ ] Env var prefix (`NEOHA_`) mapping table
- [ ] Hot reload: which keys are mutable without restart
- [ ] DCS dynamic cluster settings JSON schema (shared across MySQL/PG)
- [ ] Patroni-compatible etcd key layout as **optional** compatibility mode?

---

## 13. Related documents

- [architecture.md](./architecture.md) — runtime layers, interfaces, failover flows
- [TODO.md](./TODO.md) — phased delivery tracker
- `configs/examples/neoha-full.yaml` — exhaustive key list (today's shape)
