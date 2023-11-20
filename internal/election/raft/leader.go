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
	"strings"
	"sync"
	"time"

	"github.com/sealdb/neoha/internal/base/common"
	"github.com/sealdb/neoha/internal/base/model"
)

// Leader tuple.
type Leader struct {
	*Raft
	// the smallest binlog which slaves executed by SQL-Thread
	relayMasterLogFile string

	// leader degrade to follower
	isDegradeToFollower bool

	isReadOnly bool
	isVipSet   bool

	// Used to wait for the async job done.
	wg sync.WaitGroup

	// the binlog which we should purge to
	nextPuregeBinlog string

	// Background workers (purgeBinlog, checkSemiSync, checkGTID) each run in their
	// own goroutine driven by a time.Ticker. Stop closes a dedicated channel and
	// stops the ticker on the Leader struct; the goroutine captures the ticker in
	// a local variable before the loop. Without that local reference, Stop could
	// nil the struct field while select still evaluates tick.C, which leaked
	// goroutines and caused nil-pointer panics in unit tests.
	purgeBinlogTick   *time.Ticker
	purgeStop         chan struct{}
	checkSemiSyncTick *time.Ticker
	semiSyncStop      chan struct{}
	checkGTIDTick     *time.Ticker
	gtidCheckStop     chan struct{}

	// leader process heartbeat request handler
	processHeartbeatRequestHandler func(*model.RaftRPCRequest) *model.RaftRPCResponse

	// leader process voterequest request handler
	processRequestVoteRequestHandler func(*model.RaftRPCRequest) *model.RaftRPCResponse

	// leader send heartbeat request to other followers
	sendHeartbeatHandler func(*bool, chan *model.RaftRPCResponse)

	// leader process send heartbeat response
	processHeartbeatResponseHandler func(*int, *int, *model.RaftRPCResponse)

	// leader process ping request handler
	processPingRequestHandler func(*model.RaftRPCRequest) *model.RaftRPCResponse
}

const (
	semisyncTimeout = 1000000000000000000 // for 3 or more nodes
)

// NewLeader creates new Leader.
func NewLeader(r *Raft) *Leader {
	L := &Leader{
		Raft: r,
	}
	L.initHandlers()
	return L
}

