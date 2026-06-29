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
		newWarmMySQLCluster(t, warmSemiSyncClusterName, semiSyncMySQLPorts, nil, true),
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

		leaderCtx, leaderCancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer leaderCancel()
		t.Log("failover: wait NeoHA Raft LEADER on survivors")
		newLeaderEP, err := harness.WaitRaftLeader(leaderCtx, remainEP)
		assert.NoError(t, err)

		newPrimary := mysqlNodeForNeoHA(warm.cluster, warm.neoNodes, newLeaderEP)
		assert.NotNil(t, newPrimary)
		assert.NotEqual(t, primary.Name, newPrimary.Name)
		assert.NoError(t, warm.backend.WaitMySQLWritable(leaderCtx, newPrimary))

		other := survivors(remain, newPrimary)
		assert.Len(t, other, 1)
		assert.NoError(t, warm.backend.WaitReplicaConnectedTo(leaderCtx, other[0], newPrimary.Port))
		t.Logf("timing: failover segment %s (old=%s new=%s)", time.Since(start), primary.Name, newPrimary.Name)
	})
}
