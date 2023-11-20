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
	"github.com/sealdb/neoha/internal/base/common"
	"github.com/sealdb/neoha/internal/base/model"
	"github.com/sealdb/neoha/internal/base/nlog"
	"github.com/sealdb/neoha/internal/database/mysql"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TEST EFFECTS:
// test a hadisable command from the client
//
// TEST PROCESSES:
// 1. Start rpc server
// 2. send command to rpc server
// 3. check the response
func testRaftRPCHA(t *testing.T, replMode model.MysqlReplMode) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	names, rafts, scleanup := MockRafts(log, port, 3, -1, replMode, false)
	defer scleanup()

	// 1. Start 3 rafts state as FOLLOWER
	{
		for _, raft := range rafts {
			raft.Start()
		}

		var want, got State
		got = 0
		want = (FOLLOWER + FOLLOWER + FOLLOWER)
		for _, raft := range rafts {
			got += raft.getState()
		}

		// [FOLLOWER, FOLLOWER, FOLLOWER]
		assert.Equal(t, want, got)
	}

	// 2. all rafts to ha enable(invalid reqeust)
	{
		for i := range names {
			c, cleanup := MockGetClient(t, names[i])
			defer cleanup()

			method := model.RPCHAEnable
			req := model.NewHARPCRequest()
			rsp := model.NewHARPCResponse(model.OK)
			err := c.Call(method, req, rsp)
			assert.Nil(t, err)

			want := model.OK
			got := rsp.RetCode
			assert.Equal(t, want, got)
		}
	}

	// 3. all rafts to ha disable
	{
		for i := range rafts {
			c, cleanup := MockGetClient(t, names[i])
			defer cleanup()

			method := model.RPCHADisable
			req := model.NewHARPCRequest()
			rsp := model.NewHARPCResponse(model.OK)
			err := c.Call(method, req, rsp)
			assert.Nil(t, err)

			want := model.OK
			got := rsp.RetCode
			assert.Equal(t, want, got)
		}
	}

	// 4. check
	{
		MockWaitLeaderEggs(rafts, 0, replMode, false, -1)
		// MockWaitSomeElectionTimeout(rafts, 2)

		var want, got State
		got = 0
		want = (IDLE + IDLE + IDLE)
		for _, raft := range rafts {
			got += raft.getState()
		}
		// [IDLE, IDLE, IDLE]
		assert.Equal(t, want, got)
	}

	// 5. all rafts to HaEnable
	{
		for i := range names {
			c, cleanup := MockGetClient(t, names[i])
			defer cleanup()

			method := model.RPCHAEnable
			req := model.NewHARPCRequest()
			rsp := model.NewHARPCResponse(model.OK)
			err := c.Call(method, req, rsp)
			assert.Nil(t, err)

			want := model.OK
			got := rsp.RetCode
			assert.Equal(t, want, got)
		}
	}

	// 6. check
	{
		MockWaitLeaderEggs(rafts, 1, replMode, replMode == model.ReplModeMGR, -1)

		var want, got State
		got = 0
		want = (LEADER + FOLLOWER + FOLLOWER)
		for _, raft := range rafts {
			got += raft.getState()
		}
		// [LEADER, FOLLOWER, FOLLOWER]
		assert.Equal(t, want, got)
	}

	// 7. enable ha all
	{
		for i := range names {
			c, cleanup := MockGetClient(t, names[i])
			defer cleanup()

			method := model.RPCHAEnable
			req := model.NewHARPCRequest()
			rsp := model.NewHARPCResponse(model.OK)
			err := c.Call(method, req, rsp)
			assert.Nil(t, err)

			want := model.OK
			got := rsp.RetCode
			assert.Equal(t, want, got)
		}
	}

	// 8. all raftsto ha disable
	{
		for i := range names {
			c, cleanup := MockGetClient(t, names[i])
			defer cleanup()

			method := model.RPCHADisable
			req := model.NewHARPCRequest()
			rsp := model.NewHARPCResponse(model.OK)
			err := c.Call(method, req, rsp)
			assert.Nil(t, err)
			want := model.OK
			got := rsp.RetCode
			assert.Equal(t, want, got)
		}
	}
}

func TestRaftRPCHA_MySQL(t *testing.T) {
	forEachMySQLReplMode(t, testRaftRPCHA)
}

