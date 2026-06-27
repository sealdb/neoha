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
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/sealdb/neoha/test/integration/harness"
)

var (
	semiSyncMySQLPorts = []int{13316, 13317, 13318}
	semiSyncRaftPorts1 = []int{18111, 18112, 18113}
	semiSyncRaftPorts2 = []int{18121, 18122, 18123}
)

func newSemiSyncMySQLCluster(t *testing.T, name string) (*harness.Cluster, *harness.MySQL80, context.Context, context.CancelFunc) {
	t.Helper()
	if err := harness.EnsurePortsFree(semiSyncMySQLPorts); err != nil {
		t.Fatalf("port check: %v", err)
	}

	base := harness.MySQLBaseFromEnv()
	if base == "" {
		t.Skip("set NEOHA_IT_MYSQL_BASE to the MySQL 8.0 build root (contains bin/mysqld)")
	}
	if _, err := os.Stat(filepath.Join(base, "bin", "mysqld")); err != nil {
		t.Skipf("mysqld not found under %s/bin: %v", base, err)
	}

	backend := harness.NewMySQL80(base)
	backend.SemiSync = true
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	cluster := harness.NewCluster(name, harness.WorkDirFromEnv(), backend)
	for i, port := range semiSyncMySQLPorts {
		cluster.AddNode(fmt.Sprintf("node%d", i+1), port, 0)
	}

	t.Log("setup: init datadirs and my.cnf (semi-sync)")
	assert.NoError(t, cluster.Setup(ctx))

	t.Cleanup(func() {
		cancel()
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer stopCancel()
		_ = cluster.Teardown(stopCtx)
	})

	t.Log("setup: start mysqld x3")
	assert.NoError(t, cluster.StartAll(ctx))
	return cluster, backend, ctx, cancel
}

// TestNeoHA3NodeSemiSync forms semi-sync replication via NeoHA setupMysql + Raft leader (production path).
func TestNeoHA3NodeSemiSync(t *testing.T) {
	cluster, backend, ctx, _ := newSemiSyncMySQLCluster(t, "neoha-semisync3")
	neoNodes, endpoints := startNeoHACluster(t, ctx, cluster, semiSyncRaftPorts1, writeSemiSyncConfig)

	t.Log("neoha: wait raft leader")
	leaderEP, err := harness.WaitRaftLeader(ctx, endpoints)
	assert.NoError(t, err)
	assert.NotEmpty(t, leaderEP)

	primary := mysqlNodeForNeoHA(cluster, neoNodes, leaderEP)
	assert.NotNil(t, primary)
	assert.NoError(t, backend.WaitMySQLWritable(ctx, primary))

	replicas := survivors(cluster.Nodes, primary)
	assert.Len(t, replicas, 2)
	replCtx, replCancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer replCancel()
	for _, rep := range replicas {
		assert.NoError(t, backend.WaitReplicaConnectedTo(replCtx, rep, primary.Port))
	}
	t.Logf("semi-sync: primary=%s port=%d replicas=%d", primary.Name, primary.Port, len(replicas))
}

// TestNeoHASemiSyncFailoverMinority: primary mysqld down, 2 survivors — Raft elects new leader and re-wires replication.
func TestNeoHASemiSyncFailoverMinority(t *testing.T) {
	cluster, backend, ctx, _ := newSemiSyncMySQLCluster(t, "neoha-semisync-fail-min")
	neoNodes, endpoints := startNeoHACluster(t, ctx, cluster, semiSyncRaftPorts2, writeSemiSyncConfig)

	leaderEP, err := harness.WaitRaftLeader(ctx, endpoints)
	assert.NoError(t, err)
	primary := mysqlNodeForNeoHA(cluster, neoNodes, leaderEP)
	assert.NotNil(t, primary)
	assert.NoError(t, backend.WaitMySQLWritable(ctx, primary))
	replCtx, replCancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer replCancel()
	for _, rep := range survivors(cluster.Nodes, primary) {
		assert.NoError(t, backend.WaitReplicaConnectedTo(replCtx, rep, primary.Port))
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	t.Logf("failover: stop primary mysqld (%s port %d)", primary.Name, primary.Port)
	assert.NoError(t, backend.StopNode(stopCtx, primary))

	remain := survivors(cluster.Nodes, primary)
	remainEP := harness.EndpointsForClusterNodes(neoNodes, cluster, remain)
	assert.Len(t, remainEP, 2)

	leaderCtx, leaderCancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer leaderCancel()
	t.Log("failover: wait NeoHA Raft LEADER on survivors (semi-sync IT: ~10s after primary fault)")
	newLeaderEP, err := harness.WaitRaftLeader(leaderCtx, remainEP)
	assert.NoError(t, err)

	newPrimary := mysqlNodeForNeoHA(cluster, neoNodes, newLeaderEP)
	assert.NotNil(t, newPrimary)
	assert.NotEqual(t, primary.Name, newPrimary.Name)
	assert.NoError(t, backend.WaitMySQLWritable(leaderCtx, newPrimary))

	other := survivors(remain, newPrimary)
	assert.Len(t, other, 1)
	assert.NoError(t, backend.WaitReplicaConnectedTo(leaderCtx, other[0], newPrimary.Port))
	t.Logf("semi-sync failover: old primary=%s new primary=%s", primary.Name, newPrimary.Name)
}
