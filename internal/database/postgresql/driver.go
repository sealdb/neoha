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

package postgresql

import (
	"context"
	"fmt"
	"time"

	"github.com/sealdb/neoha/internal/base/nlog"
	"github.com/sealdb/neoha/internal/config"
	dbdriver "github.com/sealdb/neoha/internal/database/driver"
)

// Driver implements dbdriver.Driver for PostgreSQL (streaming replication path).
type Driver struct {
	pg   *Postgresql
	conf *config.PostgresqlConfig
	log  *nlog.Log
}

// NewDriver wraps *Postgresql as dbdriver.Driver.
func NewDriver(pg *Postgresql, conf *config.PostgresqlConfig, log *nlog.Log) *Driver {
	return &Driver{pg: pg, conf: conf, log: log}
}

func (d *Driver) Type() dbdriver.Engine {
	return dbdriver.EnginePostgreSQL
}

func (d *Driver) Start() {}

func (d *Driver) Stop() {
	d.pg.Close()
}

func (d *Driver) SetupBootstrap(ctx context.Context) error {
	waitCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	d.log.Info("database.postgresql.wait.for.work[maxwait:60s]")
	if err := d.pg.WaitReady(waitCtx); err != nil {
		return err
	}
	d.log.Info("database.postgresql.set.to.READONLY")
	return d.pg.SetDefaultReadOnly(ctx, true)
}

func (d *Driver) Promotable(ctx context.Context) (bool, string) {
	if err := ctx.Err(); err != nil {
		return false, err.Error()
	}
	if err := d.pg.PingDB(ctx); err != nil {
		return false, "postgresql.not.alive"
	}
	inRecovery, err := d.pg.IsInRecovery(ctx)
	if err != nil {
		return false, err.Error()
	}
	if inRecovery {
		if d.conf.MaximumLagOnFailover > 0 {
			lag, err := d.pg.ReplicationLagBytes(ctx)
			if err != nil {
				return false, err.Error()
			}
			if lag > d.conf.MaximumLagOnFailover {
				return false, "postgresql.lag.too.high"
			}
		}
		return true, ""
	}
	if d.pg.isWritable() {
		return true, ""
	}
	return false, "postgresql.already.primary.readonly"
}

func (d *Driver) ReplicationLagBytes(ctx context.Context) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return d.pg.ReplicationLagBytes(ctx)
}

func (d *Driver) ApplyPrimary(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	inRecovery, err := d.pg.IsInRecovery(ctx)
	if err != nil {
		return err
	}
	if inRecovery {
		d.log.Info("database.postgresql.promote")
		return d.pg.Promote(ctx)
	}
	return d.pg.SetDefaultReadOnly(ctx, false)
}

func (d *Driver) ApplyReplica(ctx context.Context, primary dbdriver.PrimaryRef) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if primary.Host == "" && primary.MemberID == "" {
		return fmt.Errorf("postgresql.ApplyReplica: primary endpoint required")
	}
	if d.conf.UsePGRewind {
		if err := d.pg.MaybePgRewind(ctx, primary); err != nil {
			return err
		}
	}
	if err := d.pg.SetDefaultReadOnly(ctx, true); err != nil {
		return err
	}
	useSlots := d.conf.UseSlots
	if !useSlots && d.conf.PrimarySlotName != "" {
		useSlots = true
	}
	return d.pg.ApplyStreamingReplica(ctx, primary, primary.MemberID, useSlots)
}

func (d *Driver) Demote(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return d.pg.SetDefaultReadOnly(ctx, true)
}

func (d *Driver) Status(ctx context.Context) (dbdriver.Health, error) {
	if err := ctx.Err(); err != nil {
		return dbdriver.Health{}, err
	}
	if err := d.pg.PingDB(ctx); err != nil {
		return dbdriver.Health{Alive: false, Role: dbdriver.RoleUnknown, LastError: err.Error()}, nil
	}
	inRecovery, err := d.pg.IsInRecovery(ctx)
	if err != nil {
		return dbdriver.Health{}, err
	}
	role := dbdriver.RolePrimary
	if inRecovery {
		role = dbdriver.RoleReplica
	}
	writable := d.pg.isWritable() && !inRecovery
	return dbdriver.Health{
		Alive:    true,
		Writable: writable,
		Role:     role,
	}, nil
}

// Postgresql returns the underlying instance for legacy callers.
func (d *Driver) Postgresql() *Postgresql {
	return d.pg
}
