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

package etcd

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

const envEtcdBin = "NEOHA_ETCD_BIN"

// etcdBin returns the etcd server binary path, or "" when unavailable.
// Priority: NEOHA_ETCD_BIN → PATH (etcd).
func etcdBin() string {
	if v := os.Getenv(envEtcdBin); v != "" {
		if _, err := os.Stat(v); err == nil {
			return v
		}
	}
	if path, err := exec.LookPath("etcd"); err == nil {
		return path
	}
	return ""
}

func requireEtcdServer(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("etcd server test skipped in -short mode (CI / make test)")
	}
	bin := etcdBin()
	if bin == "" {
		t.Skip("etcd not found: set NEOHA_ETCD_BIN or install etcd in PATH")
	}
	return bin
}

func waitEtcdReady(endpoint string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		cli, err := clientv3.New(clientv3.Config{
			Endpoints:   []string{endpoint},
			DialTimeout: 2 * time.Second,
		})
		if err == nil {
			_, err = cli.Status(ctx, endpoint)
			_ = cli.Close()
			if err == nil {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
	return context.DeadlineExceeded
}