// Loop used to start the loop of the state machine.
// --------------------------------------
// State Machine
// --------------------------------------
//
//	higher viewid
//
// State1. LEADER ------------------> FOLLOWER
func (r *Leader) Loop() {
	r.stateInit()
	defer r.stateExit()

	incViewID := false
	mysqlDown := false
	ackGranted := 1
	mgrCnt := 1
	checkMGR := 0
	maxCheckMGR := 5

	// wait for the maximum number of heartbeat ack
	lessHtAcks := 0
	maxLessHtAcks := r.Raft.conf.AdmitDefeatHtCnt

	// wait for the maximum number of change master to
	lessCmtHtAcks := 0
	maxLessCmtHtAcks := r.Raft.conf.AdmitDefeatHtCnt * 3

	// send heartbeat
	respChan := make(chan *model.RaftRPCResponse, r.getAllMembers())
	r.sendHeartbeatHandler(&mysqlDown, respChan)
	r.resetHeartbeatTimeout()

	for r.getState() == LEADER {
		if mysqlDown {
			r.WARNING("feel.mysql.down.degrade.to.follower")
			r.degradeToFollower()
			break
		}

		select {
		case <-r.fired:
			r.WARNING("state.machine.loop.got.fired")
		case <-r.heartbeatTick.C:
			if r.mysqlReplMode == model.ReplModeSemiSync {
				if ackGranted < r.getQuorums() {
					if r.getMembers() > 2 {
						lessHtAcks++
					}
					r.IncLessHeartbeatAcks()
					r.WARNING("heartbeat.acks.granted[%v].less.than.quorums[%v].lessHtAcks[%v].maxLessHtAcks[%v]", ackGranted, r.getQuorums(), lessHtAcks, maxLessHtAcks)
					if lessHtAcks >= maxLessHtAcks {
						r.WARNING("degrade.to.follower.lessHtAcks[%v]>=maxLessHtAcks[%v]", lessHtAcks, maxLessHtAcks)
						r.degradeToFollower()
						break
					}
				} else {
					lessHtAcks = 0

					// for brain split
					if ackGranted == r.getMembers() {
						if incViewID {
							r.WARNING("heartbeat.acks.granted[%v].equals.members[%v].again", ackGranted, r.getMembers())
							incViewID = false
						}
					} else if !incViewID {
						r.WARNING("heartbeat.acks.granted[%v].less.than.members[%v].for.the.first.time", ackGranted, r.getMembers())
						r.updateView(r.getViewID()+2, r.GetLeader())
						incViewID = true
					}
				}
			} else { // MGR TODO: 需要重构
				if r.cmtState == CmtOK {
					// TODO: ackGranted should not contain the nodes which start-as-idle
					if ackGranted < r.getQuorums() {
						if r.getMembers() > 2 {
							lessHtAcks++
						}

						r.IncLessHeartbeatAcks()
						r.WARNING("heartbeat.acks.granted[%v].less.than.quorums[%v].lessHtAcks[%v].maxLessHtAcks[%v]", ackGranted, r.getQuorums(), lessHtAcks, maxLessHtAcks)
						if lessHtAcks >= maxLessHtAcks {
							r.WARNING("degrade.to.follower.lessHtAcks[%v]>=maxLessHtAcks[%v]", lessHtAcks, maxLessHtAcks)
							r.degradeToFollower()
							break
						}
					} else {
						lessHtAcks = 0
					}

					if mgrCnt < r.getQuorums() {
						r.WARNING("MGR.running.ok[%v].less.than.quorums[%v]", mgrCnt, r.getQuorums())
						if r.GetMembers() > 2 {
							lessCmtHtAcks++
						}
						r.IncLessCmtHeartbeatAcks()
						if lessCmtHtAcks >= maxLessCmtHtAcks {
							r.WARNING("degrade.to.follower.lessCmtHtAcks[%v]>=maxLessCmtHtAcks[%v]", lessCmtHtAcks, maxLessCmtHtAcks)
							r.degradeToFollower()
							break
						}
					} else {
						r.DEBUG("MGR.running.ok[%v].is.greater.than.quorums[%v]", mgrCnt, r.getQuorums())
						lessCmtHtAcks = 0

						// set mysql to read/write
						if r.isReadOnly == true {
							r.WARNING("mysql.SetReadWrite.prepare")
							if err := r.mysql.SetReadWrite(); err == nil {
								r.isReadOnly = false
								r.WARNING("mysql.SetReadWrite.done")
							} else {
								r.ERROR("mysql.SetReadWrite.error[%v]", err)
							}
						}

						if r.isVipSet == false {
							r.WARNING("start.vip.prepare")
							if err := r.leaderStartShellCommand(); err == nil {
								r.WARNING("start.vip.done")
							} else {
								r.ERROR("leader.StartShellCommand.error[%v]", err)
							}
							// If the command fails, it will most likely continue to fail, therefore, run it only once.
							r.isVipSet = true
						}

						// for brain split
						if ackGranted == r.getMembers() {
							if incViewID {
								r.WARNING("heartbeat.acks.granted[%v].equals.members[%v].again", ackGranted, r.getMembers())
								incViewID = false
							}
						} else if !incViewID {
							r.WARNING("heartbeat.acks.granted[%v].less.than.members[%v].for.the.first.time", ackGranted, r.getMembers())
							r.updateView(r.getViewID()+2, r.GetLeader())
							incViewID = true
						}

						// If there is slave node change master to old leader, it can be detected here.
						if ok, _ := r.isMGRClusterStatusOK(); !ok {
							r.WARNING("MGR.running.not.ok.check.again")
							checkMGR = checkMGR + 1
							if checkMGR >= maxCheckMGR {
								r.WARNING("degrade.to.follower.checkMGR[%v]>=maxCheckMGR[%v]", checkMGR, maxCheckMGR)
								r.degradeToFollower()
								break
							}
						} else {
							checkMGR = 0
						}
					}
				} else {
					r.WARNING("leader.is.changing.to.master...")
				}

				mgrCnt = 1
			}

			ackGranted = 1
			respChan = make(chan *model.RaftRPCResponse, r.getAllMembers())
			r.sendHeartbeatHandler(&mysqlDown, respChan)
			r.resetHeartbeatTimeout()
		case rsp := <-respChan:
			r.processHeartbeatResponseHandler(&ackGranted, &mgrCnt, rsp)
		case e := <-r.c:
			switch e.Type {
			// 1) Heartbeat
			case MsgRaftHeartbeat:
				req := e.request.(*model.RaftRPCRequest)
				rsp := r.processHeartbeatRequestHandler(req)
				e.response <- rsp
			// 2) RequestVote
			case MsgRaftRequestVote:
				req := e.request.(*model.RaftRPCRequest)
				rsp := r.processRequestVoteRequestHandler(req)
				e.response <- rsp
			// 3) Ping
			case MsgRaftPing:
				req := e.request.(*model.RaftRPCRequest)
				rsp := r.processPingRequestHandler(req)
				e.response <- rsp
			default:
				r.ERROR("get.unknown.request[%+v]", e.Type)
			}
		}
	}
}

