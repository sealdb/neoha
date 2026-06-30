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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	dbdriver "github.com/sealdb/neoha/internal/database/driver"
)

func (p *Postgresql) pgBin(name string) string {
	if p.conf.BinDir != "" {
		return filepath.Join(p.conf.BinDir, name)
	}
	path, err := exec.LookPath(name)
	if err != nil {
		return name
	}
	return path
}

func (p *Postgresql) rewindAuth() (user, pass string) {
	if p.conf.Auth != nil && p.conf.Auth.Rewind != nil {
		if p.conf.Auth.Rewind.Username != "" {
			user = p.conf.Auth.Rewind.Username
		}
		pass = p.conf.Auth.Rewind.Password
	}
	if user == "" && p.conf.Auth != nil && p.conf.Auth.SuperUser != nil {
		user = p.conf.Auth.SuperUser.Username
		if user == "" {
			user = "postgres"
		}
		pass = p.conf.Auth.SuperUser.Password
	}
	if user == "" {
		user = "postgres"
	}
	return user, pass
}

// BuildRewindConninfo builds libpq conninfo for pg_rewind --source-server.
func (p *Postgresql) BuildRewindConninfo(primary dbdriver.PrimaryRef) string {
	host := primary.Host
	portStr := "5432"
	if host == "" && primary.MemberID != "" {
		host, portStr = splitHostPort(primary.MemberID, "5432")
	} else if primary.Port > 0 {
		portStr = fmt.Sprintf("%d", primary.Port)
	}
	user, pass := p.rewindAuth()
	if pass != "" {
		return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres sslmode=disable",
			host, portStr, user, pass)
	}
	return fmt.Sprintf("host=%s port=%s user=%s dbname=postgres sslmode=disable", host, portStr, user)
}

// ControlTimeline returns the local timeline from pg_control_checkpoint().
func (p *Postgresql) ControlTimeline(ctx context.Context) (uint32, error) {
	db, err := p.getDB()
	if err != nil {
		return 0, err
	}
	var timeline uint32
	err = db.QueryRowContext(ctx, `SELECT timeline_id FROM pg_control_checkpoint()`).Scan(&timeline)
	return timeline, err
}

// PrimaryTimeline returns the primary's current timeline.
func (p *Postgresql) PrimaryTimeline(ctx context.Context, primary dbdriver.PrimaryRef) (uint32, error) {
	db, err := p.openPrimaryDB(primary)
	if err != nil {
		return 0, err
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return 0, err
	}
	var timeline uint32
	err = db.QueryRowContext(ctx, `SELECT timeline_id FROM pg_control_checkpoint()`).Scan(&timeline)
	return timeline, err
}

// NeedsPgRewind reports whether pg_rewind should run before rejoining as replica.
func (p *Postgresql) NeedsPgRewind(ctx context.Context, primary dbdriver.PrimaryRef) (bool, error) {
	if err := p.PingDB(ctx); err != nil {
		return true, nil
	}
	inRecovery, err := p.IsInRecovery(ctx)
	if err != nil {
		return false, err
	}
	if !inRecovery {
		return true, nil
	}
	localTL, err := p.ControlTimeline(ctx)
	if err != nil {
		return false, err
	}
	primaryTL, err := p.PrimaryTimeline(ctx, primary)
	if err != nil {
		return false, err
	}
	return localTL != primaryTL, nil
}

func (p *Postgresql) pgDataDir() (string, error) {
	if p.conf.DataDir == "" {
		return "", fmt.Errorf("postgresql: data_dir required for pg_ctl/pg_rewind")
	}
	return p.conf.DataDir, nil
}

func (p *Postgresql) pgCtlStop(ctx context.Context) error {
	dataDir, err := p.pgDataDir()
	if err != nil {
		return err
	}
	args := []string{"-D", dataDir, "stop", "-m", "fast", "-w"}
	cmd := exec.CommandContext(ctx, p.pgBin("pg_ctl"), args...)
	out, err := cmd.CombinedOutput()
	msg := string(out)
	if err != nil && !strings.Contains(msg, "not running") && !strings.Contains(msg, "Is server running") {
		return fmt.Errorf("pg_ctl stop: %w: %s", err, strings.TrimSpace(msg))
	}
	return nil
}

