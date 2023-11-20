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
	"testing"
	"time"

	"github.com/sealdb/neoha/internal/base/common"
	"github.com/sealdb/neoha/internal/base/model"
	"github.com/sealdb/neoha/internal/base/nlog"
	"github.com/sealdb/neoha/internal/config"

	"github.com/stretchr/testify/assert"
)

func testMysql(t *testing.T, replMode model.MysqlReplMode) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	var mysql *Mysql
	var cleanup func()
	if replMode == model.ReplModeMGR {
		_, mysql, cleanup = MockMysql(log, port, NewMockGTIDX5())
	} else {
		_, mysql, cleanup = MockMysql(log, port, NewMockGTIDA(replMode))
	}
	defer cleanup()

	time.Sleep(time.Duration(config.DefaultMysqlConfig().PingTimeout*2) * time.Millisecond)
	got := mysql.GetState()
	want := model.MysqlAlive
	assert.Equal(t, want, got)
	mysql.PingStop()
}

func TestMysql_SemiSync(t *testing.T) {
	testMysql(t, model.ReplModeSemiSync)
}

func TestMysql_MGR(t *testing.T) {
	testMysql(t, model.ReplModeMGR)
}

func TestStateDead(t *testing.T) {
	// log
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	_, mysql, cleanup := MockMysql(log, port, NewMockGTIDPingError())
	defer cleanup()

	time.Sleep(time.Duration(config.DefaultMysqlConfig().PingTimeout*2) * time.Millisecond)
	got := mysql.GetState()
	want := model.MysqlDead
	assert.Equal(t, want, got)
	mysql.PingStop()
}

func testCreateReplUser(t *testing.T, replMode model.MysqlReplMode) {
	// log
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	_, mysql, cleanup := MockMysqlReplUser(log, port, NewMockGTIDA(replMode))
	defer cleanup()

	time.Sleep(time.Duration(config.DefaultMysqlConfig().PingTimeout*2) * time.Millisecond)
	got := mysql.GetState()
	want := model.MysqlAlive
	assert.Equal(t, want, got)
	mysql.PingStop()
}

func TestCreateReplUser_MySQL_SemiSync(t *testing.T) {
	testCreateReplUser(t, model.ReplModeSemiSync)
}

func TestCreateReplUser_MySQL_MGR(t *testing.T) {
	testCreateReplUser(t, model.ReplModeMGR)
}

/*
// TEST EFFECTS:
// test GTIDGreaterThan function
//
// TEST PROCESSES:
// 1. set mock function
// 2. greater than a
// 3. not greater than a
func TestMysqlGTIDGreatThan(t *testing.T) {
	// log
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	conf := config.DefaultMysqlConfig()
	mysql := NewMysql(conf, 10000, log)

	// Set mock functions
	mysql.SetMysqlHandler(new(MockGTIDB))

	// start ping
	mysql.PingStart()

	// wait for ping
	time.Sleep(time.Duration(conf.PingTimeout*2) * time.Millisecond)

	// 1. greater than a
	a := model.GTID{Master_Log_File: "mysql-bin.000001",
		Read_Master_Log_Pos: 121}

	want := true
	got, _ := mysql.SlaveGTIDGreaterThan(&a)
	assert.Equal(t, want, got)

	// 2. greater than a
	a = model.GTID{Master_Log_File: "",
		Read_Master_Log_Pos: 0}
	want = true
	got, _ = mysql.SlaveGTIDGreaterThan(&a)
	assert.Equal(t, want, got)

	// 3. not greater than a
	a = model.GTID{Master_Log_File: "mysql-bin.000002",
		Read_Master_Log_Pos: 0}
	want = false
	got, _ = mysql.SlaveGTIDGreaterThan(&a)
	assert.Equal(t, want, got)

	// 4. nil compare: not greater than
	// set mock: this mock sets  null GTID
	mysql.SetMysqlHandler(new(MockGTIDA))

	// wait for ping
	time.Sleep(time.Duration(conf.PingTimeout*2) * time.Millisecond)

	a = model.GTID{Master_Log_File: "",
		Read_Master_Log_Pos: 0}
	want = false
	mysql.SetMysqlHandler(new(MockGTIDB))
	assert.Equal(t, want, got)
}
*/