// TEST EFFECTS:
// test a hasetlearner command from follower by the client
//
// TEST PROCESSES:
// 1. Start rpc server
// 2. send command to rpc server
// 3. check the response
func testRaftRPCHASetLearnerFromFollower(t *testing.T, replMode model.MysqlReplMode) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	names, rafts, scleanup := MockRafts(log, port, 3, -1, replMode, false)
	learner := 2
	defer scleanup()

	// 1. Start 3 rafts state as FOLLOWER
	{
		for _, raft := range rafts {
			raft.Start()
		}

		var want, got State
		got = 0
		want = (FOLLOWER + FOLLOWER + FOLLOWER)
		for _, raft := range rafts {
			got += raft.getState()
		}

		// [FOLLOWER, FOLLOWER, FOLLOWER]
		assert.Equal(t, want, got)
	}

	// 2. set rafts[2] to LEARNER
	{
		c, cleanup := MockGetClient(t, names[learner])
		defer cleanup()

		method := model.RPCHASetLearner
		req := model.NewHARPCRequest()
		rsp := model.NewHARPCResponse(model.OK)
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)

		want := model.OK
		got := rsp.RetCode
		assert.Equal(t, want, got)
	}

	// 3. check
	{
		MockWaitLeaderEggs(rafts, 1, replMode, false, -1)

		var want, got State
		got = 0
		want = (LEADER + FOLLOWER + LEARNER)
		for _, raft := range rafts {
			got += raft.getState()
		}
		// [LEADER, FOLLOWER, LEARNER]
		assert.Equal(t, want, got)
	}

	// 4. enable ha for rafts[2]
	{
		c, cleanup := MockGetClient(t, names[learner])
		defer cleanup()

		method := model.RPCHAEnable
		req := model.NewHARPCRequest()
		rsp := model.NewHARPCResponse(model.OK)
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)

		want := model.OK
		got := rsp.RetCode
		assert.Equal(t, want, got)
	}

	// 5. check
	{
		MockWaitLeaderEggs(rafts, 1, replMode, false, -1)

		var want, got State
		got = 0
		want = (LEADER + FOLLOWER + FOLLOWER)
		for _, raft := range rafts {
			got += raft.getState()
		}
		// [LEADER, FOLLOWER, FOLLOWER]
		assert.Equal(t, want, got)
	}
}

func TestRaftRPCHASetLearnerFromFollower_MySQL(t *testing.T) {
	forEachMySQLReplMode(t, testRaftRPCHASetLearnerFromFollower)
}

// TEST EFFECTS:
// test a hasetlearner command from invalid by the client
//
// TEST PROCESSES:
// 1. Start rpc server
// 2. send command to rpc server
// 3. check the response
func testRaftRPCHASetLearnerFromInvalid(t *testing.T, replMode model.MysqlReplMode) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	names, rafts, scleanup := MockRafts(log, port, 3, -1, replMode, false)
	learner := 2
	defer scleanup()

	// 1. Start 3 rafts state as FOLLOWER and set rafts[2] to INVALID
	{
		for _, raft := range rafts {
			raft.Start()
		}

		MockStateTransition(rafts[learner], INVALID)

		var want, got State
		got = 0
		want = (FOLLOWER + FOLLOWER + INVALID)
		for _, raft := range rafts {
			got += raft.getState()
		}

		// [FOLLOWER, FOLLOWER, INVALID]
		assert.Equal(t, want, got)
	}

	// 2. set rafts[2] to LEARNER
	{
		c, cleanup := MockGetClient(t, names[learner])
		defer cleanup()

		method := model.RPCHASetLearner
		req := model.NewHARPCRequest()
		rsp := model.NewHARPCResponse(model.OK)
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)

		want := model.OK
		got := rsp.RetCode
		assert.Equal(t, want, got)
	}

	// 3. check
	{
		MockWaitLeaderEggs(rafts, 1, replMode, false, -1)

		var want, got State
		got = 0
		want = (LEADER + FOLLOWER + LEARNER)
		for _, raft := range rafts {
			got += raft.getState()
		}
		// [LEADER, FOLLOWER, LEARNER]
		assert.Equal(t, want, got)
	}

	// 4. enable ha for rafts[2]
	{
		c, cleanup := MockGetClient(t, names[learner])
		defer cleanup()

		method := model.RPCHAEnable
		req := model.NewHARPCRequest()
		rsp := model.NewHARPCResponse(model.OK)
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)

		want := model.OK
		got := rsp.RetCode
		assert.Equal(t, want, got)
	}

	// 5. check
	{
		MockWaitLeaderEggs(rafts, 1, replMode, false, -1)

		var want, got State
		got = 0
		want = (LEADER + FOLLOWER + FOLLOWER)
		for _, raft := range rafts {
			got += raft.getState()
		}
		// [LEADER, FOLLOWER, FOLLOWER]
		assert.Equal(t, want, got)
	}
}

