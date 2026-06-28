/*
 * Copyright 2022-2026 The NeoHA Authors.
 *
 * See the AUTHORS file for a list of contributors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
*/

package harness

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/sealdb/neoha/internal/base/model"
	"github.com/sealdb/neoha/internal/config"
)

// NeoHANode is one NeoHA agent process in an integration cluster.
type NeoHANode struct {
	Name       string
	Endpoint   string
	MetaDir    string
	ConfigPath string
	LogPath    string
	MySQLPort  int
	cmd        *exec.Cmd
}

const (
	EnvNeoHABin = "NEOHA_IT_BIN"

	mgrReplUser  = "repl"
	mgrReplPass  = "repl"
	mgrGroupName = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

	// Semi-sync IT Raft timing (align with Xenon): 2s heartbeat × 5 failures ≈ 10s primary fault.
	semiSyncITHeartbeatTimeoutMs = 2000
	semiSyncITElectionTimeoutMs    = 10000
	semiSyncITAdmitDefeatHtCnt     = 5

	// MGR IT uses faster timers for quicker bootstrap/election in tests.
	mgrITHeartbeatTimeoutMs = 500
	mgrITElectionTimeoutMs  = 1500
	mgrITAdmitDefeatHtCnt   = 5
)

func applySemiSyncRaftIT(conf *config.Config) {
	conf.Election.Raft.HeartbeatTimeout = semiSyncITHeartbeatTimeoutMs
	conf.Election.Raft.ElectionTimeout = semiSyncITElectionTimeoutMs
	conf.Election.Raft.AdmitDefeatHtCnt = semiSyncITAdmitDefeatHtCnt
}

func applyMGRRaftIT(conf *config.Config) {
	conf.Election.Raft.HeartbeatTimeout = mgrITHeartbeatTimeoutMs
	conf.Election.Raft.ElectionTimeout = mgrITElectionTimeoutMs
	conf.Election.Raft.AdmitDefeatHtCnt = mgrITAdmitDefeatHtCnt
}

func applyHAIT(conf *config.Config) {
	conf.HA.DelegateDBApply = true
	conf.HA.ReconcileInterval = 1
}

// BuildNeoHA compiles the neoha daemon to outPath (go build uses the module cache; always invoke it so dependency changes are picked up).
func BuildNeoHA(ctx context.Context, repoRoot, outPath string) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "go", "build", "-o", outPath, "./cmd/neoha")
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build neoha: %w: %s", err, string(out))
	}
	return nil
}

// NeoHABinFromEnv returns the neoha binary path or empty.
func NeoHABinFromEnv() string {
	return LoadIntegrationSettings().NeoHABinPath()
}

// NewNeoHANode prepares directories for a NeoHA agent.
func NewNeoHANode(name, workDir, endpoint string, mysqlPort int) *NeoHANode {
	nodeDir := filepath.Join(workDir, "neoha", name)
	return &NeoHANode{
		Name:       name,
		Endpoint:   endpoint,
		MetaDir:    filepath.Join(nodeDir, "meta"),
		ConfigPath: filepath.Join(nodeDir, "neoha.yaml"),
		LogPath:    filepath.Join(nodeDir, "neoha.log"),
		MySQLPort:  mysqlPort,
	}
}

// WritePlainConfig writes a minimal NeoHA config for backup / xtrabackup tests.
func (n *NeoHANode) WritePlainConfig(mysqlBase, defaultsFile, clusterWorkDir, mysqlDataDir string, peers []string) error {
	if err := os.MkdirAll(n.MetaDir, 0o755); err != nil {
		return err
	}
	if err := WritePeersJSON(filepath.Join(n.MetaDir, "peers.json"), peers); err != nil {
		return err
	}

	conf := config.DefaultConfig()
	conf.Scope = "neoha-it"
	conf.Name = n.Name
	conf.Endpoint = n.Endpoint
	conf.Log.Level = "INFO"

	conf.Election.Raft.MetaDatadir = n.MetaDir
	applySemiSyncRaftIT(conf)
	conf.Election.Raft.PurgeBinlogDisabled = true
	conf.Election.Raft.LeaderStartCommand = "nop"
	conf.Election.Raft.LeaderStopCommand = "nop"

	conf.Database.Mysql.Version = "mysql80"
	conf.Database.Mysql.Host = "127.0.0.1"
	conf.Database.Mysql.Port = n.MySQLPort
	conf.Database.Mysql.Basedir = mysqlBase
	conf.Database.Mysql.DefaultsFile = defaultsFile
	conf.Database.Mysql.ReplMode = model.ReplModeSemiSync
	conf.Database.Mysql.MonitorDisabled = true

	LoadIntegrationSettings().ApplyBackupConfig(conf, n.MySQLPort, mysqlBase, defaultsFile, mysqlDataDir)
	applyHAIT(conf)

	if err := config.WriteConfig(n.ConfigPath, conf); err != nil {
		return err
	}
	return writeCLIConfigPath(filepath.Dir(n.ConfigPath), n.ConfigPath)
}

