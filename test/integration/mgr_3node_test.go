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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/sealdb/neoha/test/integration/harness"
)

var (
	mgrMySQLPorts = []int{13306, 13307, 13308}
	mgrGRPorts    = []int{13361, 13362, 13363}
)

func requireMySQL80(t *testing.T) (*harness.MySQL80, string) {
	t.Helper()
	settings := harness.LoadIntegrationSettings()
	base, _ := settings.RequireMySQL80(t)
	return harness.NewMySQL80(base), base
}

func newMySQLCluster(t *testing.T, name string) (*harness.Cluster, *harness.MySQL80, context.Context, context.CancelFunc) {
	t.Helper()
	if err := harness.EnsurePortsFree(mgrMySQLPorts); err != nil {
		t.Fatalf("port check: %v", err)
	}

	backend, _ := requireMySQL80(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	cluster := harness.NewCluster(name, harness.WorkDirFromEnv(), backend)
	for i, port := range mgrMySQLPorts {
		cluster.AddNode(fmt.Sprintf("node%d", i+1), port, mgrGRPorts[i])
	}

	t.Log("setup: init datadirs and my.cnf")
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

// TestMySQL3NodeScaffold boots three MySQL 8.0 instances with the MGR plugin loaded.
func TestMySQL3NodeScaffold(t *testing.T) {
	newMySQLCluster(t, "scaffold")
}