// leaderProcessHeartbeatRequestHandler
// EFFECT
// handles the heartbeat request from the leader
//
// MYSQL
// nop
//
// RETURN
// 1. ErrorInvalidRequest: the request.From is not a member of this cluster
// 2. ErrorInvalidViewID: request leader viewid is old, he is a stale leader
// 3. OK: new leader eggs, we have to downgrade to FOLLOWER
func (r *Leader) processHeartbeatRequest(req *model.RaftRPCRequest) *model.RaftRPCResponse {
	rsp := model.NewRaftRPCResponse(model.OK)
	rsp.Raft.From = r.getID()
	rsp.Raft.ViewID = r.getViewID()
	rsp.Raft.EpochID = r.getEpochID()
	rsp.Raft.State = r.state.String()

	r.IncLeaderGetHeartbeatRequests()
	if !r.checkRequest(req) {
		rsp.RetCode = model.ErrorInvalidRequest
		return rsp
	}
	r.WARNING("get.heartbeat.from[%+v]", *req)
	vidiff := (int)(r.getViewID() - req.GetViewID())
	switch {
	case vidiff > 0:
		r.ERROR("get.heartbeat.from[N:%v, V:%v, E:%v].return.reject", req.GetFrom(), req.GetViewID(), req.GetEpochID())

		rsp.Raft.Leader = r.getLeader()
		rsp.RetCode = model.ErrorInvalidViewID

		// this case happens when two nodes all win in the same viewid
		// in the same viewid because we wait the 'reject' VOTE in random time
	case vidiff == 0:
		r.ERROR("get.heartbeat.from[N:%v, V:%v, E:%v].in.same.viewid", req.GetFrom(), req.GetViewID(), req.GetEpochID())

		// degrade to FOLLOWER
		r.degradeToFollower()

	// new leader eggs
	case vidiff < 0:
		r.WARNING("get.heartbeat.from[N:%v, V:%v, E:%v].down.follower", req.GetFrom(), req.GetViewID(), req.GetEpochID())

		// degrade to FOLLOWER
		r.degradeToFollower()
	}
	return rsp
}