func TestRaftRPCHASetLearnerFromInvalid_MySQL(t *testing.T) {
	forEachMySQLReplMode(t, testRaftRPCHASetLearnerFromInvalid)
}

// TEST EFFECTS:
// test a hasetlearner command from idle by the client
//
// TEST PROCESSES:
// 1. Start rpc server
// 2. send command to rpc server
// 3. check the response
func testRaftRPCHASetLearnerFromIdle(t *testing.T, replMode model.MysqlReplMode) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	names, rafts, scleanup := MockRafts(log, port, 3, -1, replMode, false)
	learner := 2
	defer scleanup()

	// 1. Start 3 rafts state as FOLLOWER and set rafts[2] to IDLE
	{
		for _, raft := range rafts {
			raft.Start()
		}

		MockStateTransition(rafts[learner], IDLE)

		var want, got State
		got = 0
		want = (FOLLOWER + FOLLOWER + IDLE)
		for _, raft := range rafts {
			got += raft.getState()
		}

		// [FOLLOWER, FOLLOWER, IDLE]
		assert.Equal(t, want, got)
	}

	// 2. set rafts[2] to LEARNER
	{
		c, cleanup := MockGetClient(t, names[learner])
		defer cleanup()

		method := model.RPCHASetLearner
		req := model.NewHARPCRequest()
		rsp := model.NewHARPCResponse(model.OK)
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)

		want := model.OK
		got := rsp.RetCode
		assert.Equal(t, want, got)
	}

	// 3. check
	{
		MockWaitLeaderEggs(rafts, 1, replMode, false, -1)

		var want, got State
		got = 0
		want = (LEADER + FOLLOWER + LEARNER)
		for _, raft := range rafts {
			got += raft.getState()
		}
		// [LEADER, FOLLOWER, LEARNER]
		assert.Equal(t, want, got)
	}

	// 4. enable ha for rafts[2]
	{
		c, cleanup := MockGetClient(t, names[learner])
		defer cleanup()

		method := model.RPCHAEnable
		req := model.NewHARPCRequest()
		rsp := model.NewHARPCResponse(model.OK)
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)

		want := model.OK
		got := rsp.RetCode
		assert.Equal(t, want, got)
	}

	// 5. check
	{
		MockWaitLeaderEggs(rafts, 1, replMode, false, -1)

		var want, got State
		got = 0
		want = (LEADER + FOLLOWER + FOLLOWER)
		for _, raft := range rafts {
			got += raft.getState()
		}
		// [LEADER, FOLLOWER, FOLLOWER]
		assert.Equal(t, want, got)
	}
}

func TestRaftRPCHASetLearnerFromIdle_MySQL(t *testing.T) {
	forEachMySQLReplMode(t, testRaftRPCHASetLearnerFromIdle)
}

// TEST EFFECTS:
// test TryToLeader command from the client
//
// TEST PROCESSES:
// 1. Start rpc server
// 2. send command to rpc server
// 3. check the response
func testRaftRPCHATryToLeader(t *testing.T, replMode model.MysqlReplMode) {
	if replMode == model.ReplModeMGR {
		testRaftRPCHATryToLeaderMGR(t)
		return
	}
	testRaftRPCHATryToLeaderSemiSync(t)
}

func testRaftRPCHATryToLeaderSemiSync(t *testing.T) {
	var whoisleader, whoisleadernow int
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	names, rafts, scleanup := MockRafts(log, port, 3, -1, model.ReplModeSemiSync, false)
	defer scleanup()

	// 1. Start 3 rafts state as FOLLOWER
	{
		for _, raft := range rafts {
			raft.Start()
		}

		var want, got State
		got = 0
		want = (FOLLOWER + FOLLOWER + FOLLOWER)
		for _, raft := range rafts {
			got += raft.getState()
		}

		// [FOLLOWER, FOLLOWER, FOLLOWER]
		assert.Equal(t, want, got)
	}

	// 2. check
	{
		MockWaitLeaderEggs(rafts, 1, model.ReplModeSemiSync, false, -1)

		var want, got State
		got = 0
		want = (LEADER + FOLLOWER + FOLLOWER)
		for i, raft := range rafts {
			got += raft.getState()
			if raft.getState() == LEADER {
				whoisleader = i
			}
		}
		// [LEADER, FOLLOWER, FOLLOWER]
		assert.Equal(t, want, got)
	}

	// 3. try to leader
	{
		for i := range names {
			if i == whoisleader {
				continue
			}
			func(idx int) {
				c, cleanup := MockGetClient(t, names[idx])
				defer cleanup()

				method := model.RPCHATryToLeader
				req := model.NewHARPCRequest()
				rsp := model.NewHARPCResponse(model.OK)
				err := c.Call(method, req, rsp)
				assert.Nil(t, err)

				want := model.OK
				got := rsp.RetCode
				assert.Equal(t, want, got)
				whoisleadernow = idx
			}(i)
			break
		}
	}

	// 4. check
	{
		MockWaitLeaderEggs(rafts, 1, model.ReplModeSemiSync, false, -1)
		assert.NotEqual(t, whoisleader, whoisleadernow)

		var want, got State
		got = 0
		want = (LEADER + FOLLOWER + FOLLOWER)
		for i, raft := range rafts {
			got += raft.getState()
			if raft.getState() == LEADER {
				assert.Equal(t, i, whoisleadernow)
			}
		}
		// [LEADER, FOLLOWER, FOLLOWER]
		assert.Equal(t, want, got)
	}
}

