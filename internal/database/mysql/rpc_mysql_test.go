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

	"github.com/sealdb/neoha/internal/base/common"
	"github.com/sealdb/neoha/internal/base/model"
	"github.com/sealdb/neoha/internal/base/nlog"

	"github.com/stretchr/testify/assert"
)

func testMysqlRPCStatus(t *testing.T, GTID *model.GTID, replMode model.MysqlReplMode) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	var id string
	var cleanup func()

	if replMode == model.ReplModeMGR {
		id, _, cleanup = MockMysql(log, port, NewMockGTIDX1ForMGR())
	} else {
		id, _, cleanup = MockMysql(log, port, NewMockGTIDB())
	}
	defer cleanup()

	method := model.RPCMysqlStatus
	req := model.NewMysqlStatusRPCRequest()
	rsp := model.NewMysqlStatusRPCResponse(model.OK)
	c, cleanup := MockGetClient(t, id)
	defer cleanup()
	err := c.Call(method, req, rsp)
	assert.Nil(t, err)

	want := model.NewMysqlStatusRPCResponse(model.OK)
	want.ReplMode = replMode
	want.GTID = *GTID
	want.Status = string(model.MysqlDead)
	want.Stats = &model.MysqlStats{}

	got := rsp
	assert.Equal(t, want, got)
}

func TestMysqlRPCStatus(t *testing.T) {

	// semi-sync
	{
		GTID := model.GTID{
			Master_Log_File:     "mysql-bin.000001",
			Read_Master_Log_Pos: 123,
			Executed_GTID_Set:   "c78e798a-cccc-cccc-cccc-525433e8e796:1-2",
			Slave_IO_Running:    true,
			Slave_SQL_Running:   true,
		}
		testMysqlRPCStatus(t, &GTID, model.ReplModeSemiSync)
	}

	// group-replication
	{
		GTID := model.GTID{
			Master_Log_File:     "mysql-bin.000001",
			Read_Master_Log_Pos: 123,
			Executed_GTID_Set:   "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-99",
			Retrieved_GTID_Set:  "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-102",
			Txns_Behind_Master:  "3",
			Last_Error_Message:  "",
		}
		testMysqlRPCStatus(t, &GTID, model.ReplModeMGR)
	}
}

func testMysqlRPCSetState(t *testing.T, replMode model.MysqlReplMode) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	var id string
	var cleanup func()

	if replMode == model.ReplModeMGR {
		id, _, cleanup = MockMysql(log, port, NewMockGTIDX1())
	} else {
		id, _, cleanup = MockMysql(log, port, NewMockGTIDB())
	}
	defer cleanup()

	// MysqlDead
	{
		method := model.RPCMysqlSetState
		req := model.NewMysqlSetStateRPCRequest()
		req.State = model.MysqlDead
		rsp := model.NewMysqlSetStateRPCResponse(model.OK)
		c, cleanup := MockGetClient(t, id)
		defer cleanup()
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)

		want := model.NewMysqlSetStateRPCResponse(model.OK)
		got := rsp
		assert.Equal(t, want, got)
	}

	// MysqlAlive
	{
		method := model.RPCMysqlSetState
		req := model.NewMysqlSetStateRPCRequest()
		req.State = model.MysqlAlive
		rsp := model.NewMysqlSetStateRPCResponse(model.OK)
		c, cleanup := MockGetClient(t, id)
		defer cleanup()
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)

		want := model.NewMysqlSetStateRPCResponse(model.OK)
		got := rsp
		assert.Equal(t, want, got)
	}
}

func TestMysqlRPCSetState_SemiSync(t *testing.T) {
	testMysqlRPCSetState(t, model.ReplModeSemiSync)
}

func TestMysqlRPCSetState_MGR(t *testing.T) {
	testMysqlRPCSetState(t, model.ReplModeMGR)
}

func TestMysqlRPCSetSysVar(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	id, _, cleanup := MockMysql(log, port, new(Mysql57))
	defer cleanup()

	// client
	{
		method := model.RPCMysqlSetGlobalSysVar
		req := model.NewMysqlVarRPCRequest()
		rsp := model.NewMysqlVarRPCResponse(model.OK)
		c, cleanup := MockGetClient(t, id)
		defer cleanup()
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)
		want := "[].must.be.startwith:SET GLOBAL"
		got := rsp.RetCode
		assert.Equal(t, want, got)
	}
}

func TestMysqlRPCResetMaster(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	id, _, cleanup := MockMysql(log, port, new(Mysql57))
	defer cleanup()

	// client
	{
		method := model.RPCMysqlResetMaster
		req := model.NewMysqlRPCRequest()
		rsp := model.NewMysqlRPCResponse(model.OK)
		c, cleanup := MockGetClient(t, id)
		defer cleanup()
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)
	}
}

func TestMysqlRPCResetSlave(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	id, _, cleanup := MockMysql(log, port, new(Mysql80))
	defer cleanup()

	// client
	{
		method := model.RPCMysqlResetSlave
		req := model.NewMysqlRPCRequest()
		rsp := model.NewMysqlRPCResponse(model.OK)
		c, cleanup := MockGetClient(t, id)
		defer cleanup()
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)
	}
}

func TestMysqlRPCResetSlaveAll(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	id, _, cleanup := MockMysql(log, port, new(Mysql57))
	defer cleanup()

	// client
	{
		method := model.RPCMysqlResetSlaveAll
		req := model.NewMysqlRPCRequest()
		rsp := model.NewMysqlRPCResponse(model.OK)
		c, cleanup := MockGetClient(t, id)
		defer cleanup()
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)
	}
}

func TestMysqlRPCSlaves(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	id, _, cleanup := MockMysql(log, port, new(Mysql57))
	defer cleanup()

	// start
	{
		method := model.RPCMysqlStartSlave
		req := model.NewMysqlRPCRequest()
		rsp := model.NewMysqlRPCResponse(model.OK)
		c, cleanup := MockGetClient(t, id)
		defer cleanup()
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)
	}

	// stop
	{
		method := model.RPCMysqlStopSlave
		req := model.NewMysqlRPCRequest()
		rsp := model.NewMysqlRPCResponse(model.OK)
		c, cleanup := MockGetClient(t, id)
		defer cleanup()
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)
	}
}

func TestMysqlRPCIsWorking(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	id, _, cleanup := MockMysql(log, port, new(Mysql57))
	defer cleanup()

	{
		method := model.RPCMysqlIsWorking
		req := model.NewMysqlRPCRequest()
		rsp := model.NewMysqlRPCResponse(model.OK)
		c, cleanup := MockGetClient(t, id)
		defer cleanup()
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)
		want := model.ErrorMySQLDown
		got := rsp.RetCode
		assert.Equal(t, want, got)
	}
}
