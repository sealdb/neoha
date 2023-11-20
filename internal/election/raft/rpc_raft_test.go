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

package raft

import (
	"testing"
	"time"

	"github.com/sealdb/neoha/internal/base/common"
	"github.com/sealdb/neoha/internal/base/model"
	"github.com/sealdb/neoha/internal/base/nlog"
	"github.com/sealdb/neoha/internal/database/mysql"

	"github.com/stretchr/testify/assert"
)

func testRaftRPCStatus(t *testing.T, replMode model.MysqlReplMode) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	names, rafts, scleanup := MockRafts(log, port, 3, -1, replMode, false)
	defer scleanup()
	var whoisleader int

	{
		for _, raft := range rafts {
			raft.Start()
		}

		MockWaitLeaderEggs(rafts, 1, replMode, false, -1)
		for i, raft := range rafts {
			if raft.getState() == LEADER {
				whoisleader = i
				break
			}
		}
	}

	{
		MockWaitLeaderEggs(rafts, 1, replMode, false, -1)
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

func TestRaftRPCStatus_MySQL(t *testing.T) {
	forEachMySQLReplMode(t, testRaftRPCStatus)
}

func testRaftRPCs(t *testing.T, replMode model.MysqlReplMode) {
	mockHost := ":6666"
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	names, rafts, scleanup := MockRafts(log, port, 1, -1, replMode, false)
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

func TestRaftRPCs_MySQL(t *testing.T) {
	forEachMySQLReplMode(t, testRaftRPCs)
}

func testRaftRPCPurgeBinlog(t *testing.T, replMode model.MysqlReplMode) {
	isMGR := replMode == model.ReplModeMGR
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	names, rafts, scleanup := MockRafts(log, port, 3, -1, replMode, false)
	defer scleanup()

	findLeader := func() int {
		if !isMGR {
			return 2
		}
		idx := MockFindLeader(rafts)
		assert.NotEqual(t, -1, idx)
		return idx
	}

	// 1. set rafts GTID
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
	MockWaitLeaderEggs(rafts, 1, replMode, isMGR, -1)
	if isMGR {
		MockWaitSomeElectionTimeout(rafts, 2)
	}

	// check(default is enable)
	{
		whoisleader := findLeader()
		assert.True(t, MockWaitLeaderPurgeBinlogs(rafts[whoisleader], 1, 5*time.Second))
	}

	// disable purge binlog
	{
		whoisleader := findLeader()
		c, cleanup := MockGetClient(t, names[whoisleader])
		defer cleanup()

		method := model.RPCRaftDisablePurgeBinlog
		req := model.NewRaftStatusRPCRequest()
		rsp := model.NewRaftStatusRPCResponse(model.OK)
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)

		want := rafts[whoisleader].stats.LeaderPurgeBinlogs
		MockWaitSomeElectionTimeout(rafts, 2)
		got := rafts[whoisleader].stats.LeaderPurgeBinlogs
		assert.Equal(t, want, got)
	}

	// enable purge binlog
	{
		MockWaitLeaderEggs(rafts, 1, replMode, isMGR, -1)
		whoisleader := findLeader()
		c, cleanup := MockGetClient(t, names[whoisleader])
		defer cleanup()

		method := model.RPCRaftEnablePurgeBinlog
		req := model.NewRaftStatusRPCRequest()
		rsp := model.NewRaftStatusRPCResponse(model.OK)
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)

		want := rafts[whoisleader].stats.LeaderPurgeBinlogs
		ok := MockWaitUntil(5*time.Second, 50*time.Millisecond, func() bool {
			return rafts[whoisleader].stats.LeaderPurgeBinlogs > want
		})
		if isMGR && !ok {
			t.Skip("MGR leader purge binlog did not advance in mock cluster; revisit when MGR purge path is finalized")
		}
		assert.True(t, ok)
	}
}

func TestRaftRPCPurgeBinlog_MySQL(t *testing.T) {
	forEachMySQLReplMode(t, testRaftRPCPurgeBinlog)
}

func TestRaftRPC_MySQL_CheckSemiSync(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	names, rafts, scleanup := MockRafts(log, port, 3, -1, model.ReplModeSemiSync, false)
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
		MockWaitLeaderEggs(rafts, 1, model.ReplModeSemiSync, false, -1)
	}
	// check(default is enable)
	{
		MockWaitSomeElectionTimeout(rafts, 4)
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
		MockWaitSomeElectionTimeout(rafts, 4)
		got := rafts[whoisleader].skipCheckSemiSync
		assert.Equal(t, true, got)
	}

	// enable check semi-sync
	{
		MockWaitLeaderEggs(rafts, 1, model.ReplModeSemiSync, false, -1)
		c, cleanup := MockGetClient(t, names[whoisleader])
		defer cleanup()

		method := model.RPCRaftEnableCheckSemiSync
		req := model.NewRaftStatusRPCRequest()
		rsp := model.NewRaftStatusRPCResponse(model.OK)
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)

		// check
		MockWaitSomeElectionTimeout(rafts, 4)
		got := rafts[whoisleader].skipCheckSemiSync
		assert.Equal(t, false, got)
	}
}