// leaderProcessRequestVoteRequestHandler
// EFFECT
// process the requestvote request from other peer of this cluster
// in this case, some FOLLOWER can't get the leader's heartbeat
//
// MYSQL
// nop
//
// RETURNS
// 1. ErrorInvalidRequest: the request.From is not a member of this cluster
// 2. ErrorInvalidViewID: request viewid is old
// 3. ErrorInvalidGTID: the CANDIDATE has the smaller Read_Master_Log_Pos
// 4. OK: give a vote
func (r *Leader) processRequestVoteRequest(req *model.RaftRPCRequest) *model.RaftRPCResponse {
	rsp := model.NewRaftRPCResponse(model.OK)
	rsp.Raft.From = r.getID()
	rsp.Raft.ViewID = r.getViewID()
	rsp.Raft.EpochID = r.getEpochID()
	rsp.Raft.State = r.state.String()

	r.IncLeaderGetVoteRequests()
	if !r.checkRequest(req) {
		rsp.RetCode = model.ErrorInvalidRequest
		return rsp
	}

	r.WARNING("get.voterequest.from[%+v]", *req)

	// 1. check viewid
	//    request viewid is from an old view or equal with me, reject
	{
		if req.GetViewID() <= r.getViewID() {
			r.WARNING("get.requestvote.from[N:%v, V:%v, E:%v].stale.viewid", req.GetFrom(), req.GetViewID(), req.GetEpochID())
			rsp.RetCode = model.ErrorInvalidViewID
			return rsp
		}
	}

	// 2. check master GTID
	{
		greater, thisGTID, err := r.mysql.GTIDGreaterThan(&req.GTID)
		if err != nil {
			r.ERROR("process.requestvote.get.gtid.error[%v].ret.ErrorMySQLDown", err)
			rsp.RetCode = model.ErrorMySQLDown
			return rsp
		}
		rsp.GTID = thisGTID

		// if leader get a VoteRequest, the most likely reason MySQL doesn't work
		// 'greater' means that master binlog more than you
		if greater {
			// reject cases:
			// 1. I am promotable: I am alive and GTID greater than you
			if r.Promotable() {
				r.WARNING("get.requestvote.from[%v].stale.GTID[%+v]", req.GetFrom(), req.GetGTID())
				rsp.RetCode = model.ErrorInvalidGTID
				return rsp
			}
		}
	}

	// 3. update viewid, if Candidate viewid equal with Leader viewid don't update viewid
	{
		if req.GetViewID() > r.getViewID() {
			r.WARNING("get.requestvote.from[N:%v, V:%v, E:%v].degrade.to.follower", req.GetFrom(), req.GetViewID(), req.GetEpochID())
			r.updateView(req.GetViewID(), noLeader)
			// downgrade to FOLLOWER
			r.degradeToFollower()
		}
	}

	// 4. voted for this candidate
	r.votedFor = req.GetFrom()
	return rsp
}

// leaderSendHeartbeatHandler
// broadcast hearbeat requests to other peers of the cluster
func (r *Leader) sendHeartbeat(mysqlDown *bool, c chan *model.RaftRPCResponse) {
	// check MySQL down
	if r.mysql.GetState() == model.MysqlDead {
		if r.mysql.ShouldDeferFailover() {
			r.WARNING("mysql.dead.too_many_connections.defer.failover")
			return
		}
		*mysqlDown = true
		return
	}

	// broadcast heartbeat
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	allPeers := r.peers
	for k, peer := range r.idlePeers {
		allPeers[k] = peer
	}

	for _, peer := range allPeers {
		r.wg.Add(1)
		go func(peer *Peer) {
			defer r.wg.Done()
			peer.sendHeartbeat(c)
		}(peer)
	}
}

// leaderProcessHeartbeatResponseHandler
// process the send heartbeat response comes from other peers of the cluster
func (r *Leader) processHeartbeatResponse(ackGranted *int, mgrCnt *int, rsp *model.RaftRPCResponse) {
	if rsp.RetCode != model.OK {
		r.ERROR("send.heartbeat.get.rsp[N:%v, V:%v, E:%v].error[%v]", rsp.GetFrom(), rsp.GetViewID(), rsp.GetEpochID(), rsp.RetCode)

		if rsp.RetCode == model.ErrorInvalidViewID {
			r.WARNING("send.heartbeat.get.rsp[N:%v, V:%v, E:%v].error[%v].degrade.to.follower", rsp.GetFrom(), rsp.GetViewID(), rsp.GetEpochID(), rsp.RetCode)
			// downgrade to FOLLOWER
			r.degradeToFollower()
		}
	} else {
		r.DEBUG("send.heartbeat.get.rsp[N:%v/%v, V:%v, E:%v, CmtState:%v]", rsp.GetFrom(), rsp.Raft.State, rsp.GetViewID(), rsp.GetEpochID(), rsp.Raft.CmtState)
		if rsp.Raft.State != IDLE.String() {
			*ackGranted++
		}
		// find the smallest binlog
		if r.relayMasterLogFile == "" {
			r.relayMasterLogFile = rsp.Relay_Master_Log_File
		} else if strings.Compare(r.relayMasterLogFile, rsp.Relay_Master_Log_File) > 0 {
			r.relayMasterLogFile = rsp.Relay_Master_Log_File
		}

		// to reset nextPuregeBinlog:
		// we must get all responses from the follower(s) and idle(s)
		// imagine that:
		// Master is doing backup for Slave2 restore
		// Master purged to Slave1-Relay_Master_Log_File
		// Slave2 starts up and can't find the binlog which she is want
		if *ackGranted == r.getMembers() {
			r.nextPuregeBinlog = r.relayMasterLogFile
		}

		if r.mysqlReplMode == model.ReplModeMGR && rsp.Raft.CmtState == CmtOK.String() {
			*mgrCnt++
		}
	}
}

