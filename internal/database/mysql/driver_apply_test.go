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

package mysql

import (
	"context"
	"testing"

	"github.com/sealdb/neoha/internal/base/model"
	"github.com/sealdb/neoha/internal/base/nlog"
	"github.com/sealdb/neoha/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestDriverApplyPrimarySemiSync(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	conf := config.DefaultMysqlConfig()
	conf.ReplMode = model.ReplModeSemiSync
	m := NewMysql(conf, 10000, log)
	m.SetMysqlHandler(NewMockGTIDX1())
	m.SetState(model.MysqlAlive)

	d := NewDriver(m, conf, log)
	assert.NoError(t, d.ApplyPrimary(context.Background()))
	assert.Equal(t, MysqlReadwrite, m.GetOption())
}

func TestDriverApplyPrimaryMGR(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	conf := config.DefaultMysqlConfig()
	conf.ReplMode = model.ReplModeMGR
	m := NewMysql(conf, 10000, log)
	mock := NewMockGTIDX1ForMGR()
	mock.GetLocalMGRStatFn = DefaultGetLocalMGRStat0
	m.SetMysqlHandler(mock)
	m.SetState(model.MysqlAlive)

	d := NewDriver(m, conf, log)
	assert.NoError(t, d.ApplyPrimary(context.Background()))
	assert.Equal(t, MysqlReadonly, m.GetOption())
	assert.True(t, d.mgrAlreadyPrimary())
}

func TestDriverMGRClusterWritableReady(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	conf := config.DefaultMysqlConfig()
	conf.ReplMode = model.ReplModeMGR
	m := NewMysql(conf, 10000, log)
	mock := NewMockGTIDX1MGRCaughtUp()
	mock.GetMGRStatsFn = DefaultGetMGRStats2
	mock.GetMGRMasterUUIDFn = DefaultGetMGRMasterUUID0
	m.SetMysqlHandler(mock)
	m.SetState(model.MysqlAlive)

	d := NewDriver(m, conf, log)
	ready, err := d.MGRClusterWritableReady(context.Background())
	assert.NoError(t, err)
	assert.True(t, ready)
}
