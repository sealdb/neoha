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
	"testing"

	"github.com/sealdb/neoha/internal/base/common"
	"github.com/sealdb/neoha/internal/base/model"
	"github.com/sealdb/neoha/internal/base/nlog"

	"github.com/stretchr/testify/assert"
)

func TestMysqldRPCMonitor(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	endpoint, _, cleanup := MockMysqld(log, port)
	defer cleanup()

	req := model.NewMysqldRPCRequest()
	rsp := model.NewMysqldRPCResponse(model.OK)
	c, ccleanup := MockGetClient(t, endpoint)
	defer ccleanup()

	{
		method := model.RPCMysqldStartMonitor
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)
		want := model.OK
		got := rsp.RetCode
		assert.Equal(t, want, got)
	}

	{
		method := model.RPCMysqldStopMonitor
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)
		want := model.OK
		got := rsp.RetCode
		assert.Equal(t, want, got)
	}
}

func TestMysqldRPCShutDownStartAndStatus(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	endpoint, mysqld, cleanup := MockMysqld(log, port)
	defer cleanup()
	mysqld.backup.cmd = common.NewMockCommand()
	mysqld.MonitorStop()

	// shutdown
	{
		c, ccleanup := MockGetClient(t, endpoint)
		defer ccleanup()

		method := model.RPCMysqldShutDown
		req := model.NewMysqldRPCRequest()
		rsp := model.NewMysqldRPCResponse(model.OK)
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)

		want := model.OK
		got := rsp.RetCode
		assert.Equal(t, want, got)
	}

	// isrunning
	{
		c, ccleanup := MockGetClient(t, endpoint)
		defer ccleanup()

		method := model.RPCMysqldIsRuning
		req := model.NewMysqldRPCRequest()
		rsp := model.NewMysqldRPCResponse(model.OK)
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)

		want := model.OK
		got := rsp.RetCode
		assert.Equal(t, want, got)
	}

	// status
	{
		c, ccleanup := MockGetClient(t, endpoint)
		defer ccleanup()

		method := model.RPCMysqldStatus
		req := model.NewMysqldStatusRPCRequest()
		rsp := model.NewMysqldStatusRPCResponse(model.OK)
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)

		want := &model.MysqldStatusRPCResponse{
			MonitorInfo:  "OFF",
			MysqldInfo:   "UNKNOWN",
			BackupInfo:   "NONE",
			BackupStatus: "NONE",
			RetCode:      "OK",
		}
		rsp.BackupStats = nil
		rsp.MysqldStats = nil
		got := rsp
		assert.Equal(t, want, got)
	}

	// start
	{
		req := model.NewMysqldRPCRequest()
		rsp := model.NewMysqldRPCResponse(model.OK)
		c, ccleanup := MockGetClient(t, endpoint)
		defer ccleanup()

		method := model.RPCMysqldStart
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)

		want := model.OK
		got := rsp.RetCode
		assert.Equal(t, want, got)
	}

	// isrunning
	{
		c, ccleanup := MockGetClient(t, endpoint)
		defer ccleanup()

		method := model.RPCMysqldIsRuning
		req := model.NewMysqldRPCRequest()
		rsp := model.NewMysqldRPCResponse(model.OK)
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)

		want := model.OK
		got := rsp.RetCode
		assert.Equal(t, want, got)
	}

	// kill
	{
		c, ccleanup := MockGetClient(t, endpoint)
		defer ccleanup()

		method := model.RPCMysqldKill
		req := model.NewMysqldRPCRequest()
		rsp := model.NewMysqldRPCResponse(model.OK)
		err := c.Call(method, req, rsp)
		assert.Nil(t, err)

		want := model.OK
		got := rsp.RetCode
		assert.Equal(t, want, got)
	}
}
