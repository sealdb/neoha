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

package api

import (
	"context"
	"log"
	"net/http"
	_ "net/http/pprof"

	"neoha/base/nlog"
	"neoha/base/nrpc"
	"neoha/server"

	"github.com/ant0ine/go-json-rest/rest"
)

func init() {
	go func() {
		log.Println(http.ListenAndServe(":6060", nil))
	}()
}

// Admin tuple.
type Admin struct {
	log    *nlog.Log
	server *http.Server
	neoha  *server.Server
}

// NewAdmin creates the new admin.
func NewAdmin(log *nlog.Log, neoha *server.Server) *Admin {
	return &Admin{
		log:   log,
		neoha: neoha,
	}
}

// Start starts http server.
func (admin *Admin) Start() {
	api := rest.NewApi()
	router, err := admin.NewRouter()
	if err != nil {
		panic(err)
	}

	authMiddleware := &rest.AuthBasicMiddleware{
		Realm: "neoha zone",
		Authenticator: func(userId string, password string) bool {
			if userId == admin.neoha.MySQLAdmin() && password == admin.neoha.MySQLPasswd() {
				return true
			}
			return false
		},
	}
	api.Use(authMiddleware)

	api.SetApp(router)
	handlers := api.MakeHandler()
	admin.server = &http.Server{Addr: admin.neoha.PeerAddress(), Handler: handlers}

	go func() {
		log := admin.log
		log.Info("http.server.start[%v]...", admin.neoha.PeerAddress())

		ln, err := nrpc.SetListener(admin.server.Addr)
		if err != nil {
			log.Panic("%v", err)
		}

		if err := admin.server.Serve(ln); err != http.ErrServerClosed {
			log.Panic("%v", err)
		}
	}()
}

// Stop stops http server.
func (admin *Admin) Stop() {
	log := admin.log
	admin.server.Shutdown(context.Background())
	log.Info("http.server.gracefully.stop")
}
