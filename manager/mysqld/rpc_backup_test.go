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
	"testing"

	"neoha/base/common"
	"neoha/base/model"
	"neoha/base/nlog"

	"github.com/stretchr/testify/assert"
)

func TestBackupRPCBackupDo(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	endpoint, mysqld, cleanup := MockMysqld(log, port)
	defer cleanup()

	mysqld.backup.cmd = common.NewMockCommand()

	// do backup
	{
		go func() {
			c, _ := MockGetClient(t, endpoint)
			method := model.RPCBackupDo
			req := model.NewBackupRPCRequest()
			req.SSHHost = "127.0.0.1"
			req.SSHUser = "backup"
			req.SSHPasswd = "backup"
			req.SSHPort = 22

			rsp := model.NewBackupRPCResponse(model.OK)
			err := c.Call(method, req, rsp)
			assert.Nil(t, err)
			want := model.OK
			got := rsp.RetCode
			assert.Equal(t, want, got)
		}()
	}

	// cancel
	{
		req := model.NewBackupRPCRequest()
		rsp := model.NewBackupRPCResponse(model.OK)
		c, _ := MockGetClient(t, endpoint)
		{
			method := model.RPCBackupCancel
			err := c.Call(method, req, rsp)
			assert.Nil(t, err)
			want := model.OK
			got := rsp.RetCode
			assert.Equal(t, want, got)
		}
	}
}

func TestBackupRPCApplyLog(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	endpoint, mysqld, cleanup := MockMysqld(log, port)
	defer cleanup()
	mysqld.backup.cmd = common.NewMockCommand()
	mysqld.MonitorStop()

	{
		req := model.NewBackupRPCRequest()
		rsp := model.NewBackupRPCResponse(model.OK)
		c, _ := MockGetClient(t, endpoint)

		go func() {
			method := model.RPCBackupApplyLog
			err := c.Call(method, req, rsp)
			assert.Nil(t, err)
			want := model.OK
			got := rsp.RetCode
			assert.Equal(t, want, got)
		}()
	}

	{
		req := model.NewBackupRPCRequest()
		rsp := model.NewBackupRPCResponse(model.OK)
		c, _ := MockGetClient(t, endpoint)

		{
			method := model.RPCBackupCancel
			err := c.Call(method, req, rsp)
			assert.Nil(t, err)
			want := model.OK
			got := rsp.RetCode
			assert.Equal(t, want, got)
		}
	}
}
