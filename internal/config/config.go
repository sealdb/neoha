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

package config

import (
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/sealdb/neoha/internal/base/common"
	"github.com/sealdb/neoha/internal/base/model"
)

var (
	APIPort = 8008
)

type AuthenticationConfig struct {
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
}

func DefaultAuthenticationConfig() *AuthenticationConfig {
	return &AuthenticationConfig{
		Username: "",
		Password: "",
	}
}

type RestAPIConfig struct {
	EnableAPIs     bool                  `yaml:"enable-apis" json:"enable-apis"`         // true or fase
	Listen         string                `yaml:"listen" json:"listen"`                   // ip:port
	ConnectAddress string                `yaml:"connect_address" json:"connect_address"` // ip:port
	Certfile       string                `yaml:"certfile" json:"certfile"`
	Keyfile        string                `yaml:"keyfile" json:"keyfile"`
	Authentication *AuthenticationConfig `yaml:"authentication" json:"authentication"`
}

func DefaultRestAPIConfig() *RestAPIConfig {
	return &RestAPIConfig{
		Listen:         "127.0.0.1:8008",
		ConnectAddress: "127.0.0.1:8008",
		Certfile:       "",
		Keyfile:        "",
		Authentication: DefaultAuthenticationConfig(),
	}
}

type CtlConfig struct {
	Insecure bool   `yaml:"insecure" json:"insecure"`
	Certfile string `yaml:"certfile" json:"certfile"`
	Cacert   string `yaml:"cacert" json:"cacert"`
}

func DefaultCtlConfig() *CtlConfig {
	return &CtlConfig{
		Insecure: false,
		Certfile: "",
		Cacert:   "",
	}
}

type EtcdConfig struct {
	Host       string   `yaml:"host" json:"host"`
	Hosts      []string `yaml:"hosts" json:"hosts"`
	UseProxies bool     `yaml:"use_proxies" json:"use_proxies"`
	// TTL is the DCS leader lease TTL in seconds (Patroni-style).
	TTL int `yaml:"ttl" json:"ttl"`
	/*
	 * Provide host to do the initial discovery of the cluster topology:
	 * host: 127.0.0.1:2379
	 * Or use "hosts" to provide multiple endpoints
	 * Could be a comma separated string:
	 * hosts: host1:port1,host2:port2
	 * or an actual yaml list:
	 * hosts:
	 * - host1:port1
	 * - host2:port2
	 * Once discovery is complete Patroni will use the list of advertised clientURLs
	 * It is possible to change this behavior through by setting:
	 * use_proxies: true
	 */
}

func DefaultEtcdConfig() *EtcdConfig {
	return &EtcdConfig{
		Host:  "",
		Hosts: []string{},
		TTL:   30,
	}
}

type RaftConfig struct {
	// raft meta datadir
	MetaDatadir string `yaml:"meta-datadir" json:"meta-datadir"`

	// leader heartbeat interval(ms)
	HeartbeatTimeout int `yaml:"heartbeat-timeout" json:"heartbeat-timeout"`

	// admit defeat count for hearbeat
	AdmitDefeatHtCnt int `yaml:"admit-defeat-hearbeat-count" json:"admit-defeat-hearbeat-count"`

	// election timeout(ms)
	ElectionTimeout int `yaml:"election-timeout" json:"election-timeout"`

	// purge binlog interval (ms)
	PurgeBinlogInterval int `yaml:"purge-binlog-interval" json:"purge-binlog-interval"`

	// Super IDLE can't change to FOLLOWER.
	SuperIDLE bool `yaml:"super-idle" json:"super-idle"`

	// MUST: set in init
	// the shell command when leader start
	LeaderStartCommand string `yaml:"leader-start-command" json:"leader-start-command"`

	// MUST: set in init
	// the shell command when leader stop
	LeaderStopCommand string `yaml:"leader-stop-command" json:"leader-stop-command"`

	// if true, neoha binlog-purge will be skipped, default is false.
	PurgeBinlogDisabled bool `yaml:"purge-binlog-disabled" json:"purge-binlog-disabled"`

	// rpc client request tiemout(ms)
	RequestTimeout int `yaml:"requesttimeout" json:"requesttimeout"`

	// candicate wait timeout(ms) for 2 nodes.
	CandidateWaitFor2Nodes int `yaml:"candidate-wait-for-2nodes" json:"candidate-wait-for-2nodes"`
}

