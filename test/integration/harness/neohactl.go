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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	EnvNeoHACtlBin = "NEOHA_IT_CTL_BIN"
)

// NeoHACtlBinFromEnv returns the neohactl binary path or empty.
func NeoHACtlBinFromEnv() string {
	return os.Getenv(EnvNeoHACtlBin)
}

// BuildNeoHActl compiles neohactl to outPath.
func BuildNeoHActl(ctx context.Context, repoRoot, outPath string) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "go", "build", "-o", outPath, "./cmd/neohactl")
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build neohactl: %w: %s", err, string(out))
	}
	return nil
}

// RunNeoHActl runs neohactl in ctlDir (must contain config.path).
func RunNeoHActl(ctx context.Context, ctlBin, ctlDir string, args ...string) error {
	cmd := exec.CommandContext(ctx, ctlBin, args...)
	cmd.Dir = ctlDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("neohactl %s (dir=%s): %w: %s", strings.Join(args, " "), ctlDir, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// WaitNeoHAReadyViaCLI waits until neohactl neoha ping succeeds on every node.
func WaitNeoHAReadyViaCLI(ctx context.Context, ctlBin string, nodes []*NeoHANode) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		ready := 0
		for _, na := range nodes {
			dir := filepath.Dir(na.ConfigPath)
			if err := RunNeoHActl(ctx, ctlBin, dir, "neoha", "ping"); err == nil {
				ready++
			}
		}
		if ready == len(nodes) {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// WireNeoHAClusterViaCLI registers peers using neohactl.
func WireNeoHAClusterViaCLI(ctx context.Context, ctlBin string, nodes []*NeoHANode, endpoints []string) error {
	for _, na := range nodes {
		dir := filepath.Dir(na.ConfigPath)
		if err := RunNeoHActl(ctx, ctlBin, dir, "raft", "enable"); err != nil {
			return fmt.Errorf("%s: %w", na.Name, err)
		}
	}
	if len(endpoints) <= 1 {
		return nil
	}
	leaderDir := filepath.Dir(nodes[0].ConfigPath)
	return RunNeoHActl(ctx, ctlBin, leaderDir, "cluster", "add", strings.Join(endpoints[1:], ","))
}
