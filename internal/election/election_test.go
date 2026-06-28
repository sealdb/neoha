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

package election

import (
	"testing"

	"github.com/sealdb/neoha/internal/base/nlog"
	"github.com/sealdb/neoha/internal/config"
	"github.com/sealdb/neoha/internal/coordination"
	"github.com/sealdb/neoha/internal/database"
	"github.com/sealdb/neoha/internal/election/raft"
	"github.com/stretchr/testify/assert"
)

func TestNewElectionRaft(t *testing.T) {
	conf := config.DefaultConfig()
	conf.Coordination.Provider = coordination.ProviderRaft
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	db := database.NewDatabase(conf.Database, database.MySQL, 1000, log)
	e := NewElection(conf, raft.FOLLOWER, db, database.MySQL, log)
	assert.Equal(t, ElectionRaft, e.etype)
	assert.NotNil(t, e.GetRaft())
	assert.Equal(t, coordination.ProviderRaft, e.Provider())
}

func TestNewElectionEtcd(t *testing.T) {
	conf := config.DefaultConfig()
	conf.Coordination.Provider = coordination.ProviderEtcd
	conf.Coordination.Etcd.Host = "127.0.0.1:2379"
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	db := database.NewDatabase(conf.Database, database.PostgreSQL, 1000, log)
	e := NewElection(conf, raft.FOLLOWER, db, database.PostgreSQL, log)
	assert.Equal(t, ElectionEtcd, e.etype)
	assert.Nil(t, e.GetRaft())
	assert.Equal(t, coordination.ProviderEtcd, e.Provider())
	assert.NotPanics(t, func() { e.Start(); e.Stop() })
}
