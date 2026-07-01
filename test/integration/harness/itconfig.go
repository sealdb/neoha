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
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/sealdb/neoha/internal/config"
	"gopkg.in/yaml.v3"
)

const (
	EnvITConfig           = "NEOHA_IT_CONFIG"
	EnvXtrabackupBinDir   = "NEOHA_IT_XTRABACKUP_BINDIR"
	EnvSSHPort            = "NEOHA_IT_SSH_PORT"
	EnvKeepWorkDir        = "NEOHA_IT_KEEP_WORKDIR"
	EnvTeardownWorkDir    = "NEOHA_IT_TEARDOWN"
	defaultMySQLBase      = "/home/wslu/work/mysql/mysql80-debug"
	defaultXtrabackupBase = "/home/wslu/work/mysql/xtrabackup-8.0.35"
)

// ITConfigFile is the optional integration-test settings file (YAML or JSON).
type ITConfigFile struct {
	MySQLBase        string `yaml:"mysql-base" json:"mysql-base"`
	XtrabackupBinDir string `yaml:"xtrabackup-bindir" json:"xtrabackup-bindir"`
	WorkDir          string `yaml:"workdir" json:"workdir"`
	NeoHABin         string `yaml:"neoha-bin" json:"neoha-bin"`
	NeoHACtlBin      string `yaml:"neohactl-bin" json:"neohactl-bin"`
	PGBase           string `yaml:"pg-base" json:"pg-base"`
	SSHHost          string `yaml:"ssh-host" json:"ssh-host"`
	SSHPort          int    `yaml:"ssh-port" json:"ssh-port"`
	SSHUser          string `yaml:"ssh-user" json:"ssh-user"`
	SSHPasswd        string `yaml:"ssh-passwd" json:"ssh-passwd"`
}

// IntegrationSettings resolves tool paths for integration tests.
// Priority per field: IT config file → environment variable → PATH lookup → built-in default.
type IntegrationSettings struct {
	ConfigPath string
	file       ITConfigFile
}

var (
	settingsOnce sync.Once
	settings     *IntegrationSettings
)

// LoadIntegrationSettings returns cached integration settings.
func LoadIntegrationSettings() *IntegrationSettings {
	settingsOnce.Do(func() {
		settings = &IntegrationSettings{}
		settings.loadFile()
	})
	return settings
}

func (s *IntegrationSettings) loadFile() {
	path := os.Getenv(EnvITConfig)
	if path == "" {
		if root, err := repoRootFromCWD(); err == nil {
			candidate := filepath.Join(root, "test", "integration", "it.local.yaml")
			if _, err := os.Stat(candidate); err == nil {
				path = candidate
			}
		}
	}
	if path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "integration: ignore config %s: %v\n", path, err)
		return
	}
	format, err := config.DetectConfigFormat(path, data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "integration: ignore config %s: %v\n", path, err)
		return
	}
	file := &ITConfigFile{}
	switch format {
	case ".json":
		err = json.Unmarshal(data, file)
	default:
		err = yaml.Unmarshal(data, file)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "integration: ignore config %s: %v\n", path, err)
		return
	}
	s.ConfigPath = path
	s.file = *file
}

func repoRootFromCWD() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		dir = filepath.Dir(dir)
	}
	return "", fmt.Errorf("go.mod not found from cwd")
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func binDirFromPath(name string) string {
	path, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return filepath.Dir(path)
}

func dirExists(dir string) bool {
	if dir == "" {
		return false
	}
	info, err := os.Stat(dir)
	return err == nil && info.IsDir()
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// WorkDir returns the integration test work directory.
func (s *IntegrationSettings) WorkDir() string {
	return firstNonEmpty(s.file.WorkDir, os.Getenv(EnvWorkDir), filepath.Join(os.TempDir(), "neoha-it"))
}

// MySQLBase returns the MySQL installation root (contains bin/mysqld).
func (s *IntegrationSettings) MySQLBase() string {
	return firstNonEmpty(s.file.MySQLBase, os.Getenv(EnvMySQLBase), defaultMySQLBase)
}

func xtrabackupToolDir(dir string) string {
	if fileExists(filepath.Join(dir, "xtrabackup")) {
		return dir
	}
	if fileExists(filepath.Join(dir, "bin", "xtrabackup")) {
		return filepath.Join(dir, "bin")
	}
	return dir
}

// XtrabackupBinDir returns the directory containing xtrabackup and xbstream.
func (s *IntegrationSettings) XtrabackupBinDir() string {
	dir := firstNonEmpty(s.file.XtrabackupBinDir, os.Getenv(EnvXtrabackupBinDir))
	if dir == "" {
		dir = binDirFromPath("xtrabackup")
	}
	if dir == "" {
		dir = defaultXtrabackupBase
	}
	return xtrabackupToolDir(dir)
}

func repoBinIfExists(name string) string {
	root, err := repoRootFromCWD()
	if err != nil {
		return ""
	}
	path := filepath.Join(root, "bin", name)
	if fileExists(path) {
		return path
	}
	return ""
}

// NeoHABinPath returns a pre-built neoha binary path or empty.
// Priority: IT config → NEOHA_IT_BIN → ./bin/neoha (when present).
func (s *IntegrationSettings) NeoHABinPath() string {
	return firstNonEmpty(s.file.NeoHABin, os.Getenv(EnvNeoHABin), repoBinIfExists("neoha"))
}

// NeoHACtlBinPath returns a pre-built neohactl binary path or empty.
// Priority: IT config → NEOHA_IT_CTL_BIN → ./bin/neohactl (when present).
func (s *IntegrationSettings) NeoHACtlBinPath() string {
	return firstNonEmpty(s.file.NeoHACtlBin, os.Getenv(EnvNeoHACtlBin), repoBinIfExists("neohactl"))
}

// KeepWorkDir reports whether integration tests should retain cluster workdirs
// after stop (reuse datadirs on the next run). Default true unless NEOHA_IT_TEARDOWN=1.
func KeepWorkDir() bool {
	if os.Getenv(EnvTeardownWorkDir) == "1" {
		return false
	}
	if v := os.Getenv(EnvKeepWorkDir); v != "" {
		return v != "0" && !strings.EqualFold(v, "false")
	}
	return true
}

func (s *IntegrationSettings) sshHost() string {
	return firstNonEmpty(s.file.SSHHost, "127.0.0.1")
}

func (s *IntegrationSettings) sshPort() int {
	if s.file.SSHPort > 0 {
		return s.file.SSHPort
	}
	if p := os.Getenv(EnvSSHPort); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			return n
		}
	}
	return 22
}

