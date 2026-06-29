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

package database

import (
	"context"
	"log"

	"github.com/sealdb/neoha/internal/base/nlog"
	"github.com/sealdb/neoha/internal/config"
	dbdriver "github.com/sealdb/neoha/internal/database/driver"
	"github.com/sealdb/neoha/internal/database/mysql"
	"github.com/sealdb/neoha/internal/database/postgresql"
)

// DBType identifies the database engine (alias of driver.Engine for legacy callers).
type DBType = dbdriver.Engine

const (
	MySQL      = dbdriver.EngineMySQL
	PostgreSQL = dbdriver.EnginePostgreSQL
	Unknown    = dbdriver.EngineUnknown
)

// Driver re-exports the HA database interface.
type Driver = dbdriver.Driver

type Database struct {
	dbType DBType
	log    *nlog.Log
	conf   *config.DatabaseConfig
	driver Driver
	mysql  *mysql.Mysql
	pg     *postgresql.Postgresql
}

// NewDatabase creates the new database tuple.
func NewDatabase(conf *config.DatabaseConfig, dbType DBType, queryTimeout int, log *nlog.Log) *Database {
	var my *mysql.Mysql
	var pg *postgresql.Postgresql
	var drv Driver

	if dbType == MySQL {
		my = mysql.NewMysql(conf.Mysql, queryTimeout, log)
		drv = mysql.NewDriver(my, conf.Mysql, log)
	} else if dbType == PostgreSQL {
		pg = postgresql.NewPostgresql(conf.Postgresql, queryTimeout, log)
		drv = postgresql.NewDriver(pg, conf.Postgresql, log)
	} else {
		log.Panic("unsupported database")
	}

	return &Database{
		log:    log,
		conf:   conf,
		dbType: dbType,
		driver: drv,
		mysql:  my,
		pg:     pg,
	}
}

// Driver returns the HA database facade (preferred for new code).
func (d *Database) Driver() Driver {
	return d.driver
}

// GetMysql returns the MySQL handle. Deprecated: use Driver(); retained for raft/RPC migration.
func (d *Database) GetMysql() *mysql.Mysql {
	return d.mysql
}

func (d *Database) GetPostgreSQL() *postgresql.Postgresql {
	return d.pg
}

// SetupDB used to create database
func (d *Database) SetupDB() {
	switch d.dbType {
	case MySQL:
		if err := d.driver.SetupBootstrap(context.Background()); err != nil {
			d.log.Error("database.setup.error[%v]", err)
		}
	case PostgreSQL:
		if err := d.driver.SetupBootstrap(context.Background()); err != nil {
			d.log.Error("database.setup.error[%v]", err)
		}
	default:
		log.Panic("unsupported database type")
	}
}

func (d *Database) Start() {
	d.driver.Start()
}

func (d *Database) Stop() {
	d.driver.Stop()
}
