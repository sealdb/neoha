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
	"fmt"
	"net"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

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

// KillProcessesOnWorkDir sends SIGTERM to processes whose cmdline references workDir.
func KillProcessesOnWorkDir(workDir string) {
	if workDir == "" {
		return
	}
	out, err := exec.Command("pgrep", "-f", workDir).CombinedOutput()
	if err != nil {
		return
	}
	for _, line := range strings.Fields(string(out)) {
		pid := 0
		if _, err := fmt.Sscanf(line, "%d", &pid); err != nil || pid <= 0 {
			continue
		}
		_ = syscall.Kill(pid, syscall.SIGTERM)
	}
	time.Sleep(500 * time.Millisecond)
}
