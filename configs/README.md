# NeoHA Configuration Examples

Sample **YAML and JSON** files for deployment and testing. The Go config schema and loader live in [`internal/config/`](../internal/config/).

| Path | Purpose |
|------|---------|
| `examples/neoha-full.yaml` / `.json` | **Full reference** — every supported config key |
| `examples/mysql/semisync-node1.yaml` / `.json` | Traditional semi-sync replication |
| `examples/mysql/mgr-mysql80-node1.yaml` / `.json` | MySQL Group Replication (8.0) |
| `examples/postgresql/` | Reserved for PostgreSQL (not yet supported) |

Copy an example and adjust paths (`basedir`, `defaults-file`, ports, peers) for your environment.

```bash
cp configs/examples/neoha-full.yaml /etc/neoha/db1.yaml
# or a scenario-specific template:
cp configs/examples/mysql/semisync-node1.yaml /etc/neoha/db1.yaml
cp configs/examples/mysql/semisync-node1.json /etc/neoha/db1.json
```

Load with any path — NeoHA detects format from the file extension (`.yaml`, `.yml`, `.json`), or from content when the extension is missing:

```bash
neoha -config /etc/neoha/db1.yaml
neoha -config /etc/neoha/db1.json
```
