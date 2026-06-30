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
	semiSyncMySQLPorts = []int{13316, 13317, 13318}
	semiSyncRaftPorts1 = []int{18111, 18112, 18113}
)

// TestNeoHASemiSyncWarmSuite boots one warm 3-node semi-sync cluster and reuses it for
// formation verification and minority failover.
func TestNeoHASemiSyncWarmSuite(t *testing.T) {
	warm := bootstrapWarmNeoHA(t,
		newWarmMySQLCluster(t, warmSemiSyncClusterName, semiSyncMySQLPorts, nil, true, false),
		semiSyncRaftPorts1,
		writeSemiSyncConfig,
		neoHAStartOpts{skipCLIWire: true},
	)
	primary, replicas := waitSemiSyncFormation(t, warm)
	_ = replicas

	t.Run("3NodeSemiSync", func(t *testing.T) {
		start := time.Now()
		for _, rep := range survivors(warm.cluster.Nodes, primary) {
			st, err := warm.backend.ReplicaStatus(rep)
			assert.NoError(t, err)
			assert.Equal(t, primary.Port, st.MasterPort)
			assert.Equal(t, "Yes", st.SlaveIORunning)
			assert.Equal(t, "Yes", st.SlaveSQLRunning)
		}
		t.Logf("timing: formation verify %s (primary=%s)", time.Since(start), primary.Name)
	})

	t.Run("FailoverMinority", func(t *testing.T) {
		start := time.Now()

		stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		t.Logf("failover: stop primary mysqld (%s port %d)", primary.Name, primary.Port)
		assert.NoError(t, warm.backend.StopNode(stopCtx, primary))

		remain := survivors(warm.cluster.Nodes, primary)
		remainEP := harness.EndpointsForClusterNodes(warm.neoNodes, warm.cluster, remain)
		assert.Len(t, remainEP, 2)
		oldLeaderEP := harness.EndpointsForClusterNodes(warm.neoNodes, warm.cluster, []*harness.Node{primary})
		if !assert.Len(t, oldLeaderEP, 1) {
			return
		}

		failoverCtx, failoverCancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer failoverCancel()
		t.Log("failover: wait old Raft LEADER to step down")
		if !assert.NoError(t, harness.WaitRaftNotState(failoverCtx, oldLeaderEP[0], "LEADER")) {
			return
		}

		t.Log("failover: wait NeoHA Raft LEADER on survivors")
		newLeaderEP, err := harness.WaitRaftLeader(failoverCtx, remainEP)
		if !assert.NoError(t, err) {
			for _, na := range warm.neoNodes {
				t.Logf("neoha log %s:\n%s", na.Name, na.TailNeoHALog(8000))
			}
			return
		}

		newPrimary := mysqlNodeForNeoHA(warm.cluster, warm.neoNodes, newLeaderEP)
		if !assert.NotNil(t, newPrimary) {
			return
		}
		assert.NotEqual(t, primary.Name, newPrimary.Name)
		assert.NoError(t, warm.backend.WaitMySQLWritable(failoverCtx, newPrimary))

		other := survivors(remain, newPrimary)
		if !assert.Len(t, other, 1) {
			return
		}
		assert.NoError(t, warm.backend.WaitReplicaConnectedTo(failoverCtx, other[0], newPrimary.Port))
		t.Logf("timing: failover segment %s (old=%s new=%s)", time.Since(start), primary.Name, newPrimary.Name)
	})
}
