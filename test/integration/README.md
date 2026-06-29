# NeoHA Integration Tests

Integration tests live here and are **not** run by default `make test`.
They use the Go `integration` build tag.

## Design

| Layer | Responsibility |
|-------|----------------|
| **Harness (thin)** | Process lifecycle only: init datadir, `my.cnf`, start/stop `mysqld`, start/stop `neoha` binary, ports, cleanup |
| **neorpc / neohactl** | Cluster ops via `neohactl` binary (same RPC as production); assertions via read-only SQL |
| **Read-only SQL** | Assert MGR / semi-sync state after NeoHA has configured MySQL |

NeoHA behaviour (repl user, MGR bootstrap, Raft election) is exercised through **agent startup + neorpc**, not duplicate SQL in the harness.

### Raft timers in integration tests

| Mode | `heartbeat-timeout` | `admit-defeat-hearbeat-count` | Primary fault detection |
|------|---------------------|-------------------------------|-------------------------|
| **Semi-sync** | **2000 ms** (2 s) | **5** | **~10 s** (5 consecutive failed heartbeats) |
| **MGR** | 500 ms | 5 | Faster bootstrap/election in IT only |

Semi-sync IT follows Xenon-style timing: leader heartbeats every 2 s; after 5 rounds without enough acks (or followers miss leader heartbeats for ~`election-timeout` 10 s), failover should complete within roughly **10–15 s** once the cluster is already up. MGR tests keep shorter timers for quicker cluster formation.

Constants live in `harness/neoha.go` (`semiSyncIT*` / `mgrIT*`).

## Prerequisites

- MySQL 8.0 debug build with `group_replication` plugin (MGR tests) and semi-sync plugins (`semisync_master.so`, `semisync_slave.so`)
- **Xtrabackup 8.0** (`xtrabackup`, `xbstream`) for backup tests
- **SSH to localhost** (key-based or `sshpass` + password) for xtrabackup stream

### Tool paths (config file → env → PATH → default)

Copy `test/integration/it.local.yaml.example` to `test/integration/it.local.yaml`, or export env vars:

| Setting | Env var | Default |
|---------|---------|---------|
| MySQL base | `NEOHA_IT_MYSQL_BASE` | `/home/wslu/work/mysql/mysql80-debug` |
| Xtrabackup bin dir | `NEOHA_IT_XTRABACKUP_BINDIR` | `/home/wslu/work/mysql/xtrabackup-8.0.35` or `PATH` |
| IT config file | `NEOHA_IT_CONFIG` | auto: `test/integration/it.local.yaml` |
| Work dir | `NEOHA_IT_WORKDIR` | `$TMPDIR/neoha-it` |
| SSH port | `NEOHA_IT_SSH_PORT` | `22` (example uses `2222` for WSL) |

```bash
cp test/integration/it.local.yaml.example test/integration/it.local.yaml
# edit mysql-base, xtrabackup-bindir, ssh-port (e.g. 2222), ssh-user as needed
```

## Run

### Quick start

```bash
# 1. Local config (gitignored)
cp test/integration/it.local.yaml.example test/integration/it.local.yaml
# Edit mysql-base, xtrabackup-bindir, ssh-port (WSL often 2222), ssh-user if needed

# 2. Full integration suite (~6–8 min after datadirs exist; global timeout 15m)
make test-integration

# One or more tests (comma- or space-separated names match go test -run regex)
make test-integration TestNeoHASemiSyncWarmSuite
make test-integration TestNeoHASemiSyncWarmSuite,TestNeoHAMGRWarmSuite
make test-integration TestNeoHAMGRWarmSuite TestNeoHAMGRFailoverMajorityLoss

# Same via TESTS= (useful in scripts)
make test-integration TESTS=TestNeoHASemiSyncWarmSuite,TestNeoHAMGRWarmSuite

# Subtest (regex)
make test-integration 'TestNeoHAMGRFailoverMajorityLoss/RejoinThenWritable'
```

`make test-integration` pre-builds `bin/neoha-it.test`, then runs all tests with a **15-minute** global timeout (`-test.timeout=15m`). It passes through `NEOHA_IT_MYSQL_BASE` and `NEOHA_IT_XTRABACKUP_BINDIR` when set; otherwise it uses the defaults shown in the Makefile.

Pre-build `neoha` / `neohactl` to avoid implicit compile during IT:

