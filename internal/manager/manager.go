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

package manager

import (
	"log"
	"github.com/sealdb/neoha/internal/base/nlog"
	"github.com/sealdb/neoha/internal/config"
	"github.com/sealdb/neoha/internal/database"
	"github.com/sealdb/neoha/internal/manager/mysqld"
)

type Manager struct {
	dbType database.DBType
	log    *nlog.Log
	conf   *config.DatabaseConfig
	mysqld *mysqld.Mysqld
	//pg *postgres.Postmaster
}

// NewNanager creates the new manager tuple.
func NewNanager(conf *config.DatabaseConfig, dbType database.DBType, log *nlog.Log) *Manager {
	var my *mysqld.Mysqld = nil
	// var pg

	if dbType == database.MySQL {
		my = mysqld.NewMysqld(conf.Mysql.Backup, log)
	} else if dbType == database.PostgreSQL {
		//
	} else {
		log.Panic("unsupported database")
	}

	manager := &Manager{
		log:    log,
		conf:   conf,
		dbType: dbType,
		mysqld: my,
		//pg:     pg,
	}
	return manager
}

func (m *Manager) SetupManager() {
	switch m.dbType {
	case database.MySQL:
		m.setupMysqld()
	case database.PostgreSQL:
		m.setupPostmaster()
	default:
		log.Panic("unsupported database type")
	}
}

// setupMysqld used to start mysqld and wait for it works
func (m *Manager) setupMysqld() {
	if m.conf.Mysql.MonitorDisabled {
		return
	}

	log := m.log
	log.Info("manager.prepare.setup.mysqlserver")
	if err := m.mysqld.StartMysqld(); err != nil {
		log.Error("manager.mysqlserver.start.error[%v]", err)
		return
	}
	log.Info("manager.mysqlserver.setup.done")
}

// setupPostmaster used to start postmaster and wait for it works
func (m *Manager) setupPostmaster() {

}

func (m *Manager) Start() {
	switch m.dbType {
	case database.MySQL:
		if !m.conf.Mysql.MonitorDisabled {
			m.mysqld.MonitorStart()
		}
	case database.PostgreSQL:
		// TODO: start PostgreSQL Monitor
	default:
		log.Panic("unsupported database type")
	}
}

func (m *Manager) Stop() {
	switch m.dbType {
	case database.MySQL:
		m.mysqld.MonitorStop()
	case database.PostgreSQL:
		// TODO: stop PostgreSQL Monitor
	default:
		log.Panic("unsupported database type")
	}
}

func (m *Manager) GetMysqld() *mysqld.Mysqld {
	return m.mysqld
}

func (m *Manager) SetMysqld(mysqld *mysqld.Mysqld) {
	m.mysqld = mysqld
}
