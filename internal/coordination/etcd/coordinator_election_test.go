/*
 * Copyright 2022-2026 The NeoHA Authors.
 *
 * See the AUTHORS file for a list of contributors.
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

package etcd

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/sealdb/neoha/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func etcdBin() string {
	if v := os.Getenv("NEOHA_ETCD_BIN"); v != "" {
		return v
	}
	return "/home/wslu/work/github/db/etcd/etcd/bin/etcd"
}

func startTestEtcd(t *testing.T) (endpoint string, stop func()) {
	t.Helper()
	bin := etcdBin()
	if _, err := os.Stat(bin); err != nil {
		t.Skipf("etcd binary not found at %s", bin)
	}
	dir, err := os.MkdirTemp("", "neoha-etcd-*")
	require.NoError(t, err)
	endpoint = "127.0.0.1:22379"
	cmd := exec.Command(bin,
		"--name", "neoha-test",
		"--data-dir", dir,
		"--listen-client-urls", "http://"+endpoint,
		"--advertise-client-urls", "http://"+endpoint,
		"--listen-peer-urls", "http://127.0.0.1:22380",
	)
	require.NoError(t, cmd.Start())
	stop = func() {
		_ = cmd.Process.Signal(os.Interrupt)
		_, _ = cmd.Process.Wait()
		_ = os.RemoveAll(dir)
	}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		c := New(testConfig(endpoint, "n1"))
		if err := c.Start(context.Background()); err == nil {
			_ = c.Stop()
			return endpoint, stop
		}
		time.Sleep(200 * time.Millisecond)
	}
	stop()
	t.Fatal("etcd did not become ready")
	return "", nil
}

func testConfig(endpoint, name string) *config.Config {
	conf := config.DefaultConfig()
	conf.Scope = "neoha-test"
	conf.NameSpace = "/service/"
	conf.Name = name
	switch name {
	case "n1":
		conf.Endpoint = "127.0.0.1:8081"
	case "n2":
		conf.Endpoint = "127.0.0.1:8082"
	default:
		conf.Endpoint = "127.0.0.1:8080"
	}
	conf.Coordination.Provider = "etcd"
	conf.Coordination.Etcd.Host = endpoint
	conf.Coordination.Etcd.TTL = 5
	conf.Database.Type = "postgresql"
	conf.Database.Postgresql.ConnectAddress = "127.0.0.1:5432"
	return conf
}

func TestEtcdCoordinatorLeaderElection(t *testing.T) {
	endpoint, stop := startTestEtcd(t)
	defer stop()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c1 := New(testConfig(endpoint, "n1"))
	c2 := New(testConfig(endpoint, "n2"))
	require.NoError(t, c1.Start(ctx))
	defer func() { _ = c1.Stop() }()
	require.NoError(t, c2.Start(ctx))
	defer func() { _ = c2.Stop() }()

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		l1 := c1.IsLeader()
		l2 := c2.IsLeader()
		if l1 != l2 {
			view, err := c1.ClusterView(ctx)
			assert.NoError(t, err)
			assert.NotEmpty(t, view.LeaderID)
			assert.NotEmpty(t, view.LeaderDatabase.Host)
			return
		}
		time.Sleep(300 * time.Millisecond)
	}
	t.Fatal("expected exactly one etcd leader")
}

func TestClusterPrefix(t *testing.T) {
	conf := config.DefaultConfig()
	conf.NameSpace = "/service/"
	conf.Scope = "batman"
	assert.Equal(t, "/service/batman/leader", leaderKey(conf))
	assert.Equal(t, "/service/batman/members/n1", memberKey(conf, "n1"))
}
