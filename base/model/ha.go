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
	RPCHASetLearner  = "HARPC.HASetLearner"
	RPCHADisable     = "HARPC.HADisable"
	RPCHAEnable      = "HARPC.HAEnable"
	RPCHATryToLeader = "HARPC.HATryToLeader"
)

type HARPCRequest struct {
	// My RPC client IP
	From string
}

type HARPCResponse struct {
	// Return code to rpc client
	RetCode string
}

func NewHARPCRequest() *HARPCRequest {
	return &HARPCRequest{}
}

func (req *HARPCRequest) GetFrom() string {
	return req.From
}

func NewHARPCResponse(code string) *HARPCResponse {
	return &HARPCResponse{RetCode: code}
}
