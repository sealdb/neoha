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

package mysql

import (
	"database/sql"
	"fmt"
	"testing"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"github.com/sealdb/neoha/internal/base/model"
	"github.com/sealdb/neoha/internal/base/nlog"
	"github.com/sealdb/neoha/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestIsTooManyConnections(t *testing.T) {
	assert.False(t, isTooManyConnections(nil))
	assert.False(t, isTooManyConnections(fmt.Errorf("connection refused")))

	mysqlErr := &mysqlDriver.MySQLError{Number: erTooManyConnections, Message: "Too many connections"}
	assert.True(t, isTooManyConnections(mysqlErr))
	assert.True(t, isTooManyConnections(errors.WithStack(mysqlErr)))
	assert.True(t, isTooManyConnections(fmt.Errorf("wrapped: %w", mysqlErr)))
	assert.True(t, isTooManyConnections(fmt.Errorf("Error 1040: Too many connections")))
}

func TestConfigureDBPool(t *testing.T) {
	conf := config.DefaultMysqlConfig()
	conf.MaxOpenConns = 1
	conf.MaxIdleConns = 1
	m := NewMysql(conf, 1000, nlog.NewStdLog(nlog.Level(nlog.PANIC)))
	db, err := m.getDB()
	assert.NoError(t, err)
	assert.Equal(t, 1, db.Stats().MaxOpenConnections)
}

func newPingTestMysql(t *testing.T, pingFn func(*sql.DB) (*PingEntry, error)) *Mysql {
	conf := config.DefaultMysqlConfig()
	conf.Host = "127.0.0.1"
	conf.Port = 1
	conf.AdmitDefeatPingCnt = 2
	m := NewMysql(conf, 1000, nlog.NewStdLog(nlog.Level(nlog.PANIC)))
	mock := defaultMockGTID()
	mock.PingFn = pingFn
	m.SetMysqlHandler(mock)
	_, err := m.getDB()
	assert.NoError(t, err)
	m.setState(model.MysqlAlive)
	return m
}

func TestPingTooManyConnectionsIncrementsDowns(t *testing.T) {
	m := newPingTestMysql(t, func(*sql.DB) (*PingEntry, error) {
		return nil, &mysqlDriver.MySQLError{Number: erTooManyConnections, Message: "Too many connections"}
	})
	m.Ping()
	assert.Equal(t, model.MysqlAlive, m.GetState())
	assert.Equal(t, 1, m.downs)
}

func TestPingTooManyConnectionsMarksDead(t *testing.T) {
	m := newPingTestMysql(t, func(*sql.DB) (*PingEntry, error) {
		return nil, &mysqlDriver.MySQLError{Number: erTooManyConnections, Message: "Too many connections"}
	})
	for i := 0; i < 4; i++ {
		m.Ping()
	}
	assert.Equal(t, model.MysqlDead, m.GetState())
	assert.True(t, m.ShouldDeferFailover())
}

func TestFailoverOnTooManyConnectionsEnabled(t *testing.T) {
	m := newPingTestMysql(t, func(*sql.DB) (*PingEntry, error) {
		return nil, &mysqlDriver.MySQLError{Number: erTooManyConnections, Message: "Too many connections"}
	})
	m.conf.FailoverOnTooManyConnections = true
	for i := 0; i < 4; i++ {
		m.Ping()
	}
	assert.Equal(t, model.MysqlDead, m.GetState())
	assert.False(t, m.ShouldDeferFailover())
}

func TestPingFailureIncrementsDowns(t *testing.T) {
	m := newPingTestMysql(t, func(*sql.DB) (*PingEntry, error) {
		return nil, fmt.Errorf("connection refused")
	})
	m.Ping()
	assert.Equal(t, model.MysqlAlive, m.GetState())
	assert.Equal(t, 1, m.downs)
	assert.False(t, m.ShouldDeferFailover())
}

func TestPingRecoveryClearsDeferFailover(t *testing.T) {
	calls := 0
	m := newPingTestMysql(t, func(*sql.DB) (*PingEntry, error) {
		calls++
		if calls <= 4 {
			return nil, &mysqlDriver.MySQLError{Number: erTooManyConnections, Message: "Too many connections"}
		}
		return &PingEntry{}, nil
	})
	for i := 0; i < 4; i++ {
		m.Ping()
	}
	assert.Equal(t, model.MysqlDead, m.GetState())
	m.Ping()
	assert.Equal(t, model.MysqlAlive, m.GetState())
	assert.False(t, m.ShouldDeferFailover())
}
