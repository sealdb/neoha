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

import "github.com/sealdb/neoha/internal/base/model"

// AddPeer used to add a peer to peers.
func (r *Raft) AddPeer(connStr string) error {
	if connStr == "" {
		return nil
	}
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.peers[connStr] != nil {
		r.WARNING("peer[%v].already.exists.in.peers[%+v].can't.add.repeatedly", connStr, r.peers)
		return nil
	}

	if r.idlePeers[connStr] != nil {
		r.WARNING("peer[%v].already.exists.in.idlePeers[%+v].can't.add.repeatedly", connStr, r.idlePeers)
		return nil
	}

	// we can't add ourself
	if r.getID() != connStr {
		p := NewPeer(r, connStr, r.conf.RequestTimeout, r.conf.HeartbeatTimeout)
		r.peers[connStr] = p

		// append peer to conf.Raft.Peers
		r.meta.Peers = append(r.meta.Peers, connStr)

		// write configure to file
		r.incEpochID()
		r.writePeersJSON()
	}
	r.WARNING("add.peer[%v].to.peers[%+v]", connStr, r.peers)
	return nil
}

// AddIdlePeer used to add a idle peer to peers.
func (r *Raft) AddIdlePeer(connStr string) error {
	if connStr == "" {
		return nil
	}
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.peers[connStr] != nil {
		r.WARNING("peer[%v].already.exists.in.peers[%+v].can't.add.repeatedly", connStr, r.peers)
		return nil
	}

	if r.idlePeers[connStr] != nil {
		r.WARNING("peer[%v].already.exists.in.idlePeers[%+v].can't.add.repeatedly", connStr, r.idlePeers)
		return nil
	}

	// we can't add ourself
	if r.getID() != connStr {
		p := NewPeer(r, connStr, r.conf.RequestTimeout, r.conf.HeartbeatTimeout)
		r.idlePeers[connStr] = p

		// append peer to conf.Raft.IdlePeers
		r.meta.IdlePeers = append(r.meta.IdlePeers, connStr)

		// write configure to file
		r.incEpochID()
		r.writePeersJSON()
	}
	r.WARNING("add.peer[%v].to.idlePeers[%+v]", connStr, r.idlePeers)
	return nil
}

// RemovePeer used to remove a peer from peers.
func (r *Raft) RemovePeer(connStr string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// we can't remove ourself
	if connStr != r.getID() {
		if _, ok := r.peers[connStr]; !ok {
			r.WARNING("peer[%v].not.exists.in.peers[%+v]", connStr, r.peers)
			return nil
		}
		delete(r.peers, connStr)

		// remove peer from conf.Raft.Peers
		for i, v := range r.meta.Peers {
			if v == connStr {
				r.meta.Peers = append(r.meta.Peers[:i], r.meta.Peers[i+1:]...)
				break
			}
		}

		// write configure to file
		r.incEpochID()
		r.writePeersJSON()
	}
	r.WARNING("removed.peer[%v].from.peers[%+v]", connStr, r.peers)
	return nil
}

// RemoveIdlePeer used to remove a idle peer from peers.
func (r *Raft) RemoveIdlePeer(connStr string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// we can't remove ourself
	if connStr != r.getID() {
		if _, ok := r.idlePeers[connStr]; !ok {
			r.WARNING("peer[%v].not.exists.in.idlePeers[%+v]", connStr, r.idlePeers)
			return nil
		}
		delete(r.idlePeers, connStr)

		// remove peer from conf.Raft.Peers
		for i, v := range r.meta.IdlePeers {
			if v == connStr {
				r.meta.IdlePeers = append(r.meta.IdlePeers[:i], r.meta.IdlePeers[i+1:]...)
				break
			}
		}

		// write configure to file
		r.incEpochID()
		r.writePeersJSON()
	}
	r.WARNING("removed.peer[%v].from.idlePeers[%+v]", connStr, r.idlePeers)
	return nil
}

func (r *Raft) isMGRClusterStatusOK() (bool, string) {
	var rows []map[string]string
	uuid := ""

	rows, err := r.mysql.GetMGRStats()
	if err != nil {
		r.ERROR("mgr.cluster.status.is.not.ok.get.mgr.stats[rows: %v, err: %v]", rows, err)
		return false, uuid
	}
	uuid, err = r.mysql.GetMGRMasterUUID()
	if err != nil {
		r.ERROR("mgr.cluster.status.is.not.ok.get.mgr.master.uuid[uuid: %v, err: %v]", uuid, err)
		return false, uuid
	}

	cnt := 0
	masterOK := false
	for _, row := range rows {
		if row["MEMBER_STATE"] == model.MGRStateOnline || row["MEMBER_STATE"] == model.MGRStateRecovering {
			cnt++
			if row["MEMBER_ID"] == uuid {
				masterOK = true
			}
		}
	}

	// MGR health uses reachable member count; for a 3-node group, 2 ONLINE is enough
	// (full Raft quorum is not required while one member is down).
	mgrQuorum := 2
	if !masterOK || cnt < mgrQuorum {
		r.ERROR("mgr.cluster.status.is.not.ok[masterOK:%v, cnt[%v]<mgrQuorum[%v], masteruuid:[%v], rows:[%v]]", masterOK, cnt, mgrQuorum, uuid, rows)
		return false, uuid
	}
	r.mgrClusterEverOK = true
	return true, uuid
}

// GetLeader returns leader.
func (r *Raft) GetLeader() string {
	return r.leader
}

// GetPeers returns peers string.
func (r *Raft) GetPeers() []string {
	return r.getPeers()
}

// GetIdlePeers returns idle peers string.
func (r *Raft) GetIdlePeers() []string {
	return r.getIdlePeers()
}

// GetAllPeers returns all peers string.
func (r *Raft) GetAllPeers() []string {
	return r.getAllPeers()
}

// GetQuorums returns quorums.
func (r *Raft) GetQuorums() int {
	return r.getQuorums()
}

// GetMembers returns member number.
func (r *Raft) GetMembers() int {
	return r.getMembers()
}

// GetVewiID returns view ID.
func (r *Raft) GetVewiID() uint64 {
	return r.getViewID()
}

// GetEpochID returns epoch id.
func (r *Raft) GetEpochID() uint64 {
	return r.getEpochID()
}

// GetState returns the raft state.
func (r *Raft) GetState() State {
	return r.state
}

// GetRaftRPC returns RaftRPC.
func (r *Raft) GetRaftRPC() *RaftRPC {
	return &RaftRPC{r}
}
