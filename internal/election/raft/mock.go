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
	"context"
	"fmt"
	"github.com/sealdb/neoha/internal/database"
	"os"
	"testing"
	"time"

	"github.com/sealdb/neoha/internal/base/common"
	"github.com/sealdb/neoha/internal/base/model"
	"github.com/sealdb/neoha/internal/base/nlog"
	"github.com/sealdb/neoha/internal/base/nrpc"
	"github.com/sealdb/neoha/internal/config"
	"github.com/sealdb/neoha/internal/database/mysql"

	"github.com/stretchr/testify/assert"
)

var (
	shortHeartbeatTimeoutForTest = 100
	longHeartbeatTimeoutForTest  = 500
)

func setupRPC(rpc *nrpc.Service, raft *Raft) {
	if err := rpc.RegisterService(raft.GetHARPC()); err != nil {
		raft.PANIC("server.rpc.RegisterService.HARPC.error[%+v]", err)
	}

	if err := rpc.RegisterService(raft.GetRaftRPC()); err != nil {
		raft.PANIC("server.rpc.RegisterService.RaftRPC.error[%+v]", err)
	}
}

// MockRaftsWithConfig mock.
func MockRaftsWithConfig(log *nlog.Log, conf *config.Config, port int, count int, idleStart int) ([]string, []*Raft, func()) {
	return mockRafts(log, conf, port, count, idleStart, false)
}

// MockRafts mock.
// If no idle nodes, set idleStart to -1
func MockRafts(log *nlog.Log, port int, count int, idleStart int, mode model.MysqlReplMode, isLong bool) ([]string, []*Raft, func()) {
	conf := config.DefaultConfig()
	conf.Election.Raft.PurgeBinlogInterval = 1
	conf.Election.Raft.CandidateWaitFor2Nodes = 1000
	conf.Election.Raft.MetaDatadir = "/tmp/"
	conf.Database.Mysql.ReplMode = mode

	return mockRafts(log, conf, port, count, idleStart, isLong)
}

func mockRafts(log *nlog.Log, conf *config.Config, port int, count int, idleStart int, isLong bool) ([]string, []*Raft, func()) {
	ids := []string{}
	var raft *Raft
	rafts := []*Raft{}
	rpcs := []*nrpc.Service{}
	metaDirs := []string{}
	ip, _ := common.GetLocalIP()

	for i := 0; i < count; i++ {
		id := fmt.Sprintf("%s:%d", ip, port+i)
		ids = append(ids, id)

		nodeConf := *conf
		raftCfg := *conf.Election.Raft
		if isLong {
			raftCfg.HeartbeatTimeout = longHeartbeatTimeoutForTest
			raftCfg.ElectionTimeout = longHeartbeatTimeoutForTest * 3
		} else {
			raftCfg.HeartbeatTimeout = shortHeartbeatTimeoutForTest
			raftCfg.ElectionTimeout = shortHeartbeatTimeoutForTest * 3
		}
		metaDir := fmt.Sprintf("/tmp/neoha-raft-%d-%d", port, i)
		_ = os.RemoveAll(metaDir)
		_ = os.MkdirAll(metaDir, 0o755)
		metaDirs = append(metaDirs, metaDir)
		raftCfg.MetaDatadir = metaDir
		nodeConf.Endpoint = id
		electionCfg := *conf.Election
		electionCfg.Raft = &raftCfg
		nodeConf.Election = &electionCfg

		// setup mysql
		nodeConf.Database.Mysql.Version = "mysql57"
		db := database.NewDatabase(nodeConf.Database, database.MySQL, 10000, log)
		if nodeConf.Database.Mysql.ReplMode == model.ReplModeSemiSync {
			db.GetMysql().SetMysqlHandler(mysql.NewMockGTIDA(nodeConf.Database.Mysql.ReplMode))
		} else {
			db.GetMysql().SetMysqlHandler(mysql.NewMockGTIDByNodes(count, -1, i))
		}
		db.Start()

		// setup raft
		initState := FOLLOWER
		if idleStart != -1 && i >= idleStart {
			initState = IDLE
		}
		raft = NewRaft(&nodeConf, log, db, database.MySQL, initState)
		raft.id = id
		raft.semiSyncTimeoutFor2Nodes = 10000
		rafts = append(rafts, raft)

		// setup rpc
		rpc, err := nrpc.NewService(nrpc.Log(log),
			nrpc.ConnectionStr(id))
		if err != nil {
			log.Panic("raftRPC.NewService.error[%+v]", err)
		}
		setupRPC(rpc, raft)
		rpcs = append(rpcs, rpc)
		rpc.Start()
	}

	for _, raft := range rafts {
		for i, id := range ids {
			if idleStart != -1 && i >= idleStart {
				raft.AddIdlePeer(id)
			} else {
				raft.AddPeer(id)
			}
		}
	}

	return ids, rafts, func() {
		for _, dir := range metaDirs {
			_ = os.RemoveAll(dir)
		}
		for i, r := range rafts {
			rpcs[i].Stop()
			r.Stop()
		}
	}
}

