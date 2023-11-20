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
	"errors"
	"fmt"
	"github.com/sealdb/neoha/internal/base/model"
	"testing"
	"time"

	"github.com/sealdb/neoha/internal/base/common"
	"github.com/sealdb/neoha/internal/base/nlog"
	"github.com/sealdb/neoha/internal/election/raft"
	"github.com/sealdb/neoha/internal/server"

	"github.com/stretchr/testify/assert"
)

func testCLIRaftCommand(t *testing.T, replMode model.MysqlReplMode) {
	var leader, newleader string

	err := createConfig()
	ErrorOK(err)
	defer removeConfig()

	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	servers, scleanup := server.MockServers(log, port, 3, replMode)
	defer scleanup()

	// get leader
	{
		server.MockWaitServerLeaderEggs(servers, 1, replMode)
		for _, server := range servers {
			if server.GetElection().GetRaft().GetState() == raft.LEADER {
				leader = server.Address()
				break
			}
		}
	}

	conf, err := GetConfig()

	// 1. test disable raft to leader
	{
		// setting neoha is leader
		{
			ErrorOK(err)
			conf.Endpoint = leader
			err = SaveConfig(conf)
			ErrorOK(err)
		}

		// disable raft
		{
			cmd := NewRaftCommand()
			_, err := executeCommand(cmd, "disable")
			assert.Nil(t, err)
		}

		// check the new leader
		{
			server.MockWaitServerLeaderEggs(servers, 1, replMode)
			for _, server := range servers {
				if server.GetElection().GetRaft().GetState() == raft.LEADER {
					newleader = server.Address()
					break
				}
			}

			if leader == newleader {
				ErrorOK(errors.New("leader==newleader.error"))
			}
		}

		// enable raft
		{
			cmd := NewRaftCommand()
			_, err := executeCommand(cmd, "enable")
			assert.Nil(t, err)
		}

		// trytoleader
		{
			cmd := NewRaftCommand()
			_, err := executeCommand(cmd, "trytoleader")
			assert.Nil(t, err)
		}

		// wait cli done, avoid enable cmd changes STOPPED to FOLLOWER
		time.Sleep(time.Duration(conf.Election.Raft.ElectionTimeout))
	}

	// 2. test add/remove ndoes to local
	{

		// add
		{
			ip, err := common.GetLocalIP()
			assert.Nil(t, err)
			arg := fmt.Sprintf("%s:7001,%s:7002", ip, ip)
			cmd := NewRaftCommand()
			_, err = executeCommand(cmd, "add", arg)
			assert.Nil(t, err)
		}

		// ls
		{
			cmd := NewRaftCommand()
			_, err := executeCommand(cmd, "nodes")
			assert.Nil(t, err)
		}

		// remove
		{
			ip, err := common.GetLocalIP()
			assert.Nil(t, err)
			arg := fmt.Sprintf("%s:7001,%s:7002", ip, ip)
			cmd := NewRaftCommand()
			_, err = executeCommand(cmd, "remove", arg)
			assert.Nil(t, err)
		}

		// ls
		{
			cmd := NewRaftCommand()
			_, err := executeCommand(cmd, "nodes")
			assert.Nil(t, err)
		}

		// status
		{
			cmd := NewRaftCommand()
			_, err := executeCommand(cmd, "status")
			assert.Nil(t, err)
		}

		// purge binlog
		{
			cmd := NewRaftCommand()
			_, err := executeCommand(cmd, "disablepurgebinlog")
			assert.Nil(t, err)
		}

		// purge binlog enable
		{
			cmd := NewRaftCommand()
			_, err := executeCommand(cmd, "enablepurgebinlog")
			assert.Nil(t, err)
		}

		if replMode == model.ReplModeSemiSync {
			// disable check semi-sync
			{
				cmd := NewRaftCommand()
				_, err := executeCommand(cmd, "disablechecksemisync")
				assert.Nil(t, err)
			}

			// enable check semi-sync
			{
				cmd := NewRaftCommand()
				_, err := executeCommand(cmd, "enablechecksemisync")
				assert.Nil(t, err)
			}
		}
	}
}

func TestCLIRaftCommand_MySQL_SemiSync(t *testing.T) {
	testCLIRaftCommand(t, model.ReplModeSemiSync)
}

func TestCLIRaftCommand_MySQL_MGR(t *testing.T) {
	testCLIRaftCommand(t, model.ReplModeMGR)
}
