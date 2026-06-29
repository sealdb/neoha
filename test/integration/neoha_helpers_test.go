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

type neoHAConfigWriter func(n *harness.NeoHANode, mysqlBase, defaultsFile, clusterWorkDir, mysqlDataDir string, peers []string) error

func buildNeoHABin(t *testing.T, ctx context.Context) string {
	t.Helper()
	if bin := harness.NeoHABinFromEnv(); bin != "" {
		t.Logf("neoha: using NEOHA_IT_BIN=%s", bin)
		return bin
	}
	repoRoot, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		if _, err := os.Stat(filepath.Join(repoRoot, "go.mod")); err == nil {
			break
		}
		repoRoot = filepath.Dir(repoRoot)
	}
	out := filepath.Join(harness.WorkDirFromEnv(), "bin", "neoha")
	t.Log("neoha: building binary (first run may take ~30s)...")
	assert.NoError(t, harness.BuildNeoHA(ctx, repoRoot, out))
	return out
}

func buildNeoHActl(t *testing.T, ctx context.Context) string {
	t.Helper()
	if bin := harness.NeoHACtlBinFromEnv(); bin != "" {
		t.Logf("neohactl: using NEOHA_IT_CTL_BIN=%s", bin)
		return bin
	}
	repoRoot, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		if _, err := os.Stat(filepath.Join(repoRoot, "go.mod")); err == nil {
			break
		}
		repoRoot = filepath.Dir(repoRoot)
	}
	out := filepath.Join(harness.WorkDirFromEnv(), "bin", "neohactl")
	t.Log("neohactl: building binary...")
	assert.NoError(t, harness.BuildNeoHActl(ctx, repoRoot, out))
	return out
}

type neoHAStartOpts struct {
	// skipCLIWire skips neohactl raft enable + cluster add when peers.json is pre-written.
	skipCLIWire bool
}

func startNeoHACluster(t *testing.T, ctx context.Context, cluster *harness.Cluster, raftPorts []int, writeConfig neoHAConfigWriter, opts ...neoHAStartOpts) ([]*harness.NeoHANode, []string) {
	t.Helper()
	var o neoHAStartOpts
	if len(opts) > 0 {
		o = opts[0]
	}

	if err := harness.EnsurePortsFree(raftPorts); err != nil {
		t.Fatalf("port check: %v", err)
	}

	neohaBin := buildNeoHABin(t, ctx)
	ctlBin := buildNeoHActl(t, ctx)
	_, mysqlBase := requireMySQL80(t)

	endpoints := make([]string, len(raftPorts))
	nodes := make([]*harness.NeoHANode, len(raftPorts))
	for i, p := range raftPorts {
		endpoints[i] = fmt.Sprintf("127.0.0.1:%d", p)
		nodes[i] = harness.NewNeoHANode(fmt.Sprintf("n%d", i+1), cluster.WorkDir, endpoints[i], cluster.Nodes[i].Port)
	}
	for i := range nodes {
		assert.NoError(t, writeConfig(nodes[i], mysqlBase, cluster.Nodes[i].Config, cluster.WorkDir, cluster.Nodes[i].DataDir, endpoints))
	}

	t.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		for _, na := range nodes {
			_ = na.Stop(stopCtx)
		}
	})

	// First node as LEADER matches fresh-cluster bootstrap; others FOLLOWER.
	assert.NoError(t, nodes[0].Start(ctx, neohaBin, "LEADER"))
	for _, na := range nodes[1:] {
		t.Logf("neoha: start %s on %s", na.Name, na.Endpoint)
		assert.NoError(t, na.Start(ctx, neohaBin, "FOLLOWER"))
	}

	t.Log("neoha: wait RPC ready (neohactl neoha ping)")
	if err := harness.WaitNeoHAReadyViaCLI(ctx, ctlBin, nodes); err != nil {
		for _, na := range nodes {
			t.Logf("neoha log %s:\n%s", na.Name, na.TailNeoHALog(8192))
		}
		t.Fatal(err)
	}

	if o.skipCLIWire {
		t.Log("neoha: skip CLI wire (peers.json pre-populated)")
	} else {
		t.Log("neoha: wire cluster via neohactl raft enable + cluster add")
		assert.NoError(t, harness.WireNeoHAClusterViaCLI(ctx, ctlBin, nodes, endpoints))
	}

	return nodes, endpoints
}

func writeMGRConfig(n *harness.NeoHANode, mysqlBase, defaultsFile, clusterWorkDir, mysqlDataDir string, peers []string) error {
	return n.WriteConfig(mysqlBase, defaultsFile, clusterWorkDir, mysqlDataDir, peers)
}

func writeSemiSyncConfig(n *harness.NeoHANode, mysqlBase, defaultsFile, clusterWorkDir, mysqlDataDir string, peers []string) error {
	return n.WriteSemiSyncConfig(mysqlBase, defaultsFile, clusterWorkDir, mysqlDataDir, peers)
}

func neoNodeByEndpoint(neoNodes []*harness.NeoHANode, endpoint string) *harness.NeoHANode {
	for _, na := range neoNodes {
		if na.Endpoint == endpoint {
			return na
		}
	}
	return nil
}

func mysqlNodeForNeoHA(cluster *harness.Cluster, neoNodes []*harness.NeoHANode, endpoint string) *harness.Node {
	na := neoNodeByEndpoint(neoNodes, endpoint)
	if na == nil {
		return nil
	}
	for _, n := range cluster.Nodes {
		if n.Port == na.MySQLPort {
			return n
		}
	}
	return nil
}

func survivors(all []*harness.Node, dead ...*harness.Node) []*harness.Node {
	deadSet := make(map[*harness.Node]struct{}, len(dead))
	for _, n := range dead {
		deadSet[n] = struct{}{}
	}
	out := make([]*harness.Node, 0, len(all))
	for _, n := range all {
		if _, ok := deadSet[n]; !ok {
			out = append(out, n)
		}
	}
	return out
}
