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

package v1

import (
	"encoding/base64"
	"github.com/sealdb/neoha/internal/base/model"
	"strings"
	"testing"

	"github.com/sealdb/neoha/internal/base/common"
	"github.com/sealdb/neoha/internal/base/nlog"
	"github.com/sealdb/neoha/internal/server"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/ant0ine/go-json-rest/rest/test"
	"github.com/stretchr/testify/assert"
)

func testCtlV1Raft(t *testing.T, replMode model.MysqlReplMode) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	port := common.RandomPort(8000, 9000)
	servers, cleanup := server.MockServers(log, port, 1, replMode)
	defer cleanup()

	neoha := servers[0]
	api := rest.NewApi()
	authMiddleware := &rest.AuthBasicMiddleware{
		Realm: "neoha zone",
		Authenticator: func(userId string, password string) bool {
			if userId == neoha.MySQLAdmin() && password == neoha.MySQLPasswd() {
				return true
			}
			return false
		},
	}
	api.Use(authMiddleware)

	router, _ := rest.MakeRouter(
		rest.Get("/v1/raft/status", RaftStatusHandler(log, neoha)),
		rest.Post("/v1/raft/trytoleader", RaftTryToLeaderHandler(log, neoha)),
		rest.Put("/v1/raft/disablechecksemisync", RaftDisableCheckSemiSyncHandler(log, neoha)),
		rest.Put("/v1/raft/disable", RaftDisableHandler(log, neoha)),
	)
	api.SetApp(router)
	handler := api.MakeHandler()

	// status 401.
	{
		req := test.MakeSimpleRequest("GET", "http://localhost/v1/raft/status", nil)
		recorded := test.RunRequest(t, handler, req)
		recorded.CodeIs(401)
	}

	// status 200.
	{
		req := test.MakeSimpleRequest("GET", "http://localhost/v1/raft/status", nil)
		encoded := base64.StdEncoding.EncodeToString([]byte("root:"))
		req.Header.Set("Authorization", "Basic "+encoded)
		recorded := test.RunRequest(t, handler, req)
		recorded.CodeIs(200)
		got := recorded.Recorder.Body.String()
		log.Debug(got)
		assert.True(t, strings.Contains(got, `"state":"FOLLOWER"`))
	}

	// trytoleader.
	{
		req := test.MakeSimpleRequest("POST", "http://localhost/v1/raft/trytoleader", nil)
		encoded := base64.StdEncoding.EncodeToString([]byte("root:"))
		req.Header.Set("Authorization", "Basic "+encoded)
		recorded := test.RunRequest(t, handler, req)
		recorded.CodeIs(200)
	}

	// status 200.
	{
		req := test.MakeSimpleRequest("GET", "http://localhost/v1/raft/status", nil)
		encoded := base64.StdEncoding.EncodeToString([]byte("root:"))
		req.Header.Set("Authorization", "Basic "+encoded)
		recorded := test.RunRequest(t, handler, req)
		recorded.CodeIs(200)
		got := recorded.Recorder.Body.String()
		log.Debug(got)
		assert.True(t, strings.Contains(got, `"state":"CANDIDATE"`))
	}

	// disablechecksemisync 200.
	{
		req := test.MakeSimpleRequest("PUT", "http://localhost/v1/raft/disablechecksemisync", nil)
		encoded := base64.StdEncoding.EncodeToString([]byte("root:"))
		req.Header.Set("Authorization", "Basic "+encoded)
		recorded := test.RunRequest(t, handler, req)
		recorded.CodeIs(200)
	}

	// disable 200.
	{
		req := test.MakeSimpleRequest("PUT", "http://localhost/v1/raft/disable", nil)
		encoded := base64.StdEncoding.EncodeToString([]byte("root:"))
		req.Header.Set("Authorization", "Basic "+encoded)
		recorded := test.RunRequest(t, handler, req)
		recorded.CodeIs(200)
	}
}

func TestCtlV1Raft_MySQL_SemiSync(t *testing.T) {
	testCtlV1Raft(t, model.ReplModeSemiSync)
}

func TestCtlV1Raft_MySQL_MGR(t *testing.T) {
	testCtlV1Raft(t, model.ReplModeMGR)
}
