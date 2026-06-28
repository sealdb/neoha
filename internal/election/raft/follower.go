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

	"github.com/sealdb/neoha/internal/base/model"
)

// Follower tuple.
type Follower struct {
	*Raft
	// Whether the 'change to master' is success or not when the new leader eggs.
	ChangeToMasterError bool

	// Used to wait for the async job done.
	wg sync.WaitGroup

	// follower process heartbeat request handler
	processHeartbeatRequestHandler func(*model.RaftRPCRequest) *model.RaftRPCResponse

	// follower process voterequest request handler
	processRequestVoteRequestHandler func(*model.RaftRPCRequest) *model.RaftRPCResponse

	// follower process raft ping request handler
	processPingRequestHandler func(*model.RaftRPCRequest) *model.RaftRPCResponse
}

// NewFollower creates new Follower.
func NewFollower(r *Raft) *Follower {
	F := &Follower{Raft: r}
	F.initHandlers()

	return F
}

// Loop used to start the loop of the state machine.
// --------------------------------------
// State Machine
// --------------------------------------
//
//	timeout and ping ack greater or equal to n/2+1
//
// State1. FOLLOWER ------------------> CANDIDATE
func (r *Follower) Loop() {
	r.stateInit()
	defer r.stateExit()

	r.resetElectionTimeout()
	for r.getState() == FOLLOWER {
		select {
		case <-r.fired:
			r.WARNING("state.machine.loop.got.fired")
		case <-r.electionTick.C:
			r.WARNING("timeout.to.do.new.election")
			if r.replicationPromotable() {
				mgrQuorumLost := false
				if r.mysqlReplMode == model.ReplModeMGR && r.mgrClusterEverOK {
					if ok, _ := r.isMGRClusterStatusOK(); !ok {
						mgrQuorumLost = true
					}
				}
				if mgrQuorumLost {
					r.WARNING("mgr.quorum.lost.promote.to.candidate")
					r.upgradeToCandidate()
				} else if !r.isBrainSplit {
					r.WARNING("timeout.and.ping.almost.node.successed.promote.to.candidate")
					r.upgradeToCandidate()
				}
			}

			// reset timeout
			r.resetElectionTimeout()
		case e := <-r.c:
			switch e.Type {
			case MsgRaftHeartbeat:
				req := e.request.(*model.RaftRPCRequest)
				rsp := r.processHeartbeatRequestHandler(req)
				e.response <- rsp

				if rsp.RetCode != model.OK {
					r.WARNING("process.heartbeat.request.RetCode.not.OK:%+v", rsp.RetCode)
				}
				// reset timeout
				r.resetElectionTimeout()
			case MsgRaftRequestVote:
				req := e.request.(*model.RaftRPCRequest)
				rsp := r.processRequestVoteRequestHandler(req)
				e.response <- rsp

				// reset timeout
				if rsp.RetCode == model.OK {
					r.resetElectionTimeout()
				}
			case MsgRaftPing:
				req := e.request.(*model.RaftRPCRequest)
				rsp := r.processPingRequestHandler(req)
				e.response <- rsp
			default:
				r.ERROR("get.unknown.request[%v]", e.Type)
			}
		}
	}
}

