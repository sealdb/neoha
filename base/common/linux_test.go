/*
 * Copyright 2022 The NeoHA Authors.
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

package common

import (
	"neoha/base/nlog"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunCommandWithTimeout(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	timeout := 5 * 1000
	cmds := "ls"
	args := []string{
		"-l",
	}
	_, err := runCommandWithTimeout(log, timeout, cmds, args...)
	assert.Nil(t, err)

	timeout = 1 * 1000
	cmds = "sleep"
	args = []string{
		"2",
	}
	_, err = runCommandWithTimeout(log, timeout, cmds, args...)
	assert.NotNil(t, err)
}

func TestRunCommand(t *testing.T) {
	cmds := "ls"
	args := []string{
		"-l",
	}
	_, err := RunCommand(cmds, args...)
	assert.Nil(t, err)
}

func TestCommand(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	cmds := "bash"
	cmd := NewLinuxCommand(log)
	var err error
	errSleepMsg := "sleep: missing operand" // for linux
	if runtime.GOOS == "darwin" {
		errSleepMsg = "usage: sleep seconds"
	}

	tests := []struct {
		name     string
		args     []string
		needScan bool
		timeout  int
		wantErr  bool
	}{
		{name: "sleep", args: []string{"-c", "sleep"}, needScan: true, timeout: 0, wantErr: false},
		{name: "sleepKill", args: []string{"-c", "'sleep 10'"}, needScan: false, timeout: 0, wantErr: false},
		{name: "sleepWithTimeout", args: []string{"-c", "'sleep 30'"}, needScan: false, timeout: 1, wantErr: true},
		{name: "ls", args: []string{"-c", "ls -l"}, needScan: false, timeout: 0, wantErr: false},
		{name: "lsWithTimeout", args: []string{"-c", "ls -l"}, needScan: false, timeout: 100, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.timeout == 0 {
				err = cmd.Run(cmds, tt.args)
			} else {
				_, err = cmd.RunCommandWithTimeout(tt.timeout, cmds, tt.args)
			}
			assert.Equal(t, err != nil, tt.wantErr)
			if tt.name == "sleepKill" {
				cmd.Kill()
			}
			if tt.needScan {
				err = cmd.Scan(errSleepMsg, 1)
			} else {
				err = cmd.Scan(errSleepMsg, 0)
			}
			assert.Nil(t, err)
		})
	}
}