func (s *IntegrationSettings) sshUser() string {
	if u := strings.TrimSpace(s.file.SSHUser); u != "" {
		return u
	}
	if u := os.Getenv("USER"); u != "" {
		return u
	}
	if u, err := user.Current(); err == nil {
		return u.Username
	}
	return ""
}

func (s *IntegrationSettings) sshPasswd() string {
	return s.file.SSHPasswd
}

// ApplyBackupConfig fills database.mysql.backup for integration tests.
// mysqlDataDir is the local mysqld datadir (rebuildme xbstream target).
func (s *IntegrationSettings) ApplyBackupConfig(conf *config.Config, mysqlPort int, mysqlBase, defaultsFile, mysqlDataDir string) {
	if conf == nil || conf.Database == nil || conf.Database.Mysql == nil {
		return
	}
	b := conf.Database.Mysql.Backup
	if b == nil {
		b = config.DefaultBackupConfig()
		conf.Database.Mysql.Backup = b
	}
	b.Host = "127.0.0.1"
	b.Port = mysqlPort
	b.Admin = "root"
	b.Passwd = ""
	b.Basedir = mysqlBase
	b.DefaultsFile = defaultsFile
	b.BackupDir = mysqlDataDir
	b.XtrabackupBinDir = s.XtrabackupBinDir()
	b.SSHHost = s.sshHost()
	b.SSHPort = s.sshPort()
	b.SSHUser = s.sshUser()
	b.SSHPasswd = s.sshPasswd()
}

// RequireMySQL80 skips the test when mysqld is unavailable.
func (s *IntegrationSettings) RequireMySQL80(t *testing.T) (string, string) {
	t.Helper()
	base := s.MySQLBase()
	mysqld := filepath.Join(base, "bin", "mysqld")
	if !fileExists(mysqld) {
		t.Skipf("mysqld not found at %s (set mysql-base in %s or NEOHA_IT_MYSQL_BASE)", mysqld, EnvITConfig)
	}
	return base, mysqld
}

// RequireXtrabackup skips when xtrabackup/xbstream are unavailable.
func (s *IntegrationSettings) RequireXtrabackup(t *testing.T) string {
	t.Helper()
	dir := s.XtrabackupBinDir()
	xb := filepath.Join(dir, "xtrabackup")
	stream := filepath.Join(dir, "xbstream")
	if !fileExists(xb) || !fileExists(stream) {
		t.Skipf("xtrabackup tools not found under %s (set xtrabackup-bindir in %s or NEOHA_IT_XTRABACKUP_BINDIR or PATH)",
			dir, EnvITConfig)
	}
	return dir
}

// RequireSSH skips when localhost SSH is unavailable for xbstream.
func (s *IntegrationSettings) RequireSSH(t *testing.T) {
	t.Helper()
	if !s.SSHAvailable(context.Background()) {
		t.Skipf("SSH to %s@%s:%d unavailable (configure ssh-user/key or ssh-passwd in IT config)",
			s.sshUser(), s.sshHost(), s.sshPort())
	}
}

// SSHAvailable reports whether backup SSH preflight would succeed.
func (s *IntegrationSettings) SSHAvailable(ctx context.Context) bool {
	host := s.sshHost()
	user := s.sshUser()
	port := s.sshPort()
	if user == "" {
		return false
	}
	if passwd := s.sshPasswd(); passwd != "" {
		if _, err := exec.LookPath("sshpass"); err != nil {
			return false
		}
		cmd := exec.CommandContext(ctx, "bash", "-c",
			fmt.Sprintf("sshpass -p %q ssh -o StrictHostKeyChecking=no %s@%s -p %d 'echo 1'",
				passwd, user, host, port))
		return cmd.Run() == nil
	}
	cmd := exec.CommandContext(ctx, "ssh",
		"-o", "BatchMode=yes",
		"-o", "StrictHostKeyChecking=no",
		"-p", fmt.Sprintf("%d", port),
		fmt.Sprintf("%s@%s", user, host),
		"echo", "1")
	return cmd.Run() == nil
}

// XtrabackupToolsPresent reports whether backup binaries exist (no test skip).
func (s *IntegrationSettings) XtrabackupToolsPresent() bool {
	dir := s.XtrabackupBinDir()
	return fileExists(filepath.Join(dir, "xtrabackup")) && fileExists(filepath.Join(dir, "xbstream"))
}
