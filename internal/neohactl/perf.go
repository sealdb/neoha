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

package neohactl

import (
	"encoding/json"
	"fmt"

	"github.com/sealdb/neoha/internal/base/common"
	"github.com/sealdb/neoha/internal/base/nlog"

	"github.com/spf13/cobra"
)

func quickStack(log *nlog.Log) (string, error) {
	timeout := 10 * 1000 // 10s
	cmds := "bash"
	args := []string{
		"-c",
		"sudo quickstack -s -k 10 -p `pidof mysqld`",
	}

	cmd := common.NewLinuxCommand(log)
	return cmd.RunCommandWithTimeout(timeout, cmds, args)
}

func NewPerfCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "perf <subcommand>",
		Short: "perf related commands",
	}

	cmd.AddCommand(NewQuickStackCommand())

	return cmd
}

func NewQuickStackCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "quickstack",
		Short: "capture the stack of mysqld using quickstack",
		Run:   quickStackCommandFn,
	}
	cmd.AddCommand(NewQuickStackJsonCommand())

	return cmd
}

func quickStackCommandFn(cmd *cobra.Command, args []string) {
	outs, err := quickStack(log)
	ErrorOK(err)
	fmt.Printf("%v", outs)
}

func NewQuickStackJsonCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "json",
		Short: "json format",
		Run:   quickStackJsonCommandFn,
	}

	return cmd
}

func quickStackJsonCommandFn(cmd *cobra.Command, args []string) {
	type Status struct {
		Results string `json:"status"`
	}
	status := &Status{}
	outs, err := quickStack(log)
	ErrorOK(err)
	status.Results = outs

	statusB, _ := json.Marshal(status)
	fmt.Printf("%s", string(statusB))
}
