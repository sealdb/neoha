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
	"os"
	"path/filepath"
	"testing"

	"github.com/sealdb/neoha/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestIntegrationSettingsEnvOverride(t *testing.T) {
	t.Setenv(EnvMySQLBase, "/custom/mysql")
	t.Setenv(EnvXtrabackupBinDir, "/custom/xtrabackup")

	s := &IntegrationSettings{}
	assert.Equal(t, "/custom/mysql", s.MySQLBase())
	assert.Equal(t, "/custom/xtrabackup", s.XtrabackupBinDir())
}

func TestIntegrationSettingsFileOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "it.yaml")
	content := []byte(`mysql-base: /from/file/mysql
xtrabackup-bindir: /from/file/xb
ssh-host: 10.0.0.1
ssh-port: 2222
ssh-user: backup
`)
	assert.NoError(t, os.WriteFile(path, content, 0o644))

	s := &IntegrationSettings{}
	t.Setenv(EnvITConfig, path)
	s.loadFile()

	assert.Equal(t, path, s.ConfigPath)
	assert.Equal(t, "/from/file/mysql", s.MySQLBase())
	assert.Equal(t, "/from/file/xb", s.XtrabackupBinDir())
	assert.Equal(t, "10.0.0.1", s.sshHost())
	assert.Equal(t, 2222, s.sshPort())
	assert.Equal(t, "backup", s.sshUser())
}

func TestApplyBackupConfig(t *testing.T) {
	s := &IntegrationSettings{
		file: ITConfigFile{
			XtrabackupBinDir: "/opt/xtrabackup",
			SSHHost:          "127.0.0.1",
			SSHPort:          22,
			SSHUser:          "dba",
		},
	}
	conf := defaultNeoHAConfigForTest()
	s.ApplyBackupConfig(conf, 3306, "/mysql", "/etc/my.cnf", "/data/mysql")
	b := conf.Database.Mysql.Backup
	assert.Equal(t, "127.0.0.1", b.Host)
	assert.Equal(t, 3306, b.Port)
	assert.Equal(t, "/opt/xtrabackup", b.XtrabackupBinDir)
	assert.Equal(t, "dba", b.SSHUser)
	assert.Equal(t, "/data/mysql", b.BackupDir)
}

func defaultNeoHAConfigForTest() *config.Config {
	conf := config.DefaultConfig()
	return conf
}