// followerProcessHeartbeatRequest
// EFFECT
// handles the heartbeat request from the leader
//
// MYSQL
// we should check mysql slave_io_thread is stopped(by requestvote) or not
// if stopped we start it
//
// RETURN
// 1. ErrorInvalidRequest: the request.From is not a member of this cluster
// 2. ErrorInvalidViewID: request leader viewid is old, he is a stale leader
// 3. OK: new leader eggs, we downgrade to FOLLOWER and do mysql change master
func (r *Follower) processHeartbeatRequest(req *model.RaftRPCRequest) *model.RaftRPCResponse {
	rsp := model.NewRaftRPCResponse(model.OK)
	rsp.Raft.From = r.getID()
	rsp.Raft.ViewID = r.getViewID()
	rsp.Raft.EpochID = r.getEpochID()
	rsp.Raft.State = r.state.String()
	rsp.Relay_Master_Log_File = r.mysql.RelayMasterLogFile()

	r.DEBUG("get.heartbeat.from[N:%v, V:%v, E:%v]...", req.GetFrom(), req.GetViewID(), req.GetEpochID())
	if !r.checkRequest(req) {
		rsp.RetCode = model.ErrorInvalidRequest
		return rsp
	}

	viewdiff := (int)(r.getViewID() - req.GetViewID())
	epochdiff := (int)(r.getEpochID() - req.GetEpochID())
	switch {
	case viewdiff > 0:
		r.ERROR("get.heartbeat.from[N:%v, V:%v, E:%v].stale.viewid.ret.ErrorInvalidViewID", req.GetFrom(), req.GetViewID(), req.GetEpochID())
		rsp.Raft.Leader = r.getLeader()
		rsp.RetCode = model.ErrorInvalidViewID

	case viewdiff <= 0:
		r.RecordLeaderDatabase(&req.Repl)

		if !r.DelegateDBApply() {
			if err := r.demoteReadOnly(); err != nil {
				r.ERROR("mysql.SetReadOnly.error[%v]", err)
			}
		}

		applyEpochView := func() {
			if viewdiff < 0 {
				r.WARNING("get.heartbeat.from[N:%v, V:%v, E:%v].update.view", req.GetFrom(), req.GetViewID(), req.GetEpochID())
				r.updateView(req.GetViewID(), req.GetFrom())
			}
			if epochdiff != 0 {
				r.WARNING("get.heartbeat.from[N:%v, V:%v, E:%v].update.epoch", req.GetFrom(), req.GetViewID(), req.GetEpochID())
				r.updateEpoch(req.GetEpochID(), req.GetPeers(), req.GetIdlePeers())
			}
		}

		if r.mysqlReplMode == model.ReplModeMGR {
			rsp.Raft.CmtState = r.cmtState.String()
			if r.getLeader() != req.GetFrom() {
				if gtid, err := r.mysql.GetGTID(); err == nil {
					r.WARNING("get.heartbeat.my.gtid.is:%v", gtid)
				}
				r.WARNING("get.heartbeat.from[N:%v, V:%v, E:%v, CmtState:%v].ready.to.join.mgr", req.GetFrom(), req.GetViewID(), req.GetEpochID(), req.Raft.CmtState)
				if req.Raft.CmtState != CmtOK.String() {
					r.WARNING("the.leader.not.finished.change.to.master.waiting...")
					applyEpochView()
					return rsp
				}

				if r.cmtState == CmtError || r.cmtState == CmtOK {
					r.cmtState = CmtNone
				}

				switch r.cmtState {
				case CmtNone:
					if r.DelegateDBApply() {
						if ok, err := r.mysql.IsMGRRunningOK(); err == nil && ok {
							r.cmtState = CmtOK
							r.leader = req.GetFrom()
							r.mgrClusterEverOK = true
							r.WARNING("delegated.mgr.join.ok[reconciler]")
						} else {
							r.WARNING("delegated.mgr.join.pending[reconciler]")
						}
						rsp.Raft.CmtState = r.cmtState.String()
						applyEpochView()
						return rsp
					}
					lastReq := *req
					go func() {
						r.cmtState = CmtChanging
						if err := r.mysql.MGRChangeMasterTo(&lastReq.Repl); err != nil {
							r.ERROR("change.master.to[FROM:%v, GTID:%v].error[%v]", lastReq.GetFrom(), lastReq.GetRepl(), err)
							r.cmtState = CmtError
						} else {
							r.WARNING("get.heartbeat.join.mgr.from[%v].succeed", lastReq.GetFrom())
							r.mgrClusterEverOK = true
							r.leader = lastReq.GetFrom()
							r.cmtState = CmtOK
						}
					}()
					rsp.Raft.CmtState = r.cmtState.String()
					applyEpochView()
					return rsp

				case CmtChanging:
					r.WARNING("get.heartbeat.is.joining.mgr...")
					applyEpochView()
					return rsp
				}
			}

			if r.cmtState != CmtChanging {
				if r.DelegateDBApply() {
					if ok, err := r.mysql.IsMGRRunningOK(); err == nil && ok {
						if r.cmtState != CmtOK {
							r.cmtState = CmtOK
							r.mgrClusterEverOK = true
						}
					}
				}
				if ok, err := r.mysql.IsMGRRunningOK(); err == nil && !ok {
					r.ERROR("mysql.local.MGR.is.not.running.ok[%v].error[%v]", ok, err)
					r.leader = ""
				}
			}
		} else if r.DelegateDBApply() {
			if r.getLeader() != req.GetFrom() {
				r.leader = req.GetFrom()
				r.WARNING("delegated.semisync.leader[%v].reconciler.owns.replica.apply", req.GetFrom())
			}
		} else {
			if err := r.mysql.DisableSemiSyncMaster(); err != nil {
				r.ERROR("mysql.DisableSemiSyncMaster.error[%v]", err)
			}

			if err := r.mysql.StartSlave(); err != nil {
				r.ERROR("mysql.StartSlave.error[%v]", err)
			}

			if r.getLeader() != req.GetFrom() {
				gtid, err := r.mysql.GetGTID()
				if err == nil {
					r.WARNING("get.heartbeat.my.gtid.is:%v", gtid)
				}

				if r.getMembers() > 2 {
					r.degradeToInvalid(&gtid, &req.GTID)
				}

				r.WARNING("get.heartbeat.from[N:%v, V:%v, E:%v].change.mysql.master", req.GetFrom(), req.GetViewID(), req.GetEpochID())
				req.Repl.Repl_GTID_Purged = r.Raft.mysql.GetReplGtidPurged()
				if err := r.mysql.ChangeMasterTo(&req.Repl); err != nil {
					r.ERROR("change.master.to[FROM:%v, GTID:%v].error[%v]", req.GetFrom(), req.GetRepl(), err)
					r.ChangeToMasterError = true
					rsp.RetCode = model.ErrorChangeMaster
					return rsp
				}

				r.ChangeToMasterError = false
				r.leader = req.GetFrom()
				r.WARNING("get.heartbeat.change.to.the.new.master[%v].successed", req.GetFrom())
			}
		}

		applyEpochView()
	}
	return rsp
}

