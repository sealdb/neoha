/*
 * Copyright 2022 The NeoHA Authors.
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
	"encoding/json"
	"github.com/pkg/errors"
	"neoha/base/common"

	//"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
	"io/ioutil"
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
	}
}

type RaftConfig struct {
	DataDir      string   `yaml:"data_dir" json:"data_dir"`
	SelfAddr     string   `yaml:"self_addr" json:"self_addr"`
	PartnerAddrs []string `yaml:"partner_addrs" json:"partner_addrs"`
}

func DefaultRaftConfig() *RaftConfig {
	return &RaftConfig{
		DataDir:      ".",
		SelfAddr:     "127.0.0.1:8801",
		PartnerAddrs: []string{},
	}
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

type DcsPostgresqlConfig struct {
	UsePGRewind bool              `yaml:"use_pg_rewind" json:"use_pg_rewind"`
	UseSlots    bool              `yaml:"use_slots" json:"use_slots"`
	Parameters  map[string]string `yaml:"parameters" json:"parameters"`
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

func DefaultDcsPostgresqlConfig() *DcsPostgresqlConfig {
	return &DcsPostgresqlConfig{
		UsePGRewind:  true,
		UseSlots:     true,
		Parameters:   map[string]string{},
		RecoveryConf: DefaultRecoveryConfConfig(),
	}
}

type DcsConfig struct {
	TTL                  int                   `yaml:"ttl" json:"ttl"`
	LoopWait             int                   `yaml:"loop_wait" json:"loop_wait"`
	RetryTimeout         int                   `yaml:"retry_timeout" json:"retry_timeout"`
	MaximumLagOnFailover int                   `yaml:"maximum_lag_on_failover" json:"maximum_lag_on_failover"`
	MasterStartTimeout   int                   `yaml:"master_start_timeout" json:"master_start_timeout"`
	SynchronousMode      bool                  `yaml:"synchronous_mode" json:"synchronous_mode"`
	StandbyCluster       *StandbyClusterConfig `yaml:"standby_cluster" json:"standby_cluster"`
	DcsPostgresql        *DcsPostgresqlConfig  `yaml:"postgresql" json:"postgresql"`
}

func DefaultDcsConfig() *DcsConfig {
	return &DcsConfig{
		TTL:                  30,
		LoopWait:             10,
		RetryTimeout:         10,
		MaximumLagOnFailover: 1048576,
		MasterStartTimeout:   300,
		StandbyCluster:       DefaultStandbyClusterConfig(),
		DcsPostgresql:        DefaultDcsPostgresqlConfig(),
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

type BootstrapUsersConfig struct {
	Username string
	Password string
	Options  []string
}

type BootstrapConfig struct {
	DcsConf  *DcsConfig             `yaml:"dcs" json:"dcs"`
	InitDB   *InitDBConfig          `yaml:"initdb" json:"initdb"`
	PgHba    []string               `yaml:"pg_hba" json:"pg_hba"`
	PostInit string                 `yaml:"post_init" json:"post_init"`
	Users    []BootstrapUsersConfig `yaml:"users" json:"users"`
}

func DefaultBootstrapConfig() *BootstrapConfig {
	return &BootstrapConfig{
		DcsConf:  DefaultDcsConfig(),
		InitDB:   DefaultInitDBConfig(),
		PgHba:    []string{},
		PostInit: "",
		Users:    []BootstrapUsersConfig{},
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
}

func DefaultPostgresqlConfig() *PostgresqlConfig {
	return &PostgresqlConfig{
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

type Config struct {
	Scope      string            `yaml:"scope" json:"scope"`
	NameSpace  string            `yaml:"namespace" json:"namespace"`
	Name       string            `yaml:"name" json:"name"`
	RestAPI    *RestAPIConfig    `yaml:"restapi" json:"restapi"`
	Ctl        *CtlConfig        `yaml:"ctl" json:"ctl"`
	Etcd       *EtcdConfig       `yaml:"etcd" json:"etcd"`
	Raft       *RaftConfig       `yaml:"raft" json:"raft"`
	Bootstrap  *BootstrapConfig  `yaml:"bootstrap" json:"bootstrap"`
	Postgresql *PostgresqlConfig `yaml:"postgresql" json:"postgresql"`
	Watchdog   *WatchdogConfig   `yaml:"watchdog" json:"watchdog"`
	Tags       *TagsConfig       `yaml:"tags" json:"tags"`
	Log        *LogConfig        `yaml:"log" json:"log"`
}

func DefaultConfig() *Config {
	return &Config{
		Scope:      "database",
		NameSpace:  "/service/",
		Name:       "db1",
		RestAPI:    DefaultRestAPIConfig(),
		Ctl:        DefaultCtlConfig(),
		Etcd:       DefaultEtcdConfig(),
		Raft:       DefaultRaftConfig(),
		Bootstrap:  DefaultBootstrapConfig(),
		Postgresql: DefaultPostgresqlConfig(),
		Watchdog:   DefaultWatchdogConfig(),
		Tags:       DefaultTagsConfig(),
		Log:        DefaultLogConfig(),
	}
}

func LoadConfig(filepath string) (*Config, error) {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	conf := DefaultConfig()
	fileType := common.GetFileType(filepath)
	switch fileType {
	case common.JsonType:
		err = json.Unmarshal([]byte(data), conf)
	case common.YamlType:
	case common.YmlType:
		err = yaml.Unmarshal([]byte(data), conf)
	default:
		return nil, errors.Errorf("the type [%s] of file [%s] is not supported", fileType, filepath)
	}
	if err != nil {
		return nil, errors.WithStack(err)
	}
	// If there are other non-template parameters, set them here
	return conf, nil
}

func WriteConfig(filepath string, conf *Config) error {
	var data []byte
	var err error

	fileType := common.GetFileType(filepath)
	switch fileType {
	case common.JsonType:
		data, err = json.MarshalIndent(conf, "", "\t")
	case common.YamlType:
	case common.YmlType:
		data, err = yaml.Marshal(conf)
	default:
		return errors.Errorf("the type [%s] of file [%s] is not supported", fileType, filepath)
	}

	if err = ioutil.WriteFile(filepath, data, 0644); err != nil {
		return errors.WithStack(err)
	}
	return nil
}