func (r *Leader) processPingRequest(req *model.RaftRPCRequest) *model.RaftRPCResponse {
	rsp := model.NewRaftRPCResponse(model.OK)
	rsp.Raft.From = r.getID()
	rsp.Raft.ViewID = r.getViewID()
	rsp.Raft.EpochID = r.getEpochID()
	rsp.Raft.State = r.state.String()
	return rsp
}

func (r *Leader) degradeToFollower() {
	r.WARNING("degrade.to.follower.stop.the.vip...")
	if err := r.leaderStopShellCommand(); err != nil {
		r.ERROR("degrade.to.follower.stop.the.vip.error[%v]", err)
	} else {
		r.WARNING("degrade.to.follower.stop.the.vip.done")
	}

	r.WARNING("degrade.to.follower.SetReadOnly...")
	if err := r.mysql.SetReadOnly(); err != nil {
		r.ERROR("degrade.to.follower.SetReadOnly.error[%v]", err)
	} else {
		r.WARNING("degrade.to.follower.SetReadOnly.done")
	}

	if r.mysqlReplMode == model.ReplModeMGR {
		r.WARNING("degrade.to.follower.stop.MGR...")
		if err := r.mysql.StopMGR(); err != nil {
			r.ERROR("degrade.to.follower.stop.MGR.error[%v]", err)
		} else {
			r.WARNING("degrade.to.follower.stop.MGR.done")
		}
	}
	r.purgeBinlogStop()
	if r.mysqlReplMode == model.ReplModeSemiSync {
		r.checkSemiSyncStop()
	}
	r.checkGTIDStop()
	r.IncLeaderDegrades()
	r.setState(FOLLOWER)
	r.isDegradeToFollower = true
}

func (r *Leader) checkChangeToMaster() bool {
	if ok, _ := r.isMGRClusterStatusOK(); !ok {
		return false
	}
	status, err := r.mysql.GetLocalMGRStat()
	if err == nil && status.Role == model.MGRRolePrimary {
		return true
	}
	return false
}

// prepareSettingsMGR
func (r *Leader) prepareSettingsMGR() {
	r.WARNING("MGR.setting.prepare....")

	r.cmtState = CmtNone
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		gtid, err := r.mysql.GetGTID()
		if err != nil {
			r.ERROR("mysql.get.gtid.error[%v]", err)
		} else {
			r.WARNING("my.gtid.is:%v", gtid)
		}

		// MySQL1. wait relay log replay done
		r.WARNING("1. mysql.WaitApplyRelayLog.prepare")
		r.SetRaftMysqlStatus(model.RAFTMYSQL_WAITAPPLYRELAYLOG)
		if err := r.mysql.WaitApplyRelayLog(15, 2); err != nil {
			r.ERROR("mysql.WaitApplyRelayLog.error[%v]", err)
			r.degradeToFollower()
			return
		}
		r.ResetRaftMysqlStatus()
		r.WARNING("mysql.WaitApplyRelayLog.done")

		// MySQL2. set mysql master system variables
		r.WARNING("2.mysql.SetSysVars.prepare")
		r.mysql.SetMasterGlobalSysVar()
		r.WARNING("mysql.SetSysVars.done")

		// MySQL3. start group_replication as master
		r.WARNING("3. leader.change.to.master.prepare")
		if r.checkChangeToMaster() {
			r.cmtState = CmtOK
			r.mgrClusterEverOK = true
			r.WARNING("MGR.cluster.has.met.expectations.skip.change.to.master")
		} else {
			repl := r.mysql.GetRepl()
			if err := r.mysql.ChangeToMaster(&repl); err != nil {
				r.cmtState = CmtError
				r.ERROR("leader.change.to.master.failed")
				r.degradeToFollower()
				return
			}
			r.cmtState = CmtOK
			r.mgrClusterEverOK = true
			r.WARNING("leader.change.to.master.done")
		}

		// MySQL4. set mysql to read-only
		r.WARNING("4. mysql.SetReadOnly.prepare")
		if err := r.mysql.SetReadOnly(); err != nil {
			// WTF, what can we do?
			r.ERROR("mysql.SetReadOnly.error[%v]", err)
		} else {
			r.isReadOnly = true
		}
		r.WARNING("mysql.SetReadOnly.done")
		r.WARNING("MGR.setting.all.done....")
	}()
}

