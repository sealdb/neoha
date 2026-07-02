[![NeoHA Build](https://github.com/sealdb/neoha/actions/workflows/neoha-build.yml/badge.svg?branch=main)](https://github.com/sealdb/neoha/actions/workflows/neoha-build.yml)
[![NeoHA Test](https://github.com/sealdb/neoha/actions/workflows/neoha-test.yml/badge.svg?branch=main)](https://github.com/sealdb/neoha/actions/workflows/neoha-test.yml)
[![NeoHA Coverage](https://github.com/sealdb/neoha/actions/workflows/neoha-coverage.yml/badge.svg?branch=main)](https://github.com/sealdb/neoha/actions/workflows/neoha-coverage.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/sealdb/neoha.svg)](https://pkg.go.dev/github.com/sealdb/neoha)
[![codecov](https://codecov.io/gh/sealdb/neoha/branch/main/graph/badge.svg)](https://codecov.io/gh/sealdb/neoha)

---

# NeoHA

NeoHA is a template for MySQL and PostgreSQL High Availability with Etcd, Consul, ZooKeeper, or Kubernetes written in Golang, inspired by [zalando/patroni](https://github.com/zalando/patroni) and [radondb/xenon](https://github.com/radondb/xenon).

## Authors

Copyright 2022-2026 [The NeoHA Authors](AUTHORS). See [AUTHORS](AUTHORS) for contributor details.

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE).

## Documentation


| Topic                                                                                       | Document                                                 |
| ------------------------------------------------------------------------------------------- | -------------------------------------------------------- |
| Architecture (L1–L5, Coordinator / Driver / Reconciler)                                     | [docs/architecture.md](docs/architecture.md)             |
| **MySQL HA failover** (3-node semi-sync & MGR: minority / majority loss, sequence diagrams) | [docs/ha-failover.md](docs/ha-failover.md)               |
| Config schema & validation                                                                  | [docs/config-design.md](docs/config-design.md)           |
| **Deployment** (build, 3-node wiring, neohactl, checklists)                                 | [docs/deployment.md](docs/deployment.md)                 |
| **Operations** (inspection, failover drills, production tuning)                               | [docs/operations.md](docs/operations.md)                 |
| Roadmap & IT notes                                                                          | [docs/TODO.md](docs/TODO.md)                             |
| Integration test harness                                                                    | [test/integration/README.md](test/integration/README.md) |


# Unit Tests

```bash
make test      # unit tests (~5 min; raft package is the slowest)
make coverage  # unit-test coverage (~8–15 min; raft alone ~5–10 min with -covermode=atomic)
```

CI runs **`make test`** and **`make coverage-ci`** on push/PR to `main` (unit tests only; see [.github/workflows/](.github/workflows/)).

**Integration tests are not run in CI** — they require a local MySQL 8.0 build, optional Xtrabackup, and SSH. Run them on a dev machine after significant changes:

```bash
make build && make test-integration
```

# Integration Tests

Requires MySQL 8.0 debug build, optional Xtrabackup + SSH for backup tests. See [test/integration/README.md](test/integration/README.md).

```bash
cp test/integration/it.local.yaml.example test/integration/it.local.yaml
make test-integration
```
