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

	"github.com/sealdb/neoha/internal/base/model"
)

// MysqlHandler interface.
type MysqlHandler interface {
	SetReplMode(mode model.MysqlReplMode)
	GetReplMode() *model.MysqlReplMode
	SetQueryTimeout(int)

	// check health and return log_bin_basename
	Ping(*sql.DB) (*PingEntry, error)

	// set mysql readonly variable
	SetReadOnly(*sql.DB, bool) error

	// get local UUID
	GetUUID(*sql.DB) (string, error)

	// get GTID from traversal binlog folder and find the newest one
	GetMasterGTID(*sql.DB) (*model.GTID, error)
	GetMGRGTID(*sql.DB) (*model.GTID, error)

	// whether the first GTID is a subset of another one
	GTIDBigger(*sql.DB, string, string) (bool, error)

	// get GTID from SHOW SLAVE STATUS
	GetSlaveGTID(*sql.DB) (*model.GTID, error)

	// start slave io_thread
	StartSlaveIOThread(*sql.DB) error

	// stop slave io_thread
	StopSlaveIOThread(*sql.DB) error

	// start slave
	StartSlave(*sql.DB) error

	// stop slave
	StopSlave(*sql.DB) error

	// use the provided master as the new master
	ChangeMasterTo(*sql.DB, *model.Repl) error

	// check whether local MGR is running
	IsMGRRunningOK(*sql.DB) (bool, error)

	// start group_replication
	StartMGR(*sql.DB) error

	// stop group_replication
	StopMGR(*sql.DB) error

	// use the provided master as the new master
	MGRChangeMasterTo(*sql.DB, *model.Repl) error

	// change a slave to master
	ChangeToMaster(*sql.DB, *model.Repl) error

	// get group replication status
	GetMGRStats(db *sql.DB) ([]map[string]string, error)

	// get group_replication status of local nodes
	GetLocalMGRStat(db *sql.DB) (*model.MGRStatus, error)

	// get group replication master
	GetMGRMasterUUID(db *sql.DB) (string, error)

	// waits until apply relay log completed
	WaitApplyRelayLog(*sql.DB, int, int) error

	// waits until slave replication reaches at least targetGTID
	WaitUntilAfterGTID(*sql.DB, string) error

	// get gtid subtract with slavegtid and master gtid
	GetGTIDSubtract(*sql.DB, string, string) (string, error)

	// set global variables
	SetGlobalSysVar(db *sql.DB, varsql string) error

	// reset master
	ResetMaster(db *sql.DB) error

	// reset slave
	ResetSlave(db *sql.DB) error

	// reset slave all
	ResetSlaveAll(db *sql.DB) error

	// purge binglog to
	PurgeBinlogsTo(*sql.DB, string) error

	// enable master semi sync: wait slave ack
	EnableSemiSyncMaster(db *sql.DB) error

	// disable master semi sync: don't wait slave ack
	DisableSemiSyncMaster(db *sql.DB) error

	// set semi-sync master-timeout
	SetSemiSyncMasterTimeout(db *sql.DB, timeout uint64) error

	//set rpl_semi_master_wait_for_slave_count
	SetSemiWaitSlaveCount(db *sql.DB, count int) error

	// User handlers.
	GetUser(*sql.DB) ([]model.MysqlUser, error)
	CheckUserExists(*sql.DB, string, string) (bool, error)
	CreateUser(*sql.DB, string, string, string, string) error
	DropUser(*sql.DB, string, string) error
	ChangeUserPasswd(*sql.DB, string, string, string) error
	CreateReplUserWithoutBinlog(*sql.DB, string, string) error
	GrantAllPrivileges(*sql.DB, string, string, string, string) error
	GrantNormalPrivileges(*sql.DB, string, string) error
	CreateUserWithPrivileges(db *sql.DB, user, passwd, database, table, host, privs string, ssl string) error
	GrantReplicationPrivileges(*sql.DB, string) error
}

var (
	mysqlHandlers = make(map[string]MysqlHandler)
)

func init() {
	mysqlHandlers["mysql56"] = new(Mysql56)
	mysqlHandlers["mysql57"] = new(Mysql57)
	mysqlHandlers["mysql80"] = new(Mysql80)
}

func getMysqlHandler(name string) MysqlHandler {
	handler, ok := mysqlHandlers[name]
	if !ok {
		return new(Mysql57) // default
	}
	return handler
}
