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
	"fmt"

	"github.com/sealdb/neoha/internal/neorpc"

	"github.com/spf13/cobra"
)

func NewNeoHACommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "neoha <subcommand>",
		Short: "neoha related commands",
	}

	cmd.AddCommand(NewNeoHAStatusCommand())

	return cmd
}

func NewNeoHAStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ping",
		Short: "check node work or not",
		Run:   neohaPingCommandFn,
	}

	return cmd
}

func neohaPingCommandFn(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		ErrorOK(fmt.Errorf("too.many.args"))
	}

	// send ping to self
	{
		conf, err := GetConfig()
		ErrorOK(err)
		self := conf.Endpoint
		rsp, err := neorpc.ServerPingRPC(self)
		ErrorOK(err)
		RspOK(rsp.RetCode)
	}
}
