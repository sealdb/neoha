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
	"strconv"

	"neoha/base/model"
)

// RaftRPC tuple.
type RaftRPC struct {
	raft *Raft
}

// Ping rpc.
// send MsgRaftPing
func (r *RaftRPC) Ping(req *model.RaftRPCRequest, rsp *model.RaftRPCResponse) error {
	ret, err := r.raft.send(MsgRaftPing, req, r.raft.getHeartbeatTimeout())
	if err != nil {
		return err
	}
	*rsp = *ret.(*model.RaftRPCResponse)
	return nil
}

// Heartbeat rpc.
func (r *RaftRPC) Heartbeat(req *model.RaftRPCRequest, rsp *model.RaftRPCResponse) error {
	ret, err := r.raft.send(MsgRaftHeartbeat, req, r.raft.getHeartbeatTimeout())
	if err != nil {
		return err
	}
	*rsp = *ret.(*model.RaftRPCResponse)
	return nil
}

// RequestVote rpc.
func (r *RaftRPC) RequestVote(req *model.RaftRPCRequest, rsp *model.RaftRPCResponse) error {
	ret, err := r.raft.send(MsgRaftRequestVote, req, r.raft.getHeartbeatTimeout())
	if err != nil {
		return err
	}
	*rsp = *ret.(*model.RaftRPCResponse)
	return nil
}

// Status rpc.
func (r *RaftRPC) Status(req *model.RaftStatusRPCRequest, rsp *model.RaftStatusRPCResponse) error {
	rsp.RetCode = model.OK
	rsp.State = r.raft.GetState().String()
	rsp.Stats = r.raft.getStats()
	rsp.IdleCount, _ = strconv.ParseUint(strconv.Itoa(len(r.raft.getIdlePeers())), 10, 64)
	return nil
}

// EnablePurgeBinlog rpc.
func (r *RaftRPC) EnablePurgeBinlog(req *model.RaftStatusRPCRequest, rsp *model.RaftStatusRPCResponse) error {
	r.raft.SetSkipPurgeBinlog(false)
	return nil
}

// DisablePurgeBinlog rpc.
func (r *RaftRPC) DisablePurgeBinlog(req *model.RaftStatusRPCRequest, rsp *model.RaftStatusRPCResponse) error {
	r.raft.SetSkipPurgeBinlog(true)
	return nil
}

// EnableCheckSemiSync rpc.
func (r *RaftRPC) EnableCheckSemiSync(req *model.RaftStatusRPCRequest, rsp *model.RaftStatusRPCResponse) error {
	r.raft.SetSkipCheckSemiSync(false)
	return nil
}

// DisableCheckSemiSync rpc.
func (r *RaftRPC) DisableCheckSemiSync(req *model.RaftStatusRPCRequest, rsp *model.RaftStatusRPCResponse) error {
	r.raft.SetSkipCheckSemiSync(true)
	return nil
}
