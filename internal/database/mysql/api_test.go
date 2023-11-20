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
	"fmt"
	"testing"

	"github.com/sealdb/neoha/internal/base/model"
	"github.com/sealdb/neoha/internal/base/nlog"
	"github.com/sealdb/neoha/internal/config"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func TestSetReadOnly(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	// log
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, 10000, log)
	mysql.db = db

	queryList := []string{
		"SET GLOBAL read_only = 1",
		"SET GLOBAL super_read_only = 1",
		"SET GLOBAL read_only = 0",
		"SET GLOBAL super_read_only = 0",
	}
	mock.ExpectExec(queryList[0]).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(queryList[1]).WillReturnResult(sqlmock.NewResult(1, 1))
	err = mysql.SetReadOnly()
	assert.Nil(t, err)

	mock.ExpectExec(queryList[2]).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(queryList[3]).WillReturnResult(sqlmock.NewResult(1, 1))
	err = mysql.SetReadWrite()
	assert.Nil(t, err)
}

func TestStartSlaveIOThread(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	// log
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, 10000, log)
	mysql.db = db

	query := "START SLAVE IO_THREAD"
	mock.ExpectExec(query).WillReturnResult(sqlmock.NewResult(1, 1))
	err = mysql.StartSlaveIOThread()
	assert.Nil(t, err)
}

func TestStopSlaveIOThread(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	// log
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, 10000, log)
	mysql.db = db

	query := "STOP SLAVE IO_THREAD"
	mock.ExpectExec(query).WillReturnResult(sqlmock.NewResult(1, 1))
	err = mysql.StopSlaveIOThread()
	assert.Nil(t, err)
}

func TestStartSlave(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	// log
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, 10000, log)
	mysql.db = db

	query := "START SLAVE"
	mock.ExpectExec(query).WillReturnResult(sqlmock.NewResult(1, 1))
	err = mysql.StartSlave()
	assert.Nil(t, err)
}

func TestStopSlave(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	// log
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, 10000, log)
	mysql.db = db

	query := "STOP SLAVE"
	mock.ExpectExec(query).WillReturnResult(sqlmock.NewResult(1, 1))
	err = mysql.StopSlave()
	assert.Nil(t, err)
}

func TestChangeMasterTo(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	// log
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, 10000, log)
	mysql.db = db

	queryList := []string{"STOP SLAVE",
		`CHANGE MASTER TO MASTER_HOST = '127.0.0.1', MASTER_PORT = 3306, MASTER_USER = 'repl', MASTER_PASSWORD = 'repl', MASTER_AUTO_POSITION = 1`,
		"START SLAVE",
	}

	mock.ExpectExec(queryList[0]).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(queryList[1]).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(queryList[2]).WillReturnResult(sqlmock.NewResult(1, 1))
	repl := mysql.GetRepl()
	err = mysql.ChangeMasterTo(&repl)
	assert.Nil(t, err)
}

func TestMGRChangeMasterTo(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	// log
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, 10000, log)
	mysql.db = db

	queryList := []string{"STOP GROUP_REPLICATION",
		`CHANGE MASTER TO MASTER_USER = 'repl', MASTER_PASSWORD = 'repl' FOR CHANNEL 'group_replication_recovery'`,
		"SET GLOBAL group_replication_bootstrap_group=OFF",
		"START GROUP_REPLICATION",
	}

	mock.ExpectExec(queryList[0]).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(queryList[1]).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(queryList[2]).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(queryList[3]).WillReturnResult(sqlmock.NewResult(1, 1))
	repl := mysql.GetRepl()
	err = mysql.MGRChangeMasterTo(&repl)
	assert.Nil(t, err)
}

func TestGTIDSetUnion(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, 10000, log)

	// gtid1 is null
	{
		gtid1 := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-102:1000-1002"
		gtid2 := ""
		want := gtid1
		got := mysql.GTIDSetUnion(gtid1, gtid2)
		assert.Equal(t, want, got)
	}

	// gtid2 is null
	{
		gtid1 := ""
		gtid2 := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-102:1000-1002"
		want := gtid2
		got := mysql.GTIDSetUnion(gtid1, gtid2)
		assert.Equal(t, want, got)
	}

	// gtid1 and gtid2 are not null
	{
		{
			gtid1 := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-3"
			gtid2 := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:7"
			want := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-3:7"
			got := mysql.GTIDSetUnion(gtid1, gtid2)
			assert.Equal(t, want, got)
		}

		{
			gtid1 := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb:7"
			gtid2 := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-3"
			want := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-3,bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb:7"
			got := mysql.GTIDSetUnion(gtid1, gtid2)
			assert.Equal(t, want, got)
		}

		{
			gtid1 := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:2-4:6:8-9:12,bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb:7"
			gtid2 := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-3:9-11:13"
			want := ("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-4:6:8-13,bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb:7")
			got := mysql.GTIDSetUnion(gtid1, gtid2)
			assert.Equal(t, want, got)
		}

		{
			gtid1 := "cccccccc-cccc-cccc-cccc-cccccccccccc:5,aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:2-4:6:8-9:12,bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb:7,cccccccc-cccc-cccc-cccc-cccccccccccc:1,bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb:7,cccccccc-cccc-cccc-cccc-cccccccccccc:9,cccccccc-cccc-cccc-cccc-cccccccccccc:4:1-2"
			gtid2 := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-3:9-11:13,cccccccc-cccc-cccc-cccc-cccccccccccc:9:1-2"
			want := ("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-4:6:8-13,bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb:7,cccccccc-cccc-cccc-cccc-cccccccccccc:1-2:4-5:9")
			got := mysql.GTIDSetUnion(gtid1, gtid2)
			assert.Equal(t, want, got)
		}
	}
}

func TestMGRChangeToMaster(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	// log
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	conf := config.DefaultMysqlConfig()
	conf.ReplMode = model.ReplModeMGR
	mysql := NewMysql(conf, 10000, log)
	mysql.db = db

	queryList := []string{"STOP GROUP_REPLICATION",
		`CHANGE MASTER TO MASTER_USER = 'repl', MASTER_PASSWORD = 'repl' FOR CHANNEL 'group_replication_recovery'`,
		"SET GLOBAL group_replication_bootstrap_group=ON",
		"START GROUP_REPLICATION",
		"SET GLOBAL group_replication_bootstrap_group=OFF",
	}

	mock.ExpectExec(queryList[0]).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(queryList[1]).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(queryList[2]).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(queryList[3]).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(queryList[4]).WillReturnResult(sqlmock.NewResult(1, 1))
	repl := mysql.GetRepl()
	err = mysql.ChangeToMaster(&repl)
	assert.Nil(t, err)
}

