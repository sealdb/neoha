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
	"strings"
	"testing"

	"github.com/sealdb/neoha/internal/config"

	"github.com/stretchr/testify/assert"
)

func TestLinuxStartArgs(t *testing.T) {
	conf := config.DefaultBackupConfig()
	linuxargs := NewLinuxArgs(conf)
	want := `-c /u01/mysql_20221010/bin/mysqld_safe --defaults-file=/etc/my3306.cnf > /dev/null&`
	got := strings.Join(linuxargs.Start(), " ")
	assert.Equal(t, want, got)
}

func TestLinuxStopArgs(t *testing.T) {
	conf := config.DefaultBackupConfig()
	linuxargs := NewLinuxArgs(conf)

	// 1. passwords is null
	{
		want := `-c /u01/mysql_20221010/bin/mysqladmin -hlocalhost -uroot -P3306 shutdown`
		got := strings.Join(linuxargs.Stop(), " ")
		assert.Equal(t, want, got)
	}

	// 2. with passwords
	{
		conf.Passwd = `ddd"`
		want := `-c /u01/mysql_20221010/bin/mysqladmin -hlocalhost -uroot -pddd" -P3306 shutdown`
		got := strings.Join(linuxargs.Stop(), " ")
		assert.Equal(t, want, got)
	}
}

func TestLinuxIsRunningArgs(t *testing.T) {
	linuxargs := NewLinuxArgs(config.DefaultBackupConfig())
	//want := `-c echo $(ps aux | grep '[m]ysqld_safe --defaults-file=/etc/my3306.cnf' | wc -l)`
	want := `-c ps aux | grep '[m]ysqld_safe --defaults-file=/etc/my3306.cnf' | wc -l`
	got := strings.Join(linuxargs.IsRunning(), " ")
	assert.Equal(t, want, got)
}

func TestLinuxKillArgs(t *testing.T) {
	linuxargs := NewLinuxArgs(config.DefaultBackupConfig())
	want := `-c $(ps aux | grep '[-]-defaults-file=/etc/my3306.cnf' | grep -v grep | awk '{print $2}' | xargs kill -9)`
	//want := `-c "kill -9 $(ps aux | grep '[-]-defaults-file=/etc/my3306.cnf' | awk '{print $2}')"`
	got := strings.Join(linuxargs.Kill(), " ")
	assert.Equal(t, want, got)
}
