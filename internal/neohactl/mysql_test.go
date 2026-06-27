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
	"sync/atomic"
	"testing"

	"github.com/sealdb/neoha/internal/base/common"
	"github.com/sealdb/neoha/internal/base/model"
	"github.com/sealdb/neoha/internal/base/nlog"
	"github.com/sealdb/neoha/internal/database/mysql"
	"github.com/sealdb/neoha/internal/election/raft"
	"github.com/sealdb/neoha/internal/server"

	"github.com/stretchr/testify/assert"
)

var mockMysqlPortSeq uint32 = 19200

func mockMysqlAddrs(t *testing.T, log *nlog.Log, selfHandler, fromHandler mysql.MysqlHandler) (self, from string) {
	t.Helper()
	base := int(atomic.AddUint32(&mockMysqlPortSeq, 2)) - 2
	self, _, cleanupSelf := mysql.MockMysql(log, base, selfHandler)
	from, _, cleanupFrom := mysql.MockMysql(log, base+1, fromHandler)
	t.Cleanup(cleanupSelf)
	t.Cleanup(cleanupFrom)
	return self, from
}

func TestGetLocalTrxCount(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))

	t.Run("ok_361", func(t *testing.T) {
		self, from := mockMysqlAddrs(t, log, mysql.NewMockGTIDE1(), mysql.NewMockGTIDF())
		count, err := getLocalTrxCount(self, from)
		assert.Nil(t, err)
		assert.Equal(t, 361, count)
	})

	t.Run("ok_10", func(t *testing.T) {
		self, from := mockMysqlAddrs(t, log, mysql.NewMockGTIDE2(), mysql.NewMockGTIDF())
		count, err := getLocalTrxCount(self, from)
		assert.Nil(t, err)
		assert.Equal(t, 10, count)
	})

	t.Run("ok_1", func(t *testing.T) {
		self, from := mockMysqlAddrs(t, log, mysql.NewMockGTIDE3(), mysql.NewMockGTIDF())
		count, err := getLocalTrxCount(self, from)
		assert.Nil(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("error_from_gtid", func(t *testing.T) {
		self, from := mockMysqlAddrs(t, log, mysql.NewMockGTIDE1(), mysql.NewMockGTIDError())
		count, err := getLocalTrxCount(self, from)
		assert.NotNil(t, err)
		assert.Equal(t, -1, count)
	})

	t.Run("error_self_gtid", func(t *testing.T) {
		self, from := mockMysqlAddrs(t, log, mysql.NewMockGTIDError(), mysql.NewMockGTIDF())
		count, err := getLocalTrxCount(self, from)
		assert.NotNil(t, err)
		assert.Equal(t, -1, count)
	})

	t.Run("error_from_gtid_null", func(t *testing.T) {
		self, from := mockMysqlAddrs(t, log, mysql.NewMockGTIDE1(), mysql.NewMockGTIDNull())
		count, err := getLocalTrxCount(self, from)
		assert.NotNil(t, err)
		assert.Equal(t, -1, count)
	})

	t.Run("error_self_gtid_null", func(t *testing.T) {
		self, from := mockMysqlAddrs(t, log, mysql.NewMockGTIDNull(), mysql.NewMockGTIDF())
		count, err := getLocalTrxCount(self, from)
		assert.NotNil(t, err)
		assert.Equal(t, -1, count)
	})

	t.Run("error_gtid_subtract", func(t *testing.T) {
		self, from := mockMysqlAddrs(t, log, mysql.NewMockGTIDGetGTIDSubtractError(), mysql.NewMockGTIDF())
		count, err := getLocalTrxCount(self, from)
		assert.NotNil(t, err)
		assert.Equal(t, -1, count)
	})
}

func testCLIMysqlCommand(t *testing.T, replMode model.MysqlReplMode) {
	var leader string

	err := createConfig()
	ErrorOK(err)
	defer removeConfig()

	// get leader
	{
		log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
		port := common.RandomPort(6000, 10000)
		servers, scleanup := server.MockServers(log, port, 2, replMode)
		defer scleanup()

		server.MockWaitServerLeaderEggs(servers, 1, replMode)
		for _, server := range servers {
			if server.GetElection().GetRaft().GetState() == raft.LEADER {
				leader = server.Address()
				break
			}
		}
	}

	// setting neoha is leader
	{
		conf, err := GetConfig()
		ErrorOK(err)
		conf.Endpoint = leader

		err = SaveConfig(conf)
		ErrorOK(err)
	}

	// create normal user.
	{
		cmd := NewMysqlCommand()
		_, err := executeCommand(cmd, "createuser", "userxx", "192.168.0.%", "passwdxx", "NO")
		assert.Nil(t, err)
	}

	// create super user.
	{
		cmd := NewMysqlCommand()
		_, err := executeCommand(cmd, "createsuperuser", "192.168.0.%", "userxx", "passwdxx", "NO")
		assert.Nil(t, err)
	}

	// change password
	{
		cmd := NewMysqlCommand()
		_, err := executeCommand(cmd, "changepassword", "userxx", "192.168.0.%", "passwdxx")
		assert.Nil(t, err)
	}

	// get mysql user list
	{
		cmd := NewMysqlCommand()
		_, err := executeCommand(cmd, "getuser")
		assert.Nil(t, err)
	}

	// drop normal user.
	{
		cmd := NewMysqlCommand()
		_, err := executeCommand(cmd, "dropuser", "userxx", "192.168.0.%")
		assert.Nil(t, err)
	}

	// set global sysvar
	{
		cmd := NewMysqlCommand()
		_, err := executeCommand(cmd, "sysvar", "SET GLOBAL GTID_MODE='ON'")
		assert.Nil(t, err)
	}

	// kill mysql
	{
		cmd := NewMysqlCommand()
		_, err := executeCommand(cmd, "kill")
		assert.Nil(t, err)
	}

	// status
	{
		cmd := NewMysqlCommand()
		_, err := executeCommand(cmd, "status")
		assert.Nil(t, err)
	}

	// create user with privileges
	{
		cmd := NewMysqlCommand()
		_, err := executeCommand(cmd, "createuserwithgrants", "--user", "xx", "--passwd", "xx", "--database", "db1", "--host", "192.168.0.%", "--privs", "SELECT,DROP")
		assert.Nil(t, err)
	}
}

func TestCLIMysqlCommand_SemiSync(t *testing.T) {
	testCLIMysqlCommand(t, model.ReplModeSemiSync)
}

func TestCLIMysqlCommand_MGR(t *testing.T) {
	testCLIMysqlCommand(t, model.ReplModeMGR)
}