func DefaultRaftConfig() *RaftConfig {
	return &RaftConfig{
		MetaDatadir:            ".",
		HeartbeatTimeout:       1000,
		AdmitDefeatHtCnt:       10,
		ElectionTimeout:        3000,
		PurgeBinlogInterval:    1000 * 60 * 5,
		LeaderStartCommand:     "nop", // TODO: or ""
		LeaderStopCommand:      "nop", // TODO: or ""
		RequestTimeout:         1000,  // TODO: or 500
		CandidateWaitFor2Nodes: 1000 * 60,
	}
}

type ElectionConfig struct {
	Algo string      `yaml:"algorithm" json:"algorithm"`
	Raft *RaftConfig `yaml:"raft" json:"raft"`
	Etcd *EtcdConfig `yaml:"etcd" json:"etcd"`
}

func DefaultElectionConfig() *ElectionConfig {
	return &ElectionConfig{
		Algo: "raft",
		Raft: DefaultRaftConfig(),
		Etcd: DefaultEtcdConfig(),
	}
}

// CoordinationConfig is the target name for election/coordination settings.
// Prefer this block in new configs; election is kept as a deprecated alias.
type CoordinationConfig struct {
	Provider string      `yaml:"provider" json:"provider"`
	Raft     *RaftConfig `yaml:"raft" json:"raft"`
	Etcd     *EtcdConfig `yaml:"etcd" json:"etcd"`
}

func DefaultCoordinationConfig() *CoordinationConfig {
	ec := DefaultElectionConfig()
	return &CoordinationConfig{
		Provider: ec.Algo,
		Raft:     ec.Raft,
		Etcd:     ec.Etcd,
	}
}

// EffectiveCoordination returns the active coordination settings (coordination.* or election.*).
func (c *Config) EffectiveCoordination() *CoordinationConfig {
	if c == nil {
		return DefaultCoordinationConfig()
	}
	if c.Coordination != nil && c.Coordination.Provider != "" {
		out := &CoordinationConfig{
			Provider: c.Coordination.Provider,
			Raft:     c.Coordination.Raft,
			Etcd:     c.Coordination.Etcd,
		}
		if out.Raft == nil && c.Election != nil {
			out.Raft = c.Election.Raft
		}
		if out.Etcd == nil && c.Election != nil {
			out.Etcd = c.Election.Etcd
		}
		return out
	}
	if c.Election != nil {
		return &CoordinationConfig{
			Provider: c.Election.Algo,
			Raft:     c.Election.Raft,
			Etcd:     c.Election.Etcd,
		}
	}
	return DefaultCoordinationConfig()
}

type StandbyClusterConfig struct {
	Host            string `yaml:"host" json:"host"`
	Port            int    `yaml:"port" json:"port"`
	PrimarySlotName string `yaml:"primary_slot_name" json:"primary_slot_name"`
}

func DefaultStandbyClusterConfig() *StandbyClusterConfig {
	return &StandbyClusterConfig{
		Host:            "",
		Port:            0,
		PrimarySlotName: "pgrepl",
	}
}

type RecoveryConfConfig struct {
	RestoreCommand string `yaml:"restore_command" json:"restore_command"`
}

func DefaultRecoveryConfConfig() *RecoveryConfConfig {
	return &RecoveryConfConfig{}
}

