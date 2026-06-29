/*
 * Copyright 2022-2026 The NeoHA Authors.
 *
 * See the AUTHORS file for a list of contributors.
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

package raft

import (
	"testing"

	"github.com/sealdb/neoha/internal/base/model"
	"github.com/sealdb/neoha/internal/base/nlog"
	"github.com/sealdb/neoha/internal/config"
	"github.com/sealdb/neoha/internal/database"
	"github.com/sealdb/neoha/internal/database/mysql"
	"github.com/stretchr/testify/assert"
)

func TestRaftUsesDriverForPromotable(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	conf := config.DefaultConfig()
	db := database.NewDatabase(conf.Database, database.MySQL, 10000, log)
	r := NewRaft(conf, log, db, database.MySQL, FOLLOWER)

	assert.NotNil(t, r.Driver())
	assert.NotNil(t, r.dbDriver)

	r.mysql.SetMysqlHandler(mysql.NewMockGTIDX1())
	r.mysql.SetState(model.MysqlAlive)
	assert.True(t, r.replicationPromotable())
}

func TestRaftDemoteReadOnlyViaDriver(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	conf := config.DefaultConfig()
	db := database.NewDatabase(conf.Database, database.MySQL, 10000, log)
	r := NewRaft(conf, log, db, database.MySQL, FOLLOWER)

	r.mysql.SetMysqlHandler(mysql.NewMockGTIDX1())
	r.mysql.SetState(model.MysqlAlive)
	// Mock handler: SetReadOnly succeeds without real DB.
	assert.NoError(t, r.demoteReadOnly())
}

func TestRaftDelegateDBApplySkipsDemote(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	conf := config.DefaultConfig()
	conf.HA = &config.HAConfig{DelegateDBApply: true}
	db := database.NewDatabase(conf.Database, database.MySQL, 10000, log)
	r := NewRaft(conf, log, db, database.MySQL, FOLLOWER)

	assert.True(t, r.DelegateDBApply())
	assert.NoError(t, r.demoteReadOnly())
}

func TestRaftDelegateSkipsSemiSyncChangeToMaster(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	conf := config.DefaultConfig()
	conf.HA = &config.HAConfig{DelegateDBApply: true}
	conf.Database.Mysql.ReplMode = model.ReplModeSemiSync
	db := database.NewDatabase(conf.Database, database.MySQL, 10000, log)
	r := NewRaft(conf, log, db, database.MySQL, FOLLOWER)

	assert.NoError(t, r.changeToMaster())
	assert.NoError(t, r.enableReadWrite())
}

func TestRaftDelegateSkipsMGRChangeToMaster(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	conf := config.DefaultConfig()
	conf.HA = &config.HAConfig{DelegateDBApply: true}
	conf.Database.Mysql.ReplMode = model.ReplModeMGR
	db := database.NewDatabase(conf.Database, database.MySQL, 10000, log)
	r := NewRaft(conf, log, db, database.MySQL, FOLLOWER)

	assert.True(t, r.delegateMGRApply())
	assert.NoError(t, r.changeToMaster())
	assert.NoError(t, r.enableReadWrite())
}
