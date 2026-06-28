/*
 * Copyright 2022-2026 The NeoHA Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package config

import (
	"testing"

	"github.com/sealdb/neoha/internal/base/common"
	"github.com/stretchr/testify/assert"
)

func TestEffectiveCoordinationFromElection(t *testing.T) {
	conf := DefaultConfig()
	conf.Coordination = nil
	conf.Election.Algo = "raft"

	coord := conf.EffectiveCoordination()
	assert.Equal(t, "raft", coord.Provider)
	assert.NotNil(t, coord.Raft)
}

func TestEffectiveCoordinationPrefersCoordinationBlock(t *testing.T) {
	conf := DefaultConfig()
	conf.Coordination.Provider = "raft"
	conf.Election.Algo = "etcd"

	coord := conf.EffectiveCoordination()
	assert.Equal(t, "raft", coord.Provider)
}

func TestValidateRaftRequiresEndpoint(t *testing.T) {
	conf := DefaultConfig()
	conf.Endpoint = ""
	err := conf.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "endpoint")
}

func TestValidateEtcdRequiresHosts(t *testing.T) {
	conf := DefaultConfig()
	conf.Coordination.Provider = "etcd"
	conf.Election.Algo = "etcd"
	err := conf.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "etcd")
}

func TestValidatePostgreSQLInheritsBootstrapDCS(t *testing.T) {
	conf := DefaultConfig()
	conf.Database.Type = "postgresql"
	conf.Database.Postgresql = DefaultPostgresqlConfig()
	conf.Bootstrap.BootstrapPostgresql.DcsConf.MaximumLagOnFailover = 2048
	conf.Bootstrap.BootstrapPostgresql.DcsConf.UseSlots = true
	conf.Coordination.Provider = "etcd"
	conf.Coordination.Etcd.Host = "127.0.0.1:2379"
	assert.NoError(t, conf.Validate())
	assert.Equal(t, int64(2048), conf.Database.Postgresql.MaximumLagOnFailover)
	assert.True(t, conf.Database.Postgresql.UseSlots)
}

func TestParseConfigWithCoordinationYAML(t *testing.T) {
	data := []byte(`
scope: c1
name: n1
endpoint: 127.0.0.1:9090
coordination:
  provider: raft
  raft:
    meta-datadir: /tmp/raft
database:
  type: mysql
  mysql:
    replication-mode: semi-sync
`)
	conf, err := ParseConfig(data, common.YamlType)
	assert.NoError(t, err)
	assert.Equal(t, "raft", conf.EffectiveCoordination().Provider)
	assert.Equal(t, "raft", conf.Election.Algo)
}