type DcsConfig struct {
	TTL                  int                   `yaml:"ttl" json:"ttl"`
	LoopWait             int                   `yaml:"loop_wait" json:"loop_wait"`
	RetryTimeout         int                   `yaml:"retry_timeout" json:"retry_timeout"`
	MaximumLagOnFailover int                   `yaml:"maximum_lag_on_failover" json:"maximum_lag_on_failover"`
	MasterStartTimeout   int                   `yaml:"master_start_timeout" json:"master_start_timeout"`
	SynchronousMode      bool                  `yaml:"synchronous_mode" json:"synchronous_mode"`
	StandbyCluster       *StandbyClusterConfig `yaml:"standby_cluster" json:"standby_cluster"`
	UsePGRewind          bool                  `yaml:"use_pg_rewind" json:"use_pg_rewind"`
	UseSlots             bool                  `yaml:"use_slots" json:"use_slots"`
	Parameters           map[string]string     `yaml:"parameters" json:"parameters"`
	/*
	   wal_level: hot_standby
	   hot_standby: "on"
	   max_connections: 100
	   max_worker_processes: 8
	   wal_keep_segments: 8
	   max_wal_senders: 10
	   max_replication_slots: 10
	   max_prepared_transactions: 0
	   max_locks_per_transaction: 64
	   wal_log_hints: "on"
	   track_commit_timestamp: "off"
	   archive_mode: "on"
	   archive_timeout: 1800s
	   archive_command: mkdir -p ../wal_archive && test ! -f ../wal_archive/%f && cp %p ../wal_archive/%f
	*/
	RecoveryConf *RecoveryConfConfig `yaml:"recovery_conf" json:"recovery_conf"`
}

func DefaultDcsConfig() *DcsConfig {
	return &DcsConfig{
		TTL:                  30,
		LoopWait:             10,
		RetryTimeout:         10,
		MaximumLagOnFailover: 1048576,
		MasterStartTimeout:   300,
		StandbyCluster:       DefaultStandbyClusterConfig(),
		UsePGRewind:          true,
		UseSlots:             true,
		Parameters:           map[string]string{},
		RecoveryConf:         DefaultRecoveryConfConfig(),
	}
}

type InitDBConfig struct {
	Encoding      string `yaml:"encoding" json:"encoding"`
	DataChecksums bool   `yaml:"data_checksums,omitempty" json:"data_checksums,omitempty"`
}

func DefaultInitDBConfig() *InitDBConfig {
	return &InitDBConfig{
		Encoding:      "UTF8",
		DataChecksums: true,
	}
}

type BootstrapPostgresqlUsersConfig struct {
	Username string
	Password string
	Options  []string
}

type BootstrapPostgresqlConfig struct {
	DcsConf  *DcsConfig                       `yaml:"dcs" json:"dcs"`
	InitDB   *InitDBConfig                    `yaml:"initdb" json:"initdb"`
	PgHba    []string                         `yaml:"pg_hba" json:"pg_hba"`
	PostInit string                           `yaml:"post_init" json:"post_init"`
	Users    []BootstrapPostgresqlUsersConfig `yaml:"users" json:"users"`
}

func DefaultBootstrapPostgresqlConfig() *BootstrapPostgresqlConfig {
	return &BootstrapPostgresqlConfig{
		DcsConf:  DefaultDcsConfig(),
		InitDB:   DefaultInitDBConfig(),
		PgHba:    []string{},
		PostInit: "",
		Users:    []BootstrapPostgresqlUsersConfig{},
	}
}

type BootstrapMysqlConfig struct {
}

func DefaultBootstrapMysqlConfig() *BootstrapMysqlConfig {
	return &BootstrapMysqlConfig{}
}

type BootstrapConfig struct {
	BootstrapPostgresql *BootstrapPostgresqlConfig `yaml:"postgresql" json:"postgresql"`
	BootstrapMysql      *BootstrapMysqlConfig      `yaml:"mysql" json:"mysql"`
}

func DefaultBootstrapConfig() *BootstrapConfig {
	return &BootstrapConfig{
		BootstrapPostgresql: DefaultBootstrapPostgresqlConfig(),
		BootstrapMysql:      DefaultBootstrapMysqlConfig(),
	}
}

