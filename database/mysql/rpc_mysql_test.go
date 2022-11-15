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
	"testing"

	"neoha/base/common"
	"neoha/base/model"
	"neoha/base/nlog"

	"github.com/stretchr/testify/assert"
)

func TestMysqlRPCStatus(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	id, _, cleanup := MockMysql(log, port, NewMockGTIDB())
	defer cleanup()

	// client
	{
		method := model.RPCMysqlStatus
		req := model.NewMysqlStatusRPCRequest()
		rsp := model.NewMysqlStatusRPCResponse(model.OK)
		c, cleanup := MockGetClient(t, id)
		defer cleanup()
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)

		GTID := model.GTID{
			Master_Log_File:     "mysql-bin.000001",
			Read_Master_Log_Pos: 123,
			Executed_GTID_Set:   "c78e798a-cccc-cccc-cccc-525433e8e796:1-2",
			Slave_IO_Running:    true,
			Slave_SQL_Running:   true,
		}
		want := model.NewMysqlStatusRPCResponse(model.OK)
		want.GTID = GTID
		want.Status = string(model.MysqlDead)
		want.Stats = &model.MysqlStats{}

		got := rsp
		assert.Equal(t, want, got)
	}
}

func TestMysqlRPCSetState(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	id, _, cleanup := MockMysql(log, port, NewMockGTIDB())
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
