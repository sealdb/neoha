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

package server

import (
	"github.com/sealdb/neoha/internal/base/model"
)

type NodeRPC struct {
	server *Server
}

func (s *Server) GetNodeRPC() *NodeRPC {
	return &NodeRPC{s}
}

func (n *NodeRPC) AddNodes(req *model.NodeRPCRequest, rsp *model.NodeRPCResponse) error {
	log := n.server.log
	rsp.RetCode = model.OK
	nodes := req.GetNodes()

	log.Warning("server.rpc.node.add:%+v", req)
	for _, node := range nodes {
		if err := n.server.election.GetRaft().AddPeer(node); err != nil {
			rsp.RetCode = err.Error()
			log.Error("rpc.add.peer[%v].error[%v]", node, err)
			return nil
		}
	}
	return nil
}

func (n *NodeRPC) AddIdleNodes(req *model.NodeRPCRequest, rsp *model.NodeRPCResponse) error {
	log := n.server.log
	rsp.RetCode = model.OK
	nodes := req.GetNodes()

	log.Warning("server.rpc.node.add:%+v", req)
	for _, node := range nodes {
		if err := n.server.election.GetRaft().AddIdlePeer(node); err != nil {
			rsp.RetCode = err.Error()
			log.Error("rpc.add.idle.peer[%v].error[%v]", node, err)
			return nil
		}
	}
	return nil
}

func (n *NodeRPC) RemoveNodes(req *model.NodeRPCRequest, rsp *model.NodeRPCResponse) error {
	log := n.server.log
	rsp.RetCode = model.OK
	nodes := req.GetNodes()

	log.Warning("server.rpc.node.remove:%+v", req)
	for _, node := range nodes {
		if err := n.server.election.GetRaft().RemovePeer(node); err != nil {
			rsp.RetCode = err.Error()
			log.Error("rpc.remove.peer[%v].error[%v]", node, err)
			return nil
		}
	}
	return nil
}

func (n *NodeRPC) RemoveIdleNodes(req *model.NodeRPCRequest, rsp *model.NodeRPCResponse) error {
	log := n.server.log
	rsp.RetCode = model.OK
	nodes := req.GetNodes()

	log.Warning("server.rpc.node.remove:%+v", req)
	for _, node := range nodes {
		if err := n.server.election.GetRaft().RemoveIdlePeer(node); err != nil {
			rsp.RetCode = err.Error()
			log.Error("rpc.remove.idle.peer[%v].error[%v]", node, err)
			return nil
		}
	}
	return nil
}

func (n *NodeRPC) GetNodes(req *model.NodeRPCRequest, rsp *model.NodeRPCResponse) error {
	rsp.RetCode = model.OK
	rsp.Leader = n.server.election.GetRaft().GetLeader()
	rsp.ViewID = n.server.election.GetRaft().GetVewiID()
	rsp.EpochID = n.server.election.GetRaft().GetEpochID()
	rsp.State = n.server.election.GetRaft().GetState().String()
	nodes := n.server.election.GetRaft().GetAllPeers()
	rsp.Nodes = append(rsp.Nodes, nodes...)
	return nil
}
