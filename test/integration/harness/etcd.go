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

package harness

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/sealdb/neoha/internal/config"
	etcdcoord "github.com/sealdb/neoha/internal/coordination/etcd"
)

// StartTestEtcd launches a single-node etcd for integration tests.
func StartTestEtcd(t *testing.T) (endpoint string, stop func()) {
	t.Helper()
	bin := EtcdBin()
	if _, err := os.Stat(bin); err != nil {
		t.Skipf("etcd binary not found at %s (set NEOHA_ETCD_BIN)", bin)
	}
	dir, err := os.MkdirTemp("", "neoha-it-etcd-*")
	if err != nil {
		t.Fatalf("etcd temp dir: %v", err)
	}
	endpoint = "127.0.0.1:22379"
	cmd := exec.Command(bin,
		"--name", "neoha-it",
		"--data-dir", dir,
		"--listen-client-urls", "http://"+endpoint,
		"--advertise-client-urls", "http://"+endpoint,
		"--listen-peer-urls", "http://127.0.0.1:22380",
	)
	if err := cmd.Start(); err != nil {
		_ = os.RemoveAll(dir)
		t.Fatalf("start etcd: %v", err)
	}
	stop = func() {
		_ = cmd.Process.Signal(os.Interrupt)
		_, _ = cmd.Process.Wait()
		_ = os.RemoveAll(dir)
	}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		c := etcdcoord.New(probeEtcdConfig(endpoint))
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

// EtcdBin returns the etcd binary path for integration tests.
func EtcdBin() string {
	if v := os.Getenv("NEOHA_ETCD_BIN"); v != "" {
		return v
	}
	return "/home/wslu/work/github/db/etcd/etcd/bin/etcd"
}

func probeEtcdConfig(endpoint string) *config.Config {
	conf := config.DefaultConfig()
	conf.Coordination.Provider = "etcd"
	conf.Coordination.Etcd.Host = endpoint
	conf.Coordination.Etcd.TTL = 5
	return conf
}
