//go:build integration

/*
 * Copyright 2022-2026 The NeoHA Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
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

	"github.com/sealdb/neoha/test/integration/harness"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	pgEtcdNeoHAPorts = []int{18081, 18082}
)

func startNeoHAPGEtcdCluster(t *testing.T, ctx context.Context, cluster *harness.Cluster, etcdEndpoint string) []*harness.NeoHANode {
	t.Helper()
	if err := harness.EnsurePortsFree(pgEtcdNeoHAPorts); err != nil {
		t.Fatalf("port check: %v", err)
	}
	neohaBin := buildNeoHABin(t, ctx)
	pgBase := harness.PGBaseFromEnv()
	nodes := make([]*harness.NeoHANode, len(cluster.Nodes))
	for i, p := range pgEtcdNeoHAPorts {
		nodes[i] = harness.NewNeoHANode(cluster.Nodes[i].Name, cluster.WorkDir, fmt.Sprintf("127.0.0.1:%d", p), cluster.Nodes[i].Port)
		require.NoError(t, nodes[i].WriteEtcdPGConfig(pgBase, cluster.Nodes[i].DataDir, etcdEndpoint, cluster.Nodes[i].Port))
	}
	t.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		for _, na := range nodes {
			_ = na.Stop(stopCtx)
		}
	})
	for _, na := range nodes {
		assert.NoError(t, na.Start(ctx, neohaBin, "FOLLOWER"))
	}
	ctlBin := buildNeoHActl(t, ctx)
	require.NoError(t, harness.WaitNeoHAReadyViaCLI(ctx, ctlBin, nodes))
	return nodes
}

func waitPGPrimaryCount(t *testing.T, ctx context.Context, backend *harness.PostgreSQL, cluster *harness.Cluster, want int) *harness.Node {
	t.Helper()
	for {
		if ctx.Err() != nil {
			t.Fatal(ctx.Err())
		}
		var primary *harness.Node
		count := 0
		for _, node := range cluster.Nodes {
			db, err := backend.OpenAdmin(ctx, node)
			if err != nil {
				continue
			}
			var recovery bool
			if err := db.QueryRowContext(ctx, "SELECT pg_is_in_recovery()").Scan(&recovery); err == nil && !recovery {
				count++
				primary = node
			}
			db.Close()
		}
		if count == want {
			return primary
		}
		time.Sleep(harness.ReadyPollInterval())
	}
}

// TestNeoHAPGEtcdFailoverMinority stops primary PG + NeoHA; survivor promotes via etcd + reconciler.
func TestNeoHAPGEtcdFailoverMinority(t *testing.T) {
	etcdEP, stopEtcd := harness.StartTestEtcd(t)
	defer stopEtcd()

	cluster, backend, ctx, _ := newPGCluster(t, "pg-etcd-fail")
	start := time.Now()
	t.Log("neoha: start agents (etcd DCS)")
	neoNodes := startNeoHAPGEtcdCluster(t, ctx, cluster, etcdEP)
	t.Logf("timing: neoha ready %s", time.Since(start))

	waitCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	start = time.Now()
	primary := waitPGPrimaryCount(t, waitCtx, backend, cluster, 1)
	require.NotNil(t, primary)
	t.Logf("timing: single PG primary %s (node=%s port=%d)", time.Since(start), primary.Name, primary.Port)

	var survivor *harness.Node
	for _, n := range cluster.Nodes {
		if n.Name != primary.Name {
			survivor = n
			break
		}
	}
	require.NotNil(t, survivor)

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer stopCancel()
	for _, na := range neoNodes {
		if na.MySQLPort == primary.Port {
			t.Logf("failover: stop neoha %s", na.Name)
			require.NoError(t, na.Stop(stopCtx))
		}
	}
	t.Logf("failover: stop postgres %s", primary.Name)
	require.NoError(t, backend.StopNode(stopCtx, primary))

	failCtx, failCancel := context.WithTimeout(ctx, 90*time.Second)
	defer failCancel()
	start = time.Now()
	newPrimary := waitPGPrimaryCount(t, failCtx, backend, cluster, 1)
	require.Equal(t, survivor.Name, newPrimary.Name)
	t.Logf("timing: failover promote %s (new primary=%s)", time.Since(start), newPrimary.Name)
}