// WriteSemiSyncConfig writes a NeoHA config for semi-sync integration tests.
func (n *NeoHANode) WriteSemiSyncConfig(mysqlBase, defaultsFile, clusterWorkDir, mysqlDataDir string, peers []string) error {
	if err := os.MkdirAll(n.MetaDir, 0o755); err != nil {
		return err
	}
	if err := WritePeersJSON(filepath.Join(n.MetaDir, "peers.json"), peers); err != nil {
		return err
	}

	conf := config.DefaultConfig()
	conf.Scope = "neoha-it"
	conf.Name = n.Name
	conf.Endpoint = n.Endpoint
	conf.Log.Level = "INFO"

	conf.Election.Raft.MetaDatadir = n.MetaDir
	applySemiSyncRaftIT(conf)
	conf.Election.Raft.PurgeBinlogDisabled = true
	conf.Election.Raft.LeaderStartCommand = "nop"
	conf.Election.Raft.LeaderStopCommand = "nop"

	conf.Database.Mysql.Version = "mysql80"
	conf.Database.Mysql.Host = "127.0.0.1"
	conf.Database.Mysql.Port = n.MySQLPort
	conf.Database.Mysql.Basedir = mysqlBase
	conf.Database.Mysql.DefaultsFile = defaultsFile
	conf.Database.Mysql.ReplMode = model.ReplModeSemiSync
	conf.Database.Mysql.ReplHost = "127.0.0.1"
	conf.Database.Mysql.ReplUser = mgrReplUser
	conf.Database.Mysql.ReplPasswd = mgrReplPass
	conf.Database.Mysql.MonitorDisabled = true
	conf.Database.Mysql.SemiSyncTimeoutForTwoNodes = 10000

	LoadIntegrationSettings().ApplyBackupConfig(conf, n.MySQLPort, mysqlBase, defaultsFile, mysqlDataDir)
	applyHAIT(conf)

	if err := config.WriteConfig(n.ConfigPath, conf); err != nil {
		return err
	}
	return writeCLIConfigPath(filepath.Dir(n.ConfigPath), n.ConfigPath)
}

// WriteConfig writes a NeoHA config for MGR integration tests.
func (n *NeoHANode) WriteConfig(mysqlBase, defaultsFile, clusterWorkDir, mysqlDataDir string, peers []string) error {
	if err := os.MkdirAll(n.MetaDir, 0o755); err != nil {
		return err
	}
	if err := WritePeersJSON(filepath.Join(n.MetaDir, "peers.json"), peers); err != nil {
		return err
	}

	conf := config.DefaultConfig()
	conf.Scope = "neoha-it"
	conf.Name = n.Name
	conf.Endpoint = n.Endpoint
	conf.Log.Level = "INFO"

	conf.Election.Raft.MetaDatadir = n.MetaDir
	applyMGRRaftIT(conf)
	conf.Election.Raft.PurgeBinlogDisabled = true
	conf.Election.Raft.LeaderStartCommand = "nop"
	conf.Election.Raft.LeaderStopCommand = "nop"

	conf.Database.Mysql.Version = "mysql80"
	conf.Database.Mysql.Host = "127.0.0.1"
	conf.Database.Mysql.Port = n.MySQLPort
	conf.Database.Mysql.Basedir = mysqlBase
	conf.Database.Mysql.DefaultsFile = defaultsFile
	conf.Database.Mysql.ReplMode = model.ReplModeMGR
	conf.Database.Mysql.ReplHost = "127.0.0.1"
	conf.Database.Mysql.ReplUser = mgrReplUser
	conf.Database.Mysql.ReplPasswd = mgrReplPass
	conf.Database.Mysql.MonitorDisabled = true
	conf.Database.Mysql.MasterSysVars = fmt.Sprintf(
		"group_replication_group_name='%s'", mgrGroupName)
	conf.Database.Mysql.SlaveSysVars = fmt.Sprintf(
		"group_replication_group_name='%s';group_replication_local_address='127.0.0.1:%d';group_replication_group_seeds='127.0.0.1:13361,127.0.0.1:13362,127.0.0.1:13363'",
		mgrGroupName, n.MySQLPort+55)

	LoadIntegrationSettings().ApplyBackupConfig(conf, n.MySQLPort, mysqlBase, defaultsFile, mysqlDataDir)
	applyHAIT(conf)

	if err := config.WriteConfig(n.ConfigPath, conf); err != nil {
		return err
	}
	return writeCLIConfigPath(filepath.Dir(n.ConfigPath), n.ConfigPath)
}

