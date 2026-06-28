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
	"database/sql"
	"fmt"
	"strings"
	"time"

	dbdriver "github.com/sealdb/neoha/internal/database/driver"
)

func (p *Postgresql) replUser() (string, string) {
	user, pass := "replicator", ""
	if p.conf.Auth != nil && p.conf.Auth.Repl != nil {
		if p.conf.Auth.Repl.Username != "" {
			user = p.conf.Auth.Repl.Username
		}
		pass = p.conf.Auth.Repl.Password
	}
	return user, pass
}

func (p *Postgresql) primarySlotName(memberName string) string {
	if p.conf.PrimarySlotName != "" {
		return p.conf.PrimarySlotName
	}
	if memberName != "" {
		return memberName
	}
	return "pgrepl"
}

// BuildPrimaryConninfo builds a libpq conninfo string for streaming replication.
func (p *Postgresql) BuildPrimaryConninfo(primary dbdriver.PrimaryRef) string {
	host := primary.Host
	portStr := "5432"
	if host == "" && primary.MemberID != "" {
		host, portStr = splitHostPort(primary.MemberID, "5432")
	} else if primary.Port > 0 {
		portStr = fmt.Sprintf("%d", primary.Port)
	}
	user, pass := p.replUser()
	if pass != "" {
		return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres sslmode=disable",
			host, portStr, user, pass)
	}
	return fmt.Sprintf("host=%s port=%s user=%s dbname=postgres sslmode=disable", host, portStr, user)
}

func (p *Postgresql) openPrimaryDB(primary dbdriver.PrimaryRef) (*sql.DB, error) {
	conninfo := p.BuildPrimaryConninfo(primary)
	return sql.Open("postgres", conninfo)
}

// EnsurePhysicalSlotOnPrimary creates the replication slot on the primary if missing.
func (p *Postgresql) EnsurePhysicalSlotOnPrimary(ctx context.Context, primary dbdriver.PrimaryRef, slotName string) error {
	if slotName == "" {
		return nil
	}
	db, err := p.openPrimaryDB(primary)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return err
	}
	var exists bool
	if err := db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM pg_replication_slots WHERE slot_name = $1)`, slotName).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	_, err = db.ExecContext(ctx, `SELECT pg_create_physical_replication_slot($1, true)`, slotName)
	return err
}

// SetRecoveryPrimaryConninfo writes primary_conninfo via ALTER SYSTEM on the local instance.
func (p *Postgresql) SetRecoveryPrimaryConninfo(ctx context.Context, conninfo string) error {
	db, err := p.getDB()
	if err != nil {
		return err
	}
	escaped := strings.ReplaceAll(conninfo, "'", "''")
	_, err = db.ExecContext(ctx, fmt.Sprintf("ALTER SYSTEM SET primary_conninfo = '%s'", escaped))
	return err
}

// SetRecoveryPrimarySlotName writes primary_slot_name via ALTER SYSTEM.
func (p *Postgresql) SetRecoveryPrimarySlotName(ctx context.Context, slotName string) error {
	if slotName == "" {
		return nil
	}
	db, err := p.getDB()
	if err != nil {
		return err
	}
	escaped := strings.ReplaceAll(slotName, "'", "''")
	_, err = db.ExecContext(ctx, fmt.Sprintf("ALTER SYSTEM SET primary_slot_name = '%s'", escaped))
	return err
}

// ReloadConf runs pg_reload_conf().
func (p *Postgresql) ReloadConf(ctx context.Context) error {
	db, err := p.getDB()
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, "SELECT pg_reload_conf()")
	return err
}

// ApplyStreamingReplica configures this node to follow primary (standby must already be in recovery or restart pending).
func (p *Postgresql) ApplyStreamingReplica(ctx context.Context, primary dbdriver.PrimaryRef, memberName string, useSlots bool) error {
	conninfo := p.BuildPrimaryConninfo(primary)
	slotName := p.primarySlotName(memberName)
	if useSlots {
		if err := p.EnsurePhysicalSlotOnPrimary(ctx, primary, slotName); err != nil {
			return err
		}
	}
	if err := p.SetRecoveryPrimaryConninfo(ctx, conninfo); err != nil {
		return err
	}
	if useSlots {
		if err := p.SetRecoveryPrimarySlotName(ctx, slotName); err != nil {
			return err
		}
	}
	if err := p.ReloadConf(ctx); err != nil {
		return err
	}
	inRecovery, err := p.IsInRecovery(ctx)
	if err != nil {
		return err
	}
	if !inRecovery {
		if err := p.restartIntoRecovery(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (p *Postgresql) restartIntoRecovery(ctx context.Context) error {
	p.Close()
	if err := p.pgCtlStop(ctx); err != nil {
		return err
	}
	if err := p.pgCtlStart(ctx); err != nil {
		return err
	}
	waitCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	if err := p.WaitReady(waitCtx); err != nil {
		return err
	}
	inRecovery, err := p.IsInRecovery(ctx)
	if err != nil {
		return err
	}
	if !inRecovery {
		return fmt.Errorf("postgresql: instance not in recovery after restart")
	}
	return nil
}