```bash
go build -o bin/neoha ./cmd/neoha
go build -o bin/neohactl ./cmd/neohactl
export NEOHA_IT_BIN=$PWD/bin/neoha NEOHA_IT_CTL_BIN=$PWD/bin/neohactl
make test-integration
```

### Environment variables

| Variable | Purpose |
|----------|---------|
| `NEOHA_IT_MYSQL_BASE` | MySQL 8.0 build root (`bin/mysqld`) |
| `NEOHA_IT_XTRABACKUP_BINDIR` | Directory with `xtrabackup` and `xbstream` |
| `NEOHA_IT_CONFIG` | Override IT config file path |
| `NEOHA_IT_WORKDIR` | Artifact directory (default `$TMPDIR/neoha-it`) |
| `NEOHA_IT_BIN` / `NEOHA_IT_CTL_BIN` | Pre-built `neoha` / `neohactl` (else built under `workdir/bin`) |
| `NEOHA_IT_SSH_PORT` | SSH port when not using `it.local.yaml` |

Example:

```bash
export NEOHA_IT_MYSQL_BASE=/home/wslu/work/mysql/mysql80-debug
export NEOHA_IT_XTRABACKUP_BINDIR=/home/wslu/work/mysql/xtrabackup-8.0.35
export NEOHA_IT_WORKDIR=/tmp/neoha-it
make test-integration
```

### `go test` (single test or custom flags)

Build tag **`integration`** is required.

```bash
# All integration tests (match Makefile global timeout)
go test -tags=integration -v -timeout=15m -count=1 ./test/integration

# MGR warm cluster (formation + minority failover subtests)
go test -tags=integration -v -timeout=15m -count=1 -run TestNeoHAMGRWarmSuite ./test/integration

# MGR majority loss (sole survivor + rejoin)
go test -tags=integration -v -timeout=15m -count=1 -run TestNeoHAMGRFailoverMajorityLoss ./test/integration

# Semi-sync warm cluster
go test -tags=integration -v -timeout=15m -count=1 -run TestNeoHASemiSyncWarmSuite ./test/integration

# Xtrabackup rebuildme (requires SSH + xtrabackup; see below)
go test -tags=integration -v -timeout=15m -count=1 -run TestNeoHAXtrabackupRebuildMe ./test/integration
```

Pre-compile to avoid a long silent compile phase:

```bash
go test -tags=integration -c -o bin/neoha-it.test ./test/integration
NEOHA_IT_MYSQL_BASE=... NEOHA_IT_XTRABACKUP_BINDIR=... \
  NEOHA_IT_BIN=$PWD/bin/neoha NEOHA_IT_CTL_BIN=$PWD/bin/neohactl \
  bin/neoha-it.test -test.v -test.timeout=15m -test.count=1
```

### Xtrabackup / `rebuildme` test

`TestNeoHAXtrabackupRebuildMe` exercises the production path:

1. Two-node semi-sync cluster (MySQL `13326`/`13327`, NeoHA Raft `18131`/`18132`)
2. Data sync to replica, then `neohactl mysql rebuildme --from=<leader> --force` on the follower
3. Assert replication and row data after rebuild

**Extra prerequisites:**

- `xtrabackup` and `xbstream` on PATH or under `xtrabackup-bindir`
- SSH to localhost for the `xtrabackup | ssh … xbstream` pipeline (key-based or `sshpass`)

**WSL example** (`it.local.yaml`):

```yaml
ssh-host: 127.0.0.1
ssh-port: 2222
ssh-user: ""          # defaults to $USER
ssh-passwd: ""        # empty = key auth
```

Verify SSH before running:

```bash
ssh -p 2222 -o BatchMode=yes $(whoami)@127.0.0.1 echo ok
```

Run only the rebuildme test:

```bash
go test -tags=integration -v -timeout=15m -count=1 \
  -run TestNeoHAXtrabackupRebuildMe ./test/integration
```

### Ports used by tests

| Scenario | MySQL ports | GR ports | NeoHA Raft ports |
|----------|-------------|----------|------------------|
| MGR warm (`TestNeoHAMGRWarmSuite`) | 13306–13308 | 13361–13363 | 18081–18083 |
| MGR majority loss (`TestNeoHAMGRFailoverMajorityLoss`) | 13326–13328 | 13381–13383 | 18101–18103 |
| Semi-sync warm | 13316–13318 | — | 18111–18113 |
| Xtrabackup rebuildme | 13326–13327 | — | 18131–18132 |
| PostgreSQL | 15432–15433 | — | etcd / agent per test |

