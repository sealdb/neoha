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
	"fmt"
	"testing"

	"github.com/sealdb/neoha/internal/base/common"
	"github.com/sealdb/neoha/internal/base/model"
	"github.com/sealdb/neoha/internal/base/nlog"

	"github.com/stretchr/testify/assert"
)

// TEST EFFECTS:
// test remove nodes from client rpc
//
// TEST PROCESSES:
// 1. Start rpc server
// 2. send command to rpc server
// 3. check the response
func testServerRPCAddRemoveNodes(t *testing.T, replMode model.MysqlReplMode) {
	var method = model.RPCNodesAdd

	log := nlog.NewStdLog(nlog.Level(nlog.ERROR))
	port := common.RandomPort(8000, 9000)
	servers, cleanup := MockServers(log, port, 1, replMode)
	defer cleanup()

	name := servers[0].Address()
	ip, err := common.GetLocalIP()
	assert.Nil(t, err)

	// add nodes
	{
		{
			method = model.RPCNodesAdd
			req := model.NewNodeRPCRequest()
			req.Nodes = []string{
				fmt.Sprintf("%s:%d", ip, port),
				fmt.Sprintf("%s:%d", ip, port+1),
				fmt.Sprintf("%s:%d", ip, port+2),
			}
			rsp := model.NewNodeRPCResponse(model.OK)
			c, cleanup := MockGetClient(t, name)

			if err := c.Call(method, req, rsp); err != nil {
				assert.Nil(t, err)
			}
			cleanup()
			assert.Equal(t, rsp.RetCode, model.OK)
		}

		{
			method = model.RPCNodes
			req := model.NewNodeRPCRequest()
			rsp := model.NewNodeRPCResponse(model.OK)
			c, cleanup := MockGetClient(t, name)

			if err := c.Call(method, req, rsp); err != nil {
				assert.Nil(t, err)
			}
			cleanup()

			want := []string{
				fmt.Sprintf("%s:%d", ip, port),
				fmt.Sprintf("%s:%d", ip, port+1),
				fmt.Sprintf("%s:%d", ip, port+2),
			}
			got := rsp.GetNodes()
			assert.Equal(t, want, got)
		}

	}

	// remove nodes
	{
		{
			method = model.RPCNodesRemove
			req := model.NewNodeRPCRequest()
			req.Nodes = []string{
				fmt.Sprintf("%s:%d", ip, port),
				fmt.Sprintf("%s:%d", ip, port+1),
			}
			rsp := model.NewNodeRPCResponse(model.OK)
			c, cleanup := MockGetClient(t, name)

			if err := c.Call(method, req, rsp); err != nil {
				assert.Nil(t, err)
			}
			cleanup()

			assert.Equal(t, rsp.RetCode, model.OK)
		}

		{
			method = model.RPCNodes
			req := model.NewNodeRPCRequest()
			rsp := model.NewNodeRPCResponse(model.OK)
			c, cleanup := MockGetClient(t, name)

			if err := c.Call(method, req, rsp); err != nil {
				assert.Nil(t, err)
			}
			cleanup()

			want := []string{
				fmt.Sprintf("%s:%d", ip, port),
				fmt.Sprintf("%s:%d", ip, port+2),
			}
			got := rsp.GetNodes()
			assert.Equal(t, want, got)
		}
	}
}

func TestServerRPCAddRemoveNodes_MySQL_SemiSync(t *testing.T) {
	testServerRPCAddRemoveNodes(t, model.ReplModeSemiSync)
}

func TestServerRPCAddRemoveNodes_MySQL_MGR(t *testing.T) {
	testServerRPCAddRemoveNodes(t, model.ReplModeMGR)
}