// testRaftRPCHATryToLeaderMGR covers the MGR manual TryToLeader success path.
// The candidate must be promotable and have a GTID at least as advanced as the
// current leader so RequestVote is not rejected with ErrorInvalidGTID.
func testRaftRPCHATryToLeaderMGR(t *testing.T) {
	const GTIDBIDX = 1
	const GTIDCIDX = 2

	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	names, rafts, scleanup := MockRafts(log, port, 3, -1, model.ReplModeMGR, false)
	defer scleanup()

	{
		rafts[0].mysql.SetMysqlHandler(mysql.NewMockGTIDX1GetMGRGTIDError())
		rafts[GTIDBIDX].mysql.SetMysqlHandler(mysql.NewMockGTIDX1())
		rafts[GTIDCIDX].mysql.SetMysqlHandler(mysql.NewMockGTIDX3())
	}

	for _, raft := range rafts {
		raft.Start()
	}

	MockWaitLeaderEggs(rafts[1:], 1, model.ReplModeMGR, true, -1)

	var whoisleader int
	for i, raft := range rafts {
		if raft.getState() == LEADER {
			whoisleader = i
		}
	}
	assert.Equal(t, GTIDCIDX, whoisleader)

	// Demote the current leader GTID so node 1 can win the manual election.
	rafts[GTIDCIDX].mysql.SetMysqlHandler(mysql.NewMockGTIDX1())
	rafts[GTIDBIDX].mysql.SetMysqlHandler(mysql.NewMockGTIDX3MGRCaughtUp())

	c, cleanup := MockGetClient(t, names[GTIDBIDX])
	defer cleanup()

	method := model.RPCHATryToLeader
	req := model.NewHARPCRequest()
	rsp := model.NewHARPCResponse(model.OK)
	err := c.Call(method, req, rsp)
	assert.Nil(t, err)
	assert.Equal(t, model.OK, rsp.RetCode)

	MockWaitLeaderEggs(rafts, 1, model.ReplModeMGR, true, GTIDCIDX)
	assert.Equal(t, GTIDBIDX, MockFindLeader(rafts))

	got := State(0)
	for i, raft := range rafts {
		got += raft.getState()
		if raft.getState() == LEADER {
			assert.Equal(t, GTIDBIDX, i)
		}
	}
	assert.Equal(t, LEADER+FOLLOWER+FOLLOWER, got)
}

func TestRaftRPCHATryToLeader_MySQL(t *testing.T) {
	forEachMySQLReplMode(t, testRaftRPCHATryToLeader)
}

