//go:build integration

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

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/sealdb/neoha/test/integration/harness"
)

var (
	neohaRaftPorts1 = []int{18081, 18082, 18083}
	neohaRaftPorts2 = []int{18091, 18092, 18093}
	neohaRaftPorts3 = []int{18101, 18102, 18103}
)

// TestNeoHA3NodeMGR forms MGR via NeoHA setupMysql + Raft leader (production path).
func TestNeoHA3NodeMGR(t *testing.T) {
	cluster, backend, ctx, _ := newMySQLCluster(t, "neoha3")
	neoNodes, endpoints := startNeoHACluster(t, ctx, cluster, neohaRaftPorts1, writeMGRConfig)

	t.Log("neoha: wait raft leader")
	leader, err := harness.WaitRaftLeader(ctx, endpoints)
	assert.NoError(t, err)
	assert.NotEmpty(t, leader)

	t.Log("neoha: wait MGR 3 ONLINE (via setupMysql on each agent)")
	assert.NoError(t, backend.WaitMGROnlineMembers(ctx, cluster.Nodes[0], 3))

	cnt, err := backend.OnlineMGRMembers(cluster.Nodes[0])
	assert.NoError(t, err)
	assert.Equal(t, 3, cnt)
	_ = neoNodes
}

// TestNeoHAMGRFailoverMinority: 1 mysqld down, 2 survivors — MGR auto-elects PRIMARY; NeoHA waits.
func TestNeoHAMGRFailoverMinority(t *testing.T) {
	cluster, backend, ctx, _ := newMySQLCluster(t, "neoha-failover-minor")
	_, _ = startNeoHACluster(t, ctx, cluster, neohaRaftPorts2, writeMGRConfig)

	assert.NoError(t, backend.WaitMGROnlineMembers(ctx, cluster.Nodes[0], 3))

	primaryNode, err := backend.FindMGRPrimaryNode(cluster.Nodes)
	assert.NoError(t, err)

	stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	t.Logf("failover: stop MGR primary mysqld (%s port %d)", primaryNode.Name, primaryNode.Port)
	assert.NoError(t, backend.StopNode(stopCtx, primaryNode))

	remain := survivors(cluster.Nodes, primaryNode)
	newPrimary, err := backend.WaitMGRPrimaryOnAny(ctx, remain)
	assert.NoError(t, err)
	assert.Equal(t, "PRIMARY", mustRole(t, backend, newPrimary))
	assert.NotEqual(t, primaryNode.Name, newPrimary.Name, "MGR should elect a new PRIMARY on a survivor")

	// MGR still has majority (2/3); NeoHA waits for MGR, no forced Raft re-election.
	assert.NoError(t, backend.WaitMGROnlineMembers(ctx, newPrimary, 2))
	t.Logf("mgr auto-failover: old primary=%s new primary=%s", primaryNode.Name, newPrimary.Name)
}

// TestNeoHAMGRFailoverMajorityLoss: 2 mysqld down, 1 survivor — MGR cannot quorum; NeoHA elects and bootstraps.
func TestNeoHAMGRFailoverMajorityLoss(t *testing.T) {
	cluster, backend, ctx, _ := newMySQLCluster(t, "neoha-failover-major")
	neoNodes, _ := startNeoHACluster(t, ctx, cluster, neohaRaftPorts3, writeMGRConfig)

	assert.NoError(t, backend.WaitMGROnlineMembers(ctx, cluster.Nodes[0], 3))

	survivor := cluster.Nodes[2]
	down := []*harness.Node{cluster.Nodes[0], cluster.Nodes[1]}

	stopCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	for _, n := range down {
		t.Logf("failover: stop mysqld %s port %d", n.Name, n.Port)
		assert.NoError(t, backend.StopNode(stopCtx, n))
	}

	t.Log("failover: wait MGR quorum lost on sole survivor")
	assert.NoError(t, backend.WaitMGROnlineMembersBelow(ctx, survivor, 2))

	survivorEP := harness.EndpointsForClusterNodes(neoNodes, cluster, []*harness.Node{survivor})
	assert.Len(t, survivorEP, 1)

	leaderCtx, leaderCancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer leaderCancel()
	t.Log("failover: wait NeoHA Raft LEADER on survivor (GTID max + MGR bootstrap)")
	leaderEP, err := harness.WaitRaftLeader(leaderCtx, survivorEP)
	assert.NoError(t, err)
	assert.Equal(t, neoNodes[2].Endpoint, leaderEP)

	_, err = backend.WaitMGRPrimaryOnAny(ctx, []*harness.Node{survivor})
	assert.NoError(t, err)
	assert.Equal(t, "PRIMARY", mustRole(t, backend, survivor))
	t.Logf("neoha bootstrap: raft leader=%s mgr primary=%s", leaderEP, survivor.Name)
}

func mustRole(t *testing.T, backend *harness.MySQL80, node *harness.Node) string {
	t.Helper()
	role, err := backend.MGRMemberRole(node)
	assert.NoError(t, err)
	return role
}
