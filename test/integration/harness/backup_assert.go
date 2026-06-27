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
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
)

// ExecSQL runs a statement on local mysqld (root, no password).
func ExecSQL(port int, query string) error {
	dsn := fmt.Sprintf("root@tcp(127.0.0.1:%d)/", port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec(query)
	return err
}

// QueryScalar runs a query returning a single int value on local mysqld.
func QueryScalar(port int, query string) (int, error) {
	dsn := fmt.Sprintf("root@tcp(127.0.0.1:%d)/", port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return 0, err
	}
	defer db.Close()
	var v int
	err = db.QueryRow(query).Scan(&v)
	return v, err
}

func PrepareBackupDir(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	return os.MkdirAll(dir, 0o755)
}

// AssertBackupArtifacts checks xtrabackup stream output exists after backup+prepare.
func AssertBackupArtifacts(t *testing.T, dir string) {
	t.Helper()
	assert.FileExists(t, filepath.Join(dir, "xtrabackup_checkpoints"))
	assert.FileExists(t, filepath.Join(dir, "xtrabackup_info"))
}
