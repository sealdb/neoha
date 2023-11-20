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
	"testing"

	"github.com/sealdb/neoha/internal/base/common"
	"github.com/sealdb/neoha/internal/base/nlog"

	"github.com/stretchr/testify/assert"
)

// TEST EFFECTS:
// test a single server remove and add peer
//
// TEST PROCESSES:
// 1. add peer
// 2. add same peer
// 3. remove peer
// 4. remove same peer
func testServer(t *testing.T, replMode model.MysqlReplMode) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	servers, cleanup := MockServers(log, port, 1, replMode)
	defer cleanup()

	server := servers[0]
	newpeer := "127.0.0.1:8081"
	newidlepeer := "127.0.0.1:8082"

	member := server.election.GetRaft().GetMembers()
	quorums := server.election.GetRaft().GetQuorums()
	assert.Equal(t, member, 1)
	assert.Equal(t, quorums, 1)

	peers := server.election.GetRaft().GetPeers()
	assert.Equal(t, len(peers), 1)
	idlePeers := server.election.GetRaft().GetIdlePeers()
	assert.Equal(t, len(idlePeers), 0)

	// add peer test
	err := server.election.GetRaft().AddPeer(newpeer)
	assert.Nil(t, err)
	peers = server.election.GetRaft().GetPeers()
	assert.Equal(t, len(peers), 2)

	// add idle peer test
	err = server.election.GetRaft().AddIdlePeer(newidlepeer)
	assert.Nil(t, err)
	idlePeers = server.election.GetRaft().GetIdlePeers()
	assert.Equal(t, len(idlePeers), 1)

	// add same peer to peers test
	err = server.election.GetRaft().AddPeer(newpeer)
	assert.Nil(t, err)
	peers = server.election.GetRaft().GetPeers()
	assert.Equal(t, len(peers), 2)

	// add same idle peer to peers test
	err = server.election.GetRaft().AddPeer(newidlepeer)
	assert.Nil(t, err)
	peers = server.election.GetRaft().GetPeers()
	assert.Equal(t, len(peers), 2)

	// add same peer to idle peers test
	err = server.election.GetRaft().AddIdlePeer(newpeer)
	assert.Nil(t, err)
	idlePeers = server.election.GetRaft().GetIdlePeers()
	assert.Equal(t, len(idlePeers), 1)

	// add same idle peer to idle peers test
	err = server.election.GetRaft().AddIdlePeer(newidlepeer)
	assert.Nil(t, err)
	idlePeers = server.election.GetRaft().GetIdlePeers()
	assert.Equal(t, len(idlePeers), 1)

	// remove peer from peers test
	err = server.election.GetRaft().RemovePeer(newpeer)
	assert.Nil(t, err)
	peers = server.election.GetRaft().GetPeers()
	assert.Equal(t, len(peers), 1)

	// remove idle peer from idle peers test
	err = server.election.GetRaft().RemoveIdlePeer(newidlepeer)
	assert.Nil(t, err)
	idlePeers = server.election.GetRaft().GetIdlePeers()
	assert.Equal(t, len(idlePeers), 0)

	// remove peer again
	err = server.election.GetRaft().RemovePeer(newpeer)
	assert.Nil(t, err)
	peers = server.election.GetRaft().GetPeers()
	assert.Equal(t, len(peers), 1)

	// remove idle peer again
	err = server.election.GetRaft().RemoveIdlePeer(newidlepeer)
	assert.Nil(t, err)
	idlePeers = server.election.GetRaft().GetIdlePeers()
	assert.Equal(t, len(idlePeers), 0)

	peerAddr := server.PeerAddress()
	assert.Equal(t, ":6060", peerAddr)

	mysqlAdmin := server.MySQLAdmin()
	assert.Equal(t, "root", mysqlAdmin)

	mysqlPasswd := server.MySQLPasswd()
	assert.Equal(t, "", mysqlPasswd)
}

func TestServer_MySQL_SemiSync(t *testing.T) {
	testServer(t, model.ReplModeSemiSync)
}

func TestServer_MySQL_MGR(t *testing.T) {
	testServer(t, model.ReplModeMGR)
}
