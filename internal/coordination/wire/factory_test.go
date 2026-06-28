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

package wire

import (
	"testing"

	"github.com/sealdb/neoha/internal/base/nlog"
	"github.com/sealdb/neoha/internal/config"
	"github.com/sealdb/neoha/internal/coordination"
	"github.com/sealdb/neoha/internal/database"
	"github.com/sealdb/neoha/internal/election/raft"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCoordinatorEtcd(t *testing.T) {
	conf := config.DefaultConfig()
	conf.Coordination.Provider = coordination.ProviderEtcd
	conf.Coordination.Etcd.Host = "127.0.0.1:2379"
	coord, err := NewCoordinator(conf, nil)
	require.NoError(t, err)
	require.NotNil(t, coord)
	assert.Equal(t, conf.Endpoint, coord.LocalID())
}

func TestNewCoordinatorRaftRequiresInstance(t *testing.T) {
	conf := config.DefaultConfig()
	_, err := NewCoordinator(conf, nil)
	assert.Error(t, err)
}

func TestNewCoordinatorRaft(t *testing.T) {
	conf := config.DefaultConfig()
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	db := database.NewDatabase(conf.Database, database.MySQL, 1000, log)
	r := raft.NewRaft(conf, log, db, database.MySQL, raft.FOLLOWER)
	coord, err := NewCoordinator(conf, r)
	require.NoError(t, err)
	assert.False(t, coord.IsLeader())
}
