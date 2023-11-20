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
	"github.com/sealdb/neoha/internal/base/model"
	"testing"

	"github.com/sealdb/neoha/internal/base/common"
	"github.com/sealdb/neoha/internal/base/nlog"
	"github.com/sealdb/neoha/internal/server"

	"github.com/stretchr/testify/assert"
)

func testCLINeoHACommand(t *testing.T, replMode model.MysqlReplMode) {

	err := createConfig()
	ErrorOK(err)
	defer removeConfig()

	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	servers, cleanup := server.MockServers(log, port, 1, replMode)
	defer cleanup()

	// setting neoha is leader
	{
		conf, err := GetConfig()
		ErrorOK(err)
		conf.Endpoint = servers[0].Address()
		err = SaveConfig(conf)
		ErrorOK(err)
	}

	// ping
	{
		cmd := NewNeoHACommand()
		_, err := executeCommand(cmd, "ping")
		assert.Nil(t, err)
	}
}

func TestCLINeoHACommand_MySQL_SemiSync(t *testing.T) {
	testCLINeoHACommand(t, model.ReplModeSemiSync)
}

func TestCLINeoHACommand_MySQL_MGR(t *testing.T) {
	testCLINeoHACommand(t, model.ReplModeMGR)
}
