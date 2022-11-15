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

package model

const (
	RPCNodesAdd        = "NodeRPC.AddNodes"
	RPCIdleNodesAdd    = "NodeRPC.AddIdleNodes"
	RPCNodesRemove     = "NodeRPC.RemoveNodes"
	RPCIdleNodesRemove = "NodeRPC.RemoveIdleNodes"
	RPCNodes           = "NodeRPC.GetNodes"
)

type NodeRPCRequest struct {
	// The IP of this request
	From string

	// Node endpoint lists
	Nodes []string
}

type NodeRPCResponse struct {
	// The Epoch ID of the raft
	EpochID uint64

	// The View ID of the raft
	ViewID uint64

	// The State of the raft:
	// FOLLOWER/CANDIDATE/LEADER/IDLE/INVALID
	State string

	// The Leader endpoint of the cluster
	Leader string

	// The Nodes(endpoint) of the cluster
	Nodes []string

	// Return code to rpc client:
	// OK or other errors
	RetCode string
}

func NewNodeRPCRequest() *NodeRPCRequest {
	return &NodeRPCRequest{}
}

func (req *NodeRPCRequest) GetFrom() string {
	return req.From
}

func (req *NodeRPCRequest) GetNodes() []string {
	return req.Nodes
}

func NewNodeRPCResponse(code string) *NodeRPCResponse {
	return &NodeRPCResponse{RetCode: code}
}

func (rsp *NodeRPCResponse) GetNodes() []string {
	return rsp.Nodes
}

func (rsp *NodeRPCResponse) GetLeader() string {
	return rsp.Leader
}
