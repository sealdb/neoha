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

package raft

const (
	bash = "bash"
)

// leaderStartShellCommand execute the shell commands
// when leader start, such as START-VIP command
func (r *Raft) leaderStartShellCommand() error {
	args := []string{
		"-c",
		r.conf.LeaderStartCommand,
	}

	if out, err := r.cmd.RunCommand(bash, args); err != nil {
		r.ERROR("leaderStartShellCommand[%v].out[%v].error[%+v]", args, out, err)
		return err
	}
	r.WARNING("leaderStartShellCommand[%v].done", args)
	return nil
}

// leaderStopShellCommand executes the shell commands
// when leader stop, such as STOP-VIP command
func (r *Raft) leaderStopShellCommand() error {
	args := []string{
		"-c",
		r.conf.LeaderStopCommand,
	}

	if out, err := r.cmd.RunCommand(bash, args); err != nil {
		r.ERROR("leaderStopShellCommand[%v].out[%v].error[%+v]", args, out, err)
		return err
	}
	r.WARNING("leaderStopShellCommand[%v].done", args)
	return nil
}