func (p *Postgresql) pgCtlStart(ctx context.Context) error {
	dataDir, err := p.pgDataDir()
	if err != nil {
		return err
	}
	_, port := p.ConnectHostPort()
	opts := fmt.Sprintf("-c listen_addresses=127.0.0.1 -c port=%d", port)
	configFile := filepath.Join(dataDir, "postgresql.conf")
	hbaFile := filepath.Join(dataDir, "pg_hba.conf")
	if p.conf.ConfigDir != "" && p.conf.ConfigDir != dataDir {
		configFile = filepath.Join(p.conf.ConfigDir, "postgresql.conf")
	}
	if _, err := os.Stat(configFile); err == nil {
		opts += fmt.Sprintf(" -c config_file=%s -c hba_file=%s", configFile, hbaFile)
	}
	logPath := filepath.Join(dataDir, "pg_ctl-rewind.log")
	args := []string{"-D", dataDir, "-l", logPath, "-o", opts, "start", "-w", "-t", "30"}
	cmd := exec.CommandContext(ctx, p.pgBin("pg_ctl"), args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		logTail, _ := os.ReadFile(logPath)
		return fmt.Errorf("pg_ctl start: %w: %s (log: %s)", err, strings.TrimSpace(string(out)), strings.TrimSpace(string(logTail)))
	}
	return nil
}

// RunPgRewind executes pg_rewind against primary (postgres must be stopped).
func (p *Postgresql) RunPgRewind(ctx context.Context, primary dbdriver.PrimaryRef) error {
	dataDir, err := p.pgDataDir()
	if err != nil {
		return err
	}
	conninfo := p.BuildRewindConninfo(primary)
	args := []string{
		"--target-pgdata=" + dataDir,
		"--source-server=" + conninfo,
		"--no-sync",
	}
	cmd := exec.CommandContext(ctx, p.pgBin("pg_rewind"), args...)
	if deadline, ok := ctx.Deadline(); ok {
		cmd.WaitDelay = time.Until(deadline)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pg_rewind: %w: %s", err, strings.TrimSpace(string(out)))
	}
	if p.log != nil {
		p.log.Info("database.postgresql.pg_rewind.done")
	}
	return nil
}

func (p *Postgresql) ensureStandbySignal(dataDir string) error {
	path := filepath.Join(dataDir, "standby.signal")
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	return f.Close()
}

// MaybePgRewind stops postgres, runs pg_rewind if needed, and restarts.
func (p *Postgresql) MaybePgRewind(ctx context.Context, primary dbdriver.PrimaryRef) error {
	if !p.conf.UsePGRewind {
		return nil
	}
	needs, err := p.NeedsPgRewind(ctx, primary)
	if err != nil {
		return err
	}
	if !needs {
		return nil
	}
	if p.conf.DataDir == "" {
		return fmt.Errorf("postgresql: use_pg_rewind set but data_dir is empty")
	}
	if _, err := os.Stat(filepath.Join(p.conf.DataDir, "PG_VERSION")); err != nil {
		return fmt.Errorf("postgresql: pg_rewind requires initialized data_dir: %w", err)
	}
	if p.log != nil {
		p.log.Info("database.postgresql.pg_rewind.start")
	}
	p.Close()
	if err := p.pgCtlStop(ctx); err != nil {
		return err
	}
	if err := p.RunPgRewind(ctx, primary); err != nil {
		return err
	}
	if err := p.ensureStandbySignal(p.conf.DataDir); err != nil {
		return err
	}
	if err := p.pgCtlStart(ctx); err != nil {
		return err
	}
	waitCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	return p.WaitReady(waitCtx)
}