func TestChangeToMaster(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	// log
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, 10000, log)
	mysql.db = db

	queryList := []string{"STOP SLAVE",
		"RESET SLAVE ALL",
	}

	mock.ExpectExec(queryList[0]).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(queryList[1]).WillReturnResult(sqlmock.NewResult(1, 1))

	master := model.Repl{
		Master_Host:   "localhost",
		Master_Port:   123,
		Repl_User:     "username",
		Repl_Password: "password",
	}

	err = mysql.ChangeToMaster(&master)
	assert.Nil(t, err)
}

func TestWaitApplyRelayLog(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	// log
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	conf := config.DefaultMysqlConfig()
	conf.ReplMode = model.ReplModeMGR
	mysql := NewMysql(conf, 10000, log)
	mysql.db = db

	query := "SHOW MASTER STATUS"
	columns := []string{"File",
		"Position",
		"Binlog_Do_DB",
		"Binlog_Ignore_DB",
		"Executed_Gtid_Set"}
	mockRows := sqlmock.NewRows(columns).AddRow("mysql-bin.000001",
		"147",
		"",
		"",
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-159")
	mock.ExpectQuery(query).WillReturnRows(mockRows)

	query = "SELECT RECEIVED_TRANSACTION_SET,LAST_ERROR_MESSAGE FROM performance_schema.replication_connection_status WHERE CHANNEL_NAME='group_replication_applier'"
	columns = []string{"RECEIVED_TRANSACTION_SET",
		"LAST_ERROR_MESSAGE"}
	mockRows = sqlmock.NewRows(columns).AddRow("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:154-159",
		"")
	mock.ExpectQuery(query).WillReturnRows(mockRows)

	// normal
	{
		query = "SELECT GTID_SUBTRACT\\('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:154-159', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-159'\\) RES"
		columns = []string{"RES"}
		mockRows = sqlmock.NewRows(columns).AddRow("")
		mock.ExpectQuery(query).WillReturnRows(mockRows)
		err = mysql.WaitApplyRelayLog(2, 1)
		assert.Nil(t, err)
	}

	// error
	{
		query = "SELECT GTID_SUBTRACT\\('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:154-160', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-159'\\) RES"
		columns = []string{"RES"}
		mockRows = sqlmock.NewRows(columns).AddRow("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:160")
		mock.ExpectQuery(query).WillReturnRows(mockRows)
		err = mysql.WaitApplyRelayLog(1, 1)
		assert.NotNil(t, err)
	}
}

func TestWaitUntilAfterGTID(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	// log
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, 10000, log)
	mysql.db = db

	query := "SELECT WAIT_UNTIL_SQL_THREAD_AFTER_GTIDS\\('1'\\)"
	mock.ExpectExec(query).WillReturnResult(sqlmock.NewResult(1, 1))
	err = mysql.WaitUntilAfterGTID("1")
	assert.Nil(t, err)
}

func TestGetLocalGTID(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	//log
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, 10000, log)
	mysql.db = db

	query := "SELECT @@SERVER_UUID"
	columns := []string{"@@SERVER_UUID"}
	mockRows := sqlmock.NewRows(columns).AddRow("84030605-66aa-11e6-9465-52540e7fd51c")
	mock.ExpectQuery(query).WillReturnRows(mockRows)

	want := "84030605-66aa-11e6-9465-52540e7fd51c:1-160"
	got, err := mysql.GetLocalGTID("84030605-66aa-11e6-9465-52540e7fd51c:1-160, 84030605-66bb-11e6-9465-52540e7fd51c:1-160")
	assert.Equal(t, want, got)
}

