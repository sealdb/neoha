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

package harness

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// FormatGRSeeds builds a group_replication_group_seeds value from GR ports.
func FormatGRSeeds(ports []int) string {
	parts := make([]string, len(ports))
	for i, p := range ports {
		parts[i] = fmt.Sprintf("127.0.0.1:%d", p)
	}
	return strings.Join(parts, ",")
}
func WaitPortFree(ctx context.Context, host string, port int) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if !portInUse(host, port) {
			return nil
		}
		time.Sleep(readyPollInterval)
	}
}

// portInUse reports whether something is listening on host:port.
func portInUse(host string, port int) bool {
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// EnsurePortsFree fails fast when MySQL/RPC ports are still held by a stale run.
func EnsurePortsFree(ports []int) error {
	var busy []string
	for _, p := range ports {
		if portInUse("127.0.0.1", p) {
			busy = append(busy, fmt.Sprintf("%d", p))
		}
	}
	if len(busy) == 0 {
		return nil
	}
	return fmt.Errorf("ports already in use (stale mysqld/neoha from a prior run?): %s — run: pkill -f /tmp/neoha-it/ || true", strings.Join(busy, ", "))
}

// KillProcessesOnPorts sends SIGKILL to processes listening on the given TCP ports.
func KillProcessesOnPorts(ports []int) {
	for _, p := range ports {
		// fuser exits non-zero when nothing is bound; ignore errors.
		cmd := exec.Command("fuser", "-k", fmt.Sprintf("%d/tcp", p))
		_ = cmd.Run()
	}
	if len(ports) > 0 {
		time.Sleep(300 * time.Millisecond)
	}
}

// FreePorts kills stale listeners on ports and waits until they are available.
func FreePorts(ports []int) error {
	KillProcessesOnPorts(ports)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		allFree := true
		for _, p := range ports {
			if portInUse("127.0.0.1", p) {
				allFree = false
				break
			}
		}
		if allFree {
			return nil
		}
		time.Sleep(150 * time.Millisecond)
	}
	return EnsurePortsFree(ports)
}

// KillProcessesOnWorkDir sends SIGTERM to processes whose cmdline references workDir.
func KillProcessesOnWorkDir(workDir string) {
	if workDir == "" {
		return
	}
	out, err := exec.Command("pgrep", "-f", workDir).CombinedOutput()
	if err != nil {
		return
	}
	self := os.Getpid()
	for _, line := range strings.Fields(string(out)) {
		pid := 0
		if _, err := fmt.Sscanf(line, "%d", &pid); err != nil || pid <= 0 || pid == self {
			continue
		}
		_ = syscall.Kill(pid, syscall.SIGTERM)
	}
	time.Sleep(500 * time.Millisecond)
}