// TEST EFFECTS:
// test HATryToLeader RPC failed call from the client
//
// TEST PROCESSES:
// 1. Start rpc server
// 2. send command to rpc server
// 3. check the response
func testRaftRPCHATryToLeaderFailGTIDSemiSync(t *testing.T) {
	var whoisleader int

	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	names, rafts, cleanup := MockRafts(log, port, 3, -1, model.ReplModeSemiSync, false)
	defer cleanup()

	GTIDBIDX := 1
	GTIDCIDX := 2

	// 1. set rafts GTID
	//    1.0 rafts[0]  with MockGTIDB{Master_Log_File = "", Read_Master_Log_Pos = 0}
	//    1.1 rafts[1]  with MockGTIDB{Master_Log_File = "mysql-bin.000001", Read_Master_Log_Pos = 123}
	//    1.2 rafts[2]  with MockGTIDC{Master_Log_File = "mysql-bin.000001", Read_Master_Log_Pos = 124}
	{
		rafts[GTIDBIDX].mysql.SetMysqlHandler(mysql.NewMockGTIDB())
		rafts[GTIDCIDX].mysql.SetMysqlHandler(mysql.NewMockGTIDC())
	}

	// 2. Start 3 rafts state as FOLLOWER
	{
		for _, raft := range rafts {
			raft.Start()
		}

		var got State
		want := (FOLLOWER + FOLLOWER + FOLLOWER)
		for _, raft := range rafts {
			got += raft.getState()
		}

		// [FOLLOWER, FOLLOWER, FOLLOWER]
		assert.Equal(t, want, got)
	}

	// 3. wait rafts[2] elected as leader
	{
		MockWaitSomeElectionTimeout(rafts, 2)
		MockWaitLeaderEggs(rafts, 1, model.ReplModeSemiSync, false, -1)

		var got State
		whoisleader = 0
		want := (LEADER + FOLLOWER + FOLLOWER)
		for i, raft := range rafts {
			got += raft.getState()
			if raft.getState() == LEADER {
				whoisleader = i
			}
		}
		// [LEADER, FOLLOWER, FOLLOWER]
		assert.Equal(t, want, got)

		// leader should be rafts[GTIDCIDX]
		assert.Equal(t, GTIDCIDX, whoisleader)
	}

	// 4. try rafts[1] to leader
	{
		c, cleanup := MockGetClient(t, names[1])
		defer cleanup()

		method := model.RPCHATryToLeader
		req := model.NewHARPCRequest()
		rsp := model.NewHARPCResponse(model.OK)
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)

		want := model.OK
		got := rsp.RetCode
		assert.Equal(t, want, got)
	}

	// 4.1 wait rafts[2] elected as leader again
	{
		var got, g State
		var foundLeader bool
		whoisleader = 0
		want := (LEADER + FOLLOWER + FOLLOWER)
		for !foundLeader {
			g = 0
			MockWaitLeaderEggs(rafts, 0, model.ReplModeSemiSync, false, -1)
			for i, raft := range rafts {
				g += raft.getState()
				if raft.getState() == LEADER {
					if i == GTIDCIDX {
						whoisleader = i
						foundLeader = true
					}
				}
			}
			got = g
		}
		// [LEADER, FOLLOWER, FOLLOWER]
		assert.Equal(t, want, got)

		// leader should be rafts[GTIDCIDX]
		assert.Equal(t, whoisleader, GTIDCIDX)
	}
}