func TestCheckGTID(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()
	var GTID1, GTID2 model.GTID

	//log
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, 10000, log)
	mysql.db = db

	// local is a normal follower, leader Executed_GTID_Set is ""
	{
		query := "SELECT @@SERVER_UUID"
		columns := []string{"@@SERVER_UUID"}
		mockRows := sqlmock.NewRows(columns).AddRow("84030605-66aa-11e6-9465-52540e7fd51c")
		mock.ExpectQuery(query).WillReturnRows(mockRows)

		GTID1 = model.GTID{
			Executed_GTID_Set: "84030605-66aa-11e6-9465-52540e7fd51c:154-160",
		}
		GTID2 = model.GTID{
			Executed_GTID_Set: "",
		}

		want := false
		got := mysql.CheckGTID(&GTID1, &GTID2)

		assert.Equal(t, want, got)
	}

	// local is a normal follower Executed_GTID_Set is "", leader Executed_GTID_Set is ""
	{
		GTID1 = model.GTID{
			Executed_GTID_Set: "",
		}
		GTID2 = model.GTID{
			Executed_GTID_Set: "",
		}

		want := false
		got := mysql.CheckGTID(&GTID1, &GTID2)

		assert.Equal(t, want, got)
	}

	// local is a normal follower Executed_GTID_Set is "", leader do some dml
	{
		GTID1 = model.GTID{
			Executed_GTID_Set: "",
		}

		query := "SELECT @@SERVER_UUID"
		columns := []string{"@@SERVER_UUID"}
		mockRows := sqlmock.NewRows(columns).AddRow("84030605-66aa-11e6-9465-52540e7fd51c")
		mock.ExpectQuery(query).WillReturnRows(mockRows)

		GTID2 = model.GTID{
			Executed_GTID_Set: "84030605-66aa-11e6-9465-52540e7fd51c:1-160",
		}

		want := false
		got := mysql.CheckGTID(&GTID1, &GTID2)

		assert.Equal(t, want, got)
	}

	// local is a leader bug sprain, remote has leader but has none write
	{
		query := "SELECT @@SERVER_UUID"
		columns := []string{"@@SERVER_UUID"}
		mockRows := sqlmock.NewRows(columns).AddRow("84030605-66aa-11e6-9465-52540e7fd51c")
		mock.ExpectQuery(query).WillReturnRows(mockRows)

		GTID1 = model.GTID{
			Executed_GTID_Set: "84030605-66aa-11e6-9465-52540e7fd51c:1-160",
		}

		query = "SELECT @@SERVER_UUID"
		columns = []string{"@@SERVER_UUID"}
		mockRows = sqlmock.NewRows(columns).AddRow("84030605-66aa-11e6-9465-52540e7fd51c")
		mock.ExpectQuery(query).WillReturnRows(mockRows)

		GTID2 = model.GTID{
			Executed_GTID_Set: "84030605-66aa-11e6-9465-52540e7fd51c:1-160",
		}

		query = "SELECT GTID_SUBTRACT\\('84030605-66aa-11e6-9465-52540e7fd51c:1-160','84030605-66aa-11e6-9465-52540e7fd51c:1-160'\\) as gtid_sub"
		columns = []string{"gtid_sub"}
		mockRows = sqlmock.NewRows(columns).AddRow("")
		mock.ExpectQuery(query).WillReturnRows(mockRows)

		want := false
		got := mysql.CheckGTID(&GTID1, &GTID2)
		assert.Equal(t, want, got)
	}

	// local is a leader bug sprain, remote has leader has writen
	{
		query := "SELECT @@SERVER_UUID"
		columns := []string{"@@SERVER_UUID"}
		mockRows := sqlmock.NewRows(columns).AddRow("84030605-66aa-11e6-9465-52540e7fd51c")
		mock.ExpectQuery(query).WillReturnRows(mockRows)

		GTID1 = model.GTID{
			Executed_GTID_Set: "84030605-66aa-11e6-9465-52540e7fd51c:1-160",
		}

		query = "SELECT @@SERVER_UUID"
		columns = []string{"@@SERVER_UUID"}
		mockRows = sqlmock.NewRows(columns).AddRow("84030605-66aa-11e6-9465-52540e7fd51c")
		mock.ExpectQuery(query).WillReturnRows(mockRows)

		GTID2 = model.GTID{
			Executed_GTID_Set: "84030605-66aa-11e6-9465-52540e7fd51c:1-160, 84030605-77bb-11e6-9465-52540e7fd51c:1-10",
		}

		query = "SELECT GTID_SUBTRACT\\('84030605-66aa-11e6-9465-52540e7fd51c:1-160','84030605-66aa-11e6-9465-52540e7fd51c:1-160'\\) as gtid_sub"
		columns = []string{"gtid_sub"}
		mockRows = sqlmock.NewRows(columns).AddRow("")
		mock.ExpectQuery(query).WillReturnRows(mockRows)

		want := false
		got := mysql.CheckGTID(&GTID1, &GTID2)

		assert.Equal(t, want, got)
	}

	// local is a leader bug sprain and localcommitted, remote has leader has writen
	{
		query := "SELECT @@SERVER_UUID"
		columns := []string{"@@SERVER_UUID"}
		mockRows := sqlmock.NewRows(columns).AddRow("84030605-66aa-11e6-9465-52540e7fd51c")
		mock.ExpectQuery(query).WillReturnRows(mockRows)

		GTID1 = model.GTID{
			Executed_GTID_Set: "84030605-66aa-11e6-9465-52540e7fd51c:1-161",
		}

		query = "SELECT @@SERVER_UUID"
		columns = []string{"@@SERVER_UUID"}
		mockRows = sqlmock.NewRows(columns).AddRow("84030605-66aa-11e6-9465-52540e7fd51c")
		mock.ExpectQuery(query).WillReturnRows(mockRows)

		GTID2 = model.GTID{
			Executed_GTID_Set: "84030605-66aa-11e6-9465-52540e7fd51c:1-160, 84030605-77bb-11e6-9465-52540e7fd51c:1-10",
		}

		query = "SELECT GTID_SUBTRACT\\('84030605-66aa-11e6-9465-52540e7fd51c:1-161','84030605-66aa-11e6-9465-52540e7fd51c:1-160'\\) as gtid_sub"
		columns = []string{"gtid_sub"}
		mockRows = sqlmock.NewRows(columns).AddRow("84030605-66aa-11e6-9465-52540e7fd51c:161")
		mock.ExpectQuery(query).WillReturnRows(mockRows)

		want := true
		got := mysql.CheckGTID(&GTID1, &GTID2)

		assert.Equal(t, want, got)
	}
}