// MockCountPeerInLists counts how many rafts include peer in GetPeers().
func MockCountPeerInLists(rafts []*Raft, peer string) int {
	cnt := 0
	for _, raft := range rafts {
		for _, p := range raft.GetPeers() {
			if p == peer {
				cnt++
				break
			}
		}
	}
	return cnt
}

// MockWaitPeerListCount waits until peer appears in want rafts' peer lists.
func MockWaitPeerListCount(rafts []*Raft, peer string, want int) int {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cnt := MockCountPeerInLists(rafts, peer); cnt == want {
			return cnt
		}
		time.Sleep(50 * time.Millisecond)
	}
	return MockCountPeerInLists(rafts, peer)
}

// MockResetMGRHandlers resets per-node MGR mocks before a fresh election round.
func MockResetMGRHandlers(rafts []*Raft) {
	for i, raft := range rafts {
		raft.mysql.SetMysqlHandler(mysql.NewMockGTIDByNodes(len(rafts), -1, i))
	}
}

// MockUseSimpleMGRHandlers uses lightweight MGR mocks that do not block Raft election.
func MockUseSimpleMGRHandlers(rafts []*Raft) {
	for _, raft := range rafts {
		raft.mysql.SetMysqlHandler(mysql.NewMockGTIDX1ForMGR())
	}
}

// MockSetMysqlHandler used to set mysql repl hander for test.
func MockSetMysqlHandler(raft *Raft, h mysql.MysqlHandler) {
	raft.mysql.SetMysqlHandler(h)
}

// MockWaitMySQLPingTimeout used to wait mysql ping timeout.
func MockWaitMySQLPingTimeout() {
	pingTimeout := config.DefaultMysqlConfig().PingTimeout * 6
	time.Sleep(time.Millisecond * time.Duration(pingTimeout))
}

// MockWaitHeartBeatTimeout used to wait mysql ping timeout.
func MockWaitHeartBeatTimeout() {
	hbTimeout := config.DefaultRaftConfig().HeartbeatTimeout * 6
	time.Sleep(time.Millisecond * time.Duration(hbTimeout))
}

// MockWaitSomeElectionTimeout mock.
// we just want to sleep for a heartbeat broadcast
func MockWaitSomeElectionTimeout(rafts []*Raft, nums int) {
	time.Sleep(time.Millisecond * time.Duration(rafts[0].getElectionTimeout()*nums))
}

// MockWaitUntil polls cond until true or timeout.
func MockWaitUntil(timeout, interval time.Duration, cond func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(interval)
	}
	return false
}

// MockWaitLeaderPurgeBinlogs waits until leader purge counter reaches at least want.
func MockWaitLeaderPurgeBinlogs(raft *Raft, want uint64, timeout time.Duration) bool {
	return MockWaitUntil(timeout, 50*time.Millisecond, func() bool {
		return raft.stats.LeaderPurgeBinlogs >= want
	})
}