// prepareSettingsAsync
// wait mysql WAIT_UNTIL_SQL_THREAD_AFTER_GTIDS done and other mysql settings
// since leader must periodically send heartbeat to followers, so setMysqlAsync is asynchronous
// WAIT_UNTIL_SQL_THREAD_AFTER_GTIDS maybe a long operation here
func (r *Leader) prepareSettingsAsync() {
	r.WARNING("async.setting.prepare....")

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		gtid, err := r.mysql.GetGTID()
		if err != nil {
			r.ERROR("mysql.get.gtid.error[%v]", err)
		} else {
			r.WARNING("my.gtid.is:%v", gtid)
		}

		// MySQL1. wait relay log replay done
		r.WARNING("1. mysql.WaitUntilAfterGTID.prepare")
		r.SetRaftMysqlStatus(model.RAFTMYSQL_WAITUNTILAFTERGTID)
		if err := r.mysql.WaitUntilAfterGTID(gtid.Retrieved_GTID_Set); err != nil {
			r.ERROR("mysql.WaitUntilAfterGTID.error[%v]", err)
			// TODO: Why not use r.degradeToFollower()
			r.setState(FOLLOWER)
			r.isDegradeToFollower = true
			return
		}
		r.ResetRaftMysqlStatus()
		r.WARNING("mysql.WaitUntilAfterGTID.done")

		// MySQL2. change to master
		r.WARNING("2. mysql.ChangeToMaster.prepare")
		repl := r.mysql.GetRepl()
		if err := r.mysql.ChangeToMaster(&repl); err != nil {
			r.ERROR("mysql.ChangeToMaster.error[%v]", err)
			// TODO: Why not use r.degradeToFollower()
			r.setState(FOLLOWER)
			r.isDegradeToFollower = true
			return
		}
		r.WARNING("mysql.ChangeToMaster.done")

		// MySQL3. enable semi-sync on master
		// wait slave ack
		r.WARNING("3. mysql.EnableSemiSyncMaster.prepare")
		if err := r.mysql.EnableSemiSyncMaster(); err != nil {
			// WTF, what can we do?
			r.ERROR("mysql.EnableSemiSyncMaster.error[%v]", err)
		}
		r.WARNING("mysql.EnableSemiSyncMaster.done")

		// MySQL4. set mysql master system variables
		r.WARNING("4.mysql.SetSysVars.prepare")
		r.mysql.SetMasterGlobalSysVar()
		r.WARNING("mysql.SetSysVars.done")

		// MySQL5. set mysql to read/write
		r.WARNING("5. mysql.SetReadWrite.prepare")
		if err := r.mysql.SetReadWrite(); err != nil {
			// WTF, what can we do?
			r.ERROR("mysql.SetReadWrite.error[%v]", err)
		}
		r.WARNING("mysql.SetReadWrite.done")
		r.WARNING("6. start.vip.prepare")
		if r.initRole == LEADER {
			// The -r LEADER is specified at startup server, ndicates that the current node
			// has previously executed leaderStartShellCommand，skip this one.
			r.initRole = UNKNOWN
			r.WARNING("the.init.role.is.leader.skip")
		} else if err := r.leaderStartShellCommand(); err != nil {
			// TODO(array): what todo?
			r.ERROR("leader.StartShellCommand.error[%v]", err)
		}
		r.WARNING("start.vip.done")
		r.WARNING("async.setting.all.done....")
	}()
}

