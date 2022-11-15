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

package v1

import (
	"net/http"

	"neoha/base/model"
	"neoha/base/nlog"
	"neoha/cmd/neorpc"
	"neoha/server"

	"github.com/ant0ine/go-json-rest/rest"
)

// RaftStatusHandler impl.
func RaftStatusHandler(log *nlog.Log, neoha *server.Server) rest.HandlerFunc {
	f := func(w rest.ResponseWriter, r *rest.Request) {
		raftStatusHandler(log, neoha, w, r)
	}
	return f
}

func raftStatusHandler(log *nlog.Log, neoha *server.Server, w rest.ResponseWriter, r *rest.Request) {
	type Status struct {
		State  string   `json:"state"`
		Leader string   `json:"leader"`
		Nodes  []string `json:"nodes"`
	}
	status := &Status{}
	address := neoha.Address()

	state, nodes, err := neorpc.GetRaftState(address)
	if err != nil {
		log.Error("api.v1.raft.status.error:%+v", err)
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	status.State = state
	status.Nodes = nodes

	rsp, err := neorpc.GetNodesRPC(address)
	if err != nil {
		log.Error("api.v1.raft.status.error:%+v", err)
		rest.Error(w, err.Error(), http.StatusInternalServerError)
	}
	if rsp == nil {
		log.Error("api.v1.raft.status.error:rsp[nil] != [OK]")
		rest.Error(w, err.Error(), http.StatusInternalServerError)
	}
	status.Leader = rsp.GetLeader()

	w.WriteJson(status)
}

// RaftTryToLeaderHandler impl.
func RaftTryToLeaderHandler(log *nlog.Log, neoha *server.Server) rest.HandlerFunc {
	f := func(w rest.ResponseWriter, r *rest.Request) {
		raftTryToLeaderHandler(log, neoha, w, r)
	}
	return f
}

func raftTryToLeaderHandler(log *nlog.Log, neoha *server.Server, w rest.ResponseWriter, r *rest.Request) {
	address := neoha.Address()
	log.Warning("api.v1.raft.trytoleader.[%v].prepare.to.propose.this.raft.to.leader", address)
	rsp, err := neorpc.TryToLeaderRPC(address)
	if err != nil {
		log.Error("api.v1.raft.trytoleader.error:%+v", err)
		rest.Error(w, err.Error(), http.StatusInternalServerError)
	}
	if rsp == nil {
		log.Error("api.v1.raft.trytoleader.error:rsp[nil] != [OK]")
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if rsp.RetCode != model.OK {
		log.Error("api.v1.raft.trytoleader.error:rsp[%v] != [OK]", rsp.RetCode)
		rest.Error(w, err.Error(), http.StatusInternalServerError)
	}
	log.Warning("api.v1.raft.trytoleader.[%v].propose.done", address)
}

// RaftDisableCheckSemiSyncHandler impl.
func RaftDisableCheckSemiSyncHandler(log *nlog.Log, neoha *server.Server) rest.HandlerFunc {
	f := func(w rest.ResponseWriter, r *rest.Request) {
		raftDisableCheckSemiSyncHandler(log, neoha, w, r)
	}
	return f
}

func raftDisableCheckSemiSyncHandler(log *nlog.Log, neoha *server.Server, w rest.ResponseWriter, r *rest.Request) {
	address := neoha.Address()
	log.Warning("api.v1.raft.disablechecksemisync.[%v].prepare.to.disable.check.semi-sync", address)
	if err := neorpc.RaftDisableCheckSemiSyncRPC(address); err != nil {
		log.Error("api.v1.raft.disablechecksemisync.error:%+v", err)
		rest.Error(w, err.Error(), http.StatusInternalServerError)
	}
	log.Warning("api.v1.raft.disablechecksemisync.[%v].disable.check.semi-sync.done", address)
}

// RaftDisableHandler impl.
func RaftDisableHandler(log *nlog.Log, neoha *server.Server) rest.HandlerFunc {
	f := func(w rest.ResponseWriter, r *rest.Request) {
		raftDisableHandler(log, neoha, w, r)
	}
	return f
}

func raftDisableHandler(log *nlog.Log, neoha *server.Server, w rest.ResponseWriter, r *rest.Request) {
	address := neoha.Address()
	log.Warning("api.v1.raft.disable.[%v].prepare.to.disable.raft", address)
	rsp, err := neorpc.DisableRaftRPC(address)
	if err != nil {
		log.Error("api.v1.raft.disable.error:%+v", err)
		rest.Error(w, err.Error(), http.StatusInternalServerError)
	}
	if rsp == nil {
		log.Error("api.v1.raft.disable.error:rsp[nil] != [OK]")
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if rsp.RetCode != model.OK {
		log.Error("api.v1.raft.disable.error:rsp[%v] != [OK]", rsp.RetCode)
		rest.Error(w, err.Error(), http.StatusInternalServerError)
	}
	log.Warning("api.v1.raft.disable.[%v].done", address)
}