// TEST EFFECTS:
// test HATryToLeader RPC failed call from the client
//
// TEST PROCESSES:
// 1. Start rpc server
// 2. send command to rpc server
// 3. check the response
func testRaftRPCHATryToLeaderFailGTIDMGR(t *testing.T) {
	var whoisleader int

	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	names, rafts, cleanup := MockRafts(log, port, 3, -1, model.ReplModeMGR, false)
	defer cleanup()

	GTIDBIDX := 1
	GTIDCIDX := 2

	/*
		1. set rafts GTID
		1.0 rafts[0] with MockGTID{
			Master_Log_File = "mysql-bin.000001",
			Read_Master_Log_Pos = 123,
			Executed_GTID_Set = "",
			Retrieved_GTID_Set = "",
			Txns_Behind_Master = ""
		}
		1.1 rafts[1]  with MockGTID{
			Master_Log_File = "mysql-bin.000001",
			Read_Master_Log_Pos = 123,
			Executed_GTID_Set = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-99",
			Retrieved_GTID_Set = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-102",
			Txns_Behind_Master = "3"
		}
		1.2 rafts[2]  with MockGTID{
			Master_Log_File = "mysql-bin.000003",
			Read_Master_Log_Pos = 123,
			Executed_GTID_Set = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-99:1000-1002",
			Retrieved_GTID_Set = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-102:1000-1002",
			Txns_Behind_Master = "3"
		}
	*/
	{
		rafts[0].mysql.SetMysqlHandler(mysql.NewMockGTIDX1GetMGRGTIDError())
		rafts[GTIDBIDX].mysql.SetMysqlHandler(mysql.NewMockGTIDX1())
		rafts[GTIDCIDX].mysql.SetMysqlHandler(mysql.NewMockGTIDX3())
	}

	// 2. Start 3 rafts state as FOLLOWER
	{
		for _, raft := range rafts {
			raft.Start()
		}

		var got State
		want := (FOLLOWER + FOLLOWER + FOLLOWER)
		for _, raft := range rafts {
			got += raft.getState()
		}

		// [FOLLOWER, FOLLOWER, FOLLOWER]
		assert.Equal(t, want, got)
	}

	// 3. wait rafts[2] elected as leader
	{
		MockWaitLeaderEggs(rafts[1:], 1, model.ReplModeMGR, true, -1)

		var got State
		whoisleader = 0
		want := (LEADER + FOLLOWER + FOLLOWER)
		for i, raft := range rafts {
			got += raft.getState()
			if raft.getState() == LEADER {
				whoisleader = i
			}
		}
		// [LEADER, FOLLOWER, FOLLOWER]
		assert.Equal(t, want, got)

		// leader should be rafts[GTIDCIDX]
		assert.Equal(t, whoisleader, GTIDCIDX)
	}

	// 4. try rafts[1] to leader
	{
		rafts[GTIDBIDX].mysql.SetMysqlHandler(mysql.NewMockGTIDX1MGRCaughtUp())

		c, cleanup := MockGetClient(t, names[1])
		defer cleanup()

		method := model.RPCHATryToLeader
		req := model.NewHARPCRequest()
		rsp := model.NewHARPCResponse(model.OK)
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)

		want := model.OK
		got := rsp.RetCode
		assert.Equal(t, want, got)
	}

	// 4.1 wait rafts[2] elected as leader again
	{
		var got State
		want := (LEADER + FOLLOWER + FOLLOWER)
		found := false
		for attempt := 0; attempt < 10 && !found; attempt++ {
			rafts[0].mysql.SetMysqlHandler(mysql.NewMockGTIDX1GetMGRGTIDError())
			rafts[GTIDBIDX].mysql.SetMysqlHandler(mysql.NewMockGTIDX1())
			rafts[GTIDCIDX].mysql.SetMysqlHandler(mysql.NewMockGTIDX3())

			MockWaitSomeElectionTimeout(rafts, 2)
			MockWaitLeaderEggs(rafts[1:], 1, model.ReplModeMGR, true, -1)

			got = 0
			whoisleader = -1
			for i, raft := range rafts {
				got += raft.getState()
				if raft.getState() == LEADER {
					whoisleader = i
				}
			}
			if whoisleader == GTIDCIDX {
				found = true
			}
		}
		assert.True(t, found, "rafts[%d] should become leader again", GTIDCIDX)
		assert.Equal(t, want, got)
	}
}

func TestRaftRPCHATryToLeaderFailGTID_MySQL(t *testing.T) {
	forEachMySQLReplMode(t, func(t *testing.T, replMode model.MysqlReplMode) {
		if replMode == model.ReplModeSemiSync {
			testRaftRPCHATryToLeaderFailGTIDSemiSync(t)
		} else {
			testRaftRPCHATryToLeaderFailGTIDMGR(t)
		}
	})
}

// TEST EFFECTS:
// test TryToLeader command from the client
//
// TEST PROCESSES:
// 1. Start rpc server
// 2. send command to rpc server
// 3. check the response
func testRaftRPCHATryToLeaderFailUnpromotbleSemiSync(t *testing.T) {
	var whoisleader int

	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	names, rafts, cleanup := MockRafts(log, port, 3, -1, model.ReplModeSemiSync, false)
	defer cleanup()

	GTIDERRIDX := 0
	GTIDBIDX := 1
	GTIDCIDX := 2

	// 1. set rafts GTID
	//    1.0 rafts[0]  with Ping error(mocks MySQL down)
	//    1.1 rafts[1]  with MockGTIDB{Master_Log_File = "mysql-bin.000001", Read_Master_Log_Pos = 123}
	//    1.2 rafts[2]  with MockGTIDC{Master_Log_File = "mysql-bin.000001", Read_Master_Log_Pos = 124}
	{
		rafts[GTIDERRIDX].mysql.SetMysqlHandler(mysql.NewMockGTIDPingError())
		rafts[GTIDBIDX].mysql.SetMysqlHandler(mysql.NewMockGTIDB())
		rafts[GTIDCIDX].mysql.SetMysqlHandler(mysql.NewMockGTIDC())
	}

	// 2. Start 3 rafts state as FOLLOWER
	{
		for _, raft := range rafts {
			raft.Start()
		}

		var got State
		want := (FOLLOWER + FOLLOWER + FOLLOWER)
		for _, raft := range rafts {
			got += raft.getState()
		}

		// [FOLLOWER, FOLLOWER, FOLLOWER]
		assert.Equal(t, want, got)
	}

	// 3. wait rafts[2] elected as leader
	{
		MockWaitLeaderEggs(rafts, 1, model.ReplModeSemiSync, false, -1)

		var got State
		whoisleader = 0
		want := (LEADER + FOLLOWER + FOLLOWER)
		for i, raft := range rafts {
			got += raft.getState()
			if raft.getState() == LEADER {
				whoisleader = i
			}
		}
		// [LEADER, FOLLOWER, FOLLOWER]
		assert.Equal(t, want, got)

		// leader should be rafts[GTIDCIDX]
		assert.Equal(t, GTIDCIDX, whoisleader)
	}

	// 4. try rafts[2](already leader) to leader
	{
		c, cleanup := MockGetClient(t, names[2])
		defer cleanup()

		method := model.RPCHATryToLeader
		req := model.NewHARPCRequest()
		rsp := model.NewHARPCResponse(model.OK)
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)

		want := model.OK
		got := rsp.RetCode
		assert.Equal(t, want, got)
	}

	// 4. try rafts[0] to leader
	{
		c, cleanup := MockGetClient(t, names[0])
		defer cleanup()

		method := model.RPCHATryToLeader
		req := model.NewHARPCRequest()
		rsp := model.NewHARPCResponse(model.OK)
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)

		want := model.RPCError_MySQLUnpromotable
		got := rsp.RetCode
		assert.Equal(t, want, got)
	}

	// 4.1 wait rafts[2] elected as leader again
	{
		MockWaitSomeElectionTimeout(rafts, 4)
		MockWaitLeaderEggs(rafts, 1, model.ReplModeSemiSync, false, -1)

		var got State
		whoisleader = 0
		want := (LEADER + FOLLOWER + FOLLOWER)
		for i, raft := range rafts {
			got += raft.getState()
			if raft.getState() == LEADER {
				whoisleader = i
			}
		}
		// [LEADER, FOLLOWER, FOLLOWER]
		assert.Equal(t, want, got)

		// leader should be rafts[GTIDCIDX]
		assert.Equal(t, whoisleader, GTIDCIDX)
	}
}