func TestMGRGTIDGreaterThan(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	// log
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, 10000, log)
	mysql.db = db
	mysql.mysqlHandler.SetReplMode(model.ReplModeMGR)

	// 1. the local Retrieved_GTID_Set is null
	{
		GTID := model.GTID{Master_Log_File: "mysql-bin.000001",
			Read_Master_Log_Pos: 148,
			Retrieved_GTID_Set:  "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:154-160",
			Executed_GTID_Set:   "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-159",
			Txns_Behind_Master:  "1",
			Last_Error_Message:  ""}

		query := "SHOW MASTER STATUS"
		columns := []string{"File",
			"Position",
			"Binlog_Do_DB",
			"Binlog_Ignore_DB",
			"Executed_Gtid_Set"}

		mockRows := sqlmock.NewRows(columns).AddRow("mysql-bin.000001",
			"148",
			"",
			"",
			"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-159")
		mock.ExpectQuery(query).WillReturnRows(mockRows)

		query = "SELECT RECEIVED_TRANSACTION_SET,LAST_ERROR_MESSAGE FROM performance_schema.replication_connection_status WHERE CHANNEL_NAME='group_replication_applier'"
		columns = []string{"RECEIVED_TRANSACTION_SET",
			"LAST_ERROR_MESSAGE"}
		mockRows = sqlmock.NewRows(columns).AddRow("",
			"")
		mock.ExpectQuery(query).WillReturnRows(mockRows)

		query = "SELECT GTID_SUBTRACT\\('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-159', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-160'\\) RES"
		columns = []string{"RES"}
		mockRows = sqlmock.NewRows(columns).AddRow("")
		mock.ExpectQuery(query).WillReturnRows(mockRows)

		want := false
		got, _, err := mysql.GTIDGreaterThan(&GTID)
		assert.Nil(t, err)
		assert.Equal(t, want, got)
	}

	// 2. the local Retrieved_GTID_Set is bigger than specified
	{
		// 2.1. the Executed_GTID_Set includes more GTIDs
		{
			GTID := model.GTID{Master_Log_File: "mysql-bin.000001",
				Read_Master_Log_Pos: 148,
				Retrieved_GTID_Set:  "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-160:1000",
				Executed_GTID_Set:   "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-160:1000-1010",
				Txns_Behind_Master:  "10",
				Last_Error_Message:  ""}

			query := "SHOW MASTER STATUS"
			columns := []string{"File",
				"Position",
				"Binlog_Do_DB",
				"Binlog_Ignore_DB",
				"Executed_Gtid_Set"}

			mockRows := sqlmock.NewRows(columns).AddRow("mysql-bin.000001",
				"148",
				"",
				"",
				"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-160:1000-1005")
			mock.ExpectQuery(query).WillReturnRows(mockRows)

			query = "SELECT RECEIVED_TRANSACTION_SET,LAST_ERROR_MESSAGE FROM performance_schema.replication_connection_status WHERE CHANNEL_NAME='group_replication_applier'"
			columns = []string{"RECEIVED_TRANSACTION_SET",
				"LAST_ERROR_MESSAGE"}
			mockRows = sqlmock.NewRows(columns).AddRow("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-160:1000-1006",
				"")
			mock.ExpectQuery(query).WillReturnRows(mockRows)

			query = "SELECT GTID_SUBTRACT\\('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-160:1000-1006', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-160:1000-1010'\\) RES"
			columns = []string{"RES"}
			mockRows = sqlmock.NewRows(columns).AddRow("")
			mock.ExpectQuery(query).WillReturnRows(mockRows)

			want := false
			got, _, err := mysql.GTIDGreaterThan(&GTID)
			assert.Nil(t, err)
			assert.Equal(t, want, got)
		}

		// 2.2. the Executed_GTID_Set includes same GTIDs
		{
			GTID := model.GTID{Master_Log_File: "mysql-bin.000001",
				Read_Master_Log_Pos: 148,
				Retrieved_GTID_Set:  "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-160:1000:10000-10001",
				Executed_GTID_Set:   "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-160:1000:10000-10001",
				Txns_Behind_Master:  "0",
				Last_Error_Message:  ""}

			query := "SHOW MASTER STATUS"
			columns := []string{"File",
				"Position",
				"Binlog_Do_DB",
				"Binlog_Ignore_DB",
				"Executed_Gtid_Set"}

			mockRows := sqlmock.NewRows(columns).AddRow("mysql-bin.000001",
				"148",
				"",
				"",
				"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-160:1000-1002:10000-10001")
			mock.ExpectQuery(query).WillReturnRows(mockRows)

			query = "SELECT RECEIVED_TRANSACTION_SET,LAST_ERROR_MESSAGE FROM performance_schema.replication_connection_status WHERE CHANNEL_NAME='group_replication_applier'"
			columns = []string{"RECEIVED_TRANSACTION_SET",
				"LAST_ERROR_MESSAGE"}
			mockRows = sqlmock.NewRows(columns).AddRow("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-160:1000-1002:10000-10001",
				"")
			mock.ExpectQuery(query).WillReturnRows(mockRows)

			query = "SELECT GTID_SUBTRACT\\('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-160:1000-1002:10000-10001', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-160:1000:10000-10001'\\) RES"
			columns = []string{"RES"}
			mockRows = sqlmock.NewRows(columns).AddRow("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1001-1002")
			mock.ExpectQuery(query).WillReturnRows(mockRows)

			want := true
			got, _, err := mysql.GTIDGreaterThan(&GTID)
			assert.Nil(t, err)
			assert.Equal(t, want, got)
		}

		// 2.3. the Executed_GTID_Set includes less GTIDs
		{
			GTID := model.GTID{Master_Log_File: "mysql-bin.000001",
				Read_Master_Log_Pos: 148,
				Retrieved_GTID_Set:  "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-160:1000-1010",
				Executed_GTID_Set:   "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-150:1000-1001",
				Txns_Behind_Master:  "19",
				Last_Error_Message:  ""}

			query := "SHOW MASTER STATUS"
			columns := []string{"File",
				"Position",
				"Binlog_Do_DB",
				"Binlog_Ignore_DB",
				"Executed_Gtid_Set"}

			mockRows := sqlmock.NewRows(columns).AddRow("mysql-bin.000001",
				"148",
				"",
				"",
				"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-100")
			mock.ExpectQuery(query).WillReturnRows(mockRows)

			query = "SELECT RECEIVED_TRANSACTION_SET,LAST_ERROR_MESSAGE FROM performance_schema.replication_connection_status WHERE CHANNEL_NAME='group_replication_applier'"
			columns = []string{"RECEIVED_TRANSACTION_SET",
				"LAST_ERROR_MESSAGE"}
			mockRows = sqlmock.NewRows(columns).AddRow("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-160:1000-1020",
				"")
			mock.ExpectQuery(query).WillReturnRows(mockRows)

			query = "SELECT GTID_SUBTRACT\\('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-160:1000-1020', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-160:1000-1010'\\) RES"
			columns = []string{"RES"}
			mockRows = sqlmock.NewRows(columns).AddRow("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1011-1020")
			mock.ExpectQuery(query).WillReturnRows(mockRows)

			want := true
			got, _, err := mysql.GTIDGreaterThan(&GTID)
			assert.Nil(t, err)
			assert.Equal(t, want, got)
		}
	}

	// 2. the local Retrieved_GTID_Set is smaller than specified
	{
		// 2.1. the Executed_GTID_Set includes more GTIDs
		{
			GTID := model.GTID{Master_Log_File: "mysql-bin.000001",
				Read_Master_Log_Pos: 148,
				Retrieved_GTID_Set:  "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-160:1000-1010:10000",
				Executed_GTID_Set:   "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-160:1000-1010:10000:100000-100020",
				Txns_Behind_Master:  "0",
				Last_Error_Message:  ""}

			query := "SHOW MASTER STATUS"
			columns := []string{"File",
				"Position",
				"Binlog_Do_DB",
				"Binlog_Ignore_DB",
				"Executed_Gtid_Set"}

			mockRows := sqlmock.NewRows(columns).AddRow("mysql-bin.000001",
				"148",
				"",
				"",
				"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-160:1000-1010:10000-10020:100000-100020")
			mock.ExpectQuery(query).WillReturnRows(mockRows)

			query = "SELECT RECEIVED_TRANSACTION_SET,LAST_ERROR_MESSAGE FROM performance_schema.replication_connection_status WHERE CHANNEL_NAME='group_replication_applier'"
			columns = []string{"RECEIVED_TRANSACTION_SET",
				"LAST_ERROR_MESSAGE"}
			mockRows = sqlmock.NewRows(columns).AddRow("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-160:1000:10000",
				"")
			mock.ExpectQuery(query).WillReturnRows(mockRows)

			query = "SELECT GTID_SUBTRACT\\('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-160:1000-1010:10000-10020:100000-100020', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-160:1000-1010:10000:100000-100020'\\) RES"
			columns = []string{"RES"}
			mockRows = sqlmock.NewRows(columns).AddRow("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:10001-10020")
			mock.ExpectQuery(query).WillReturnRows(mockRows)

			want := true
			got, _, err := mysql.GTIDGreaterThan(&GTID)
			assert.Nil(t, err)
			assert.Equal(t, want, got)
		}

		// 2.2. the Executed_GTID_Set includes same GTIDs
		{
			GTID := model.GTID{Master_Log_File: "mysql-bin.000001",
				Read_Master_Log_Pos: 148,
				Retrieved_GTID_Set:  "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-160:1000:10000-10001",
				Executed_GTID_Set:   "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-160:1000:10000-10001",
				Txns_Behind_Master:  "0",
				Last_Error_Message:  ""}

			query := "SHOW MASTER STATUS"
			columns := []string{"File",
				"Position",
				"Binlog_Do_DB",
				"Binlog_Ignore_DB",
				"Executed_Gtid_Set"}

			mockRows := sqlmock.NewRows(columns).AddRow("mysql-bin.000001",
				"148",
				"",
				"",
				"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-160:10000-10001")
			mock.ExpectQuery(query).WillReturnRows(mockRows)

			query = "SELECT RECEIVED_TRANSACTION_SET,LAST_ERROR_MESSAGE FROM performance_schema.replication_connection_status WHERE CHANNEL_NAME='group_replication_applier'"
			columns = []string{"RECEIVED_TRANSACTION_SET",
				"LAST_ERROR_MESSAGE"}
			mockRows = sqlmock.NewRows(columns).AddRow("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-160:10000-10001",
				"")
			mock.ExpectQuery(query).WillReturnRows(mockRows)

			query = "SELECT GTID_SUBTRACT\\('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-160:10000-10001', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-160:1000:10000-10001'\\) RES"
			columns = []string{"RES"}
			mockRows = sqlmock.NewRows(columns).AddRow("")
			mock.ExpectQuery(query).WillReturnRows(mockRows)

			want := false
			got, _, err := mysql.GTIDGreaterThan(&GTID)
			assert.Nil(t, err)
			assert.Equal(t, want, got)
		}

		// 2.3. the Executed_GTID_Set includes less GTIDs
		{
			GTID := model.GTID{Master_Log_File: "mysql-bin.000001",
				Read_Master_Log_Pos: 148,
				Retrieved_GTID_Set:  "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1000-1010:10000",
				Executed_GTID_Set:   "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-160:1000-1001",
				Txns_Behind_Master:  "10",
				Last_Error_Message:  ""}

			query := "SHOW MASTER STATUS"
			columns := []string{"File",
				"Position",
				"Binlog_Do_DB",
				"Binlog_Ignore_DB",
				"Executed_Gtid_Set"}

			mockRows := sqlmock.NewRows(columns).AddRow("mysql-bin.000001",
				"148",
				"",
				"",
				"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-150:1000")
			mock.ExpectQuery(query).WillReturnRows(mockRows)

			query = "SELECT RECEIVED_TRANSACTION_SET,LAST_ERROR_MESSAGE FROM performance_schema.replication_connection_status WHERE CHANNEL_NAME='group_replication_applier'"
			columns = []string{"RECEIVED_TRANSACTION_SET",
				"LAST_ERROR_MESSAGE"}
			mockRows = sqlmock.NewRows(columns).AddRow("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:100-160:1000-1010",
				"")
			mock.ExpectQuery(query).WillReturnRows(mockRows)

			query = "SELECT GTID_SUBTRACT\\('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-160:1000-1010', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-160:1000-1010:10000'\\) RES"
			columns = []string{"RES"}
			mockRows = sqlmock.NewRows(columns).AddRow("")
			mock.ExpectQuery(query).WillReturnRows(mockRows)

			want := false
			got, _, err := mysql.GTIDGreaterThan(&GTID)
			assert.Nil(t, err)
			assert.Equal(t, want, got)
		}
	}
}

