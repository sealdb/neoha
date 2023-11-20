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
	"sync/atomic"

	"github.com/sealdb/neoha/internal/base/model"

	"github.com/pkg/errors"
)

const (
	noVote   = ""
	noLeader = ""
)

// State enum.
type State int

const (
	// FOLLOWER state.
	FOLLOWER State = 1 << iota

	// CANDIDATE state.
	CANDIDATE

	// LEADER state.
	LEADER

	// IDLE state.
	// neither process heartbeat nor voterequest(return ErrorInvalidRequest)
	IDLE

	// INVALID state.
	// neither process heartbeat nor voterequest(return ErrorInvalidRequest)
	INVALID

	// LEARNER state.
	LEARNER

	// STOPPED state.
	STOPPED

	// UNKNOWN state.
	UNKNOWN
)

func (s State) String() string {
	switch s {
	case 1 << 0:
		return "FOLLOWER"
	case 1 << 1:
		return "CANDIDATE"
	case 1 << 2:
		return "LEADER"
	case 1 << 3:
		return "IDLE"
	case 1 << 4:
		return "INVALID"
	case 1 << 5:
		return "LEARNER"
	case 1 << 6:
		return "STOPPED"
	}
	return "UNKNOWN"
}

// CmtState enum.
type CmtState int

const (
	// CmtNone represents initial state.
	CmtNone CmtState = 1 << iota

	// CmtOK represents change master to successfully.
	CmtOK

	// CmtChanging represents it's changing master.
	CmtChanging

	// CmtError represents there was an error changing master to.
	CmtError
)

func (s CmtState) String() string {
	switch s {
	case 1 << 0:
		return "CmtNone"
	case 1 << 1:
		return "CmtOK"
	case 1 << 2:
		return "CmtChanging"
	case 1 << 3:
		return "CmtError"
	}
	return "UNKNOWN"
}

const (
	// MsgNone type.
	MsgNone = iota + 1

	// MsgRaftHeartbeat type.
	MsgRaftHeartbeat

	// MsgRaftRequestVote type.
	MsgRaftRequestVote

	// MsgRaftPing type.
	MsgRaftPing
)

var (
	errStop = errors.New("raft.has.been.stopped")
	errSend = errors.New("raft.send.timeout")
)

// raft attributes
func (r *Raft) getState() State {
	return r.state
}

func (r *Raft) setState(state State) {
	r.setLeader(noLeader)
	r.state = state
}

func (r *Raft) getCmtState() CmtState {
	return r.cmtState
}

func (r *Raft) getID() string {
	return r.id
}

func (r *Raft) getQuorums() int {
	return (len(r.meta.Peers) / 2) + 1
}

// all members include me and exclude idle nodes
func (r *Raft) getMembers() int {
	return len(r.meta.Peers)
}

// all members include me and idle nodes
func (r *Raft) getAllMembers() int {
	return len(r.meta.Peers) + len(r.meta.IdlePeers)
}

func (r *Raft) getPeers() []string {
	return r.meta.Peers
}

func (r *Raft) getIdlePeers() []string {
	return r.meta.IdlePeers
}

func (r *Raft) getAllPeers() []string {
	allPeers := r.meta.Peers
	allPeers = append(allPeers, r.meta.IdlePeers...)
	return allPeers
}

func (r *Raft) getElectionTimeout() int {
	return r.conf.ElectionTimeout
}

func (r *Raft) getHeartbeatTimeout() int {
	return r.conf.HeartbeatTimeout
}

func (r *Raft) incViewID() {
	atomic.AddUint64(&r.meta.ViewID, 1)
}

func (r *Raft) getViewID() uint64 {
	return atomic.LoadUint64(&r.meta.ViewID)
}

func (r *Raft) incEpochID() {
	atomic.AddUint64(&r.meta.EpochID, 1)
}

func (r *Raft) getEpochID() uint64 {
	return atomic.LoadUint64(&r.meta.EpochID)
}

func (r *Raft) getGTID() model.GTID {
	return r.gtid
}

func (r *Raft) getUUID() (string, error) {
	return r.mysql.GetUUID()
}

func (r *Raft) getLeader() string {
	return r.leader
}

func (r *Raft) setLeader(leader string) {
	r.leader = leader
}