// TEST EFFECTS:
// test TryToLeader command from the client
//
// TEST PROCESSES:
// 1. Start rpc server
// 2. send command to rpc server
// 3. check the response
func testRaftRPCHATryToLeaderFailUnpromotbleMGR(t *testing.T) {
	var whoisleader int

	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	names, rafts, cleanup := MockRafts(log, port, 3, -1, model.ReplModeMGR, false)
	defer cleanup()

	GTIDERRIDX := 0
	GTIDBIDX := 1
	GTIDCIDX := 2

	/*
		1. set rafts GTID
		1.0 rafts[0] with MockGTID{
			Master_Log_File = "mysql-bin.000001",
			Read_Master_Log_Pos = 123,
			Executed_GTID_Set = "",
			Retrieved_GTID_Set = "",
			Txns_Behind_Master = ""
		}
		1.1 rafts[1]  with MockGTID{
			Master_Log_File = "mysql-bin.000001",
			Read_Master_Log_Pos = 123,
			Executed_GTID_Set = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-99",
			Retrieved_GTID_Set = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-102",
			Txns_Behind_Master = "3"
		}
		1.2 rafts[2]  with MockGTID{
			Master_Log_File = "mysql-bin.000003",
			Read_Master_Log_Pos = 123,
			Executed_GTID_Set = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-99:1000-1002",
			Retrieved_GTID_Set = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa:1-102:1000-1002",
			Txns_Behind_Master = "3"
		}
	*/
	{
		rafts[GTIDERRIDX].mysql.SetMysqlHandler(mysql.NewMockGTIDPingError())
		rafts[GTIDBIDX].mysql.SetMysqlHandler(mysql.NewMockGTIDX1())
		rafts[GTIDCIDX].mysql.SetMysqlHandler(mysql.NewMockGTIDX3())
	}

	// 2. Start 3 rafts state as FOLLOWER
	{
		for _, raft := range rafts {
			raft.Start()
		}

		var got State
		want := (FOLLOWER + FOLLOWER + FOLLOWER)
		for _, raft := range rafts {
			got += raft.getState()
		}

		// [FOLLOWER, FOLLOWER, FOLLOWER]
		assert.Equal(t, want, got)
	}

	// 3. wait rafts[2] elected as leader
	{
		MockWaitLeaderEggs(rafts[1:], 1, model.ReplModeMGR, true, -1)

		var got State
		whoisleader = 0
		want := (LEADER + FOLLOWER + FOLLOWER)
		for i, raft := range rafts {
			got += raft.getState()
			if raft.getState() == LEADER {
				whoisleader = i
			}
		}
		// [LEADER, FOLLOWER, FOLLOWER]
		assert.Equal(t, want, got)

		// leader should be rafts[GTIDCIDX]
		assert.Equal(t, whoisleader, GTIDCIDX)
	}

	// 4. try rafts[2](already leader) to leader
	{
		c, cleanup := MockGetClient(t, names[2])
		defer cleanup()

		method := model.RPCHATryToLeader
		req := model.NewHARPCRequest()
		rsp := model.NewHARPCResponse(model.OK)
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)

		want := model.OK
		got := rsp.RetCode
		assert.Equal(t, want, got)
	}

	// 4. try rafts[0] to leader
	{
		c, cleanup := MockGetClient(t, names[0])
		defer cleanup()

		method := model.RPCHATryToLeader
		req := model.NewHARPCRequest()
		rsp := model.NewHARPCResponse(model.OK)
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)

		want := model.RPCError_MySQLUnpromotable
		got := rsp.RetCode
		assert.Equal(t, want, got)
	}

	// 4.1 wait rafts[2] elected as leader again
	{
		MockWaitSomeElectionTimeout(rafts, 4)
		MockWaitLeaderEggs(rafts[1:], 1, model.ReplModeMGR, true, -1)

		var got State
		whoisleader = 0
		want := (LEADER + FOLLOWER + FOLLOWER)
		for i, raft := range rafts {
			got += raft.getState()
			if raft.getState() == LEADER {
				whoisleader = i
			}
		}
		// [LEADER, FOLLOWER, FOLLOWER]
		assert.Equal(t, want, got)

		// leader should be rafts[GTIDCIDX]
		assert.Equal(t, whoisleader, GTIDCIDX)
	}
}