// followerProcessRequestVoteRequest
// EFFECT
// handles the requestvote request from other CANDIDATEs
//
// MYSQL
// stop mysql slave_io_thread to get a GTID coordinate of this view
//
// RETURN
// 1. ErrorInvalidRequest: the request.From is not a member of this cluster
// 2. ErrorInvalidViewID: request viewid is old
// 3. ErrorInvalidGTID: the CANDIDATE has the smaller Read_Master_Log_Pos
// 4. OK: give a vote
func (r *Follower) processRequestVoteRequest(req *model.RaftRPCRequest) *model.RaftRPCResponse {
	rsp := model.NewRaftRPCResponse(model.OK)
	rsp.Raft.From = r.getID()
	rsp.Raft.ViewID = r.getViewID()
	rsp.Raft.EpochID = r.getEpochID()
	rsp.Raft.State = r.state.String()

	if !r.checkRequest(req) {
		rsp.RetCode = model.ErrorInvalidRequest
		return rsp
	}

	r.WARNING("get.voterequest.from[%+v].request[%v]", req.GetFrom(), req.GetGTID())
	// 1. check viewid(req.viewid < thisnode.viewid)
	{
		if req.GetViewID() < r.getViewID() {
			r.WARNING("get.requestvote.from[N:%v, V:%v, E:%v].stale.viewid.ret.reject", req.GetFrom(), req.GetViewID(), req.GetEpochID())
			rsp.RetCode = model.ErrorInvalidViewID
			return rsp
		}
	}

	// 2. check GTID
	{
		if r.mysqlReplMode == model.ReplModeSemiSync {
			// stop io thread; it will re-start again when heartbeat received
			if err := r.mysql.StopSlaveIOThread(); err != nil {
				r.ERROR("mysql.StopSlaveIOThread.error[%+v]", err)
			}
		}

		greater, thisGTID, err := r.mysql.GTIDGreaterThan(&req.GTID)
		if err != nil {
			r.ERROR("process.requestvote.get.gtid.error[%v].ret.ErrorMySQLDown", err)
			rsp.RetCode = model.ErrorMySQLDown
			return rsp
		}
		rsp.GTID = thisGTID

		if greater {
			// reject cases:
			// 1. I am promotable: I am alive and GTID greater than you
			if r.Promotable() {
				r.WARNING("get.requestvote.from[N:%v, V:%v, E:%v].stale.ret.ErrorInvalidGTID", req.GetFrom(), req.GetViewID(), req.GetEpochID())
				rsp.RetCode = model.ErrorInvalidGTID
				return rsp
			}
		}
	}

	// 3. check viewid(req.viewid > thisnode.viewid)
	// if the req.viewid is larger than this node, update the viewid
	// if the req.viewid is equal with this node and we have voted for other one then
	// don't voted for this candidate
	{
		if req.GetViewID() > r.getViewID() {
			r.updateView(req.GetViewID(), noLeader)
		} else {
			if (r.votedFor != noVote) && (r.votedFor != req.GetFrom()) {
				r.WARNING("get.requestvote.from[N:%v, V:%v, E:%v].already.vote.for[%v].ret.reject", req.GetFrom(), req.GetViewID(), req.GetEpochID(), r.votedFor)
				rsp.RetCode = model.ErrorVoteNotGranted
				return rsp
			}
		}
	}

	// 4. check MGR cluster status
	if r.mysqlReplMode == model.ReplModeMGR {
		if ok, uuid := r.isMGRClusterStatusOK(); ok && uuid != "" && uuid != req.GetUUID() {
			r.WARNING("get.requestvote.from[N:%v, V:%v, E:%v, UUID:%v].stale.ret.ErrorMGRAlreadyRunningOK[MGRMasterUUID:%v]", req.GetFrom(), req.GetViewID(), req.GetEpochID(), req.GetUUID(), uuid)
			rsp.RetCode = model.ErrorMGRAlreadyRunningOK
			return rsp
		}
	}

	// 5. voted for this candidate
	r.votedFor = req.GetFrom()
	r.WARNING("get.requestvote.from[N:%v, V:%v, E:%v].vote.for.this.candidate", req.GetFrom(), req.GetViewID(), req.GetEpochID())
	return rsp
}

