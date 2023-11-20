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
	"testing"

	"github.com/sealdb/neoha/internal/base/common"
	"github.com/sealdb/neoha/internal/base/nlog"
	"github.com/sealdb/neoha/internal/server"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/ant0ine/go-json-rest/rest/test"
)

func testCtlV1ClusterAddRemove(t *testing.T, replMode model.MysqlReplMode) {
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
		rest.Post("/v1/cluster/add", ClusterAddHandler(log, neoha)),
		rest.Post("/v1/cluster/remove", ClusterRemoveHandler(log, neoha)),
	)
	api.SetApp(router)
	handler := api.MakeHandler()

	p := &peerParams{
		Address: "192.168.0.1:8080",
	}

	// 500.
	{
		req := test.MakeSimpleRequest("POST", "http://localhost/v1/cluster/add", nil)
		encoded := base64.StdEncoding.EncodeToString([]byte("root:"))
		req.Header.Set("Authorization", "Basic "+encoded)
		recorded := test.RunRequest(t, handler, req)
		recorded.CodeIs(500)
	}

	// 500.
	{

		req := test.MakeSimpleRequest("POST", "http://localhost/v1/cluster/add", &peerParams{})
		encoded := base64.StdEncoding.EncodeToString([]byte("root:"))
		req.Header.Set("Authorization", "Basic "+encoded)
		recorded := test.RunRequest(t, handler, req)
		recorded.CodeIs(500)
	}

	// 200.
	{
		req := test.MakeSimpleRequest("POST", "http://localhost/v1/cluster/add", p)
		encoded := base64.StdEncoding.EncodeToString([]byte("root:"))
		req.Header.Set("Authorization", "Basic "+encoded)
		recorded := test.RunRequest(t, handler, req)
		recorded.CodeIs(200)
	}

	// 500.
	{
		req := test.MakeSimpleRequest("POST", "http://localhost/v1/cluster/remove", nil)
		encoded := base64.StdEncoding.EncodeToString([]byte("root:"))
		req.Header.Set("Authorization", "Basic "+encoded)
		recorded := test.RunRequest(t, handler, req)
		recorded.CodeIs(500)
	}

	// 500.
	{

		req := test.MakeSimpleRequest("POST", "http://localhost/v1/cluster/remove", &peerParams{})
		encoded := base64.StdEncoding.EncodeToString([]byte("root:"))
		req.Header.Set("Authorization", "Basic "+encoded)
		recorded := test.RunRequest(t, handler, req)
		recorded.CodeIs(500)
	}

	// 200.
	{
		req := test.MakeSimpleRequest("POST", "http://localhost/v1/cluster/remove", p)
		encoded := base64.StdEncoding.EncodeToString([]byte("root:"))
		req.Header.Set("Authorization", "Basic "+encoded)
		recorded := test.RunRequest(t, handler, req)
		recorded.CodeIs(200)
	}
}

func TestCtlV1ClusterAddRemove_MySQL_SemiSync(t *testing.T) {
	testCtlV1ClusterAddRemove(t, model.ReplModeSemiSync)
}

func TestCtlV1ClusterAddRemove_MySQL_MGR(t *testing.T) {
	testCtlV1ClusterAddRemove(t, model.ReplModeMGR)
}
