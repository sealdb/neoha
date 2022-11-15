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

package raft

import (
	"testing"

	"neoha/base/common"
	"neoha/base/model"
	"neoha/base/nlog"
	"neoha/database/mysql"

	"github.com/stretchr/testify/assert"
)

func TestRaftRPCStatus(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	names, rafts, scleanup := MockRafts(log, port, 3, -1)
	defer scleanup()
	var whoisleader int

	{
		for _, raft := range rafts {
			raft.Start()
		}

		MockWaitLeaderEggs(rafts, 1)
		for i, raft := range rafts {
			if raft.getState() == LEADER {
				whoisleader = i
				break
			}
		}
	}

	{
		MockWaitLeaderEggs(rafts, 1)
		c, cleanup := MockGetClient(t, names[whoisleader])
		defer cleanup()

		method := model.RPCRaftStatus
		req := model.NewRaftStatusRPCRequest()
		rsp := model.NewRaftStatusRPCResponse(model.OK)
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)

		want := 1
		got := int(rsp.Stats.LeaderPromotes)
		assert.Equal(t, want, got)
	}
}

func TestRaftRPCs(t *testing.T) {
	mockHost := ":6666"
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	names, rafts, scleanup := MockRafts(log, port, 1, -1)
	defer scleanup()

	// start
	{
		for _, raft := range rafts {
			raft.Start()
		}
		rafts[0].AddPeer(mockHost)
	}

	// test: heartbeat with large ViewID to CANDIDATE
	{
		MockStateTransition(rafts[0], CANDIDATE)
		c, cleanup := MockGetClient(t, names[0])
		defer cleanup()

		method := model.RPCRaftHeartbeat
		req := model.NewRaftRPCRequest()
		req.Raft.From = mockHost
		req.Raft.ViewID = 1000

		rsp := model.NewRaftRPCResponse(model.OK)
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)

		want := model.OK
		got := rsp.RetCode
		assert.Equal(t, want, got)
	}

	// test: requestvote with small ViewID to LEADER
	{
		MockStateTransition(rafts[0], LEADER)
		c, cleanup := MockGetClient(t, names[0])
		defer cleanup()

		method := model.RPCRaftRequestVote
		req := model.NewRaftRPCRequest()
		req.Raft.From = mockHost
		req.Raft.ViewID = 0

		rsp := model.NewRaftRPCResponse(model.OK)
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)

		want := model.ErrorInvalidViewID
		got := rsp.RetCode
		assert.Equal(t, want, got)
	}
}

func TestRaftRPCPurgeBinlog(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	names, rafts, scleanup := MockRafts(log, port, 3, -1)
	defer scleanup()
	whoisleader := 2

	// 1. set rafts GTID
	//    1.0 rafts[0]  with MockGTIDB{Master_Log_File = "mysql-bin.000001", Read_Master_Log_Pos = 123}
	//    1.1 rafts[1]  with MockGTIDB{Master_Log_File = "mysql-bin.000003", Read_Master_Log_Pos = 123}
	//    1.2 rafts[2]  with MockGTIDC{Master_Log_File = "mysql-bin.000005", Read_Master_Log_Pos = 123}
	{
		rafts[0].mysql.SetMysqlHandler(mysql.NewMockGTIDX1())
		rafts[1].mysql.SetMysqlHandler(mysql.NewMockGTIDX3())
		rafts[2].mysql.SetMysqlHandler(mysql.NewMockGTIDX5())
	}

	// 2. Start 3 rafts state as FOLLOWER
	for _, raft := range rafts {
		raft.Start()
	}

	// wait leader eggs
	{
		MockWaitLeaderEggs(rafts, 1)
	}
	// check(default is enable)
	{
		MockWaitLeaderEggs(rafts, 0)
		MockWaitLeaderEggs(rafts, 0)
		purged := rafts[whoisleader].stats.LeaderPurgeBinlogs
		assert.NotZero(t, purged)
	}

	// disable purge binlog
	{
		c, cleanup := MockGetClient(t, names[whoisleader])
		defer cleanup()

		method := model.RPCRaftDisablePurgeBinlog
		req := model.NewRaftStatusRPCRequest()
		rsp := model.NewRaftStatusRPCResponse(model.OK)
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)

		// check
		want := rafts[whoisleader].stats.LeaderPurgeBinlogs
		MockWaitLeaderEggs(rafts, 0)
		MockWaitLeaderEggs(rafts, 0)
		got := rafts[whoisleader].stats.LeaderPurgeBinlogs
		assert.Equal(t, want, got)
	}

	// enable purge binlog
	{
		MockWaitLeaderEggs(rafts, 1)
		c, cleanup := MockGetClient(t, names[whoisleader])
		defer cleanup()

		method := model.RPCRaftEnablePurgeBinlog
		req := model.NewRaftStatusRPCRequest()
		rsp := model.NewRaftStatusRPCResponse(model.OK)
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)

		// check
		want := rafts[whoisleader].stats.LeaderPurgeBinlogs
		MockWaitLeaderEggs(rafts, 0)
		MockWaitLeaderEggs(rafts, 0)
		got := rafts[whoisleader].stats.LeaderPurgeBinlogs
		assert.NotEqual(t, want, got)
	}
}

