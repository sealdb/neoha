/*
 * Copyright 2022-2025 The NeoHA Authors.
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
	"sync"
	"time"

	"neoha/base/common"
	"neoha/base/model"
	"neoha/base/nlog"
	"neoha/config"
)

type (
	// Option enum.
	Option string
)

const (
	// MysqlReadonly enum.
	MysqlReadonly Option = "READONLY"
	// MysqlReadwrite enum.
	MysqlReadwrite Option = "READWRITE"
)

// PingEntry tuple.
type PingEntry struct {
	Relay_Master_Log_File string
}

// Mysql tuple.
type Mysql struct {
	db           *sql.DB
	conf         *config.MysqlConfig
	log          *nlog.Log
	state        model.MysqlState
	option       Option
	mutex        sync.RWMutex
	dbmutex      sync.RWMutex
	mysqlHandler MysqlHandler
	pingEntry    PingEntry
	pingTicker   *time.Ticker
	stats        model.MysqlStats
	downs        int
}

// NewMysql creates the new Mysql.
func NewMysql(conf *config.MysqlConfig, queryTimeout int, log *nlog.Log) *Mysql {
	mysql := &Mysql{
		db:           nil,
		log:          log,
		conf:         conf,
		state:        model.MysqlDead,
		mysqlHandler: getMysqlHandler(conf.Version),
		pingTicker:   common.NormalTicker(conf.PingTimeout),
	}
	mysql.mysqlHandler.SetQueryTimeout(queryTimeout)
	return mysql
}

// SetMysqlHandler used to set the repl handler.
func (m *Mysql) SetMysqlHandler(h MysqlHandler) {
	m.mysqlHandler = h
}

// Ping used to get the master binlog every ping.
func (m *Mysql) Ping() {
	var err error
	var db *sql.DB
	var pe *PingEntry
	log := m.log

	downsLimits := m.conf.AdmitDefeatPingCnt

	if db, err = m.getDB(); err != nil {
		log.Error("mysql[%v].ping.getdb.error[%v].downs:%v,downslimits:%v", m.getConnStr(), err, m.downs, downsLimits)
		if m.downs > downsLimits {
			log.Error("mysql.dead.downs:%v,downslimits:%v", m.downs, downsLimits)
			m.setState(model.MysqlDead)
		}
		m.IncMysqlDowns()
		m.downs++
		return
	}

	if pe, err = m.mysqlHandler.Ping(db); err != nil {
		log.Error("mysql[%v].ping.error[%v].downs:%v,downslimits:%v", m.getConnStr(), err, m.downs, downsLimits)
		if m.downs > downsLimits {
			log.Error("mysql.dead.downs:%v,downslimits:%v", m.downs, downsLimits)
			m.setState(model.MysqlDead)
		}
		m.IncMysqlDowns()
		m.downs++
		return
	}

	// check replication users
	if exists, err := m.mysqlHandler.CheckUserExists(db, m.conf.ReplUser, "%"); err == nil {
		if !exists {
			log.Info("mysql[%v].ping.create.replication.user[%v]", m.getConnStr(), m.conf.ReplUser)
			if err = m.mysqlHandler.CreateReplUserWithoutBinlog(db, m.conf.ReplUser, m.conf.ReplPasswd); err != nil {
				log.Error("server.mysql.create.replication.user[%v].error[%+v]", m.conf.ReplUser, err)
			}
		}
	}

	// reset downs.
	m.downs = 0
	m.setState(model.MysqlAlive)
	m.pingEntry = *pe
}

// GetUUID used to get local uuid.
func (m *Mysql) GetUUID() (string, error) {
	var err error
	var db *sql.DB
	var uuid string
	log := m.log

	if db, err = m.getDB(); err != nil {
		log.Error("mysql.get.local.uuid.error[%v]", err)
		return "", err
	}

	if uuid, err = m.mysqlHandler.GetUUID(db); err != nil {
		log.Error("mysql.get.local.uuid.error[%v]", err)
		return "", err
	}
	log.Info("mysql.get.local.uuid:[%v]", uuid)

	return uuid, nil
}

// GetMasterGTID used to get master binlog info.
func (m *Mysql) GetMasterGTID() (*model.GTID, error) {
	var err error
	var db *sql.DB
	var gtid *model.GTID

	if db, err = m.getDB(); err != nil {
		return nil, err
	}

	if gtid, err = m.mysqlHandler.GetMasterGTID(db); err != nil {
		return nil, err
	}
	return gtid, nil
}

// GetSlaveGTID used to get Relay_Master_Log_File and read_master_binlog_pos.
func (m *Mysql) GetSlaveGTID() (*model.GTID, error) {
	var err error
	var db *sql.DB
	var gtid *model.GTID

	if db, err = m.getDB(); err != nil {
		return nil, err
	}

	if gtid, err = m.mysqlHandler.GetSlaveGTID(db); err != nil {
		return nil, err
	}
	return gtid, nil
}

// getDB get the database connection.
func (m *Mysql) getDB() (*sql.DB, error) {
	var err error
	var db *sql.DB

	m.dbmutex.Lock()
	defer m.dbmutex.Unlock()

	if m.db == nil {
		connstr := fmt.Sprintf("%s:%s@tcp(%s:%d)/", m.conf.Admin, m.conf.Passwd, m.conf.Host, m.conf.Port)
		if db, err = sql.Open("mysql", connstr); err != nil {
			return nil, err
		}
		m.db = db
	}
	return m.db, nil
}

// Get ReplGtidPurged
func (m *Mysql) GetReplGtidPurged() string {
	return m.conf.ReplGtidPurged
}
