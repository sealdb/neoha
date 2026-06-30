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
		newWarmMySQLCluster(t, warmMGRClusterName, mgrMySQLPorts, mgrGRPorts, false, false),
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
		failoverCtx, failoverCancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer failoverCancel()
		newPrimary, err := warm.backend.WaitMGRPrimaryOnAny(failoverCtx, remain)
		if !assert.NoError(t, err) {
			return
		}
		if !assert.NotNil(t, newPrimary) {
			return
		}
		assert.Equal(t, "PRIMARY", mustRole(t, warm.backend, newPrimary))
		assert.NotEqual(t, primaryNode.Name, newPrimary.Name, "MGR should elect a new PRIMARY on a survivor")

		assert.NoError(t, warm.backend.WaitMGROnlineMembers(failoverCtx, newPrimary, 2))
		t.Logf("timing: failover segment %s (old=%s new=%s)", time.Since(start), primaryNode.Name, newPrimary.Name)
	})
}

// TestNeoHAMGRFailoverMajorityLoss: 2 mysqld down, 1 survivor — MGR cannot quorum;
// NeoHA Raft elects GTID-max survivor, force-bootstraps read-only PRIMARY, then opens
// writes only after enough members rejoin (2+ ONLINE).
func TestNeoHAMGRFailoverMajorityLoss(t *testing.T) {
	warm := bootstrapWarmNeoHA(t,
		newWarmMySQLCluster(t, warmMGRMajorClusterName, mgrMajorMySQLPorts, mgrMajorGRPorts, false, true),
		neohaRaftPorts3,
		writeMGRMajorConfig,
		neoHAStartOpts{skipCLIWire: true},
	)
	waitMGRFormation(t, warm)

	survivor := warm.cluster.Nodes[2]
	down := []*harness.Node{warm.cluster.Nodes[0], warm.cluster.Nodes[1]}

	phase1OK := t.Run("SoleSurvivorBootstrap", func(t *testing.T) {
		start := time.Now()
		stopMajorityLossMysqld(t, warm, down)
		waitSurvivorRaftLeader(t, warm, survivor)

		t.Log("failover: wait MGR PRIMARY on survivor (read-only until quorum returns)")
		mgrCtx, mgrCancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer mgrCancel()
		_, err := warm.backend.WaitMGRPrimaryOnAny(mgrCtx, []*harness.Node{survivor})
		if !assert.NoError(t, err) {
			for _, na := range warm.neoNodes {
				t.Logf("neoha log %s:\n%s", na.Name, na.TailNeoHALog(12000))
			}
			return
		}
		assert.Equal(t, "PRIMARY", mustRole(t, warm.backend, survivor))

		ro, err := warm.backend.MySQLReadOnly(survivor)
		assert.NoError(t, err)
		assert.True(t, ro, "sole-survivor PRIMARY must stay read_only until 2+ ONLINE")

		cnt, err := warm.backend.OnlineMGRMembers(survivor)
		assert.NoError(t, err)
		assert.Equal(t, 1, cnt)
		t.Logf("timing: sole-survivor bootstrap %s (primary=%s read_only=%v)", time.Since(start), survivor.Name, ro)
	})

	if phase1OK {
		t.Run("RejoinThenWritable", func(t *testing.T) {
			start := time.Now()

			for _, n := range down {
				ports := []int{n.Port}
				if n.GRPort > 0 {
					ports = append(ports, n.GRPort)
				}
				if err := harness.FreePorts(ports); err != nil {
					t.Fatalf("rejoin: free ports for %s: %v", n.Name, err)
				}
				t.Logf("rejoin: start mysqld %s port %d", n.Name, n.Port)
				assert.NoError(t, warm.backend.StartNode(context.Background(), n))
				readyCtx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
				assert.NoError(t, warm.backend.Ready(readyCtx, n))
				cancel()
			}

			rejoinCtx, rejoinCancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer rejoinCancel()
			t.Log("rejoin: wait 2+ MGR ONLINE (Phase 2 may open writes)")
			if err := warm.backend.WaitMGROnlineMembers(rejoinCtx, survivor, 2); err != nil {
				cnt, _ := warm.backend.OnlineMGRMembers(survivor)
				for _, na := range warm.neoNodes {
					t.Logf("neoha log %s:\n%s", na.Name, na.TailNeoHALog(8000))
				}
				assert.Failf(t, "MGR rejoin", "online=%d: %v", cnt, err)
				return
			}
			assert.NoError(t, warm.backend.WaitMySQLWritable(rejoinCtx, survivor))

			ro, err := warm.backend.MySQLReadOnly(survivor)
			assert.NoError(t, err)
			assert.False(t, ro)
			t.Logf("timing: rejoin then writable %s (primary=%s)", time.Since(start), survivor.Name)
		})
	}
}

func stopMajorityLossMysqld(t *testing.T, warm warmNeoHACluster, down []*harness.Node) {
	t.Helper()
	stopCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	for _, n := range down {
		t.Logf("failover: stop mysqld %s port %d", n.Name, n.Port)
		assert.NoError(t, warm.backend.StopNode(stopCtx, n))
	}
	freeCtx, freeCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer freeCancel()
	for _, n := range down {
		assert.NoError(t, harness.WaitPortFree(freeCtx, "127.0.0.1", n.Port))
	}
	t.Log("failover: wait MGR quorum lost on sole survivor")
	survivor := warm.cluster.Nodes[2]
	assert.NoError(t, warm.backend.WaitMGROnlineMembersBelow(warm.ctx, survivor, 2))
}

func waitSurvivorRaftLeader(t *testing.T, warm warmNeoHACluster, survivor *harness.Node) {
	t.Helper()
	survivorEP := harness.EndpointsForClusterNodes(warm.neoNodes, warm.cluster, []*harness.Node{survivor})
	assert.Len(t, survivorEP, 1)

	stepDownCtx, stepDownCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer stepDownCancel()
	t.Log("failover: wait old Raft LEADER (n1/n2 NeoHA) to step down")
	for _, ep := range []string{warm.neoNodes[0].Endpoint, warm.neoNodes[1].Endpoint} {
		assert.NoError(t, harness.WaitRaftNotState(stepDownCtx, ep, "LEADER"))
		assert.NoError(t, harness.WaitRaftState(stepDownCtx, ep, "FOLLOWER"))
	}

	leaderCtx, leaderCancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer leaderCancel()
	t.Log("failover: wait NeoHA Raft LEADER on survivor (GTID max + MGR force bootstrap)")
	leaderEP, err := harness.WaitRaftLeader(leaderCtx, warm.endpoints)
	assert.NoError(t, err)
	assert.Equal(t, warm.neoNodes[2].Endpoint, leaderEP)

	t.Log("failover: wait survivor mysqld ready (NeoHA may restart it during bootstrap)")
	assert.NoError(t, warm.backend.Ready(leaderCtx, survivor))
}

func mustRole(t *testing.T, backend *harness.MySQL80, node *harness.Node) string {
	t.Helper()
	role, err := backend.MGRMemberRole(node)
	assert.NoError(t, err)
	return role
}
