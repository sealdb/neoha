//go:build integration

/*
 * Copyright 2022-2026 The NeoHA Authors.
 *
 * See the AUTHORS file for a list of contributors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/sealdb/neoha/test/integration/harness"
)

const (
	warmMGRClusterName      = "neoha-mgr-warm"
	warmMGRMajorClusterName = "neoha-mgr-major-warm"
	warmSemiSyncClusterName = "neoha-semisync-warm"
)

type warmMySQLCluster struct {
	cluster *harness.Cluster
	backend *harness.MySQL80
	ctx     context.Context
	cancel  context.CancelFunc
}

type warmNeoHACluster struct {
	warmMySQLCluster
	neoNodes  []*harness.NeoHANode
	endpoints []string
}

func newWarmMySQLCluster(t *testing.T, name string, mysqlPorts []int, grPorts []int, semiSync bool, resetDatadir bool) warmMySQLCluster {
	t.Helper()
	allPorts := append(append([]int{}, mysqlPorts...), grPorts...)
	if resetDatadir {
		if err := harness.FreePorts(allPorts); err != nil {
			t.Fatalf("free ports before reset: %v", err)
		}
	} else if err := harness.EnsurePortsFree(mysqlPorts); err != nil {
		t.Fatalf("port check: %v", err)
	}

	backend, _ := requireMySQL80(t)
	if semiSync {
		backend.SemiSync = true
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	cluster := harness.NewCluster(name, harness.WorkDirFromEnv(), backend)
	for i, port := range mysqlPorts {
		grPort := 0
		if grPorts != nil {
			grPort = grPorts[i]
		}
		cluster.AddNode(fmt.Sprintf("node%d", i+1), port, grPort)
	}

	start := time.Now()
	if resetDatadir {
		t.Log("warm: reset datadirs and my.cnf (force clean)")
		assert.NoError(t, cluster.SetupReset(ctx))
	} else {
		t.Log("warm: init datadirs and my.cnf (reuse when ready)")
		assert.NoError(t, cluster.Setup(ctx))
	}
	t.Logf("timing: init datadirs %s", time.Since(start))

	t.Cleanup(func() {
		cancel()
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer stopCancel()
		_ = cluster.Teardown(stopCtx)
	})

	start = time.Now()
	t.Log("warm: start mysqld x3")
	assert.NoError(t, cluster.StartAll(ctx))
	t.Logf("timing: mysqld ready %s", time.Since(start))

	return warmMySQLCluster{cluster: cluster, backend: backend, ctx: ctx, cancel: cancel}
}

func bootstrapWarmNeoHA(
	t *testing.T,
	warm warmMySQLCluster,
	raftPorts []int,
	writeConfig neoHAConfigWriter,
	opts ...neoHAStartOpts,
) warmNeoHACluster {
	t.Helper()
	var o neoHAStartOpts
	if len(opts) > 0 {
		o = opts[0]
	}

	start := time.Now()
	neoNodes, endpoints := startNeoHACluster(t, warm.ctx, warm.cluster, raftPorts, writeConfig, o)
	t.Logf("timing: neoha ready %s", time.Since(start))

	return warmNeoHACluster{
		warmMySQLCluster: warm,
		neoNodes:         neoNodes,
		endpoints:        endpoints,
	}
}

func waitMGRFormation(t *testing.T, warm warmNeoHACluster) {
	t.Helper()
	start := time.Now()
	leaderEP, err := harness.WaitRaftLeader(warm.ctx, warm.endpoints)
	if err != nil {
		t.Fatalf("warm: raft leader: %v", err)
	}
	primary := mysqlNodeForNeoHA(warm.cluster, warm.neoNodes, leaderEP)
	if primary == nil {
		t.Fatalf("warm: no mysql node for raft leader %s", leaderEP)
	}
	t.Logf("warm: raft leader %s (mysql %s)", leaderEP, primary.Name)

	t.Log("warm: wait MGR 3 ONLINE")
	formCtx, cancel := context.WithTimeout(warm.ctx, 5*time.Minute)
	defer cancel()
	if err := warm.backend.WaitMGROnlineMembers(formCtx, primary, 3); err != nil {
		cnt, _ := warm.backend.OnlineMGRMembers(primary)
		for _, na := range warm.neoNodes {
			t.Logf("neoha log %s:\n%s", na.Name, na.TailNeoHALog(8000))
		}
		t.Fatalf("warm: MGR 3 online (have %d): %v", cnt, err)
	}
	t.Logf("timing: MGR 3 online %s", time.Since(start))
}

func waitSemiSyncFormation(t *testing.T, warm warmNeoHACluster) (*harness.Node, []*harness.Node) {
	t.Helper()
	start := time.Now()
	t.Log("warm: wait raft leader + semi-sync replication")
	leaderEP, err := harness.WaitRaftLeader(warm.ctx, warm.endpoints)
	assert.NoError(t, err)
	assert.NotEmpty(t, leaderEP)

	primary := mysqlNodeForNeoHA(warm.cluster, warm.neoNodes, leaderEP)
	assert.NotNil(t, primary)
	assert.NoError(t, warm.backend.WaitMySQLWritable(warm.ctx, primary))

	replicas := survivors(warm.cluster.Nodes, primary)
	assert.Len(t, replicas, 2)
	replCtx, replCancel := context.WithTimeout(warm.ctx, 90*time.Second)
	defer replCancel()
	for _, rep := range replicas {
		assert.NoError(t, warm.backend.WaitReplicaConnectedTo(replCtx, rep, primary.Port))
	}
	t.Logf("timing: semi-sync formation %s (primary=%s)", time.Since(start), primary.Name)
	return primary, replicas
}
