# NeoHA Configuration Examples

Sample **YAML and JSON** files for deployment and testing. The Go config schema and loader live in [`internal/config/`](../internal/config/).

## 注释说明

| 格式 | 行内注释 |
|------|----------|
| **`.yaml` / `.yml`** | 每个配置项均有 `#` 注释，**请以 YAML 为权威参考** |
| **`.json`** | 标准 JSON **不支持**注释；字段含义见同名 `.yaml` |

同步 JSON：修改 YAML 后可运行：

```bash
NEOHA_REGEN_CONFIG_EXAMPLES=1 go test ./internal/config/ -run TestRegenerateExampleJSON
```

## 示例文件

| Path | Purpose |
|------|---------|
| `examples/neoha-full.yaml` / `.json` | **Full reference** — every supported config key |
| `examples/mysql/semisync-node1.yaml` / `.json` | Traditional semi-sync replication |
| `examples/mysql/mgr-mysql80-node1.yaml` / `.json` | MySQL Group Replication (8.0) |
| `examples/postgresql/etcd-node1.yaml` | PostgreSQL streaming + `coordination.provider: etcd` |

详细设计见 [docs/config-design.md](../docs/config-design.md)。

Copy an example and adjust paths (`basedir`, `defaults-file`, ports, peers) for your environment.

```bash
cp configs/examples/neoha-full.yaml /etc/neoha/db1.yaml
cp configs/examples/mysql/semisync-node1.yaml /etc/neoha/db1.yaml
```

Load with any path — NeoHA detects format from the file extension (`.yaml`, `.yml`, `.json`), or from content when the extension is missing:

```bash
neoha -config /etc/neoha/db1.yaml
neoha -config /etc/neoha/db1.json
```