func TestGTIDGreaterThan(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	// log
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, 10000, log)
	mysql.db = db

	// 1. show slave status OK
	{
		query := "SHOW SLAVE STATUS"
		columns := []string{"Master_Log_File",
			"Read_Master_Log_Pos",
			"Retrieved_Gtid_Set",
			"Executed_Gtid_Set",
			"Slave_IO_Running",
			"Slave_SQL_Running"}

		GTID := model.GTID{Master_Log_File: "mysql-bin.000001",
			Read_Master_Log_Pos: 147,
			Retrieved_GTID_Set:  "84030605-66aa-11e6-9465-52540e7fd51c:154-160",
			Executed_GTID_Set:   "84030605-66aa-11e6-9465-52540e7fd51c:1-159,ebd03dad-69ad-11e6-aa22-52540e7fd51c:1",
			Slave_IO_Running:    true,
			Slave_SQL_Running:   true}

		mockRows := sqlmock.NewRows(columns).AddRow("mysql-bin.000001",
			"148",
			"84030605-66aa-11e6-9465-52540e7fd51c:154-160",
			"84030605-66aa-11e6-9465-52540e7fd51c:1-159,ebd03dad-69ad-11e6-aa22-52540e7fd51c:1",
			"Yes",
			"Yes")

		mock.ExpectQuery(query).WillReturnRows(mockRows)
		want := true
		got, _, err := mysql.GTIDGreaterThan(&GTID)
		assert.Nil(t, err)
		assert.Equal(t, want, got)
	}

	// 2. Seconds_Behind_Master
	{
		query := "SHOW SLAVE STATUS"
		columns := []string{"Master_Log_File",
			"Read_Master_Log_Pos",
			"Retrieved_Gtid_Set",
			"Executed_Gtid_Set",
			"Seconds_Behind_Master",
			"Slave_IO_Running",
			"Slave_SQL_Running"}

		GTID := model.GTID{Master_Log_File: "mysql-bin.000001",
			Read_Master_Log_Pos:   147,
			Retrieved_GTID_Set:    "84030605-66aa-11e6-9465-52540e7fd51c:154-160",
			Executed_GTID_Set:     "84030605-66aa-11e6-9465-52540e7fd51c:1-159,ebd03dad-69ad-11e6-aa22-52540e7fd51c:1",
			Seconds_Behind_Master: "13",
			Slave_IO_Running:      true,
			Slave_SQL_Running:     true}

		mockRows := sqlmock.NewRows(columns).AddRow("mysql-bin.000001",
			"147",
			"84030605-66aa-11e6-9465-52540e7fd51c:154-160",
			"84030605-66aa-11e6-9465-52540e7fd51c:1-159,ebd03dad-69ad-11e6-aa22-52540e7fd51c:1",
			"12",
			"Yes",
			"Yes")

		mock.ExpectQuery(query).WillReturnRows(mockRows)
		want := true
		got, _, err := mysql.GTIDGreaterThan(&GTID)
		assert.Nil(t, err)
		assert.Equal(t, want, got)
	}

	// 3. Seconds_Behind_Master error
	{
		query := "SHOW SLAVE STATUS"
		columns := []string{"Master_Log_File",
			"Read_Master_Log_Pos",
			"Retrieved_Gtid_Set",
			"Executed_Gtid_Set",
			"Seconds_Behind_Master",
			"Slave_IO_Running",
			"Slave_SQL_Running"}

		GTID := model.GTID{Master_Log_File: "mysql-bin.000001",
			Read_Master_Log_Pos:   147,
			Retrieved_GTID_Set:    "84030605-66aa-11e6-9465-52540e7fd51c:154-160",
			Executed_GTID_Set:     "84030605-66aa-11e6-9465-52540e7fd51c:1-159,ebd03dad-69ad-11e6-aa22-52540e7fd51c:1",
			Seconds_Behind_Master: "13",
			Slave_IO_Running:      true,
			Slave_SQL_Running:     true}

		mockRows := sqlmock.NewRows(columns).AddRow("mysql-bin.000001",
			"147",
			"84030605-66aa-11e6-9465-52540e7fd51c:154-160",
			"84030605-66aa-11e6-9465-52540e7fd51c:1-159,ebd03dad-69ad-11e6-aa22-52540e7fd51c:1",
			"NULL",
			"Yes",
			"Yes")

		mock.ExpectQuery(query).WillReturnRows(mockRows)
		want := false
		got, _, err := mysql.GTIDGreaterThan(&GTID)
		assert.Nil(t, err)
		assert.Equal(t, want, got)
	}

	// 4. Seconds_Behind_Master error too
	{
		query := "SHOW SLAVE STATUS"
		columns := []string{"Master_Log_File",
			"Read_Master_Log_Pos",
			"Retrieved_Gtid_Set",
			"Executed_Gtid_Set",
			"Seconds_Behind_Master",
			"Slave_IO_Running",
			"Slave_SQL_Running"}

		GTID := model.GTID{Master_Log_File: "mysql-bin.000001",
			Read_Master_Log_Pos:   147,
			Retrieved_GTID_Set:    "84030605-66aa-11e6-9465-52540e7fd51c:154-160",
			Executed_GTID_Set:     "84030605-66aa-11e6-9465-52540e7fd51c:1-159,ebd03dad-69ad-11e6-aa22-52540e7fd51c:1",
			Seconds_Behind_Master: "NULL",
			Slave_IO_Running:      true,
			Slave_SQL_Running:     true}

		mockRows := sqlmock.NewRows(columns).AddRow("mysql-bin.000001",
			"147",
			"84030605-66aa-11e6-9465-52540e7fd51c:154-160",
			"84030605-66aa-11e6-9465-52540e7fd51c:1-159,ebd03dad-69ad-11e6-aa22-52540e7fd51c:1",
			"14",
			"Yes",
			"Yes")

		mock.ExpectQuery(query).WillReturnRows(mockRows)
		want := false
		got, _, err := mysql.GTIDGreaterThan(&GTID)
		assert.Nil(t, err)
		assert.Equal(t, want, got)
	}
}

