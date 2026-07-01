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
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/sealdb/neoha/test/integration/harness"
)

// TestITPrepWarmDatadirs initializes MySQL datadirs for warm integration clusters
// without starting NeoHA. Run via: make test-integration-prep
func TestITPrepWarmDatadirs(t *testing.T) {
	if os.Getenv("NEOHA_IT_PREP") == "" {
		t.Skip("set NEOHA_IT_PREP=1 (use make test-integration-prep)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	prepCluster := func(name string, ports []int, grPorts []int, semiSync bool) {
		t.Helper()
		backend, _ := requireMySQL80(t)
		if semiSync {
			backend.SemiSync = true
		}
		cluster := harness.NewCluster(name, harness.WorkDirFromEnv(), backend)
		for i, port := range ports {
			grPort := 0
			if grPorts != nil {
				grPort = grPorts[i]
			}
			cluster.AddNode(fmt.Sprintf("node%d", i+1), port, grPort)
		}

		start := time.Now()
		assert.NoError(t, cluster.Setup(ctx))
		t.Logf("test-integration-prep: %s datadirs ready in %s (%s)", name, time.Since(start), cluster.WorkDir)

		stopCtx, stopCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer stopCancel()
		assert.NoError(t, cluster.StopAndMaybeTeardown(stopCtx, true))
	}

	t.Run("MGRWarm", func(t *testing.T) {
		prepCluster(warmMGRClusterName, mgrMySQLPorts, mgrGRPorts, false)
	})
	t.Run("SemiSyncWarm", func(t *testing.T) {
		prepCluster(warmSemiSyncClusterName, semiSyncMySQLPorts, nil, true)
	})
}
