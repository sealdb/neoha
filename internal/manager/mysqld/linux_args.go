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

package mysqld

import (
	"fmt"
	"github.com/sealdb/neoha/internal/config"
	"path/filepath"
)

var (
	_ ArgsHandler = &LinuxArgs{}
)

const (
	bash       = "bash"
	mysqldsafe = "bin/mysqld_safe"
	mysqladmin = "bin/mysqladmin"
)

// LinuxArgs tuple.
type LinuxArgs struct {
	conf *config.BackupConfig
	ArgsHandler
}

// NewLinuxArgs creates new LinuxArgs.
func NewLinuxArgs(conf *config.BackupConfig) *LinuxArgs {
	return &LinuxArgs{
		conf: conf,
	}
}

// Start used to start mysqld.
func (l *LinuxArgs) Start() []string {
	safe57 := filepath.Join(l.conf.Basedir, mysqldsafe)
	args := []string{
		"-c",
		fmt.Sprintf("%s --defaults-file=%s > /dev/null&", safe57, l.conf.DefaultsFile),
	}
	return args
}

// Stop used to stop the mysqld.
func (l *LinuxArgs) Stop() []string {
	admin57 := filepath.Join(l.conf.Basedir, mysqladmin)
	args := []string{
		"-c",
	}
	if l.conf.Passwd == "" {
		args = append(args, fmt.Sprintf("%s -h%s -u%s -P%d shutdown", admin57, l.conf.Host, l.conf.Admin, l.conf.Port))
	} else {
		args = append(args, fmt.Sprintf("%s -h%s -u%s -p%s -P%d shutdown", admin57, l.conf.Host, l.conf.Admin, l.conf.Passwd, l.conf.Port))
	}
	return args
}

// IsRunning used to check the mysqld is running or not.
func (l *LinuxArgs) IsRunning() []string {
	// [m] is a trick to stop you picking up the actual grep process itself
	safe57 := fmt.Sprintf("[m]ysqld_safe --defaults-file=%s", l.conf.DefaultsFile)
	args := []string{
		"-c",
		//fmt.Sprintf("echo $(ps aux | grep '%s' | wc -l)", safe57),
		fmt.Sprintf("ps aux | grep '%s' | wc -l", safe57),
	}
	return args
}

// Kill used to kill -9 the mysqld process.
func (l *LinuxArgs) Kill() []string {
	args := []string{
		"-c",
	}
	//args = append(args,
	//	fmt.Sprintf("\"$(ps aux | grep '[-]-defaults-file=%s' | grep -v grep | awk '{print $2}' | xargs kill -9)\"", l.conf.DefaultsFile))
	args = append(args,
		fmt.Sprintf("$(ps aux | grep '[-]-defaults-file=%s' | grep -v grep | awk '{print $2}' | xargs kill -9)", l.conf.DefaultsFile))
	//args = append(args,
	//	fmt.Sprintf("\"kill -9 $(ps aux | grep '[-]-defaults-file=%s' | awk '{print $2}')\"", l.conf.DefaultsFile))
	return args
}
