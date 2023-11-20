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

// WaitRaftLeader polls RaftStatusRPC until one endpoint reports LEADER.
func WaitRaftLeader(ctx context.Context, endpoints []string) (string, error) {
	deadline := time.Now().Add(90 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
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
		time.Sleep(500 * time.Millisecond)
	}
	return "", fmt.Errorf("no Raft LEADER on %v before timeout", endpoints)
}