Majority-loss and xtrabackup both use MySQL `13326–13327`; tests run sequentially and tear down workdirs between suites. If a run aborts mid-test, free ports or use a fresh `NEOHA_IT_WORKDIR`.

### First run vs later runs

The harness **initializes MySQL datadirs** on first use (roughly 20–30s per node, parallelized across nodes). Later runs **reuse** existing datadirs under `$NEOHA_IT_WORKDIR/<cluster-name>/` when the `mysql` system schema is already present.

IT `my.cnf` puts `socket` and `pid-file` inside `datadir` so `rebuildme`’s datadir wipe does not leave stale socket files.

To force a clean slate:

```bash
rm -rf "${NEOHA_IT_WORKDIR:-/tmp/neoha-it}"
pkill -f 'defaults-file=/tmp/neoha-it' || true
```

## Scenarios

| Test | What it verifies |
|------|------------------|
| `TestMySQL3NodeScaffold` | 3 mysqld instances start with MGR plugin loaded |
| `TestNeoHAMGRWarmSuite` | Warm 3-node MGR: formation + minority failover (failover subtest times only the fault segment) |
| `TestNeoHAMGRFailoverMajorityLoss` | 2 mysqld down: sole-survivor read-only PRIMARY, then rejoin → 2+ ONLINE → writable |
| `TestPostgreSQLApplyReplicaPgRewind` | `ApplyReplica` with `pg_rewind` after promote |
| `TestPostgreSQL2NodeScaffold` | 2-node PG primary + streaming standby |
| `TestNeoHAPGEtcdFailoverMinority` | PG + etcd DCS: minority failover promote |
| `TestNeoHASemiSyncWarmSuite` | Warm 3-node semi-sync: formation + minority failover |
| `TestNeoHAXtrabackupRebuildMe` | 2-node semi-sync: `neohactl mysql rebuildme --from=<leader> --force` |

MGR / semi-sync warm suites pre-write `peers.json` and skip `neohactl raft enable` + `cluster add` (production CLI wire is still covered by the xtrabackup test).

## Repository layout

```
cmd/
  neoha/                 # neoha daemon main
  neohactl/              # CLI main
internal/                # application-private packages
  config/ server/ election/ database/ manager/ base/ neohactl/ neorpc/
api/                     # REST API for external callers (e.g. Kubernetes)
configs/examples/        # sample YAML (see configs/README.md)
test/integration/        # integration tests (build tag: integration)
build/                   # ldflags / build metadata
pkg/                     # reserved for future public libraries
```

## Layout

```
test/integration/
├── harness/
│   backend.go       # Cluster, Node, Backend interface
│   mysql80.go       # MySQL 8.0 start/stop
│   mysql_assert.go  # Read-only MGR assertions
│   neoha.go         # NeoHA config + process lifecycle
│   neorpc.go        # neorpc helpers (neohactl-equivalent)
├── mysql_warm_test.go   # warm fixture helpers (MGR / semi-sync)
├── mgr_3node_test.go
├── mgr_neoha_test.go
├── semisync_neoha_test.go
├── xtrabackup_neoha_test.go
├── postgresql_apply_replica_test.go
├── postgresql_etcd_neoha_test.go
├── it.local.yaml.example
└── README.md
```

## Troubleshooting

### `go test` appears stuck with no output

The first `go test` compiles the whole module and can take 30s+ with no logs. Prefer:

```bash
make test-integration
# or
go test -tags=integration -c -o bin/neoha-it.test ./test/integration && bin/neoha-it.test -test.v ...
```

### Stale processes / ports

```bash
pkill -f 'defaults-file=/tmp/neoha-it' || true
rm -rf "${NEOHA_IT_WORKDIR:-/tmp/neoha-it}/<cluster-name>"
```

Common ports: MGR warm `13306–13308`, MGR majority-loss `13326–13328`, semi-sync `13316–13318`, xtrabackup `13326–13327`.

### Xtrabackup test hangs on `rebuildme`

- Confirm SSH: `ssh -p <port> $(whoami)@127.0.0.1 echo ok`
- Confirm tools: `ls "$NEOHA_IT_XTRABACKUP_BINDIR"/xtrabackup "$NEOHA_IT_XTRABACKUP_BINDIR"/xbstream`
- Check NeoHA logs under `$NEOHA_IT_WORKDIR/<cluster>/neoha/*/neoha.log` for backup or socket errors
