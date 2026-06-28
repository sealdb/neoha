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

// Package ha implements the L3 reconcile loop (Patroni-style run_cycle).
// See docs/architecture.md.
package ha

import (
	"context"
	"fmt"
	"time"

	"github.com/sealdb/neoha/internal/base/nlog"
	"github.com/sealdb/neoha/internal/config"
	"github.com/sealdb/neoha/internal/coordination"
	dbdriver "github.com/sealdb/neoha/internal/database/driver"
)

const DefaultReconcileInterval = 3 * time.Second

// DesiredState is the outcome of one reconcile iteration.
type DesiredState struct {
	IsCoordLeader bool
	DBRole        dbdriver.Role
	Primary       dbdriver.PrimaryRef
	Reason        string
}

// Reconciler aligns coordination leadership with database role via the Driver.
type Reconciler struct {
	log               *nlog.Log
	coord             coordination.Coordinator
	driver            dbdriver.Driver
	tags              *config.TagsConfig
	hooks             *PrimaryHooks
	interval          time.Duration
	applyPromote       bool
	defensiveStopDone  bool
	lastReplicaPrimary string
	mysqlPort          int
}

// Option configures a Reconciler.
type Option func(*Reconciler)

// WithInterval sets the reconcile loop tick interval.
func WithInterval(d time.Duration) Option {
	return func(r *Reconciler) {
		if d > 0 {
			r.interval = d
		}
	}
}

// WithApplyPromote enables ApplyPrimary from reconcile (off by default in v0.3).
func WithApplyPromote(enable bool) Option {
	return func(r *Reconciler) {
		r.applyPromote = enable
	}
}

// WithPrimaryHooks attaches primary accessibility hooks (VIP, LB, …).
func WithPrimaryHooks(hooks *PrimaryHooks) Option {
	return func(r *Reconciler) {
		r.hooks = hooks
	}
}

// WithMySQLPort sets the local MySQL port used when deriving ApplyReplica targets from NeoHA endpoints.
func WithMySQLPort(port int) Option {
	return func(r *Reconciler) {
		r.mysqlPort = port
	}
}

