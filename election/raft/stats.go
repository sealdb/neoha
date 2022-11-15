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
	"sync/atomic"
	"time"

	"neoha/base/model"
)

// IncLeaderPromotes counter.
func (s *Raft) IncLeaderPromotes() {
	atomic.AddUint64(&s.stats.LeaderPromotes, 1)
}

// IncLeaderDegrades counter.
func (s *Raft) IncLeaderDegrades() {
	atomic.AddUint64(&s.stats.LeaderDegrades, 1)
}

// IncLeaderGetHeartbeatRequests counter.
func (s *Raft) IncLeaderGetHeartbeatRequests() {
	atomic.AddUint64(&s.stats.LeaderGetHeartbeatRequests, 1)
}

// IncLeaderPurgeBinlogs counter.
func (s *Raft) IncLeaderPurgeBinlogs() {
	atomic.AddUint64(&s.stats.LeaderPurgeBinlogs, 1)
}

// IncLeaderPurgeBinlogFails counter.
func (s *Raft) IncLeaderPurgeBinlogFails() {
	atomic.AddUint64(&s.stats.LeaderPurgeBinlogFails, 1)
}

// IncLeaderGetVoteRequests counter.
func (s *Raft) IncLeaderGetVoteRequests() {
	atomic.AddUint64(&s.stats.LeaderGetVoteRequests, 1)
}

// IncLessHeartbeatAcks counter.
func (s *Raft) IncLessHeartbeatAcks() {
	atomic.AddUint64(&s.stats.LessHearbeatAcks, 1)
}

// IncCandidatePromotes counter.
func (s *Raft) IncCandidatePromotes() {
	atomic.AddUint64(&s.stats.CandidatePromotes, 1)
}

// IncCandidateDegrades counter.
func (s *Raft) IncCandidateDegrades() {
	atomic.AddUint64(&s.stats.CandidateDegrades, 1)
}

// SetRaftMysqlStatus used to set mysql status.
func (s *Raft) SetRaftMysqlStatus(rms model.RAFTMYSQL_STATUS) {
	s.stats.RaftMysqlStatus = rms
}

// ResetRaftMysqlStatus used to reset mysql status.
func (s *Raft) ResetRaftMysqlStatus() {
	s.stats.RaftMysqlStatus = model.RAFTMYSQL_NONE
}

func (s *Raft) getStats() *model.RaftStats {
	return &model.RaftStats{
		HaEnables:                  atomic.LoadUint64(&s.stats.HaEnables),
		LeaderPromotes:             atomic.LoadUint64(&s.stats.LeaderPromotes),
		LeaderDegrades:             atomic.LoadUint64(&s.stats.LeaderDegrades),
		LeaderGetHeartbeatRequests: atomic.LoadUint64(&s.stats.LeaderGetHeartbeatRequests),
		LeaderGetVoteRequests:      atomic.LoadUint64(&s.stats.LeaderGetVoteRequests),
		LeaderPurgeBinlogs:         atomic.LoadUint64(&s.stats.LeaderPurgeBinlogs),
		LeaderPurgeBinlogFails:     atomic.LoadUint64(&s.stats.LeaderPurgeBinlogFails),
		LessHearbeatAcks:           atomic.LoadUint64(&s.stats.LessHearbeatAcks),
		CandidatePromotes:          atomic.LoadUint64(&s.stats.CandidatePromotes),
		CandidateDegrades:          atomic.LoadUint64(&s.stats.CandidateDegrades),
		StateUptimes:               uint64(time.Since(s.stateBegin).Seconds()),
		RaftMysqlStatus:            s.stats.RaftMysqlStatus,
	}
}
