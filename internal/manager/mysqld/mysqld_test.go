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
	"testing"
	"time"

	"github.com/sealdb/neoha/internal/base/common"
	"github.com/sealdb/neoha/internal/base/model"
	"github.com/sealdb/neoha/internal/base/nlog"
	"github.com/sealdb/neoha/internal/config"

	"github.com/stretchr/testify/assert"
)

func TestStartMysqld(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	mysqld := NewMysqld(config.DefaultBackupConfig(), log)
	err := mysqld.StartMysqld()
	assert.Nil(t, err)
}

func TestStopMysqld(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	conf := config.DefaultBackupConfig()
	mysqld := NewMysqld(conf, log)
	err := mysqld.StopMysqld()
	assert.Nil(t, err)
}

func TestKillMysqld(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.WARNING))
	conf := config.DefaultBackupConfig()
	mysqld := NewMysqld(conf, log)

	// mock a mysqld running
	go func() {
		args := []string{
			"-c",
			fmt.Sprintf("watch -n 0.1 -d 'echo --defaults-file=%v'", conf.DefaultsFile)}
		common.RunCommand("bash", args...)
		//o, _ := common.RunCommand("bash", args...)
		//log.Warning("watch command output: [%+v]", o)
	}()

	// Wait for watch process to start, only for github workflows
	time.Sleep(time.Duration(50 * time.Millisecond))

	err := mysqld.KillMysqld()
	assert.Nil(t, err)
}

func TestIsMysqldRunning(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	conf := config.DefaultBackupConfig()
	mysqld := NewMysqld(conf, log)

	// mock a mysqld running
	go func() {
		args := []string{
			"-c",
			fmt.Sprintf("watch -d 'mysqld_safe --defaults-file=%v'", conf.DefaultsFile)}
		common.RunCommand("bash", args...)
	}()

	// Wait for watch process to start, only for github workflows
	time.Sleep(time.Duration(50 * time.Millisecond))

	want := true
	got := mysqld.isMysqldRunning()
	assert.Equal(t, want, got)
}

func TestMonitor(t *testing.T) {
	conf := config.DefaultBackupConfig()
	// 100ms
	conf.MysqldMonitorInterval = 100
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	conf.DefaultsFile = "/etc/my.cnf"
	mysqld := NewMysqld(conf, log)

	{
		want := model.MYSQLD_NOTRUNNING
		got := mysqld.getStatus()
		assert.Equal(t, want, got)
		mysqld.MonitorStart()
		time.Sleep(500 * time.Millisecond)

		// TODO: expect running
		want = model.MYSQLD_NOTRUNNING
		got = mysqld.getStatus()
		assert.Equal(t, want, got)
	}

	{
		mysqld.MonitorStop()

		wantstatus := model.MYSQLD_UNKNOWN
		gotstatus := mysqld.getStatus()
		assert.Equal(t, wantstatus, gotstatus)

		want := false
		got := mysqld.monitorRunning
		assert.Equal(t, want, got)
	}

	{
		want := false
		got := mysqld.monitorRunning
		assert.Equal(t, want, got)
		mysqld.MonitorStart()
		time.Sleep(500 * time.Millisecond)

		// TODO: expect running
		wantstatus := model.MYSQLD_NOTRUNNING
		gotstatus := mysqld.getStatus()
		assert.Equal(t, wantstatus, gotstatus)
	}
}
