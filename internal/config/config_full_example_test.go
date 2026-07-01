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
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func fullExampleWant() *Config {
	want := DefaultConfig()
	want.Scope = "neoha"
	return want
}

func TestLoadFullExampleConfig(t *testing.T) {
	examples := filepath.Join("..", "..", "configs", "examples")
	for _, name := range []string{"neoha-full.yaml", "neoha-full.json"} {
		t.Run(name, func(t *testing.T) {
			conf, err := LoadConfig(filepath.Join(examples, name))
			assert.NoError(t, err)
			assert.Equal(t, fullExampleWant(), conf)
		})
	}
}

func TestFullExampleCoversAllSections(t *testing.T) {
	conf, err := LoadConfig(filepath.Join("..", "..", "configs", "examples", "neoha-full.yaml"))
	assert.NoError(t, err)

	assert.NotNil(t, conf.RestAPI)
	assert.NotNil(t, conf.RestAPI.Authentication)
	assert.NotNil(t, conf.Ctl)
	assert.NotNil(t, conf.Election)
	assert.NotNil(t, conf.Election.Raft)
	assert.NotNil(t, conf.Election.Etcd)
	assert.NotNil(t, conf.Coordination)
	assert.Equal(t, conf.Election.Algo, conf.Coordination.Provider)
	assert.NotNil(t, conf.Bootstrap)
	assert.NotNil(t, conf.Bootstrap.BootstrapPostgresql)
	assert.NotNil(t, conf.Bootstrap.BootstrapPostgresql.DcsConf)
	assert.NotNil(t, conf.Bootstrap.BootstrapPostgresql.DcsConf.StandbyCluster)
	assert.NotNil(t, conf.Bootstrap.BootstrapPostgresql.DcsConf.RecoveryConf)
	assert.NotNil(t, conf.Bootstrap.BootstrapPostgresql.InitDB)
	assert.NotNil(t, conf.Bootstrap.BootstrapMysql)
	assert.NotNil(t, conf.Database)
	assert.NotNil(t, conf.Database.Mysql)
	assert.NotNil(t, conf.Database.Mysql.Backup)
	assert.NotNil(t, conf.Database.Postgresql)
	assert.NotNil(t, conf.Database.Postgresql.Auth)
	assert.NotNil(t, conf.Database.Postgresql.Auth.Rewind)
	assert.False(t, conf.Database.Postgresql.UseSlots)
	assert.False(t, conf.Database.Postgresql.UsePGRewind)
	assert.Equal(t, int64(0), conf.Database.Postgresql.MaximumLagOnFailover)
	assert.NotNil(t, conf.Coordination.Etcd)
	assert.Equal(t, 30, conf.Coordination.Etcd.TTL)
	assert.Equal(t, 10, conf.Coordination.Etcd.LoopWait)
	assert.Equal(t, 10, conf.Coordination.Etcd.RetryTimeout)
	assert.NotNil(t, conf.Watchdog)
	assert.NotNil(t, conf.Tags)
	assert.NotNil(t, conf.HA)
	assert.NotNil(t, conf.HA.PrimaryHooks)
	assert.NotNil(t, conf.Log)
}

// TestRegenerateFullExampleJSON refreshes configs/examples/neoha-full.json from YAML.
// Run: NEOHA_REGEN_CONFIG_EXAMPLES=1 go test ./internal/config/ -run TestRegenerateExampleJSON
func TestRegenerateFullExampleJSON(t *testing.T) {
	regenerateExampleJSON(t, filepath.Join("..", "..", "configs", "examples", "neoha-full.yaml"))
}

func TestRegenerateExampleJSON(t *testing.T) {
	if os.Getenv("NEOHA_REGEN_CONFIG_EXAMPLES") == "" {
		t.Skip("set NEOHA_REGEN_CONFIG_EXAMPLES=1 to regenerate")
	}
	root := filepath.Join("..", "..", "configs", "examples")
	regenerateExampleJSON(t, filepath.Join(root, "neoha-full.yaml"))
	regenerateExampleJSON(t, filepath.Join(root, "mysql", "semisync-node1.yaml"))
	regenerateExampleJSON(t, filepath.Join(root, "mysql", "mgr-mysql80-node1.yaml"))
}

func regenerateExampleJSON(t *testing.T, yamlPath string) {
	t.Helper()
	if os.Getenv("NEOHA_REGEN_CONFIG_EXAMPLES") == "" {
		t.Skip("set NEOHA_REGEN_CONFIG_EXAMPLES=1 to regenerate")
	}
	conf, err := LoadConfig(yamlPath)
	assert.NoError(t, err)
	jsonPath := yamlPath[:len(yamlPath)-len(filepath.Ext(yamlPath))] + ".json"
	assert.NoError(t, WriteConfig(jsonPath, conf))
}
