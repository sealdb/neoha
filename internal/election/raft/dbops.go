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

package raft

import (
	"context"

	"github.com/sealdb/neoha/internal/base/model"
	dbdriver "github.com/sealdb/neoha/internal/database/driver"
	mysqldriver "github.com/sealdb/neoha/internal/database/mysql"
)

// DelegateDBApply reports whether DB role apply is owned by L3 Reconciler.
func (r *Raft) DelegateDBApply() bool {
	return r.delegateDBApply
}

func (r *Raft) delegateSemiSyncApply() bool {
	return r.delegateDBApply && r.mysqlReplMode == model.ReplModeSemiSync
}

func (r *Raft) delegateMGRApply() bool {
	return r.delegateDBApply && r.mysqlReplMode == model.ReplModeMGR
}

// demoteReadOnly sets the database to a non-primary role via L4 Driver when available.
func (r *Raft) demoteReadOnly() error {
	if r.delegateDBApply {
		return nil
	}
	if r.dbDriver != nil {
		return r.dbDriver.Demote(context.Background())
	}
	return r.mysql.SetReadOnly()
}

// changeToMaster runs semi-sync/MGR "become primary" replication setup.
func (r *Raft) changeToMaster() error {
	if r.delegateDBApply {
		return nil
	}
	if md, ok := r.dbDriver.(*mysqldriver.Driver); ok {
		return md.ChangeToMasterRepl(context.Background())
	}
	repl := r.mysql.GetRepl()
	return r.mysql.ChangeToMaster(&repl)
}

// enableReadWrite opens writes on the database instance after promotion steps complete.
func (r *Raft) enableReadWrite() error {
	if r.delegateDBApply {
		return nil
	}
	if md, ok := r.dbDriver.(*mysqldriver.Driver); ok {
		return md.EnableReadWrite(context.Background())
	}
	return r.mysql.SetReadWrite()
}

func (r *Raft) driverPromotable() bool {
	if r.dbDriver == nil {
		return r.mysql.Promotable()
	}
	ok, _ := r.dbDriver.Promotable(context.Background())
	return ok
}

// Driver returns the L4 database facade (may be nil in partial tests).
func (r *Raft) Driver() dbdriver.Driver {
	return r.dbDriver
}

// isDelegatedPrimaryReady returns true when the driver reports a writable primary.
func (r *Raft) isDelegatedPrimaryReady() bool {
	if r.dbDriver == nil {
		return false
	}
	health, err := r.dbDriver.Status(context.Background())
	return err == nil && health.Writable
}

// isDelegatedMGRPhase1Ready reports MGR phase-1 (group primary, still read-only) completion.
func (r *Raft) isDelegatedMGRPhase1Ready() bool {
	mgr, ok := r.dbDriver.(dbdriver.MGRLifecycle)
	if !ok || !mgr.IsMGRMode() {
		return false
	}
	done, err := mgr.MGRPrimaryPhase1Done(context.Background())
	return err == nil && done
}
