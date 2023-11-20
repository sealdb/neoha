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

package server

import (
	"testing"

	"github.com/sealdb/neoha/internal/base/common"
	"github.com/sealdb/neoha/internal/base/model"
	"github.com/sealdb/neoha/internal/base/nlog"

	"github.com/stretchr/testify/assert"
)

// TEST EFFECTS:
// test a ping command from the client
//
// TEST PROCESSES:
// 1. Start rpc server
// 2. send command to rpc server
// 3. check the response
func testServerRPCPing(t *testing.T, replMode model.MysqlReplMode) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	servers, cleanup := MockServers(log, port, 1, replMode)
	defer cleanup()
	name := servers[0].Address()

	// rpc call
	{
		req := model.NewServerRPCRequest()
		rsp := model.NewServerRPCResponse(model.OK)
		c, cleanup := MockGetClient(t, name)
		defer cleanup()

		method := model.RRCServerPing
		if err := c.Call(method, req, rsp); err != nil {
			assert.Nil(t, err)
		}

		want := model.OK
		got := rsp.RetCode
		assert.Equal(t, want, got)
	}
}

func TestServerRPCPing_MySQL_SemiSync(t *testing.T) {
	testServerRPCPing(t, model.ReplModeSemiSync)
}

func TestServerRPCPing_MySQL_MGR(t *testing.T) {
	testServerRPCPing(t, model.ReplModeMGR)
}

func testServerRPCStatus(t *testing.T, replMode model.MysqlReplMode) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	servers, cleanup := MockServers(log, port, 1, replMode)
	defer cleanup()
	name := servers[0].Address()

	// rpc call
	{
		req := model.NewServerRPCRequest()
		rsp := model.NewServerRPCResponse(model.OK)
		c, cleanup := MockGetClient(t, name)
		defer cleanup()

		method := model.RPCServerStatus
		if err := c.Call(method, req, rsp); err != nil {
			assert.Nil(t, err)
		}

		config := &model.ConfigStatus{
			LogLevel:              "INFO",
			BackupDir:             "/u01/backup",
			BackupIOPSLimits:      100000,
			XtrabackupBinDir:      ".",
			MysqldBaseDir:         "/u01/mysql_20221010/",
			MysqldDefaultsFile:    "/etc/my3306.cnf",
			MysqlAdmin:            "root",
			MysqlHost:             "localhost",
			MysqlPort:             3306,
			MysqlReplUser:         "repl",
			MysqlPingTimeout:      1000,
			RaftDataDir:           ".",
			RaftHeartbeatTimeout:  100,
			RaftElectionTimeout:   300,
			RaftRPCRequestTimeout: 1000,
			RaftStartVipCommand:   "nop",
			RaftStopVipCommand:    "nop",
		}
		want := config
		got := rsp.Config
		assert.Equal(t, want, got)
	}
}

func TestServerRPCStatus_MySQL_SemiSync(t *testing.T) {
	testServerRPCStatus(t, model.ReplModeSemiSync)
}

func TestServerRPCStatus_MySQL_MGR(t *testing.T) {
	testServerRPCStatus(t, model.ReplModeMGR)
}
