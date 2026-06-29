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

// Package coordination defines the L2 cluster coordination abstraction.
// See docs/architecture.md.
package coordination

import (
	"context"
)

// Member is a registered cluster member (NeoHA agent).
type Member struct {
	ID       string
	Name     string
	Database string // mysql | postgresql
	Tags     map[string]bool
	Meta     map[string]string
}

// DatabaseEndpoint is the replication target for the current primary mysqld/postgres.
type DatabaseEndpoint struct {
	Host string
	Port int
}

// ClusterView is a snapshot of coordination state.
type ClusterView struct {
	LeaderID         string
	LeaderDatabase   DatabaseEndpoint
	Members          []Member
	Epoch            uint64
	ViewID           uint64
}

// Coordinator abstracts cluster membership and leadership regardless of
// backend (embedded Raft, etcd, …).
type Coordinator interface {
	Start(ctx context.Context) error
	Stop() error

	LocalID() string
	IsLeader() bool
	ClusterView(ctx context.Context) (ClusterView, error)
	Watch(ctx context.Context) (<-chan ClusterView, error)

	AddMember(ctx context.Context, id string) error
	RemoveMember(ctx context.Context, id string) error

	// Dynamic cluster config blob — schema TBD (docs/architecture.md §4.1).
	GetClusterConfig(ctx context.Context) ([]byte, error)
	SetClusterConfig(ctx context.Context, cfg []byte) error
}
