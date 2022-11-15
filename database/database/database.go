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

package database

import (
	"log"
	"neoha/base/nlog"
	"neoha/config"
	"neoha/database/mysql"
	"neoha/database/postgresql"
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

// setupMysql used to create replication user where not exists
func (d *Database) setupMysql() {
	log := d.log
	log.Info("database.mysql.wait.for.work[maxwait:60s]")
	if err := d.mysql.WaitMysqlWorks(60 * 1000); err != nil {
		log.Error("database.mysql.WaitMysqlWorks.error[%v]", err)
		return
	}

	gtid, _ := d.mysql.GetGTID()
	log.Info("database.mysql.gtid:%+v", gtid)

	log.Info("database.mysql.set.to.READONLY")
	if err := d.mysql.SetReadOnly(); err != nil {
		log.Error("database.mysql.SetReadOnly.error[%+v]", err)
		return
	}

	log.Info("database.mysql.start.slave")
	if err := d.mysql.StartSlave(); err != nil {
		log.Error("database.mysql.start.slave.error[%+v]", err)
	}

	log.Info("database.mysql.check.replication.user...")
	ret, err := d.mysql.CheckUserExists(d.conf.Mysql.ReplUser, "%")
	if err != nil {
		log.Error("database.mysql.CheckUserExists.error[%+v]", err)
		return
	}
	if !ret {
		log.Info("setupMysql.database.mysql.prepare.to.create.replication.user[%v]", d.conf.Mysql.ReplUser)
		user := d.conf.Mysql.ReplUser
		pwd := d.conf.Mysql.ReplPasswd
		if err = d.mysql.CreateReplUserWithoutBinlog(user, pwd); err != nil {
			log.Error("server.mysql.create.replication.user[%v, %v].error[%+v]", user, pwd, err)
		}
	}
	log.Info("server.mysql.setup.done")
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
