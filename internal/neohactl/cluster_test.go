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

package neohactl

import (
	"fmt"
	"github.com/sealdb/neoha/internal/base/model"
	"testing"

	"github.com/sealdb/neoha/internal/base/common"
	"github.com/sealdb/neoha/internal/base/nlog"
	"github.com/sealdb/neoha/internal/election/raft"
	"github.com/sealdb/neoha/internal/server"

	"github.com/stretchr/testify/assert"
)

func testCLIClusterCommand(t *testing.T, replMode model.MysqlReplMode) {
	var leader string
	var follower string

	err := createConfig()
	ErrorOK(err)
	defer removeConfig()

	// get leader
	{
		log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
		port := common.RandomPort(6000, 10000)
		servers, cleanup := server.MockServers(log, port, 2, replMode)
		defer cleanup()

		server.MockWaitServerLeaderEggs(servers, 1, replMode)
		for _, server := range servers {
			if server.GetElection().GetRaft().GetState() == raft.LEADER {
				leader = server.Address()
			} else {
				follower = server.Address()
			}
		}
	}

	// 1. test add node direct to leader
	{
		// setting neoha is leader
		{
			conf, err := GetConfig()
			ErrorOK(err)
			conf.Endpoint = leader
			err = SaveConfig(conf)
			ErrorOK(err)
		}

		// status.
		{
			cmd := NewClusterCommand()
			_, err := executeCommand(cmd, "status")
			assert.Nil(t, err)
		}

		// add node.
		{
			ip, err := common.GetLocalIP()
			cmd := NewClusterCommand()
			_, err = executeCommand(cmd, "add", fmt.Sprintf("%s:6001,%s:6002", ip, ip))
			assert.Nil(t, err)
		}

		// status.
		{
			cmd := NewClusterCommand()
			_, err := executeCommand(cmd, "status")
			assert.Nil(t, err)
		}
	}

	// 2. test add node forward to leader
	{
		// setting neoha is follower
		{
			conf, err := GetConfig()
			ErrorOK(err)
			conf.Endpoint = follower
			err = SaveConfig(conf)
			ErrorOK(err)
		}

		// add nodes.
		{
			ip, err := common.GetLocalIP()
			cmd := NewClusterCommand()
			_, err = executeCommand(cmd, "add", fmt.Sprintf("%s:7001,%s:7002", ip, ip))
			assert.Nil(t, err)
		}

		// status.
		{
			cmd := NewClusterCommand()
			_, err := executeCommand(cmd, "status")
			assert.Nil(t, err)
		}
	}

	// 3. test remove node forward to leader
	{
		// setting neoha is follower
		{
			conf, err := GetConfig()
			ErrorOK(err)
			conf.Endpoint = follower
			err = SaveConfig(conf)
			ErrorOK(err)
		}

		// remove nodes.
		{
			ip, err := common.GetLocalIP()
			cmd := NewClusterCommand()
			_, err = executeCommand(cmd, "remove", fmt.Sprintf("%s:7001,%s:7002", ip, ip))
			assert.Nil(t, err)
		}

		// status.
		{
			cmd := NewClusterCommand()
			_, err := executeCommand(cmd, "status")
			assert.Nil(t, err)
		}

		// json status
		{
			cmd := NewClusterCommand()
			_, err := executeCommand(cmd, "status", "json")
			assert.Nil(t, err)
		}

		// msyql status.
		{
			cmd := NewClusterCommand()
			_, err := executeCommand(cmd, "mysql")
			assert.Nil(t, err)
		}

		// msyql GTID.
		{
			cmd := NewClusterCommand()
			_, err := executeCommand(cmd, "gtid")
			assert.Nil(t, err)
		}

		// raft
		{
			cmd := NewClusterCommand()
			_, err := executeCommand(cmd, "raft")
			assert.Nil(t, err)
		}

		// neoha
		{
			cmd := NewClusterCommand()
			_, err := executeCommand(cmd, "neoha")
			assert.Nil(t, err)
		}

	}
}

func TestCLIClusterCommand_MySQL_SemiSync(t *testing.T) {
	testCLIClusterCommand(t, model.ReplModeSemiSync)
}

func TestCLIClusterCommand_MySQL_MGR(t *testing.T) {
	testCLIClusterCommand(t, model.ReplModeMGR)
}