func (r *Follower) processPingRequest(req *model.RaftRPCRequest) *model.RaftRPCResponse {
	rsp := model.NewRaftRPCResponse(model.OK)
	rsp.Raft.From = r.getID()
	rsp.Raft.ViewID = r.getViewID()
	rsp.Raft.EpochID = r.getEpochID()
	rsp.Raft.State = r.state.String()
	return rsp
}

// startCheckBrainSplit check for split brain
func (r *Follower) startCheckBrainSplit() {
	r.isBrainSplit = true
	r.INFO("start.CheckBrainSplit")

	cnt := 1
	respChan := make(chan *model.RaftRPCResponse, r.getMembers())
	r.resetCheckBrainSplitTimeout()
	go func() {
		for r.getState() == FOLLOWER {
			select {
			case <-r.fired:
				r.WARNING("check.brain.split.loop.got.fired")
			case <-r.checkBrainSplitTick.C:
				r.DEBUG("timeout.to.check.brain.split")

				if r.isBrainSplit {
					r.WARNING("ping.responses[%v].is.less.than.half.maybe.brain.split", cnt)
				}

				cnt = 1
				respChan = make(chan *model.RaftRPCResponse, r.getMembers())
				r.sendClusterPing(respChan)
				r.resetCheckBrainSplitTimeout()

			case rsp := <-respChan:
				if rsp.RetCode == model.OK {
					if rsp.Raft.State == "LEADER" {
						r.DEBUG("receive.ping.responses.from.leader[%v].skip.check.brain.split", rsp.GetFrom())
						continue
					}
					if strings.Contains("FOLLOWER CANDIDATE LEARNER", rsp.Raft.State) {
						cnt++
						r.DEBUG("receive.ping.responses[%v].from[N:%v, R:%v]", cnt, rsp.GetFrom(), rsp.Raft.State)
					}
				}

				if cnt < r.GetQuorums() {
					r.isBrainSplit = true
					continue
				}

				if r.isBrainSplit {
					r.isBrainSplit = false
					r.WARNING("ping.responses[%v].is.greater.than.half.again", cnt)
				}
			}
		}
	}()
}

func (r *Follower) sendClusterPing(respChan chan *model.RaftRPCResponse) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	for _, peer := range r.peers {
		go func(peer *Peer) {
			peer.SendPing(respChan)
		}(peer)
	}
}

func (r *Follower) upgradeToCandidate() {
	// only you
	if len(r.peers) == 0 {
		r.WARNING("peers.is.null.can.not.upgrade.to.candidate")
		return
	}

	if r.ChangeToMasterError {
		r.WARNING("change.to.master.error.can.not.upgrade.to.candidate")
		return
	}

	// stop io thread
	// it will re-start again when heartbeat received
	if err := r.mysql.StopSlaveIOThread(); err != nil {
		r.ERROR("mysql.StopSlaveIOThread.error[%v]", err)
	}
	r.setState(CANDIDATE)
	r.IncCandidatePromotes()
}