type PGAuthReplConfig struct {
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
}

func DefaultPGAuthReplConfig() *PGAuthReplConfig {
	return &PGAuthReplConfig{}
}

type PGAuthSuperUserConfig struct {
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
}

func DefaultPGAuthSuperUserConfig() *PGAuthSuperUserConfig {
	return &PGAuthSuperUserConfig{}
}

type PGAuthRewindConfig struct {
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
}

func DefaultPGAuthRewindConfig() *PGAuthRewindConfig {
	return &PGAuthRewindConfig{}
}

type PGAuthConfig struct {
	Repl      *PGAuthReplConfig      `yaml:"replication" json:"replication"`
	SuperUser *PGAuthSuperUserConfig `yaml:"superuser" json:"superuser"`
	Rewind    *PGAuthRewindConfig    `yaml:"rewind" json:"rewind"`
}

func DefaultPGAuthConfig() *PGAuthConfig {
	return &PGAuthConfig{
		Repl:      DefaultPGAuthReplConfig(),
		SuperUser: DefaultPGAuthSuperUserConfig(),
		Rewind:    DefaultPGAuthRewindConfig(),
	}
}

type PostgresqlConfig struct {
	Version        string            `yaml:"version" json:"version"`
	Listen         string            `yaml:"listen" json:"listen"`
	ConnectAddress string            `yaml:"connect_address" json:"connect_address"`
	DataDir        string            `yaml:"data_dir" json:"data_dir"`
	BinDir         string            `yaml:"bin_dir" json:"bin_dir"`
	ConfigDir      string            `yaml:"config_dir" json:"config_dir"`
	PGPass         string            `yaml:"pgpass" json:"pgpass"`
	Auth           *PGAuthConfig     `yaml:"authentication" json:"authentication"`
	Krbsrvname     string            `yaml:"krbsrvname" json:"krbsrvname"`
	Parameters     map[string]string `yaml:"parameters" json:"parameters"`
	PrePromote     string            `yaml:"pre_promote" json:"pre_promote"`
	// UseSlots enables primary_slot_name in recovery settings.
	UseSlots bool `yaml:"use_slots" json:"use_slots"`
	// PrimarySlotName is the physical replication slot on the primary for this replica.
	PrimarySlotName string `yaml:"primary_slot_name" json:"primary_slot_name"`
}

func DefaultPostgresqlConfig() *PostgresqlConfig {
	return &PostgresqlConfig{
		Version:        "postgresql14",
		Listen:         "127.0.0.1:5432",
		ConnectAddress: "127.0.0.1:5432",
		DataDir:        "",
		BinDir:         "",
		ConfigDir:      "",
		PGPass:         "",
		Auth:           DefaultPGAuthConfig(),
		Krbsrvname:     "postgres",
		Parameters:     map[string]string{},
		PrePromote:     "",
	}
}

type BackupConfig struct {
	// MUST: set in init
	SSHHost                 string `yaml:"ssh-host" json:"ssh-host"`
	SSHUser                 string `yaml:"ssh-user" json:"ssh-user"`
	SSHPasswd               string `yaml:"ssh-passwd" json:"ssh-passwd"`
	SSHPort                 int    `yaml:"ssh-port" json:"ssh-port"`
	BackupDir               string `yaml:"backupdir" json:"backupdir"`
	XtrabackupBinDir        string `yaml:"xtrabackup-bindir" json:"xtrabackup-bindir"`
	BackupIOPSLimits        int    `yaml:"backup-iops-limits" json:"backup-iops-limits"`
	UseMemory               string `yaml:"backup-use-memory" json:"backup-use-memory"`
	Parallel                int    `yaml:"backup-parallel" json:"backup-parallel"`
	MysqldMonitorInterval   int    `yaml:"mysqld-monitor-interval" json:"mysqld-monitor-interval"`
	MaxAllowedLocalTrxCount int    `yaml:"max-allowed-local-trx-count" json:"max-allowed-local-trx-count"`

	// mysql admin
	Admin string `yaml:"admin" json:"admin"`

	// mysql passwd
	Passwd string `yaml:"passwd" json:"passwd"`

	// mysql host
	Host string `yaml:"host" json:"host"`

	// mysql port
	Port int `yaml:"port" json:"port"`

	// mysql basedir
	Basedir string `yaml:"basedir" json:"basedir"`

	// mysql default file
	DefaultsFile string `yaml:"defaults-file" json:"defaults-file"`
}