// WriteEtcdPGConfig writes NeoHA config for PostgreSQL + etcd integration tests.
func (n *NeoHANode) WriteEtcdPGConfig(pgBase, dataDir, etcdEndpoint string, pgPort int) error {
	if err := os.MkdirAll(n.MetaDir, 0o755); err != nil {
		return err
	}

	conf := config.DefaultConfig()
	conf.Scope = "neoha-pg-it"
	conf.Name = n.Name
	conf.Endpoint = n.Endpoint
	conf.Log.Level = "INFO"

	conf.Coordination.Provider = "etcd"
	conf.Coordination.Etcd.Host = etcdEndpoint
	conf.Coordination.Etcd.TTL = 5

	conf.Database.Type = "postgresql"
	conf.Database.Postgresql.Version = "postgresql14"
	conf.Database.Postgresql.Listen = fmt.Sprintf("127.0.0.1:%d", pgPort)
	conf.Database.Postgresql.ConnectAddress = fmt.Sprintf("127.0.0.1:%d", pgPort)
	conf.Database.Postgresql.DataDir = dataDir
	conf.Database.Postgresql.BinDir = filepath.Join(pgBase, "bin")
	conf.Database.Postgresql.UseSlots = true
	conf.Database.Postgresql.PrimarySlotName = n.Name
	conf.Database.Postgresql.UsePGRewind = true
	conf.Database.Postgresql.Auth.Repl.Username = PGReplUser
	conf.Database.Postgresql.Auth.Repl.Password = PGReplPass
	conf.Database.Postgresql.Auth.SuperUser.Username = PGSuperUser
	conf.Database.Postgresql.Auth.SuperUser.Password = PGSuperPass
	conf.Database.Postgresql.Auth.Rewind.Username = PGSuperUser
	conf.Database.Postgresql.Auth.Rewind.Password = PGSuperPass

	applyHAIT(conf)
	conf.HA.PrimaryHooks.OnPrimaryStart = "nop"
	conf.HA.PrimaryHooks.OnPrimaryStop = "nop"

	if err := config.WriteConfig(n.ConfigPath, conf); err != nil {
		return err
	}
	return writeCLIConfigPath(filepath.Dir(n.ConfigPath), n.ConfigPath)
}

// writeCLIConfigPath writes config.path for neohactl.
func writeCLIConfigPath(dir, configFile string) error {
	return os.WriteFile(filepath.Join(dir, "config.path"), []byte(configFile), 0o644)
}

// WritePeersJSON writes raft peers metadata consumed on NeoHA startup.
func WritePeersJSON(path string, peers []string) error {
	clean := make([]string, 0, len(peers))
	seen := make(map[string]bool, len(peers))
	for _, p := range peers {
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		clean = append(clean, p)
	}
	payload := map[string][]string{
		"peers":     clean,
		"idlepeers": {},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// Start launches the neoha daemon. role is LEADER, FOLLOWER, or IDLE (default FOLLOWER).
func (n *NeoHANode) Start(ctx context.Context, neohaBin, role string) error {
	if role == "" {
		role = "FOLLOWER"
	}
	logFile, err := os.Create(n.LogPath)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, neohaBin, "-config", n.ConfigPath, "-role", role)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		logFile.Close()
		return err
	}
	n.cmd = cmd
	return nil
}

// Stop terminates the neoha process.
func (n *NeoHANode) Stop(ctx context.Context) error {
	if n.cmd == nil || n.cmd.Process == nil {
		return nil
	}
	_ = n.cmd.Process.Signal(os.Interrupt)
	done := make(chan error, 1)
	go func() { done <- n.cmd.Wait() }()
	select {
	case <-ctx.Done():
		_ = n.cmd.Process.Kill()
	case <-done:
	}
	n.cmd = nil
	return nil
}

// EndpointsForClusterNodes maps surviving MySQL cluster nodes to NeoHA RPC endpoints.
func EndpointsForClusterNodes(neoNodes []*NeoHANode, cluster *Cluster, nodes []*Node) []string {
	byName := make(map[string]string, len(neoNodes))
	for i, na := range neoNodes {
		if i < len(cluster.Nodes) {
			byName[cluster.Nodes[i].Name] = na.Endpoint
		}
	}
	out := make([]string, 0, len(nodes))
	for _, n := range nodes {
		if ep, ok := byName[n.Name]; ok {
			out = append(out, ep)
		}
	}
	return out
}

// TailNeoHALog returns the tail of a NeoHA agent log file.
func (n *NeoHANode) TailNeoHALog(maxBytes int) string {
	data, err := os.ReadFile(n.LogPath)
	if err != nil {
		return fmt.Sprintf("(no log: %v)", err)
	}
	if len(data) > maxBytes {
		data = data[len(data)-maxBytes:]
	}
	return string(data)
}