func TestRaftRPCCheckSemiSync(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	names, rafts, scleanup := MockRafts(log, port, 3, -1)
	defer scleanup()
	whoisleader := 2

	// 1. set rafts GTID
	//    1.0 rafts[0]  with MockGTIDB{Master_Log_File = "mysql-bin.000001", Read_Master_Log_Pos = 123}
	//    1.1 rafts[1]  with MockGTIDB{Master_Log_File = "mysql-bin.000003", Read_Master_Log_Pos = 123}
	//    1.2 rafts[2]  with MockGTIDC{Master_Log_File = "mysql-bin.000005", Read_Master_Log_Pos = 123}
	{
		rafts[0].mysql.SetMysqlHandler(mysql.NewMockGTIDX1())
		rafts[1].mysql.SetMysqlHandler(mysql.NewMockGTIDX3())
		rafts[2].mysql.SetMysqlHandler(mysql.NewMockGTIDX5())
	}

	// 2. Start 3 rafts state as FOLLOWER
	for _, raft := range rafts {
		raft.Start()
	}

	// wait leader eggs
	{
		MockWaitLeaderEggs(rafts, 1)
	}
	// check(default is enable)
	{
		MockWaitLeaderEggs(rafts, 0)
		MockWaitLeaderEggs(rafts, 0)
		check := rafts[whoisleader].skipCheckSemiSync
		assert.Equal(t, false, check)
	}

	// disable check semi-sync
	{
		c, cleanup := MockGetClient(t, names[whoisleader])
		defer cleanup()

		method := model.RPCRaftDisableCheckSemiSync
		req := model.NewRaftStatusRPCRequest()
		rsp := model.NewRaftStatusRPCResponse(model.OK)
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)

		// check
		MockWaitLeaderEggs(rafts, 0)
		MockWaitLeaderEggs(rafts, 0)
		got := rafts[whoisleader].skipCheckSemiSync
		assert.Equal(t, true, got)
	}

	// enable check semi-sync
	{
		MockWaitLeaderEggs(rafts, 1)
		c, cleanup := MockGetClient(t, names[whoisleader])
		defer cleanup()

		method := model.RPCRaftEnableCheckSemiSync
		req := model.NewRaftStatusRPCRequest()
		rsp := model.NewRaftStatusRPCResponse(model.OK)
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)

		// check
		MockWaitLeaderEggs(rafts, 0)
		MockWaitLeaderEggs(rafts, 0)
		got := rafts[whoisleader].skipCheckSemiSync
		assert.Equal(t, false, got)
	}
}
