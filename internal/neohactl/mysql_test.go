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
	"github.com/sealdb/neoha/internal/database/mysql"
	"github.com/sealdb/neoha/internal/election/raft"
	"github.com/sealdb/neoha/internal/server"

	"github.com/stretchr/testify/assert"
)

func TestGetLocalTrxCount(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))

	// ok
	{
		// setGTID: c78e798a-cccc-cccc-cccc-525433e8e796:1-10, df24366e-inva-bbbb-bbbb-525433b6dbaa:1-30
		port := common.RandomPort(8100, 8200)
		from, _, cleanup2 := mysql.MockMysql(log, port, mysql.NewMockGTIDF())
		defer cleanup2()

		// subsetGTID: c78e798a-cccc-cccc-cccc-525433e8e796:1-200, df24366e-inva-bbbb-bbbb-525433b6dbaa:1-200, ef24366e-aaaa-aaaa-aaaa-525433b6deee:100
		// result: c78e798a-cccc-cccc-cccc-525433e8e796:11-200,\ndf24366e-inva-bbbb-bbbb-525433b6dbaa:31-200,\nef24366e-aaaa-aaaa-aaaa-525433b6deee:100
		port = common.RandomPort(8000, 8100)
		self, _, cleanup1 := mysql.MockMysql(log, port, mysql.NewMockGTIDE1())
		defer cleanup1()
		count, err := getLocalTrxCount(self, from)
		assert.Nil(t, err)
		assert.Equal(t, 361, count)

		// subsetGTID: c78e798a-cccc-cccc-cccc-525433e8e796:1-10, df24366e-inva-bbbb-bbbb-525433b6dbaa:1-40
		// result: df24366e-inva-bbbb-bbbb-525433b6dbaa:31-40
		port = common.RandomPort(8000, 8100)
		self, _, cleanup1 = mysql.MockMysql(log, port, mysql.NewMockGTIDE2())
		defer cleanup1()
		count, err = getLocalTrxCount(self, from)
		assert.Nil(t, err)
		assert.Equal(t, 10, count)

		// subsetGTID: df24366e-inva-bbbb-bbbb-525433b6dbaa:1-31
		// result: df24366e-inva-bbbb-bbbb-525433b6dbaa:31
		port = common.RandomPort(8000, 8100)
		self, _, cleanup1 = mysql.MockMysql(log, port, mysql.NewMockGTIDE3())
		defer cleanup1()
		count, err = getLocalTrxCount(self, from)
		assert.Nil(t, err)
		assert.Equal(t, 1, count)
	}

	// error
	{
		// get setGTID error
		{
			port := common.RandomPort(8000, 8100)
			self, _, cleanup1 := mysql.MockMysql(log, port, mysql.NewMockGTIDE1())
			defer cleanup1()
			port = common.RandomPort(8100, 8200)
			from, _, cleanup2 := mysql.MockMysql(log, port, mysql.NewMockGTIDError())
			defer cleanup2()
			count, err := getLocalTrxCount(self, from)
			assert.NotNil(t, err)
			assert.Equal(t, -1, count)
		}

		// get subsetGTID error
		{
			port := common.RandomPort(8000, 8100)
			self, _, cleanup1 := mysql.MockMysql(log, port, mysql.NewMockGTIDError())
			defer cleanup1()
			port = common.RandomPort(8100, 8200)
			from, _, cleanup2 := mysql.MockMysql(log, port, mysql.NewMockGTIDF())
			defer cleanup2()
			count, err := getLocalTrxCount(self, from)
			assert.NotNil(t, err)
			assert.Equal(t, -1, count)
		}

		// from.Executed_GTID_Set is null
		{
			port := common.RandomPort(8000, 8100)
			self, _, cleanup1 := mysql.MockMysql(log, port, mysql.NewMockGTIDE1())
			defer cleanup1()
			port = common.RandomPort(8100, 8200)
			from, _, cleanup2 := mysql.MockMysql(log, port, mysql.NewMockGTIDNull())
			defer cleanup2()
			count, err := getLocalTrxCount(self, from)
			assert.NotNil(t, err)
			assert.Equal(t, -1, count)
		}

		// self.Executed_GTID_Set is null
		{
			port := common.RandomPort(8000, 8100)
			self, _, cleanup1 := mysql.MockMysql(log, port, mysql.NewMockGTIDNull())
			defer cleanup1()
			port = common.RandomPort(8100, 8200)
			from, _, cleanup2 := mysql.MockMysql(log, port, mysql.NewMockGTIDF())
			defer cleanup2()
			count, err := getLocalTrxCount(self, from)
			assert.NotNil(t, err)
			assert.Equal(t, -1, count)
		}

		// GetGTIDSubtract error
		{
			port := common.RandomPort(8000, 8100)
			self, _, cleanup1 := mysql.MockMysql(log, port, mysql.NewMockGTIDGetGTIDSubtractError())
			defer cleanup1()
			port = common.RandomPort(8100, 8200)
			from, _, cleanup2 := mysql.MockMysql(log, port, mysql.NewMockGTIDF())
			defer cleanup2()
			count, err := getLocalTrxCount(self, from)
			assert.NotNil(t, err)
			assert.Equal(t, -1, count)
		}
	}

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
