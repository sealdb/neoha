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
	"net/http"

	"github.com/sealdb/neoha/internal/base/model"
	"github.com/sealdb/neoha/internal/base/nlog"
	"github.com/sealdb/neoha/internal/neorpc"
	"github.com/sealdb/neoha/internal/server"

	"github.com/ant0ine/go-json-rest/rest"
)

// RaftStatusHandler impl.
func NeoHAPingHandler(log *nlog.Log, neoha *server.Server) rest.HandlerFunc {
	f := func(w rest.ResponseWriter, r *rest.Request) {
		neohaPingHandler(log, neoha, w, r)
	}
	return f
}

func neohaPingHandler(log *nlog.Log, neoha *server.Server, w rest.ResponseWriter, r *rest.Request) {
	address := neoha.Address()
	rsp, err := neorpc.ServerPingRPC(address)
	if err != nil {
		log.Error("api.v1.neoha.ping.error:%+v", err)
		rest.Error(w, err.Error(), http.StatusInternalServerError)
	}
	if rsp == nil {
		log.Error("api.v1.neoha.ping.error:rsp[nil] != [OK]")
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if rsp.RetCode != model.OK {
		log.Error("api.v1.neoha.ping.error:rsp[%v] != [OK]", rsp.RetCode)
		rest.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
