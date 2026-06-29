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

// Package driver defines the L4 database abstraction for HA orchestration.
// See docs/architecture.md.
package driver

import (
	"context"
	"errors"
)

// ErrNotImplemented is returned by stub driver methods.
var ErrNotImplemented = errors.New("database driver: not implemented")

// Engine identifies which database engine a driver serves.
type Engine int

const (
	EngineMySQL Engine = 1 << iota
	EnginePostgreSQL
	EngineUnknown
)

// Role is the desired or observed replication role from the HA layer's view.
type Role int

const (
	RoleUnknown Role = iota
	RolePrimary
	RoleReplica
	RoleReadOnly
	RoleInvalid
)

// PrimaryRef identifies the cluster primary for replica setup.
type PrimaryRef struct {
	MemberID string
	Host     string
	Port     int
}

// Health is a coarse health snapshot for status RPC / neohactl.
type Health struct {
	Alive     bool
	Writable  bool
	Role      Role
	LastError string
	Details   any
}

// CandidacyEvaluator answers whether this node may become primary.
type CandidacyEvaluator interface {
	Promotable(ctx context.Context) (ok bool, reason string)
	ReplicationLagBytes(ctx context.Context) (int64, error)
}

// RoleApplier executes role transitions on the database instance.
type RoleApplier interface {
	ApplyPrimary(ctx context.Context) error
	ApplyReplica(ctx context.Context, primary PrimaryRef) error
	Demote(ctx context.Context) error
}

// Lifecycle covers setup and background health probes.
type Lifecycle interface {
	Start()
	Stop()
	SetupBootstrap(ctx context.Context) error
}

// StatusReporter exposes read-only status for API/CLI.
type StatusReporter interface {
	Status(ctx context.Context) (Health, error)
}

// Driver is the unified database facade for HA orchestration.
type Driver interface {
	Type() Engine
	Lifecycle
	CandidacyEvaluator
	RoleApplier
	StatusReporter
}

// MGRLifecycle covers MySQL Group Replication two-phase primary apply.
// Implemented by internal/database/mysql.Driver when repl_mode is MGR.
type MGRLifecycle interface {
	IsMGRMode() bool
	MGRPrimaryPhase1Done(ctx context.Context) (bool, error)
	MGRClusterWritableReady(ctx context.Context) (bool, error)
	ApplyPrimaryMGRPhase2(ctx context.Context) error
	// MGRReplicaJoined reports whether local mysqld is a live MGR secondary.
	MGRReplicaJoined(ctx context.Context) (bool, error)
}
