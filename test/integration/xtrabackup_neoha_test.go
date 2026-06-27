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
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/sealdb/neoha/test/integration/harness"
)

var (
	xtrabackupMySQLPorts = []int{13326, 13327}
	xtrabackupRaftPorts  = []int{18131, 18132}
)

func newXtrabackupSemiSyncCluster(t *testing.T) (*harness.Cluster, *harness.MySQL80, context.Context, context.CancelFunc) {
	t.Helper()
	settings := harness.LoadIntegrationSettings()
	settings.RequireXtrabackup(t)
	settings.RequireSSH(t)
	mysqlBase, _ := settings.RequireMySQL80(t)

	if err := harness.EnsurePortsFree(append(xtrabackupMySQLPorts, xtrabackupRaftPorts...)); err != nil {
		t.Fatalf("port check: %v", err)
	}

	backend := harness.NewMySQL80(mysqlBase)
	backend.SemiSync = true
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	cluster := harness.NewCluster("xtrabackup-rebuild", settings.WorkDir(), backend)
	for i, port := range xtrabackupMySQLPorts {
		cluster.AddNode(fmt.Sprintf("node%d", i+1), port, 0)
	}

	t.Log("setup: init datadirs and my.cnf (semi-sync x2)")
	assert.NoError(t, cluster.Setup(ctx))

	t.Cleanup(func() {
		cancel()
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer stopCancel()
		_ = cluster.Teardown(stopCtx)
	})

	t.Log("setup: start mysqld x2")
	assert.NoError(t, cluster.StartAll(ctx))
	return cluster, backend, ctx, cancel
}

// TestNeoHAXtrabackupRebuildMe rebuilds a semi-sync replica via neohactl mysql rebuildme.
func TestNeoHAXtrabackupRebuildMe(t *testing.T) {
	cluster, backend, ctx, _ := newXtrabackupSemiSyncCluster(t)
	neoNodes, endpoints := startNeoHACluster(t, ctx, cluster, xtrabackupRaftPorts, writeSemiSyncConfig)

	t.Log("rebuildme: wait raft leader and semi-sync topology")
	leaderEP, err := harness.WaitRaftLeader(ctx, endpoints)
	assert.NoError(t, err)

	primary := mysqlNodeForNeoHA(cluster, neoNodes, leaderEP)
	assert.NotNil(t, primary)
	replicas := survivors(cluster.Nodes, primary)
	assert.Len(t, replicas, 1)
	replica := replicas[0]
	replicaNA := neoNodeByEndpoint(neoNodes, endpoints[0])
	for _, na := range neoNodes {
		if na.MySQLPort == replica.Port {
			replicaNA = na
			break
		}
	}
	assert.NotNil(t, replicaNA)
	assert.NotEqual(t, primary.Name, replica.Name)

	replCtx, replCancel := context.WithTimeout(ctx, 90*time.Second)
	defer replCancel()
	assert.NoError(t, backend.WaitReplicaConnectedTo(replCtx, replica, primary.Port))

	assert.NoError(t, harness.ExecSQL(primary.Port, "CREATE DATABASE IF NOT EXISTS neoha_it"))
	assert.NoError(t, harness.ExecSQL(primary.Port, "CREATE TABLE IF NOT EXISTS neoha_it.rebuild_marker (id INT PRIMARY KEY, note VARCHAR(64))"))
	assert.NoError(t, harness.ExecSQL(primary.Port, "INSERT INTO neoha_it.rebuild_marker VALUES (42, 'before-rebuild') ON DUPLICATE KEY UPDATE note='before-rebuild'"))

	syncCtx, syncCancel := context.WithTimeout(ctx, 60*time.Second)
	defer syncCancel()
	for syncCtx.Err() == nil {
		if v, err := harness.QueryScalar(replica.Port, "SELECT COUNT(*) FROM neoha_it.rebuild_marker WHERE id=42"); err == nil && v == 1 {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	got, err := harness.QueryScalar(replica.Port, "SELECT COUNT(*) FROM neoha_it.rebuild_marker WHERE id=42")
	assert.NoError(t, err)
	assert.Equal(t, 1, got)

	ctlBin := buildNeoHActl(t, ctx)
	replicaCtlDir := filepath.Dir(replicaNA.ConfigPath)
	t.Logf("rebuildme: neohactl mysql rebuildme on %s from %s", replicaNA.Endpoint, leaderEP)
	err = harness.RunNeoHActlMysqlRebuildMe(ctx, ctlBin, replicaCtlDir, leaderEP, true)
	if err != nil {
		for _, na := range neoNodes {
			t.Logf("neoha log %s:\n%s", na.Name, na.TailNeoHALog(16384))
		}
		t.Fatal(err)
	}

	afterCtx, afterCancel := context.WithTimeout(ctx, 120*time.Second)
	defer afterCancel()
	assert.NoError(t, backend.WaitReplicaConnectedTo(afterCtx, replica, primary.Port))

	got, err = harness.QueryScalar(replica.Port, "SELECT COUNT(*) FROM neoha_it.rebuild_marker WHERE id=42")
	assert.NoError(t, err)
	assert.Equal(t, 1, got)

	t.Logf("rebuildme: replica %s re-synced from primary %s", replica.Name, primary.Name)
}
