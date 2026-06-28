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

	"github.com/sealdb/neoha/internal/base/nlog"
	"github.com/sealdb/neoha/internal/config"
	"github.com/sealdb/neoha/internal/database/driver"
	"github.com/sealdb/neoha/internal/database/postgresql"
	"github.com/sealdb/neoha/test/integration/harness"
	"github.com/stretchr/testify/require"
)

var pgITPorts = []int{15432, 15433}

func newPGCluster(t *testing.T, name string) (*harness.Cluster, *harness.PostgreSQL, context.Context, context.CancelFunc) {
	t.Helper()
	if err := harness.FreePorts(pgITPorts); err != nil {
		t.Fatalf("port check: %v", err)
	}
	backend := harness.RequirePostgreSQL(t, "")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	cluster := harness.NewCluster(name, harness.WorkDirFromEnv(), backend)
	for i, port := range pgITPorts {
		cluster.AddNode(fmt.Sprintf("node%d", i+1), port, 0)
	}
	_ = os.RemoveAll(cluster.WorkDir)
	start := time.Now()
	t.Log("setup: init pgdata")
	require.NoError(t, cluster.Setup(ctx))
	t.Logf("timing: init pgdata %s", time.Since(start))

	t.Cleanup(func() {
		cancel()
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer stopCancel()
		_ = cluster.Teardown(stopCtx)
	})

	start = time.Now()
	t.Log("setup: start primary + streaming standby")
	require.NoError(t, backend.StartNode(ctx, cluster.Nodes[0]))
	require.NoError(t, backend.Ready(ctx, cluster.Nodes[0]))
	require.NoError(t, backend.BootstrapPrimary(ctx, cluster.Nodes[0]))
	require.NoError(t, backend.CloneStandby(ctx, cluster.Nodes[0], cluster.Nodes[1]))
	require.NoError(t, backend.StartNode(ctx, cluster.Nodes[1]))
	require.NoError(t, backend.Ready(ctx, cluster.Nodes[1]))
	require.NoError(t, backend.WaitInRecovery(ctx, cluster.Nodes[1], true))
	require.NoError(t, backend.WaitReplicationConnected(ctx, cluster.Nodes[0], cluster.Nodes[1]))
	t.Logf("timing: postgres ready %s", time.Since(start))
	return cluster, backend, ctx, cancel
}

func pgDriver(t *testing.T, node *harness.Node, pgBase string) *postgresql.Driver {
	t.Helper()
	conf := config.DefaultPostgresqlConfig()
	conf.ConnectAddress = fmt.Sprintf("127.0.0.1:%d", node.Port)
	conf.DataDir = node.DataDir
	conf.ConfigDir = node.WorkDir
	conf.BinDir = pgBase + "/bin"
	conf.UsePGRewind = true
	conf.UseSlots = false
	conf.Auth.Repl.Username = harness.PGReplUser
	conf.Auth.Repl.Password = harness.PGReplPass
	conf.Auth.SuperUser.Username = harness.PGSuperUser
	conf.Auth.SuperUser.Password = harness.PGSuperPass
	conf.Auth.Rewind.Username = harness.PGSuperUser
	conf.Auth.Rewind.Password = harness.PGSuperPass
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	pg := postgresql.NewPostgresql(conf, 0, log)
	return postgresql.NewDriver(pg, conf, log)
}

// TestPostgreSQLApplyReplicaPgRewind verifies pg_rewind rejoin after timeline fork.
func TestPostgreSQLApplyReplicaPgRewind(t *testing.T) {
	cluster, backend, ctx, _ := newPGCluster(t, "pg-rewind")
	primary := cluster.Nodes[0]
	standby := cluster.Nodes[1]
	pgBase := harness.PGBaseFromEnv()

	_, err := backend.ExecPrimary(ctx, primary, "CREATE TABLE IF NOT EXISTS rewtest(id int); INSERT INTO rewtest VALUES (1)")
	require.NoError(t, err)

	t.Log("promote standby (simulate failover)")
	require.NoError(t, backend.StopNode(ctx, primary))
	require.NoError(t, backend.Promote(ctx, standby))
	require.NoError(t, backend.WaitInRecovery(ctx, standby, false))

	_, err = backend.ExecPrimary(ctx, standby, "INSERT INTO rewtest VALUES (2)")
	require.NoError(t, err)

	drv := pgDriver(t, primary, pgBase)
	applyCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	t.Log("ApplyReplica with pg_rewind to new primary")
	require.NoError(t, drv.ApplyReplica(applyCtx, driver.PrimaryRef{
		Host:     "127.0.0.1",
		Port:     standby.Port,
		MemberID: standby.Name,
	}))
	t.Log("verify old primary rejoined as standby")
	require.NoError(t, backend.WaitInRecovery(ctx, primary, true))
}

// TestPostgreSQL2NodeScaffold boots two PostgreSQL instances with streaming replication.
func TestPostgreSQL2NodeScaffold(t *testing.T) {
	newPGCluster(t, "pg-scaffold")
}
