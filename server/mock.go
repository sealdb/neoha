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

package server

import (
	"fmt"
	"os"
	"testing"

	"neoha/base/common"
	"neoha/base/nlog"
	"neoha/base/nrpc"
	"neoha/config"
	"neoha/database/mysql"
	"neoha/election/raft"
	"neoha/manager/mysqld"

	"github.com/stretchr/testify/assert"
)

var (
	shortHeartbeatTimeoutForTest = 100
)

func MockServers(log *nlog.Log, port int, count int) ([]*Server, func()) {
	names := []string{}
	servers := []*Server{}
	ip, _ := common.GetLocalIP()

	os.Remove("peers.json")
	for i := 0; i < count; i++ {
		name := fmt.Sprintf("%s:%d", ip, port+i)
		names = append(names, name)

		conf := config.DefaultConfig()
		conf.Endpoint = name
		conf.Election.Raft.HeartbeatTimeout = shortHeartbeatTimeoutForTest
		conf.Election.Raft.ElectionTimeout = shortHeartbeatTimeoutForTest * 3

		server := NewServer(conf, log, ServerFollower)

		// mock mysqld
		_, mysqld, _ := mysqld.MockMysqld(log, port)
		server.manager.SetMysqld(mysqld)

		// mock mysql
		server.db.GetMysql().SetMysqlHandler(mysql.NewMockGTIDA())

		server.Init()
		servers = append(servers, server)
	}

	for _, server := range servers {
		for _, name := range names {
			server.election.GetRaft().AddPeer(name)
		}
	}

	for _, server := range servers {
		server.Start()
	}

	return servers, func() {
		os.Remove("peers.json")
		for i, s := range servers {
			log.Info("mock.server[%v].shutdown", names[i])
			s.Shutdown()
		}
	}
}

// wait the leader eggs when leadernums >0
// if leadernums == 0, we just want to sleep for a heartbeat broadcast
func MockWaitLeaderEggs(servers []*Server, leadernums int) {
	rafts := []*raft.Raft{}
	for _, server := range servers {
		rafts = append(rafts, server.election.GetRaft())
	}
	raft.MockWaitLeaderEggs(rafts, leadernums)
}

// nrpc client
func MockGetClient(t *testing.T, svrConn string) (*nrpc.Client, func()) {
	client, err := nrpc.NewClient(svrConn, 100)
	assert.Nil(t, err)

	return client, func() {
		client.Close()
	}
}