func DefaultBackupConfig() *BackupConfig {
	return &BackupConfig{
		SSHHost:                 "127.0.0.1",
		SSHUser:                 "backup",
		SSHPasswd:               "backup",
		SSHPort:                 22,
		BackupDir:               "/u01/backup",
		XtrabackupBinDir:        ".",
		BackupIOPSLimits:        100000,
		UseMemory:               "2GB",
		Parallel:                2,
		MysqldMonitorInterval:   1000 * 1,
		MaxAllowedLocalTrxCount: 0,
		Admin:                   "root",
		Passwd:                  "",
		Host:                    "localhost",
		Port:                    3306,
		Basedir:                 "/u01/mysql_20221010/",
		DefaultsFile:            "/etc/my3306.cnf",
	}
}

type MysqlConfig struct {
	// mysql admin user
	Admin string `yaml:"admin" json:"admin"`

	// mysql admin passwd
	Passwd string `yaml:"passwd" json:"passwd"`

	// mysql localhost
	Host string `yaml:"host" json:"host"`

	// mysql local port
	Port int `yaml:"port" json:"port"`

	// mysql basedir
	Basedir string `yaml:"basedir" json:"basedir"`

	// mysql version
	Version string `yaml:"version" json:"version"`

	// mysql default file path
	DefaultsFile string `yaml:"defaults-file" json:"defaults-file"`

	// replication mode, semi-sync or group-replication
	ReplMode model.MysqlReplMode `yaml:"replication-mode" json:"replication-mode"`

	// ping mysql interval(ms)
	PingTimeout int `yaml:"ping-timeout" json:"ping-timeout"`

	// admit defeat count for ping mysql
	AdmitDefeatPingCnt int `yaml:"admit-defeat-ping-count" json:"admit-defeat-ping-count"`

	// max-open-conns limits NeoHA admin connections to MySQL (0 = default 2).
	MaxOpenConns int `yaml:"max-open-conns" json:"max-open-conns"`

	// max-idle-conns keeps idle admin connections in the pool (0 = same as max-open-conns).
	MaxIdleConns int `yaml:"max-idle-conns" json:"max-idle-conns"`

	// When true, MysqlDead caused by Error 1040 triggers automatic leader failover.
	FailoverOnTooManyConnections bool `yaml:"failover-on-too-many-connections" json:"failover-on-too-many-connections"`

	// rpl_semi_sync_master_timeout for 2 nodes
	SemiSyncTimeoutForTwoNodes uint64 `yaml:"semi-sync-timeout-for-two-nodes" json:"semi-sync-timeout-for-two-nodes"`

	// master system variables configure(separated by ;)
	MasterSysVars string `yaml:"master-sysvars" json:"master-sysvars"`

	// slave system variables configure(separated by ;)
	SlaveSysVars string `yaml:"slave-sysvars" json:"slave-sysvars"`

	// If true, the mysql monitor will disabled, default is false.
	MonitorDisabled bool `yaml:"monitor-disabled" json:"monitor-disabled"`

	// mysql intranet ip, other replicas Master_Host
	ReplHost string `yaml:"repl-host" json:"repl-host"`

	// mysql replication user
	ReplUser string `yaml:"repl-user" json:"repl-user"`

	// mysql replication user pwd
	ReplPasswd string `yaml:"repl-passwd" json:"repl-passwd"`

	// replication Gtid Purged
	ReplGtidPurged string `yaml:"repl-gtid-purged" json:"repl-gtid-purged"`

	Backup *BackupConfig `yaml:"backup" json:"backup"`
}