func TestMGRGetGTID(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	// log
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	conf := config.DefaultMysqlConfig()
	conf.ReplMode = model.ReplModeMGR
	mysql := NewMysql(conf, 10000, log)
	mysql.db = db

	// 1. normal
	{
		query := "SHOW MASTER STATUS"
		columns := []string{"File",
			"Position",
			"Binlog_Do_DB",
			"Binlog_Ignore_DB",
			"Executed_Gtid_Set"}
		mockRows := sqlmock.NewRows(columns).AddRow("mysql-bin.000001",
			"147",
			"",
			"",
			"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-159")
		mock.ExpectQuery(query).WillReturnRows(mockRows)

		query = "SELECT RECEIVED_TRANSACTION_SET,LAST_ERROR_MESSAGE FROM performance_schema.replication_connection_status WHERE CHANNEL_NAME='group_replication_applier'"
		columns = []string{"RECEIVED_TRANSACTION_SET",
			"LAST_ERROR_MESSAGE"}
		mockRows = sqlmock.NewRows(columns).AddRow("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:154-160",
			"")
		mock.ExpectQuery(query).WillReturnRows(mockRows)

		query = "SELECT GTID_SUBTRACT\\('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:154-160', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-159'\\) RES"
		columns = []string{"RES"}
		mockRows = sqlmock.NewRows(columns).AddRow("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:160")
		mock.ExpectQuery(query).WillReturnRows(mockRows)

		want := model.GTID{Master_Log_File: "mysql-bin.000001",
			Read_Master_Log_Pos: 147,
			Retrieved_GTID_Set:  "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:154-160",
			Executed_GTID_Set:   "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-159",
			Txns_Behind_Master:  "1",
			Last_Error_Message:  ""}

		got, _ := mysql.GetGTID()
		assert.Equal(t, want, got)
	}
}

func TestGetGTID(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	// log
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, 10000, log)
	mysql.db = db

	// 1. show slave status OK
	{
		query := "SHOW SLAVE STATUS"
		columns := []string{"Master_Log_File",
			"Read_Master_Log_Pos",
			"Retrieved_Gtid_Set",
			"Executed_Gtid_Set",
			"Slave_IO_Running",
			"Slave_SQL_Running"}

		want := model.GTID{Master_Log_File: "mysql-bin.000001",
			Read_Master_Log_Pos:   147,
			Retrieved_GTID_Set:    "84030605-66aa-11e6-9465-52540e7fd51c:154-160",
			Executed_GTID_Set:     "84030605-66aa-11e6-9465-52540e7fd51c:1-159,ebd03dad-69ad-11e6-aa22-52540e7fd51c:1",
			Slave_IO_Running:      true,
			Slave_IO_Running_Str:  "Yes",
			Slave_SQL_Running:     true,
			Slave_SQL_Running_Str: "Yes",
		}

		mockRows := sqlmock.NewRows(columns).AddRow("mysql-bin.000001",
			"147",
			"84030605-66aa-11e6-9465-52540e7fd51c:154-160",
			"84030605-66aa-11e6-9465-52540e7fd51c:1-159,ebd03dad-69ad-11e6-aa22-52540e7fd51c:1",
			"Yes",
			"Yes")

		mock.ExpectQuery(query).WillReturnRows(mockRows)
		got, _ := mysql.GetGTID()
		assert.Equal(t, want, got)
	}

	// 2. show slave status returns null
	//    will hit GetMasterGTID
	{
		query := "SHOW SLAVE STATUS"
		columns := []string{"Master_Log_File",
			"Read_Master_Log_Pos",
			"Retrieved_Gtid_Set",
			"Executed_Gtid_Set",
			"Slave_IO_Running",
			"Slave_SQL_Running"}
		mockRows := sqlmock.NewRows(columns).AddRow("",
			"",
			"",
			"",
			"",
			"")
		mock.ExpectQuery(query).WillReturnRows(mockRows)

		// show master status
		query = "SHOW MASTER STATUS"
		columns = []string{"File",
			"Position",
			"Binlog_Do_DB",
			"Binlog_Ignore_DB",
			"Executed_Gtid_Set",
		}
		mockRows = sqlmock.NewRows(columns).AddRow("mysql-bin.000001",
			"147",
			"",
			"",
			"84030605-66aa-11e6-9465-52540e7fd51c:154-160",
		)
		mock.ExpectQuery(query).WillReturnRows(mockRows)

		want := model.GTID{Master_Log_File: "mysql-bin.000001",
			Read_Master_Log_Pos:     147,
			Retrieved_GTID_Set:      "",
			Executed_GTID_Set:       "84030605-66aa-11e6-9465-52540e7fd51c:154-160",
			Slave_IO_Running:        true,
			Slave_SQL_Running:       true,
			Seconds_Behind_Master:   "0",
			Last_Error:              "",
			Slave_SQL_Running_State: "",
		}

		got, _ := mysql.GetGTID()
		assert.Equal(t, want, got)
	}

	// 3. show slave status returns Str No
	{
		query := "SHOW SLAVE STATUS"
		columns := []string{"Master_Log_File",
			"Read_Master_Log_Pos",
			"Retrieved_Gtid_Set",
			"Executed_Gtid_Set",
			"Slave_IO_Running",
			"Slave_SQL_Running"}
		mockRows := sqlmock.NewRows(columns).AddRow("",
			"",
			"",
			"",
			"No",
			"No")
		mock.ExpectQuery(query).WillReturnRows(mockRows)

		want := model.GTID{
			Slave_IO_Running:      false,
			Slave_SQL_Running:     false,
			Slave_IO_Running_Str:  "No",
			Slave_SQL_Running_Str: "No",
		}

		got, _ := mysql.GetGTID()
		assert.Equal(t, want, got)
	}
}

