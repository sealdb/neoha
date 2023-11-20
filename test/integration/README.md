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

## Prerequisites

- MySQL 8.0 debug build with `group_replication` plugin (MGR tests) and semi-sync plugins (`semisync_master.so`, `semisync_slave.so`)
- Default path: `/home/wslu/work/mysql/mysql80-debug`

## Run

```bash
export NEOHA_IT_MYSQL_BASE=/home/wslu/work/mysql/mysql80-debug
export NEOHA_IT_WORKDIR=/tmp/neoha-it   # optional
export NEOHA_IT_BIN=/path/to/neoha
export NEOHA_IT_CTL_BIN=/path/to/neohactl  # optional

make test-integration
```

## Scenarios

| Test | What it verifies |
|------|------------------|
| `TestMySQL3NodeScaffold` | 3 mysqld instances start with MGR plugin loaded |
| `TestNeoHA3NodeMGR` | 3 NeoHA agents → Raft leader → MGR 3 ONLINE via `setupMysql` |
| `TestNeoHAMGRFailoverMinority` | MGR primary loss + Raft leader survives |
| `TestNeoHAMGRFailoverMajorityLoss` | MGR quorum lost; NeoHA Raft bootstrap on sole survivor |
| `TestNeoHA3NodeSemiSync` | 3 NeoHA agents → Raft leader → semi-sync replicas running |
| `TestNeoHASemiSyncFailoverMinority` | Primary mysqld loss → Raft re-elects → replication re-wired |

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
├── mgr_3node_test.go
├── mgr_neoha_test.go
├── semisync_neoha_test.go
└── README.md
```

## Troubleshooting

### `go test` appears stuck with no output

Use `make test-integration` or precompile: `go test -tags=integration -c -o bin/neoha-it.test ./test/integration`.

### Stale processes / ports

Set `NEOHA_IT_WORKDIR` to a fresh directory or kill leftover mysqld on ports 13306–13308 (MGR) / 13316–13318 (semi-sync).