func (r *Leader) purgeBinlogStart() {
	r.purgeBinlogStop()
	r.purgeStop = make(chan struct{})
	stop := r.purgeStop
	tick := common.NormalTicker(r.conf.PurgeBinlogInterval)
	r.purgeBinlogTick = tick
	go func(leader *Leader) {
		for {
			select {
			case <-stop:
				return
			case _, ok := <-tick.C:
				if !ok {
					return
				}
				leader.purgeBinlog()
			}
		}
	}(r)
	r.INFO("purge.binlog.start[%vms]...", r.conf.PurgeBinlogInterval)
}

func (r *Leader) purgeBinlogStop() {
	r.relayMasterLogFile = ""
	r.nextPuregeBinlog = ""
	if r.purgeStop != nil {
		close(r.purgeStop)
		r.purgeStop = nil
	}
	if r.purgeBinlogTick != nil {
		r.purgeBinlogTick.Stop()
		r.purgeBinlogTick = nil
	}
}

func (r *Leader) purgeBinlog() {
	if r.skipPurgeBinlog {
		r.WARNING("purge.binlog.skipped[skipPurgeBinlog is true]")
		return
	}

	if r.conf.PurgeBinlogDisabled {
		r.WARNING("purge.binlog.skipped[conf.PurgeBinlogDisabled is true]")
		return
	}

	if r.nextPuregeBinlog != "" {
		if err := r.mysql.PurgeBinlogsTo(r.nextPuregeBinlog); err != nil {
			r.ERROR("purge.binlogs.to[%v].error[%v]", r.nextPuregeBinlog, err)
			r.IncLeaderPurgeBinlogFails()
		} else {
			r.WARNING("purged.binlogs.to[%v]...", r.nextPuregeBinlog)
			r.relayMasterLogFile = ""
			r.nextPuregeBinlog = ""
			r.IncLeaderPurgeBinlogs()
		}
	}
}

func (r *Leader) checkSemiSyncStart() {
	r.checkSemiSyncStop()
	interval := r.getElectionTimeout() / 2
	r.semiSyncStop = make(chan struct{})
	stop := r.semiSyncStop
	tick := common.NormalTicker(interval)
	r.checkSemiSyncTick = tick
	go func(leader *Leader) {
		for {
			select {
			case <-stop:
				return
			case _, ok := <-tick.C:
				if !ok {
					return
				}
				leader.checkSemiSync()
			}
		}
	}(r)
	r.INFO("check.semi-sync.thread.start[%vms]...", interval)
}

func (r *Leader) checkSemiSyncStop() {
	if r.semiSyncStop != nil {
		close(r.semiSyncStop)
		r.semiSyncStop = nil
	}
	if r.checkSemiSyncTick != nil {
		r.checkSemiSyncTick.Stop()
		r.checkSemiSyncTick = nil
	}
	r.INFO("check.semi-sync.thread.stop...")
}

// Disable the semi-sync if the nodes number less than 3.
func (r *Leader) checkSemiSync() {
	if r.skipCheckSemiSync {
		r.WARNING("check.semi-sync.skipped[skipCheckSemiSync is true]")
		return
	}

	min := 3
	cur := r.getMembers()
	if cur < min {
		if err := r.mysql.SetSemiSyncMasterTimeout(r.semiSyncTimeoutFor2Nodes); err != nil {
			r.ERROR("mysql.set.semi-sync.master.timeout.to.default.error[%v]", err)
		}
	} else {
		if err := r.mysql.EnableSemiSyncMaster(); err != nil {
			r.ERROR("mysql.enable.semi-sync.error[%v]", err)
		}
		if err := r.mysql.SetSemiWaitSlaveCount((cur - 1) / 2); err != nil {
			r.ERROR("mysql.set.semi.wait.slave.count.error[%v]", err)
		}
		if err := r.mysql.SetSemiSyncMasterTimeout(semisyncTimeout); err != nil {
			r.ERROR("mysql.set.semi.sync.master.timeout.to.infinite.error[%v]", err)
		}
	}
}

