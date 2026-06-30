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
	"time"

	"github.com/sealdb/neoha/internal/neorpc"
)

// WaitNeoHAReady waits until all endpoints accept ServerPingRPC (same RPC as neohactl neoha ping).
func WaitNeoHAReady(ctx context.Context, endpoints []string) error {
	deadline := time.Now().Add(90 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		ready := 0
		for _, ep := range endpoints {
			if _, err := neorpc.ServerPingRPC(ep); err == nil {
				ready++
			}
		}
		if ready == len(endpoints) {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("NeoHA not ready on %v after timeout (check neoha.log under workdir)", endpoints)
}

// GetRaftStates returns the current Raft state per endpoint (for diagnostics).
func GetRaftStates(endpoints []string) map[string]string {
	out := make(map[string]string, len(endpoints))
	for _, ep := range endpoints {
		rsp, err := neorpc.GetRaftStatusRPC(ep)
		if err != nil {
			out[ep] = fmt.Sprintf("rpc_err:%v", err)
			continue
		}
		out[ep] = rsp.State
	}
	return out
}

// WaitRaftNotState waits until endpoint reports a Raft state other than avoid.
func WaitRaftNotState(ctx context.Context, endpoint, avoid string) error {
	lastLog := time.Now()
	for {
		if ctx.Err() != nil {
			return fmt.Errorf("%s still %s (states: %v): %w", endpoint, avoid, GetRaftStates([]string{endpoint}), ctx.Err())
		}
		rsp, err := neorpc.GetRaftStatusRPC(endpoint)
		if err == nil && rsp.State != avoid {
			return nil
		}
		if time.Since(lastLog) >= 15*time.Second {
			state := "unknown"
			if err == nil {
				state = rsp.State
			}
			fmt.Fprintf(os.Stderr, "harness: waiting %s to leave %s (now %s)\n", endpoint, avoid, state)
			lastLog = time.Now()
		}
		time.Sleep(ReadyPollInterval())
	}
}

// WaitRaftState waits until endpoint reports the wanted Raft state.
func WaitRaftState(ctx context.Context, endpoint, want string) error {
	lastLog := time.Now()
	for {
		if ctx.Err() != nil {
			return fmt.Errorf("%s not %s (states: %v): %w", endpoint, want, GetRaftStates([]string{endpoint}), ctx.Err())
		}
		rsp, err := neorpc.GetRaftStatusRPC(endpoint)
		if err == nil && rsp.State == want {
			return nil
		}
		if time.Since(lastLog) >= 15*time.Second {
			state := "unknown"
			if err == nil {
				state = rsp.State
			}
			fmt.Fprintf(os.Stderr, "harness: waiting %s to become %s (now %s)\n", endpoint, want, state)
			lastLog = time.Now()
		}
		time.Sleep(ReadyPollInterval())
	}
}

// WaitRaftLeader polls RaftStatusRPC until one endpoint reports LEADER.
func WaitRaftLeader(ctx context.Context, endpoints []string) (string, error) {
	deadline := time.Now().Add(90 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	lastLog := time.Now()
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		for _, ep := range endpoints {
			rsp, err := neorpc.GetRaftStatusRPC(ep)
			if err == nil && rsp.State == "LEADER" {
				return ep, nil
			}
		}
		if time.Since(lastLog) >= 15*time.Second {
			fmt.Fprintf(os.Stderr, "harness: waiting for Raft LEADER on %v (states: %v)\n", endpoints, GetRaftStates(endpoints))
			lastLog = time.Now()
		}
		time.Sleep(ReadyPollInterval())
	}
	return "", fmt.Errorf("no Raft LEADER on %v before timeout (states: %v)", endpoints, GetRaftStates(endpoints))
}
