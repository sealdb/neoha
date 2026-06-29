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
	neohaRaftPorts3 = []int{18101, 18102, 18103}
)

// TestNeoHAMGRWarmSuite boots one warm 3-node MGR cluster and reuses it for formation
// verification and minority failover (failover subtest measures only the fault segment).
func TestNeoHAMGRWarmSuite(t *testing.T) {
	warm := bootstrapWarmNeoHA(t,
		newWarmMySQLCluster(t, warmMGRClusterName, mgrMySQLPorts, mgrGRPorts, false),
		neohaRaftPorts1,
		writeMGRConfig,
		neoHAStartOpts{skipCLIWire: true},
	)
	waitMGRFormation(t, warm)

	t.Run("3NodeMGR", func(t *testing.T) {
		start := time.Now()
		leader, err := harness.WaitRaftLeader(warm.ctx, warm.endpoints)
		assert.NoError(t, err)
		assert.NotEmpty(t, leader)

		cnt, err := warm.backend.OnlineMGRMembers(warm.cluster.Nodes[0])
		assert.NoError(t, err)
		assert.Equal(t, 3, cnt)
		t.Logf("timing: formation verify %s", time.Since(start))
	})

	t.Run("FailoverMinority", func(t *testing.T) {
		start := time.Now()

	primaryNode, err := warm.backend.FindMGRPrimaryNode(warm.cluster.Nodes)
	assert.NoError(t, err)
	if !assert.NotNil(t, primaryNode, "MGR PRIMARY must exist before failover") {
		return
	}

		stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		t.Logf("failover: stop MGR primary mysqld (%s port %d)", primaryNode.Name, primaryNode.Port)
		assert.NoError(t, warm.backend.StopNode(stopCtx, primaryNode))

		remain := survivors(warm.cluster.Nodes, primaryNode)
		newPrimary, err := warm.backend.WaitMGRPrimaryOnAny(warm.ctx, remain)
		assert.NoError(t, err)
		assert.Equal(t, "PRIMARY", mustRole(t, warm.backend, newPrimary))
		assert.NotEqual(t, primaryNode.Name, newPrimary.Name, "MGR should elect a new PRIMARY on a survivor")

		assert.NoError(t, warm.backend.WaitMGROnlineMembers(warm.ctx, newPrimary, 2))
		t.Logf("timing: failover segment %s (old=%s new=%s)", time.Since(start), primaryNode.Name, newPrimary.Name)
	})
}

// TestNeoHAMGRFailoverMajorityLoss: 2 mysqld down, 1 survivor — MGR cannot quorum; NeoHA elects and bootstraps.
func TestNeoHAMGRFailoverMajorityLoss(t *testing.T) {
	warm := bootstrapWarmNeoHA(t,
		newWarmMySQLCluster(t, warmMGRMajorClusterName, mgrMySQLPorts, mgrGRPorts, false),
		neohaRaftPorts3,
		writeMGRConfig,
		neoHAStartOpts{skipCLIWire: false},
	)
	waitMGRFormation(t, warm)

	survivor := warm.cluster.Nodes[2]
	down := []*harness.Node{warm.cluster.Nodes[0], warm.cluster.Nodes[1]}

	start := time.Now()
	stopCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	for _, n := range down {
		t.Logf("failover: stop mysqld %s port %d", n.Name, n.Port)
		assert.NoError(t, warm.backend.StopNode(stopCtx, n))
	}

	t.Log("failover: wait MGR quorum lost on sole survivor")
	assert.NoError(t, warm.backend.WaitMGROnlineMembersBelow(warm.ctx, survivor, 2))

	survivorEP := harness.EndpointsForClusterNodes(warm.neoNodes, warm.cluster, []*harness.Node{survivor})
	assert.Len(t, survivorEP, 1)

	leaderCtx, leaderCancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer leaderCancel()
	t.Log("failover: wait NeoHA Raft LEADER on survivor (GTID max + MGR bootstrap)")
	leaderEP, err := harness.WaitRaftLeader(leaderCtx, survivorEP)
	assert.NoError(t, err)
	assert.Equal(t, warm.neoNodes[2].Endpoint, leaderEP)

	_, err = warm.backend.WaitMGRPrimaryOnAny(leaderCtx, []*harness.Node{survivor})
	assert.NoError(t, err)
	assert.Equal(t, "PRIMARY", mustRole(t, warm.backend, survivor))
	t.Logf("timing: majority-loss failover segment %s (raft leader=%s mgr primary=%s)", time.Since(start), leaderEP, survivor.Name)
}

func mustRole(t *testing.T, backend *harness.MySQL80, node *harness.Node) string {
	t.Helper()
	role, err := backend.MGRMemberRole(node)
	assert.NoError(t, err)
	return role
}