// NewReconciler builds a reconciler.
func NewReconciler(log *nlog.Log, coord coordination.Coordinator, driver dbdriver.Driver, tags *config.TagsConfig, opts ...Option) *Reconciler {
	if tags == nil {
		tags = config.DefaultTagsConfig()
	}
	r := &Reconciler{
		log:      log,
		coord:    coord,
		driver:   driver,
		tags:     tags,
		interval: DefaultReconcileInterval,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// RunOnce executes a single reconcile cycle.
func (r *Reconciler) RunOnce(ctx context.Context) error {
	if r.coord == nil || r.driver == nil {
		return nil
	}
	desired, err := r.computeDesired(ctx)
	if err != nil {
		return err
	}
	if !r.defensiveStopDone && desired.DBRole != dbdriver.RolePrimary {
		r.defensiveStopDone = true
		if r.hooks != nil {
			if err := r.hooks.RunStopIfConfigured(ctx); err != nil {
				r.log.Warning("reconcile.defensive.stop.error[%v]", err)
			}
		}
	}
	return r.applyDesired(ctx, desired)
}

// Loop runs reconcile cycles until ctx is cancelled.
func (r *Reconciler) Loop(ctx context.Context) {
	if r.coord == nil || r.driver == nil {
		return
	}
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		if err := r.RunOnce(ctx); err != nil {
			r.log.Warning("reconcile.error[%v]", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (r *Reconciler) computeDesired(ctx context.Context) (DesiredState, error) {
	var out DesiredState
	out.IsCoordLeader = r.coord.IsLeader()
	if r.tags.Nofailover {
		out.Reason = "tag.nofailover"
	}
	if out.IsCoordLeader && !r.tags.Nofailover {
		ok, reason := r.driver.Promotable(ctx)
		if ok {
			out.DBRole = dbdriver.RolePrimary
			out.Reason = "coord.leader.promotable"
		} else {
			out.DBRole = dbdriver.RoleReplica
			out.Reason = reason
		}
	} else {
		out.DBRole = dbdriver.RoleReplica
		if !out.IsCoordLeader {
			out.Reason = "not.coord.leader"
		}
	}
	view, err := r.coord.ClusterView(ctx)
	if err == nil && view.LeaderID != "" {
		out.Primary = dbdriver.PrimaryRef{MemberID: view.LeaderID}
		if view.LeaderDatabase.Host != "" && view.LeaderDatabase.Port > 0 {
			out.Primary.Host = view.LeaderDatabase.Host
			out.Primary.Port = view.LeaderDatabase.Port
		}
	}
	return out, nil
}

func (r *Reconciler) applyDesired(ctx context.Context, desired DesiredState) error {
	health, err := r.driver.Status(ctx)
	if err != nil {
		return err
	}

	if desired.DBRole == dbdriver.RolePrimary {
		if r.applyPromote && !health.Writable {
			if mgr, ok := r.driver.(dbdriver.MGRLifecycle); ok && mgr.IsMGRMode() {
				if err := r.applyMGRPrimary(ctx, desired.Reason, mgr, health); err != nil {
					return err
				}
			} else if health.Role != dbdriver.RolePrimary {
				r.log.Info("reconcile.apply.primary reason[%s]", desired.Reason)
				if err := r.driver.ApplyPrimary(ctx); err != nil {
					return err
				}
			}
			health, err = r.driver.Status(ctx)
			if err != nil {
				return err
			}
		}
		if health.Writable {
			if r.hooks != nil {
				return r.hooks.EnsurePrimary(ctx)
			}
		}
		return nil
	}

	if r.hooks != nil {
		if err := r.hooks.EnsureReplica(ctx); err != nil {
			return err
		}
	}
	if health.Writable || health.Role == dbdriver.RolePrimary {
		r.log.Info("reconcile.apply.demote reason[%s]", desired.Reason)
		if err := r.driver.Demote(ctx); err != nil {
			return err
		}
	}
	if r.applyPromote {
		return r.applyReplicaIfNeeded(ctx, desired)
	}
	return nil
}

func (r *Reconciler) applyReplicaIfNeeded(ctx context.Context, desired DesiredState) error {
	pid := desired.Primary.MemberID
	if pid == "" || pid == r.coord.LocalID() {
		r.lastReplicaPrimary = ""
		return nil
	}
	ref := enrichPrimaryRef(desired.Primary, r.mysqlPort)
	targetKey := replicaTargetKey(ref)
	if targetKey == "" || targetKey == r.lastReplicaPrimary {
		return nil
	}
	r.log.Info("reconcile.apply.replica primary[%s] db[%s] reason[%s]", pid, targetKey, desired.Reason)
	if err := r.driver.ApplyReplica(ctx, ref); err != nil {
		return err
	}
	r.lastReplicaPrimary = targetKey
	return nil
}

func replicaTargetKey(ref dbdriver.PrimaryRef) string {
	if ref.Host != "" && ref.Port > 0 {
		return fmt.Sprintf("%s:%d", ref.Host, ref.Port)
	}
	return ref.MemberID
}

func (r *Reconciler) applyMGRPrimary(ctx context.Context, reason string, mgr dbdriver.MGRLifecycle, health dbdriver.Health) error {
	phase1, err := mgr.MGRPrimaryPhase1Done(ctx)
	if err != nil {
		return err
	}
	if !phase1 {
		r.log.Info("reconcile.apply.primary.mgr.phase1 reason[%s]", reason)
		if err := r.driver.ApplyPrimary(ctx); err != nil {
			return err
		}
	}
	health, err = r.driver.Status(ctx)
	if err != nil {
		return err
	}
	if health.Writable {
		return nil
	}
	ready, err := mgr.MGRClusterWritableReady(ctx)
	if err != nil {
		return err
	}
	if !ready {
		return nil
	}
	r.log.Info("reconcile.apply.primary.mgr.phase2 reason[%s]", reason)
	return mgr.ApplyPrimaryMGRPhase2(ctx)
}