func TestPromotableYes(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	// log
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, 10000, log)
	mysql.db = db

	// 2. mock Slave_SQL_Running OK
	query := "SHOW SLAVE STATUS"
	columns := []string{"Master_Log_File",
		"Read_Master_Log_Pos",
		"Retrieved_Gtid_Set",
		"Executed_Gtid_Set",
		"Slave_IO_Running",
		"Slave_SQL_Running"}
	mockRows := sqlmock.NewRows(columns).AddRow("mysql-bin.000001",
		"147",
		"84030605-66aa-11e6-9465-52540e7fd51c:154-160",
		"84030605-66aa-11e6-9465-52540e7fd51c:1-159,ebd03dad-69ad-11e6-aa22-52540e7fd51c:1",
		"Yes",
		"Yes")
	// for ping
	mock.ExpectQuery(query).WillReturnRows(mockRows)
	mysql.Ping()

	columns = []string{"Master_Log_File",
		"Read_Master_Log_Pos",
		"Retrieved_Gtid_Set",
		"Executed_Gtid_Set",
		"Slave_IO_Running",
		"Slave_SQL_Running"}
	mockRows = sqlmock.NewRows(columns).AddRow("mysql-bin.000001",
		"147",
		"84030605-66aa-11e6-9465-52540e7fd51c:154-160",
		"84030605-66aa-11e6-9465-52540e7fd51c:1-159,ebd03dad-69ad-11e6-aa22-52540e7fd51c:1",
		"Yes",
		"Yes")

	// for getgtid
	mock.ExpectQuery(query).WillReturnRows(mockRows)

	want := true
	got := mysql.Promotable()
	assert.Equal(t, want, got)
}

func TestPromotableNot(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	// log
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, 10000, log)
	mysql.db = db

	// 1. mock MysqlDead
	{
		query := "SHOW SLAVE STATUS"
		mock.ExpectQuery(query).WillReturnError(fmt.Errorf("mock.mysql.ping.error"))
		want := false
		mysql.Ping()
		got := mysql.Promotable()
		assert.Equal(t, want, got)
	}

	// 2. mock mysql is MysqlAlive
	{
		query := "SHOW SLAVE STATUS"
		columns := []string{"Master_Log_File",
			"Read_Master_Log_Pos",
			"Retrieved_Gtid_Set",
			"Executed_Gtid_Set",
			"Slave_IO_Running",
			"Slave_SQL_Running"}
		mockRows := sqlmock.NewRows(columns).AddRow("mysql-bin.000001",
			"147",
			"84030605-66aa-11e6-9465-52540e7fd51c:154-160",
			"84030605-66aa-11e6-9465-52540e7fd51c:1-159,ebd03dad-69ad-11e6-aa22-52540e7fd51c:1",
			"Yes",
			"Yes")
		mock.ExpectQuery(query).WillReturnRows(mockRows)
		mysql.Ping()
	}

	// 3. mock Slave_SQL_Running NO
	{
		query := "SHOW SLAVE STATUS"
		columns := []string{"Master_Log_File",
			"Read_Master_Log_Pos",
			"Retrieved_Gtid_Set",
			"Executed_Gtid_Set",
			"Slave_IO_Running",
			"Slave_SQL_Running"}
		mockRows := sqlmock.NewRows(columns).AddRow("mysql-bin.000001",
			"147",
			"84030605-66aa-11e6-9465-52540e7fd51c:154-160",
			"84030605-66aa-11e6-9465-52540e7fd51c:1-159,ebd03dad-69ad-11e6-aa22-52540e7fd51c:1",
			"Yes",
			"No")
		mock.ExpectQuery(query).WillReturnRows(mockRows)

		want := false
		got := mysql.Promotable()
		assert.Equal(t, want, got)
	}

	// 4. mock Slave_IO_Running NO
	{
		query := "SHOW SLAVE STATUS"
		columns := []string{"Master_Log_File",
			"Read_Master_Log_Pos",
			"Retrieved_Gtid_Set",
			"Executed_Gtid_Set",
			"Slave_IO_Running",
			"Slave_SQL_Running"}
		mockRows := sqlmock.NewRows(columns).AddRow("mysql-bin.000001",
			"147",
			"84030605-66aa-11e6-9465-52540e7fd51c:154-160",
			"84030605-66aa-11e6-9465-52540e7fd51c:1-159,ebd03dad-69ad-11e6-aa22-52540e7fd51c:1",
			"No",
			"Yes")
		mock.ExpectQuery(query).WillReturnRows(mockRows)

		want := true
		got := mysql.Promotable()
		assert.Equal(t, want, got)
	}

	// 5. mock Slave_IO/SQL_Running NO
	{
		query := "SHOW SLAVE STATUS"
		columns := []string{"Master_Log_File",
			"Read_Master_Log_Pos",
			"Retrieved_Gtid_Set",
			"Executed_Gtid_Set",
			"Slave_IO_Running",
			"Slave_SQL_Running"}
		mockRows := sqlmock.NewRows(columns).AddRow("mysql-bin.000001",
			"147",
			"84030605-66aa-11e6-9465-52540e7fd51c:154-160",
			"84030605-66aa-11e6-9465-52540e7fd51c:1-159,ebd03dad-69ad-11e6-aa22-52540e7fd51c:1",
			"No",
			"No")
		mock.ExpectQuery(query).WillReturnRows(mockRows)

		want := false
		got := mysql.Promotable()
		assert.Equal(t, want, got)
	}
}

