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

package postgresql

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

func (p *Postgresql) dsn() string {
	host, port := splitHostPort(p.conf.ConnectAddress, "5432")
	user := "postgres"
	pass := ""
	if p.conf.Auth != nil && p.conf.Auth.SuperUser != nil {
		if p.conf.Auth.SuperUser.Username != "" {
			user = p.conf.Auth.SuperUser.Username
		}
		pass = p.conf.Auth.SuperUser.Password
	}
	if pass != "" {
		return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres sslmode=disable",
			host, port, user, pass)
	}
	return fmt.Sprintf("host=%s port=%s user=%s dbname=postgres sslmode=disable", host, port, user)
}

func splitHostPort(addr, defaultPort string) (string, string) {
	if addr == "" {
		return "127.0.0.1", defaultPort
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr, defaultPort
	}
	if port == "" {
		port = defaultPort
	}
	return host, port
}

func (p *Postgresql) getDB() (*sql.DB, error) {
	p.dbmutex.RLock()
	db := p.db
	p.dbmutex.RUnlock()
	if db != nil {
		return db, nil
	}
	p.dbmutex.Lock()
	defer p.dbmutex.Unlock()
	if p.db != nil {
		return p.db, nil
	}
	db, err := sql.Open("postgres", p.dsn())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(2)
	p.db = db
	return p.db, nil
}

// PingDB checks PostgreSQL connectivity.
func (p *Postgresql) PingDB(ctx context.Context) error {
	db, err := p.getDB()
	if err != nil {
		return err
	}
	return db.PingContext(ctx)
}

// WaitReady polls until PostgreSQL accepts connections or ctx expires.
func (p *Postgresql) WaitReady(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		if err := p.PingDB(ctx); err == nil {
			p.setAlive(true)
			return nil
		} else if ctx.Err() != nil {
			return ctx.Err()
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (p *Postgresql) setAlive(alive bool) {
	p.mutex.Lock()
	if alive {
		p.alive = true
	} else {
		p.alive = false
	}
	p.mutex.Unlock()
}

func (p *Postgresql) IsAlive() bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.alive
}

// IsInRecovery reports pg_is_in_recovery().
func (p *Postgresql) IsInRecovery(ctx context.Context) (bool, error) {
	db, err := p.getDB()
	if err != nil {
		return false, err
	}
	var inRecovery bool
	err = db.QueryRowContext(ctx, "SELECT pg_is_in_recovery()").Scan(&inRecovery)
	return inRecovery, err
}

// SetDefaultReadOnly sets default_transaction_read_only for new sessions.
func (p *Postgresql) SetDefaultReadOnly(ctx context.Context, readOnly bool) error {
	db, err := p.getDB()
	if err != nil {
		return err
	}
	val := "off"
	if readOnly {
		val = "on"
	}
	_, err = db.ExecContext(ctx, "SET default_transaction_read_only = "+val)
	if err == nil {
		p.mutex.Lock()
		p.writable = !readOnly
		p.mutex.Unlock()
	}
	return err
}

// Promote runs pg_promote() on a standby.
func (p *Postgresql) Promote(ctx context.Context) error {
	db, err := p.getDB()
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, "SELECT pg_promote(true)")
	if err != nil {
		// Some builds expose pg_promote() without args.
		_, err = db.ExecContext(ctx, "SELECT pg_promote()")
	}
	if err == nil {
		p.mutex.Lock()
		p.writable = true
		p.mutex.Unlock()
	}
	return err
}

func (p *Postgresql) isWritable() bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.writable
}

// ConnectHostPort returns host/port parsed from connect_address.
func (p *Postgresql) ConnectHostPort() (string, int) {
	host, portStr := splitHostPort(p.conf.ConnectAddress, "5432")
	port := 5432
	if n, err := parsePort(portStr); err == nil {
		port = n
	}
	return host, port
}

func parsePort(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(strings.TrimSpace(s), "%d", &n)
	return n, err
}