func (r *Leader) checkGTIDStart() {
	r.checkGTIDStop()
	interval := r.getElectionTimeout() / 2
	r.gtidCheckStop = make(chan struct{})
	stop := r.gtidCheckStop
	tick := common.NormalTicker(interval)
	r.checkGTIDTick = tick
	go func(leader *Leader) {
		for {
			select {
			case <-stop:
				return
			case _, ok := <-tick.C:
				if !ok {
					return
				}
				leader.checkGTID()
			}
		}
	}(r)
	r.INFO("check.gtid.thread.start[%vms]...", interval)
}

func (r *Leader) checkGTIDStop() {
	if r.gtidCheckStop != nil {
		close(r.gtidCheckStop)
		r.gtidCheckStop = nil
	}
	if r.checkGTIDTick != nil {
		r.checkGTIDTick.Stop()
		r.checkGTIDTick = nil
	}
	r.INFO("check.gtid.thread.stop...")
}

func (r *Leader) checkGTID() {
	if gtid, err := r.mysql.GetGTID(); err != nil {
		r.ERROR("mysql.get.gtid.error[%v]", err)
	} else {
		r.gtid = gtid
		r.DEBUG("mysql.get.gtid[%v]", r.gtid)
	}
}

func (r *Leader) stateInit() {
	r.WARNING("state.init")
	r.isReadOnly = false
	r.updateStateBegin()
	r.purgeBinlogStart()
	r.checkGTIDStart()
	if r.mysqlReplMode == model.ReplModeSemiSync {
		r.checkSemiSyncStart()
		r.prepareSettingsAsync()
	} else {
		r.prepareSettingsMGR()
	}
	r.isVipSet = false
	r.isDegradeToFollower = false

	r.WARNING("state.machine.run")
}

func (r *Leader) stateExit() {
	if !r.isDegradeToFollower {
		r.WARNING("state.machine.exit.stop.the.vip...")
		if err := r.leaderStopShellCommand(); err != nil {
			r.ERROR("state.machine.exit.stop.the.vip.error[%v]", err)
		} else {
			r.WARNING("state.machine.exit.stop.the.vip.done")
		}

		r.WARNING("state.machine.exit.SetReadOnly...")
		if err := r.mysql.SetReadOnly(); err != nil {
			r.ERROR("state.machine.exit.SetReadOnly.error[%v]", err)
		} else {
			r.WARNING("state.machine.exit.SetReadOnly.done")
		}

		r.purgeBinlogStop()
		if r.mysqlReplMode == model.ReplModeSemiSync {
			r.checkSemiSyncStop()
		}
		r.checkGTIDStop()
	}
	// Wait for the LEADER state-machine async work done.
	r.wg.Wait()
	r.WARNING("leader.state.machine.exit.done")
}

// leader handlers
func (r *Leader) initHandlers() {
	// heartbeat request
	r.setProcessHeartbeatRequestHandler(r.processHeartbeatRequest)

	// vote request
	r.setProcessRequestVoteRequestHandler(r.processRequestVoteRequest)

	// send heartbeat
	r.setSendHeartbeatHandler(r.sendHeartbeat)
	r.setProcessHeartbeatResponseHandler(r.processHeartbeatResponse)

	// ping request
	r.setProcessPingRequestHandler(r.processPingRequest)
}

// for tests
func (r *Leader) setProcessHeartbeatRequestHandler(f func(*model.RaftRPCRequest) *model.RaftRPCResponse) {
	r.processHeartbeatRequestHandler = f
}

func (r *Leader) setProcessRequestVoteRequestHandler(f func(*model.RaftRPCRequest) *model.RaftRPCResponse) {
	r.processRequestVoteRequestHandler = f
}

func (r *Leader) setSendHeartbeatHandler(f func(*bool, chan *model.RaftRPCResponse)) {
	r.sendHeartbeatHandler = f
}

func (r *Leader) setProcessHeartbeatResponseHandler(f func(*int, *int, *model.RaftRPCResponse)) {
	r.processHeartbeatResponseHandler = f
}

func (r *Leader) setProcessPingRequestHandler(f func(*model.RaftRPCRequest) *model.RaftRPCResponse) {
	r.processPingRequestHandler = f
}
