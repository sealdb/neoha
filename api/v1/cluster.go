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
	"strings"

	"github.com/sealdb/neoha/internal/base/nlog"
	"github.com/sealdb/neoha/internal/neorpc"
	"github.com/sealdb/neoha/internal/server"

	"github.com/ant0ine/go-json-rest/rest"
)

type peerParams struct {
	Address string `json:"address"`
}

func ClusterAddHandler(log *nlog.Log, neoha *server.Server) rest.HandlerFunc {
	f := func(w rest.ResponseWriter, r *rest.Request) {
		clusterAddHandler(log, neoha, w, r)
	}
	return f
}

func clusterAddHandler(log *nlog.Log, neoha *server.Server, w rest.ResponseWriter, r *rest.Request) {
	p := peerParams{}
	err := r.DecodeJsonPayload(&p)
	if err != nil {
		log.Error("api.v1.cluster.add.error:%+v", err)
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if p.Address == "" {
		rest.Error(w, "api.v1.cluster.add.request.address.is.null", http.StatusInternalServerError)
		return
	}

	self := neoha.Address()
	nodes := strings.Split(strings.Trim(p.Address, ","), ",")
	leader, err := neorpc.GetClusterLeader(self)
	if err != nil {
		log.Warning("%v", err)
	}

	log.Warning("api.v1.cluster.prepare.to.add.nodes[%v].to.leader[%v]", p.Address, leader)
	if leader != "" {
		if err := neorpc.AddNodeRPC(leader, nodes); err != nil {
			log.Error("api.v1.cluster.add[%+v].error:%+v", p, err)
			rest.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		log.Warning("api.v1.cluster.add.canot.found.leader.forward.to[%v]", self)
		if err := neorpc.AddNodeRPC(self, nodes); err != nil {
			log.Error("api.v1.cluster.add[%+v].error:%+v", p, err)
			rest.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	log.Warning("api.v1.cluster.add.nodes.to.leader[%v].done", leader)
}

func ClusterRemoveHandler(log *nlog.Log, neoha *server.Server) rest.HandlerFunc {
	f := func(w rest.ResponseWriter, r *rest.Request) {
		clusterRemoveHandler(log, neoha, w, r)
	}
	return f
}

func clusterRemoveHandler(log *nlog.Log, neoha *server.Server, w rest.ResponseWriter, r *rest.Request) {
	p := peerParams{}
	err := r.DecodeJsonPayload(&p)
	if err != nil {
		log.Error("api.v1.cluster.remove.error:%+v", err)
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if p.Address == "" {
		rest.Error(w, "api.v1.cluster.remove.request.address.is.null", http.StatusInternalServerError)
		return
	}

	self := neoha.Address()
	nodes := strings.Split(strings.Trim(p.Address, ","), ",")
	leader, err := neorpc.GetClusterLeader(self)
	if err != nil {
		log.Warning("%v", err)
	}

	log.Warning("api.v1.cluster.prepare.to.remove.nodes[%v].from.leader[%v]", p.Address, leader)
	if leader != "" {
		if err := neorpc.RemoveNodeRPC(leader, nodes); err != nil {
			log.Error("api.v1.cluster.remove[%+v].error:%+v", p, err)
			rest.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		log.Warning("api.v1.cluster.remove.canot.found.leader.forward.to[%v]", self)
		if err := neorpc.RemoveNodeRPC(self, nodes); err != nil {
			log.Error("api.v1.cluster.remove[%+v].error:%+v", p, err)
			rest.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	log.Warning("api.v1.cluster.remove.nodes.from.leader[%v].done", leader)
}
