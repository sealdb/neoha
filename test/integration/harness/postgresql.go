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

package harness

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

const (
	EnvPGBase    = "NEOHA_IT_PG_BASE"
	defaultPGBase = "/home/wslu/work/pg/pgsql"
	PGSuperUser  = "postgres"
	PGSuperPass  = "NeoHAIT1!Pass"
	PGReplUser   = "replicator"
	PGReplPass   = "NeoHAIT1!Repl"
)

// PostgreSQL implements Backend for a source-tree or package PostgreSQL build.
type PostgreSQL struct {
	BaseDir string
}

// NewPostgreSQL returns a backend using baseDir (pg_ctl lives in baseDir/bin/pg_ctl).
func NewPostgreSQL(baseDir string) *PostgreSQL {
	if baseDir == "" {
		baseDir = PGBaseFromEnv()
	}
	return &PostgreSQL{BaseDir: baseDir}
}

func (b *PostgreSQL) Name() string { return "postgresql" }

func (b *PostgreSQL) bin(name string) string {
	return filepath.Join(b.BaseDir, "bin", name)
}

func (b *PostgreSQL) NodeDatadirReady(node *Node) bool {
	if node.DataDir == "" {
		node.DataDir = filepath.Join(node.WorkDir, "pgdata")
	}
	_, err := os.Stat(filepath.Join(node.DataDir, "PG_VERSION"))
	return err == nil
}

func (b *PostgreSQL) InitNode(ctx context.Context, node *Node) error {
	if err := os.MkdirAll(node.WorkDir, 0o755); err != nil {
		return err
	}
	node.DataDir = filepath.Join(node.WorkDir, "pgdata")
	if b.NodeDatadirReady(node) {
		return nil
	}
	if err := os.MkdirAll(node.DataDir, 0o755); err != nil {
		return err
	}
	initdb := exec.CommandContext(ctx, b.bin("initdb"), "-D", node.DataDir, "-U", PGSuperUser, "--auth-local=trust", "--auth-host=trust")
	out, err := initdb.CombinedOutput()
	if err != nil {
		return fmt.Errorf("initdb: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return b.writeNodeConfig(node)
}

func (b *PostgreSQL) writeNodeConfig(node *Node) error {
	conf := fmt.Sprintf(`listen_addresses = '127.0.0.1'
port = %d
wal_level = replica
hot_standby = on
max_wal_senders = 10
max_replication_slots = 10
wal_log_hints = on
archive_mode = off
unix_socket_directories = '%s'
`, node.Port, node.DataDir)
	node.Config = filepath.Join(node.WorkDir, "postgresql.conf")
	if err := os.WriteFile(node.Config, []byte(conf), 0o644); err != nil {
		return err
	}
	hba := fmt.Sprintf(`local   all             all                                     trust
host    all             all             127.0.0.1/32            trust
host    replication     %s              127.0.0.1/32            trust
`, PGReplUser)
	return os.WriteFile(filepath.Join(node.DataDir, "pg_hba.conf"), []byte(hba), 0o600)
}

func (b *PostgreSQL) StartNode(ctx context.Context, node *Node) error {
	logPath := filepath.Join(node.WorkDir, "postgres.log")
	args := []string{
		"-D", node.DataDir, "-l", logPath,
		"-o", fmt.Sprintf("-c config_file=%s -c hba_file=%s/pg_hba.conf", node.Config, node.DataDir),
		"start", "-w", "-t", "120",
	}
	cmd := exec.CommandContext(ctx, b.bin("pg_ctl"), args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		logTail, _ := os.ReadFile(logPath)
		return fmt.Errorf("pg_ctl start: %w: %s (log: %s)", err, strings.TrimSpace(string(out)), strings.TrimSpace(string(logTail)))
	}
	return nil
}

func (b *PostgreSQL) StopNode(ctx context.Context, node *Node) error {
	args := []string{"-D", node.DataDir, "stop", "-m", "fast", "-w"}
	cmd := exec.CommandContext(ctx, b.bin("pg_ctl"), args...)
	out, err := cmd.CombinedOutput()
	if err != nil && !strings.Contains(string(out), "not running") {
		return fmt.Errorf("pg_ctl stop: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (b *PostgreSQL) Ready(ctx context.Context, node *Node) error {
	dsn := fmt.Sprintf("host=127.0.0.1 port=%d user=%s dbname=postgres sslmode=disable", node.Port, PGSuperUser)
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(60 * time.Second)
		ctx, _ = context.WithDeadline(ctx, deadline)
	}
	for {
		if ctx.Err() != nil {
			return fmt.Errorf("postgresql %s not ready on port %d", node.Name, node.Port)
		}
		db, err := sql.Open("postgres", dsn)
		if err == nil {
			err = db.PingContext(ctx)
			db.Close()
			if err == nil {
				return nil
			}
		}
		time.Sleep(readyPollInterval)
	}
}

// OpenAdmin connects as the integration superuser.
func (b *PostgreSQL) OpenAdmin(ctx context.Context, node *Node) (*sql.DB, error) {
	dsn := fmt.Sprintf("host=127.0.0.1 port=%d user=%s dbname=postgres sslmode=disable", node.Port, PGSuperUser)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// BootstrapPrimary creates replication users and sets superuser password on a fresh primary.
func (b *PostgreSQL) BootstrapPrimary(ctx context.Context, node *Node) error {
	db, err := b.OpenAdmin(ctx, node)
	if err != nil {
		return err
	}
	defer db.Close()
	stmts := []string{
		fmt.Sprintf("ALTER USER %s WITH PASSWORD '%s'", PGSuperUser, PGSuperPass),
		fmt.Sprintf("CREATE USER %s REPLICATION LOGIN PASSWORD '%s'", PGReplUser, PGReplPass),
	}
	for _, q := range stmts {
		if _, err := db.ExecContext(ctx, q); err != nil && !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("bootstrap primary: %w (sql: %s)", err, q)
		}
	}
	return nil
}

// CloneStandby runs pg_basebackup from primary into standby node.
func (b *PostgreSQL) CloneStandby(ctx context.Context, primary, standby *Node) error {
	if err := os.RemoveAll(standby.DataDir); err != nil {
		return err
	}
	args := []string{
		"-h", "127.0.0.1",
		"-p", fmt.Sprintf("%d", primary.Port),
		"-U", PGReplUser,
		"-D", standby.DataDir,
		"-Fp", "-Xs", "-P", "-R",
	}
	cmd := exec.CommandContext(ctx, b.bin("pg_basebackup"), args...)
	cmd.Env = append(os.Environ(), "PGPASSWORD="+PGReplPass)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pg_basebackup: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return b.writeNodeConfig(standby)
}

// WaitInRecovery polls until node reports pg_is_in_recovery().
func (b *PostgreSQL) WaitInRecovery(ctx context.Context, node *Node, inRecovery bool) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		db, err := b.OpenAdmin(ctx, node)
		if err == nil {
			var recovery bool
			qerr := db.QueryRowContext(ctx, "SELECT pg_is_in_recovery()").Scan(&recovery)
			db.Close()
			if qerr == nil && recovery == inRecovery {
				return nil
			}
		}
		time.Sleep(readyPollInterval)
	}
}

// WaitReplicationConnected waits until primary shows a connected standby.
func (b *PostgreSQL) WaitReplicationConnected(ctx context.Context, primary, standby *Node) error {
	_ = standby
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		db, err := b.OpenAdmin(ctx, primary)
		if err == nil {
			var count int
			qerr := db.QueryRowContext(ctx,
				`SELECT count(*) FROM pg_stat_replication WHERE client_addr IS NOT NULL`,
			).Scan(&count)
			db.Close()
			if qerr == nil && count > 0 {
				return nil
			}
		}
		time.Sleep(readyPollInterval)
	}
}

