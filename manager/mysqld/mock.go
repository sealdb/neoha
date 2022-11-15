/*
 * Copyright 2022-2025 The NeoHA Authors.
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

	"neoha/base/common"
	"neoha/base/nlog"
	"neoha/base/nrpc"
	"neoha/config"

	"github.com/stretchr/testify/assert"
)

var (
	_ ArgsHandler = &MockArgs{}
)

// mock mysqld with rpc server
func setupRPC(rpc *nrpc.Service, mysqld *Mysqld) {
	if err := rpc.RegisterService(mysqld.GetBackupRPC()); err != nil {
		mysqld.log.Panic("server.rpc.RegisterService.GetBackupRPC.error[%v]", err)
	}
	if err := rpc.RegisterService(mysqld.GetMysqldRPC()); err != nil {
		mysqld.log.Panic("server.rpc.RegisterService.GetMysqldRPC.error[%v]", err)
	}
}

// MockMysqld used to mock a mysqld.
func MockMysqld(log *nlog.Log, port int) (string, *Mysqld, func()) {
	id := fmt.Sprintf("127.0.0.1:%d", port)
	conf := config.DefaultBackupConfig()
	mysqld := NewMysqld(conf, log)
	mysqld.SetArgsHandler(NewMockArgs())
	mysqld.backup.SetCMDHandler(common.NewMockCommand())

	// setup rpc
	rpc, err := nrpc.NewService(nrpc.Log(log),
		nrpc.ConnectionStr(id))
	if err != nil {
		log.Panic("mysqldRPC.NewService.error[%v]", err)
	}
	setupRPC(rpc, mysqld)
	rpc.Start()

	return id, mysqld, func() {
		rpc.Stop()
	}
}

// MockGetClient used to mock client.
func MockGetClient(t *testing.T, svrConn string) (*nrpc.Client, func()) {
	client, err := nrpc.NewClient(svrConn, 100)
	assert.Nil(t, err)

	return client, func() {
		client.Close()
	}
}

// MockArgs tuple.
type MockArgs struct {
	ArgsHandler
}

// NewMockArgs creates the new MockArgs.
func NewMockArgs() *MockArgs {
	return &MockArgs{}
}

// Start used to start the mock.
func (l *MockArgs) Start() []string {
	args := []string{
		"-c",
		"ls -l",
	}

	return args
}

// Stop used to stop the mock.
func (l *MockArgs) Stop() []string {
	args := []string{
		"-c",
		"ls -l",
	}

	return args
}

// IsRunning used to check the mock running.
func (l *MockArgs) IsRunning() []string {
	args := []string{
		"-c",
		"ls -l",
	}

	return args
}

// Kill used to kill the mock.
func (l *MockArgs) Kill() []string {
	args := []string{
		"-c",
		"ls -l",
	}
	return args
}