func DefaultMysqlConfig() *MysqlConfig {
	return &MysqlConfig{
		Admin:                      "root",
		Passwd:                     "",
		Host:                       "localhost", // TODO: or "127.0.0.1"
		Port:                       3306,        // TODO: or 8080
		Version:                    "mysql57",
		ReplMode:                   "semi-sync",
		PingTimeout:                1000,
		AdmitDefeatPingCnt:         2,
		MaxOpenConns:                 2,
		MaxIdleConns:                 2,
		FailoverOnTooManyConnections: false,
		SemiSyncTimeoutForTwoNodes:   10000,
		Basedir:                    "/u01/mysql_20221010/",
		DefaultsFile:               "/etc/my3306.cnf",
		ReplHost:                   "127.0.0.1",
		ReplUser:                   "repl",
		ReplPasswd:                 "repl",
		ReplGtidPurged:             "",
		Backup:                     DefaultBackupConfig(),
	}
}

type DatabaseConfig struct {
	Type       string            `yaml:"type" json:"type"`
	Postgresql *PostgresqlConfig `yaml:"postgresql" json:"postgresql"`
	Mysql      *MysqlConfig      `yaml:"mysql" json:"mysql"`
}

func DefaultDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		Type:       "mysql",
		Postgresql: DefaultPostgresqlConfig(),
		Mysql:      DefaultMysqlConfig(),
	}
}

type WatchdogConfig struct {
	Mode         string `yaml:"mode" json:"mode"`
	Device       string `yaml:"device" json:"device"`
	SafetyMargin int    `yaml:"safety_margin" json:"safety_margin"`
}

func DefaultWatchdogConfig() *WatchdogConfig {
	return &WatchdogConfig{
		Mode:         "automatic",
		Device:       "/dev/watchdog",
		SafetyMargin: 5,
	}
}

type TagsConfig struct {
	Nofailover    bool `yaml:"nofailover" json:"nofailover"`
	Noloadbalance bool `yaml:"noloadbalance" json:"noloadbalance"`
	Clonefrom     bool `yaml:"clonefrom" json:"clonefrom"`
	Nosync        bool `yaml:"nosync" json:"nosync"`
}

func DefaultTagsConfig() *TagsConfig {
	return &TagsConfig{
		Nofailover:    false,
		Noloadbalance: false,
		Clonefrom:     false,
		Nosync:        false,
	}
}

type LogConfig struct {
	Level string `yaml:"level" json:"level"`
}

func DefaultLogConfig() *LogConfig {
	return &LogConfig{
		Level: "INFO",
	}
}

// HAConfig controls the L3 reconcile loop (see internal/ha, docs/architecture.md).
type HAConfig struct {
	// ReconcileInterval is seconds between reconcile ticks; 0 uses ha.DefaultReconcileInterval.
	ReconcileInterval int `yaml:"reconcile_interval" json:"reconcile_interval"`
	// DelegateDBApply lets Reconciler own demote and promotion (semi-sync + MGR).
	DelegateDBApply bool `yaml:"delegate_db_apply" json:"delegate_db_apply"`
	// PrimaryHooks run when this node becomes/stops serving as the writable primary (VIP, LB, …).
	PrimaryHooks *PrimaryHooksConfig `yaml:"primary_hooks" json:"primary_hooks"`
}

// PrimaryHooksConfig configures shell hooks after the DB is promoted/demoted.
// Prefer this over election.raft leader-start/stop-command (legacy alias).
type PrimaryHooksConfig struct {
	OnPrimaryStart string `yaml:"on_primary_start" json:"on_primary_start"`
	OnPrimaryStop  string `yaml:"on_primary_stop" json:"on_primary_stop"`
}

func DefaultPrimaryHooksConfig() *PrimaryHooksConfig {
	return &PrimaryHooksConfig{
		OnPrimaryStart: "nop",
		OnPrimaryStop:  "nop",
	}
}