// ExecPrimary runs SQL on a node using the superuser.
func (b *PostgreSQL) ExecPrimary(ctx context.Context, node *Node, query string) (int64, error) {
	db, err := b.OpenAdmin(ctx, node)
	if err != nil {
		return 0, err
	}
	defer db.Close()
	res, err := db.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// Promote runs pg_promote on standby.
func (b *PostgreSQL) Promote(ctx context.Context, node *Node) error {
	db, err := b.OpenAdmin(ctx, node)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.ExecContext(ctx, "SELECT pg_promote(true)")
	return err
}

// PGBaseFromEnv returns the PostgreSQL installation root.
func PGBaseFromEnv() string {
	if v := os.Getenv(EnvPGBase); v != "" {
		return v
	}
	s := LoadIntegrationSettings()
	if s.file.PGBase != "" {
		return s.file.PGBase
	}
	return defaultPGBase
}

// RequirePostgreSQL skips the test when pg_ctl is missing.
func RequirePostgreSQL(t testingT, base string) *PostgreSQL {
	t.Helper()
	if base == "" {
		base = PGBaseFromEnv()
	}
	pgCtl := filepath.Join(base, "bin", "pg_ctl")
	if _, err := os.Stat(pgCtl); err != nil {
		t.Skipf("pg_ctl not found at %s (set %s)", pgCtl, EnvPGBase)
	}
	return NewPostgreSQL(base)
}

type testingT interface {
	Helper()
	Skipf(format string, args ...interface{})
}
