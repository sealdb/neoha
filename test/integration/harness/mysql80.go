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
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const (
	defaultMySQLBase = "/home/wslu/work/mysql/mysql80-debug"
)

// MySQL80 implements Backend for a debug/source-tree MySQL 8.0 build.
type MySQL80 struct {
	BaseDir  string
	SemiSync bool // when true, load semi-sync plugins instead of group_replication
}

// NewMySQL80 returns a backend using baseDir (mysqld lives in baseDir/bin/mysqld).
func NewMySQL80(baseDir string) *MySQL80 {
	if baseDir == "" {
		baseDir = defaultMySQLBase
	}
	return &MySQL80{BaseDir: baseDir}
}

func (b *MySQL80) Name() string { return "mysql80" }

func (b *MySQL80) mysqld() string {
	return filepath.Join(b.BaseDir, "bin", "mysqld")
}

func (b *MySQL80) InitNode(ctx context.Context, node *Node) error {
	if err := os.MkdirAll(node.WorkDir, 0o755); err != nil {
		return err
	}
	node.DataDir = filepath.Join(node.WorkDir, "data")
	node.Socket = filepath.Join(node.WorkDir, "mysql.sock")
	node.Config = filepath.Join(node.WorkDir, "my.cnf")

	if err := os.MkdirAll(node.DataDir, 0o755); err != nil {
		return err
	}

	var cnf string
	if b.SemiSync {
		cnf = fmt.Sprintf(`[mysqld]
basedir=%s
plugin_dir=%s/lib/plugin
datadir=%s
port=%d
socket=%s
pid-file=%s/mysqld.pid
bind-address=127.0.0.1
mysqlx=0
skip-log-bin=0
log-bin=mysql-bin
relay-log=%s/relay-bin
binlog_format=ROW
binlog_checksum=NONE
gtid_mode=ON
enforce_gtid_consistency=ON
log_replica_updates=ON
transaction_write_set_extraction=XXHASH64
replica_preserve_commit_order=ON
server_id=%d
report_host=127.0.0.1
report_port=%d
default_authentication_plugin=mysql_native_password
plugin_load_add='semisync_master.so;semisync_slave.so'
`, b.BaseDir, b.BaseDir, node.DataDir, node.Port, node.Socket, node.WorkDir, node.WorkDir, node.Port, node.Port)
	} else {
		cnf = fmt.Sprintf(`[mysqld]
basedir=%s
plugin_dir=%s/lib/plugin
datadir=%s
port=%d
socket=%s
pid-file=%s/mysqld.pid
bind-address=127.0.0.1
mysqlx=0
skip-log-bin=0
log-bin=mysql-bin
relay-log=%s/relay-bin
binlog_format=ROW
binlog_checksum=NONE
gtid_mode=ON
enforce_gtid_consistency=ON
log_replica_updates=ON
transaction_write_set_extraction=XXHASH64
replica_preserve_commit_order=ON
server_id=%d
report_host=127.0.0.1
report_port=%d
loose-group_replication_group_name="%s"
loose-group_replication_start_on_boot=OFF
loose-group_replication_local_address="127.0.0.1:%d"
loose-group_replication_group_seeds="127.0.0.1:13361,127.0.0.1:13362,127.0.0.1:13363"
loose-group_replication_bootstrap_group=OFF
loose-group_replication_single_primary_mode=ON
loose-group_replication_recovery_get_public_key=ON
plugin_load_add='group_replication.so'
`, b.BaseDir, b.BaseDir, node.DataDir, node.Port, node.Socket, node.WorkDir, node.WorkDir, node.Port, node.Port, mgrGroupName, node.GRPort)
	}

	if err := os.WriteFile(node.Config, []byte(cnf), 0o644); err != nil {
		return err
	}

	// Initialize datadir if empty.
	entries, err := os.ReadDir(node.DataDir)
	if err != nil {
		return err
	}
	if len(entries) > 0 {
		return nil
	}

	cmd := exec.CommandContext(ctx, b.mysqld(), "--defaults-file="+node.Config, "--initialize-insecure")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mysqld --initialize-insecure: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (b *MySQL80) StartNode(ctx context.Context, node *Node) error {
	cmd := exec.CommandContext(ctx, b.mysqld(), "--defaults-file="+node.Config)
	logPath := filepath.Join(node.WorkDir, "mysqld.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		return err
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return err
	}
	node.cmd = cmd.Process
	return nil
}

func (b *MySQL80) StopNode(ctx context.Context, node *Node) error {
	if node.cmd == nil {
		return nil
	}
	proc := node.cmd
	node.cmd = nil
	_ = proc.Signal(os.Interrupt)
	done := make(chan error, 1)
	go func() {
		_, err := proc.Wait()
		done <- err
	}()
	select {
	case <-ctx.Done():
		_ = proc.Kill()
		<-done
	case <-done:
	}
	return nil
}

func (b *MySQL80) Ready(ctx context.Context, node *Node) error {
	dsn := fmt.Sprintf("root@tcp(127.0.0.1:%d)/", node.Port)
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(60 * time.Second)
		ctx, _ = context.WithDeadline(ctx, deadline)
	}

	for {
		if ctx.Err() != nil {
			return fmt.Errorf("mysql %s not ready on port %d", node.Name, node.Port)
		}
		db, err := sql.Open("mysql", dsn)
		if err == nil {
			err = db.PingContext(ctx)
			db.Close()
			if err == nil {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// TailLog returns the last n bytes of a node's mysqld log (for diagnostics).
func (b *MySQL80) TailLog(node *Node, maxBytes int) string {
	path := filepath.Join(node.WorkDir, "mysqld.log")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("(no log: %v)", err)
	}
	if len(data) > maxBytes {
		data = data[len(data)-maxBytes:]
	}
	return string(data)
}