// MockWaitLeaderEggs mock.
// Wait the leader eggs when leadernums >0.
// When leadernums == 0, sleep for two heartbeat periods (epoch broadcast).
// The variables isXn and ignore are valid only for MGR.
func MockWaitLeaderEggs(rafts []*Raft, leadernums int, replMode model.MysqlReplMode, isXn bool, ignore int) int {
	if leadernums == 0 {
		if len(rafts) > 0 {
			time.Sleep(time.Millisecond * time.Duration(rafts[0].getElectionTimeout()*2))
		}
		return -1
	}

	maxRunTime := 15 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), maxRunTime)
	defer cancel()

	done := make(chan int, 1)
	go func() {
		whoisleader := -1
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			nums := 0
			for i, raft := range rafts {
				if raft.getState() == LEADER && i != ignore {
					nums++
					whoisleader = i
					if replMode == model.ReplModeMGR {
						for j, r := range rafts {
							if j == ignore {
								continue
							}
							if isXn {
								r.mysql.SetMysqlHandler(mysql.NewMockGTIDByNodesXn(len(rafts), whoisleader, j))
							} else {
								r.mysql.SetMysqlHandler(mysql.NewMockGTIDByNodes(len(rafts), whoisleader, j))
							}
						}
					}
					if nums == leadernums {
						time.Sleep(time.Millisecond * time.Duration(rafts[0].getElectionTimeout()))
						select {
						case done <- whoisleader:
						case <-ctx.Done():
						}
						return
					}
				}
			}
			time.Sleep(50 * time.Millisecond)
		}
	}()

	select {
	case <-ctx.Done():
		return -1
	case idx := <-done:
		return idx
	}
}

// MockInitGTID used to reinit Mysql handler.
func MockInitGTID(rafts []*Raft) {
	for _, raft := range rafts {
		raft.mysql.SetMysqlHandler(mysql.NewMockGTIDX1())
	}
}

// MockStateTransition use to transfer the raft.state to state.
func MockStateTransition(raft *Raft, state State) {
	raft.setState(state)
	raft.loopFired()
	// Allow the state loop to exit the prior state and enter the new one.
	time.Sleep(time.Duration(shortHeartbeatTimeoutForTest*4) * time.Millisecond)
}

// MockGetClient mock.
func MockGetClient(t *testing.T, svrConn string) (*nrpc.Client, func()) {
	client, err := nrpc.NewClient(svrConn, 3000)
	assert.Nil(t, err)

	return client, func() {
		client.Close()
	}
}

// MockFindLeader returns the index of the current LEADER raft, or -1.
func MockFindLeader(rafts []*Raft) int {
	for i, raft := range rafts {
		if raft.getState() == LEADER {
			return i
		}
	}
	return -1
}

// mock leader process heartbeat request who comes from the new leader
// nop here, so our leader state won't be changed(degrade)
func (r *Raft) mockLeaderProcessHeartbeatRequest(req *model.RaftRPCRequest) *model.RaftRPCResponse {
	rsp := model.NewRaftRPCResponse(model.ErrorInvalidRequest)
	rsp.Raft.From = r.getID()
	rsp.Raft.ViewID = r.getViewID()
	rsp.Raft.EpochID = r.getEpochID()
	r.INFO("mock.get.heartbeat.from[N:%v, V:%v, E:%v]", req.GetFrom(), req.GetViewID(), req.GetEpochID())

	return rsp
}

// mock leader process requestvote request from other candidate
// nop here, so our leader state won't be changed(degrade)
func (r *Raft) mockLeaderProcessRequestVoteRequest(req *model.RaftRPCRequest) *model.RaftRPCResponse {
	rsp := model.NewRaftRPCResponse(model.ErrorInvalidRequest)
	rsp.Raft.From = r.getID()
	rsp.Raft.ViewID = r.getViewID()
	rsp.Raft.EpochID = r.getEpochID()

	r.INFO("mock.get.requestvote.from[N:%v, V:%v, E:%v]", req.GetFrom(), req.GetViewID(), req.GetEpochID())

	return rsp
}

// mock leader send heartbeat request
// nop here, so other followers will start a new leader election
func (r *Raft) mockLeaderSendHeartbeat(mysqlDown *bool, c chan *model.RaftRPCResponse) {
	r.DEBUG("mock.send.nop.heartbeat.request")
}

// mock leader prpcess heartbeat response
func (r *Raft) mockLeaderProcessSendHeartbeatResponse(ackGranted *int, mgrCnt *int, rsp *model.RaftRPCResponse) {
	r.DEBUG("mock.send.heartbeat.get.rsp[N:%v, V:%v, E:%v].retcode[%v]", rsp.GetFrom(), rsp.GetViewID(), rsp.GetEpochID(), rsp.RetCode)
}