func (r *Follower) degradeToInvalid(followerGTID *model.GTID, candidateGTID *model.GTID) {
	// only you
	if len(r.peers) == 0 {
		r.WARNING("peers.is.null.can.not.upgrade.to.candidate")
		return
	}

	// stop io thread
	// it will re-start again when heartbeat received
	if err := r.mysql.StopSlaveIOThread(); err != nil {
		r.ERROR("mysql.StopSlaveIOThread.error[%v]", err)
	}

	// if error can not vote candidate
	greater := r.mysql.CheckGTID(followerGTID, candidateGTID)
	if greater {
		// degrade to INVALID
		r.setState(INVALID)
		return
	}
}

// setMySQLAsync used to setting mysql in async
func (r *Follower) setMySQLAsync() {
	r.WARNING("mysql.waitMysqlDoneAsync.prepare")

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		// MySQL1: set readonly
		if err := r.demoteReadOnly(); err != nil {
			r.ERROR("mysql.SetReadOnly.error[%v]", err)
		}
		r.WARNING("mysql.SetReadOnly.done")

		// MySQL2. set mysql slave system variables
		if err := r.mysql.SetSlaveGlobalSysVar(); err != nil {
			r.ERROR("mysql.SetSlaveGlobalSysVar.error[%v]", err)
		}
		r.WARNING("mysql.SetSlaveGlobalSysVar.done")
		r.WARNING("prepareAsync.done")

		// Log the gtid info.
		if gtid, err := r.mysql.GetGTID(); err != nil {
			r.ERROR("init.get.mysql.gtid.error:%v", err)
		} else {
			r.WARNING("init.my.gtid.is:%v", gtid)
		}
	}()
}

func (r *Follower) stateInit() {
	r.WARNING("state.init")
	r.updateStateBegin()
	if r.DelegateDBApply() {
		if r.mysqlReplMode == model.ReplModeMGR {
			mgrJoined := false
			if stat, err := r.mysql.GetLocalMGRStat(); err == nil && stat != nil {
				mgrJoined = stat.State == model.MGRStateOnline || stat.State == model.MGRStateRecovering
			}
			if !mgrJoined {
				r.WARNING("delegated.follower.stop.MGR.before.reconciler")
				if err := r.mysql.StopMGR(); err != nil {
					r.ERROR("stop.MGR.error[%v]", err)
				}
			}
		}
		r.WARNING("follower.state.init.delegated.skip[mysql setup owned by reconciler]")
		r.WARNING("state.machine.run")
		return
	}
	if r.mysqlReplMode == model.ReplModeMGR {
		mgrJoined := false
		mgrState := ""
		if stat, err := r.mysql.GetLocalMGRStat(); err == nil && stat != nil {
			mgrState = stat.State
			mgrJoined = stat.State == model.MGRStateOnline || stat.State == model.MGRStateRecovering
		}
		if mgrJoined {
			r.WARNING("MGR.member.online.skip.stop.on.follower.init[state:%v]", mgrState)
			r.cmtState = CmtOK
		} else {
			r.WARNING("stop.MGR.begin")
			if err := r.mysql.StopMGR(); err != nil {
				r.ERROR("stop.MGR.error[%v]", err)
			}
			r.WARNING("stop.MGR.end")
			r.cmtState = CmtNone
		}
		if err := r.demoteReadOnly(); err != nil {
			r.ERROR("mysql.SetReadOnly.error[%v]", err)
		}
	} else {
		r.setMySQLAsync()
	}
	r.WARNING("state.machine.run")
}

func (r *Follower) stateExit() {
	// Wait for the FOLLOWER state-machine async work done.
	r.wg.Wait()
	r.WARNING("follower.state.machine.exit")
}

// follower handlers
func (r *Follower) initHandlers() {
	r.setProcessHeartbeatRequestHandler(r.processHeartbeatRequest)
	r.setProcessRequestVoteRequestHandler(r.processRequestVoteRequest)
	r.setProcessPingRequestHandler(r.processPingRequest)
}

// for tests
func (r *Follower) setProcessHeartbeatRequestHandler(f func(*model.RaftRPCRequest) *model.RaftRPCResponse) {
	r.processHeartbeatRequestHandler = f
}

func (r *Follower) setProcessRequestVoteRequestHandler(f func(*model.RaftRPCRequest) *model.RaftRPCResponse) {
	r.processRequestVoteRequestHandler = f
}

func (r *Follower) setProcessPingRequestHandler(f func(*model.RaftRPCRequest) *model.RaftRPCResponse) {
	r.processPingRequestHandler = f
}