func TestRaftRPCHATryToLeaderFailUnpromotble_MySQL(t *testing.T) {
	forEachMySQLReplMode(t, func(t *testing.T, replMode model.MysqlReplMode) {
		if replMode == model.ReplModeSemiSync {
			testRaftRPCHATryToLeaderFailUnpromotbleSemiSync(t)
		} else {
			testRaftRPCHATryToLeaderFailUnpromotbleMGR(t)
		}
	})
}

func testRaftSuperIDLEEnableHA(t *testing.T, replMode model.MysqlReplMode) {
	var testName = "TestRaftSuperIDLEEnableHA"
	var want, got State
	var whoisleader int
	var leader *Raft
	var idler *Raft

	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	names, rafts, cleanup := MockRafts(log, port, 3, 2, replMode, false)
	defer cleanup()

	// 1.  Start 3 rafts.
	{
		for i, raft := range rafts {
			if i == 2 {
				raft.conf.SuperIDLE = true
				idler = raft
			}
			raft.Start()
		}
	}

	// 2.  wait leader election.
	{
		MockWaitLeaderEggs(rafts, 1, replMode, replMode == model.ReplModeMGR, -1)
		whoisleader = 0
		got = 0
		want = (LEADER + FOLLOWER + IDLE)
		for i, raft := range rafts {
			got += raft.getState()
			if raft.getState() == LEADER {
				whoisleader = i
			}
		}
		// [LEADER, FOLLOWER, IDLE]
		assert.Equal(t, want, got)
	}
	idlerLeader1 := idler.getLeader()

	// 3.  set leader handlers to mock
	{
		leader = rafts[whoisleader]
		log.Warning("%v.leader[%v].set.mock.functions", testName, rafts[whoisleader].getID())
		leader.L.setProcessHeartbeatRequestHandler(leader.mockLeaderProcessHeartbeatRequest)
		leader.L.setProcessRequestVoteRequestHandler(leader.mockLeaderProcessRequestVoteRequest)
	}

	// 4.  Stop leader hearbeat
	{
		log.Warning("%v.leader[%v].Stop.heartbeat", testName, leader.getID())
		leader.L.setSendHeartbeatHandler(leader.mockLeaderSendHeartbeat)
	}

	// 5.  Wait new leader.
	{
		MockWaitLeaderEggs(rafts, 1, replMode, false, -1)
	}

	{
		c, cleanup := MockGetClient(t, names[2])
		defer cleanup()

		method := model.RPCHAEnable
		req := model.NewHARPCRequest()
		rsp := model.NewHARPCResponse(model.OK)
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)

		want := model.OK
		got := rsp.RetCode
		assert.Equal(t, want, got)
	}

	idlerLeader2 := idler.getLeader()
	assert.NotEqual(t, idlerLeader1, idlerLeader2)
}

func TestRaftSuperIDLEEnableHA_MySQL(t *testing.T) {
	forEachMySQLReplMode(t, testRaftSuperIDLEEnableHA)
}
