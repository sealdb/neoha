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
	"testing"

	"github.com/sealdb/neoha/internal/base/common"
	"github.com/sealdb/neoha/internal/base/model"
	"github.com/sealdb/neoha/internal/base/nlog"
	"github.com/sealdb/neoha/internal/election/raft"

	"github.com/stretchr/testify/assert"
)

func testServerRPCUser(t *testing.T, replMode model.MysqlReplMode) {
	var leader, follower string
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	servers, cleanup := MockServers(log, port, 2, replMode)
	defer cleanup()

	// leader and follower
	{
		MockWaitServerLeaderEggs(servers, 1, replMode)
		for i, server := range servers {
			if server.election.GetRaft().GetState() == raft.LEADER {
				leader = servers[i].Address()
			} else {
				follower = servers[i].Address()
			}
		}
	}

	// send to follower
	{
		c, cleanup := MockGetClient(t, follower)
		defer cleanup()

		req := model.NewMysqlUserRPCRequest()
		rsp := model.NewMysqlUserRPCResponse(model.OK)
		method := model.RPCMysqlCreateNormalUser
		if err := c.Call(method, req, rsp); err != nil {
			assert.Nil(t, err)
		}

		want := "nonleader.can.not.createuser"
		got := rsp.RetCode
		assert.Equal(t, want, got)
	}

	// send to leader
	{
		c, cleanup := MockGetClient(t, leader)
		defer cleanup()

		req := model.NewMysqlUserRPCRequest()
		rsp := model.NewMysqlUserRPCResponse(model.OK)
		method := model.RPCMysqlCreateNormalUser
		if err := c.Call(method, req, rsp); err != nil {
			assert.Nil(t, err)
		}

		want := model.OK
		got := rsp.RetCode
		assert.Equal(t, want, got)
	}

	// Create super user.
	{
		c, cleanup := MockGetClient(t, leader)
		defer cleanup()

		req := model.NewMysqlUserRPCRequest()
		rsp := model.NewMysqlUserRPCResponse(model.OK)
		method := model.RPCMysqlCreateSuperUser
		if err := c.Call(method, req, rsp); err != nil {
			assert.Nil(t, err)
		}

		want := model.OK
		got := rsp.RetCode
		assert.Equal(t, want, got)
	}

	// Get user.
	{
		c, cleanup := MockGetClient(t, leader)
		defer cleanup()

		req := model.NewMysqlUserRPCRequest()
		rsp := model.NewMysqlUserRPCResponse(model.OK)
		method := model.RPCMysqlGetUser
		if err := c.Call(method, req, rsp); err != nil {
			assert.Nil(t, err)
		}

		want := model.OK
		got := rsp.RetCode
		assert.Equal(t, want, got)

		want1 := []model.MysqlUser{
			{User: "user1",
				Host:      "localhost",
				SuperPriv: "N"},
			{User: "root",
				Host:      "localhost",
				SuperPriv: "Y"},
		}
		got1 := rsp.Users
		assert.Equal(t, want1, got1)
	}

	// Drop user.
	{
		c, cleanup := MockGetClient(t, leader)
		defer cleanup()

		req := model.NewMysqlUserRPCRequest()
		rsp := model.NewMysqlUserRPCResponse(model.OK)
		method := model.RPCMysqlDropUser
		if err := c.Call(method, req, rsp); err != nil {
			assert.Nil(t, err)
		}

		want := model.OK
		got := rsp.RetCode
		assert.Equal(t, want, got)
	}

	// Create user with privileges.
	{
		c, cleanup := MockGetClient(t, leader)
		defer cleanup()

		req := model.NewMysqlUserRPCRequest()
		rsp := model.NewMysqlUserRPCResponse(model.OK)
		method := model.RPCMysqlCreateUserWithPrivileges
		if err := c.Call(method, req, rsp); err != nil {
			assert.Nil(t, err)
		}

		want := model.OK
		got := rsp.RetCode
		assert.Equal(t, want, got)
	}

	// Change password.
	{
		c, cleanup := MockGetClient(t, leader)
		defer cleanup()

		req := model.NewMysqlUserRPCRequest()
		rsp := model.NewMysqlUserRPCResponse(model.OK)
		method := model.RPCMysqlChangePassword
		if err := c.Call(method, req, rsp); err != nil {
			assert.Nil(t, err)
		}

		want := model.OK
		got := rsp.RetCode
		assert.Equal(t, want, got)
	}

	// send to leader
	{
		c, cleanup := MockGetClient(t, leader)
		defer cleanup()

		req := model.NewMysqlUserRPCRequest()
		rsp := model.NewMysqlUserRPCResponse(model.OK)
		method := model.RPCMysqlCreateNormalUser
		if err := c.Call(method, req, rsp); err != nil {
			assert.Nil(t, err)
		}

		want := model.OK
		got := rsp.RetCode
		assert.Equal(t, want, got)
	}
}

func TestServerRPCUser_MySQL_SemiSync(t *testing.T) {
	testServerRPCUser(t, model.ReplModeSemiSync)
}

func TestServerRPCUser_MySQL_MGR(t *testing.T) {
	testServerRPCUser(t, model.ReplModeMGR)
}