func DefaultHAConfig() *HAConfig {
	return &HAConfig{
		ReconcileInterval: 3,
		DelegateDBApply:   false,
		PrimaryHooks:      DefaultPrimaryHooksConfig(),
	}
}

type Config struct {
	Scope     string `yaml:"scope" json:"scope"`
	NameSpace string `yaml:"namespace" json:"namespace"`
	Name      string `yaml:"name" json:"name"`

	// connection string(format ip:port)
	Endpoint string `yaml:"endpoint" json:"endpoint"`

	// HTTP APIs address.
	PeerAddress string `yaml:"peer-address,omitempty" json:"peer-address,omitempty"`

	RestAPI       *RestAPIConfig       `yaml:"restapi" json:"restapi"`
	Ctl           *CtlConfig           `yaml:"ctl" json:"ctl"`
	Election      *ElectionConfig      `yaml:"election" json:"election"`           // deprecated: use coordination
	Coordination  *CoordinationConfig  `yaml:"coordination" json:"coordination"`
	Bootstrap     *BootstrapConfig     `yaml:"bootstrap" json:"bootstrap"`
	Database  *DatabaseConfig  `yaml:"database" json:"database"`
	Watchdog  *WatchdogConfig  `yaml:"watchdog" json:"watchdog"`
	Tags      *TagsConfig      `yaml:"tags" json:"tags"`
	HA        *HAConfig        `yaml:"ha" json:"ha"`
	Log       *LogConfig       `yaml:"log" json:"log"`
}

func DefaultConfig() *Config {
	return &Config{
		Scope:       "database",
		NameSpace:   "/service/",
		Name:        "db1",
		Endpoint:    "127.0.0.1:8080",
		PeerAddress: ":6060",

		RestAPI:       DefaultRestAPIConfig(),
		Ctl:           DefaultCtlConfig(),
		Election:      DefaultElectionConfig(),
		Coordination:  DefaultCoordinationConfig(),
		Bootstrap:     DefaultBootstrapConfig(),
		Database:  DefaultDatabaseConfig(),
		Watchdog:  DefaultWatchdogConfig(),
		Tags:      DefaultTagsConfig(),
		HA:        DefaultHAConfig(),
		Log:       DefaultLogConfig(),
	}
}

func LoadConfig(filepath string) (*Config, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	format, err := DetectConfigFormat(filepath, data)
	if err != nil {
		return nil, err
	}

	conf, err := ParseConfig(data, format)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return conf, nil
}

func WriteConfig(filepath string, conf *Config) error {
	format := common.GetFileType(filepath)
	if !isSupportedConfigFormat(format) {
		return errors.Errorf("the type [%s] of file [%s] is not supported", format, filepath)
	}

	data, err := MarshalConfig(conf, format)
	if err != nil {
		return errors.WithStack(err)
	}

	if err = os.WriteFile(filepath, data, 0644); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// EffectivePrimaryHooks returns primary accessibility hooks, preferring ha.primary_hooks
// and falling back to legacy election.raft leader-start/stop-command.
func (c *Config) EffectivePrimaryHooks() PrimaryHooksConfig {
	if c == nil {
		return *DefaultPrimaryHooksConfig()
	}
	_ = c.Normalize()
	out := *c.HA.PrimaryHooks
	if strings.TrimSpace(out.OnPrimaryStart) == "" && c.Election != nil && c.Election.Raft != nil {
		out.OnPrimaryStart = c.Election.Raft.LeaderStartCommand
	}
	if strings.TrimSpace(out.OnPrimaryStop) == "" && c.Election != nil && c.Election.Raft != nil {
		out.OnPrimaryStop = c.Election.Raft.LeaderStopCommand
	}
	if strings.TrimSpace(out.OnPrimaryStart) == "" {
		out.OnPrimaryStart = "nop"
	}
	if strings.TrimSpace(out.OnPrimaryStop) == "" {
		out.OnPrimaryStop = "nop"
	}
	return out
}
