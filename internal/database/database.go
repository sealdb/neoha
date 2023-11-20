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

package database

import (
	"log"

	"github.com/sealdb/neoha/internal/base/model"
	"github.com/sealdb/neoha/internal/base/nlog"
	"github.com/sealdb/neoha/internal/config"
	"github.com/sealdb/neoha/internal/database/mysql"
	"github.com/sealdb/neoha/internal/database/postgresql"
)

type DBType int

const (
	MySQL DBType = 1 << iota
	PostgreSQL
	Unknown
)

type Database struct {
	dbType DBType
	log    *nlog.Log
	conf   *config.DatabaseConfig
	mysql  *mysql.Mysql
	pg     *postgresql.Postgresql
}

// NewDatabase creates the new database tuple.
func NewDatabase(conf *config.DatabaseConfig, dbType DBType, queryTimeout int, log *nlog.Log) *Database {
	var my *mysql.Mysql = nil
	var pg *postgresql.Postgresql = nil

	if dbType == MySQL {
		my = mysql.NewMysql(conf.Mysql, queryTimeout, log)
	} else if dbType == PostgreSQL {
		pg = postgresql.NewPostgresql(conf.Postgresql, queryTimeout, log)
	} else {
		log.Panic("unsupported database")
	}

	db := &Database{
		log:    log,
		conf:   conf,
		dbType: dbType,
		mysql:  my,
		pg:     pg,
	}
	return db
}

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
		d.setupMysql()
	case PostgreSQL:
		d.setupPostgresql()
	default:
		log.Panic("unsupported database type")
	}
}

// setupMysql waits for mysqld, prepares the replication user on this node (Semi-Sync / MGR),
// sets read-only, then starts replication.
func (d *Database) setupMysql() {
	log := d.log
	log.Info("database.mysql.wait.for.work[maxwait:60s]")
	if err := d.mysql.WaitMysqlWorks(60 * 1000); err != nil {
		log.Error("database.mysql.WaitMysqlWorks.error[%v]", err)
		return
	}

	gtid, _ := d.mysql.GetGTID()
	log.Info("database.mysql.gtid:%+v", gtid)

	if err := d.ensureReplUser(); err != nil {
		return
	}

	log.Info("database.mysql.set.to.READONLY")
	if err := d.mysql.SetReadOnly(); err != nil {
		log.Error("database.mysql.SetReadOnly.error[%+v]", err)
		return
	}

	if d.conf.Mysql.ReplMode != model.ReplModeMGR {
		d.setupMysqlSemiSync()
	}
	// MGR bootstrap/join is driven by Raft (prepareSettingsMGR / follower heartbeat).
	log.Info("server.mysql.setup.done")
}

// ensureReplUser creates the configured replication account with sql_log_bin=0 on every NeoHA node
// before change master / MGR recovery needs it (avoids local GTIDs on joiners).
func (d *Database) ensureReplUser() error {
	log := d.log
	log.Info("database.mysql.check.replication.user...")
	ret, err := d.mysql.CheckUserExists(d.conf.Mysql.ReplUser, "%")
	if err != nil {
		log.Error("database.mysql.CheckUserExists.error[%+v]", err)
		return err
	}
	if !ret {
		log.Info("setupMysql.database.mysql.prepare.to.create.replication.user[%v]", d.conf.Mysql.ReplUser)
		user := d.conf.Mysql.ReplUser
		pwd := d.conf.Mysql.ReplPasswd
		if err = d.mysql.CreateReplUserWithoutBinlog(user, pwd); err != nil {
			log.Error("server.mysql.create.replication.user[%v, %v].error[%+v]", user, pwd, err)
			return err
		}
	}
	return nil
}

func (d *Database) setupMysqlSemiSync() {
	log := d.log
	log.Info("database.mysql.start.slave")
	if err := d.mysql.StartSlave(); err != nil {
		log.Error("database.mysql.start.slave.error[%+v]", err)
	}
}

func (d *Database) setupPostgresql() {

}

func (d *Database) Start() {
	switch d.dbType {
	case MySQL:
		d.mysql.PingStart()
	case PostgreSQL:
		// TODO: start PostgreSQL ping
	default:
		log.Panic("unsupported database type")
	}
}

func (d *Database) Stop() {
	switch d.dbType {
	case MySQL:
		d.mysql.PingStop()
	case PostgreSQL:
		// TODO: stop PostgreSQL ping
	default:
		log.Panic("unsupported database type")
	}
}