func TestWaitMysqlWorks(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	// log
	log := nlog.NewStdLog(nlog.Level(nlog.DEBUG))
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, 10000, log)
	mysql.db = db
	mysql.PingStart()
	defer mysql.PingStop()

	// works
	{
		conf := config.DefaultMysqlConfig()
		mysql := NewMysql(conf, 10000, log)
		mysql.db = db

		query := "SHOW SLAVE STATUS"
		columns := []string{"Master_Log_File",
			"Read_Master_Log_Pos",
			"Retrieved_Gtid_Set",
			"Executed_Gtid_Set",
			"Slave_IO_Running",
			"Slave_SQL_Running"}
		mockRows := sqlmock.NewRows(columns).AddRow("mysql-bin.000001",
			"147",
			"84030605-66aa-11e6-9465-52540e7fd51c:154-160",
			"84030605-66aa-11e6-9465-52540e7fd51c:1-159,ebd03dad-69ad-11e6-aa22-52540e7fd51c:1",
			//"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:154-160",
			//"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-159,bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb:1",
			"Yes",
			"Yes")
		mock.ExpectQuery(query).WillReturnRows(mockRows)
		err = mysql.WaitMysqlWorks(10000)
		assert.Nil(t, err)
	}

	// timeouts
	{
		conf := config.DefaultMysqlConfig()
		mysql := NewMysql(conf, 10000, log)
		mysql.db = db
		mysql.PingStart()
		defer mysql.PingStop()

		query := "SHOW SLAVE STATUS"
		mock.ExpectQuery(query).WillReturnError(fmt.Errorf("mock.mysql.ping.error"))

		err = mysql.WaitMysqlWorks(1000)
		want := err.Error()
		got := "WaitMysqlWorks.Timeout[1s]"
		assert.Equal(t, want, got)
	}
}

func TestGlobalSysVar(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	// log
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, 10000, log)
	mysql.db = db

	queryList := []string{
		"SET GLOBAL read_only = 1",
		"SET GLOBAL gtid_mode = 'ON'",
		"XET GLOBAL gtid_mode = 'ON'",
	}

	mock.ExpectExec(queryList[0]).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(queryList[1]).WillReturnResult(sqlmock.NewResult(1, 1))
	err = mysql.SetGlobalSysVar(queryList[0])
	assert.Nil(t, err)

	err = mysql.SetGlobalSysVar(queryList[1])
	assert.Nil(t, err)

	err = mysql.SetGlobalSysVar(queryList[2])
	want := "[XET GLOBAL gtid_mode = 'ON'].must.be.startwith:SET GLOBAL"
	got := err.Error()
	assert.Equal(t, want, got)
}

func TestSemiSyncMaster(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	// log
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, 10000, log)
	mysql.db = db

	queryList := []string{
		"SET GLOBAL rpl_semi_sync_master_enabled=ON",
		"SET GLOBAL rpl_semi_sync_master_enabled=OFF",
		"SET GLOBAL rpl_semi_sync_master_timeout=10000",
	}

	mock.ExpectExec(queryList[0]).WillReturnResult(sqlmock.NewResult(1, 1))
	err = mysql.EnableSemiSyncMaster()
	assert.Nil(t, err)

	mock.ExpectExec(queryList[1]).WillReturnResult(sqlmock.NewResult(1, 1))
	err = mysql.DisableSemiSyncMaster()
	assert.Nil(t, err)

	mock.ExpectExec(queryList[2]).WillReturnResult(sqlmock.NewResult(1, 1))
	err = mysql.SetSemiSyncMasterTimeout(10000)
	assert.Nil(t, err)
}

func TestPurgeBinlogsTo(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	// log
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, 10000, log)
	mysql.db = db

	// 1. show slave status OK
	query := "PURGE BINARY LOGS TO 'mysql-bin.000032'"

	mock.ExpectExec(query).WillReturnResult(sqlmock.NewResult(1, 1))
	mysql.PurgeBinlogsTo("mysql-bin.000032")
	assert.Nil(t, err)
}

func TestSetMasterSysVars(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	// log
	log := nlog.NewStdLog(nlog.Level(nlog.ERROR))
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, 10000, log)
	mysql.db = db
	err = mysql.SetMasterGlobalSysVar()
	assert.Nil(t, err)

	{
		query := "SET GLOBAL tokudb_fsync_log_period=default"
		mock.ExpectExec(query).WillReturnResult(sqlmock.NewResult(1, 1))
	}

	{
		query := "SET GLOBAL sync_binlog=default"
		mock.ExpectExec(query).WillReturnResult(sqlmock.NewResult(1, 1))
	}

	{
		query := "SET GLOBAL innodb_flush_log_at_trx_commit=default"
		mock.ExpectExec(query).WillReturnResult(sqlmock.NewResult(1, 1))
	}
	{
		conf.MasterSysVars = "tokudb_fsync_log_period=default;sync_binlog=default;innodb_flush_log_at_trx_commit=default"
		err = mysql.SetMasterGlobalSysVar()
		assert.Nil(t, err)
	}
}

func TestSetSlaveSysVars(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.Nil(t, err)
	defer db.Close()

	// log
	log := nlog.NewStdLog(nlog.Level(nlog.ERROR))
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, 10000, log)
	mysql.db = db

	err = mysql.SetSlaveGlobalSysVar()
	assert.Nil(t, err)
	{
		query := "SET GLOBAL tokudb_fsync_log_period=1000"
		mock.ExpectExec(query).WillReturnResult(sqlmock.NewResult(1, 1))
	}

	{
		query := "SET GLOBAL sync_binlog=1000"
		mock.ExpectExec(query).WillReturnResult(sqlmock.NewResult(1, 1))
	}

	{
		query := "SET GLOBAL innodb_flush_log_at_trx_commit=2"
		mock.ExpectExec(query).WillReturnResult(sqlmock.NewResult(1, 1))
	}
	{
		conf.SlaveSysVars = "tokudb_fsync_log_period=1000;sync_binlog=1000;innodb_flush_log_at_trx_commit=2"
		err = mysql.SetSlaveGlobalSysVar()
		assert.Nil(t, err)
	}
}
