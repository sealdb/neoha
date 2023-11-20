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

package api

import (
	"github.com/sealdb/neoha/api/v1"

	"github.com/ant0ine/go-json-rest/rest"
)

// NewRouter creates the new router.
func (admin *Admin) NewRouter() (rest.App, error) {
	log := admin.log
	neoha := admin.neoha

	return rest.MakeRouter(
		// cluster.
		rest.Post("/v1/cluster/add", v1.ClusterAddHandler(log, neoha)),
		rest.Post("/v1/cluster/remove", v1.ClusterRemoveHandler(log, neoha)),

		// raft.
		rest.Get("/v1/raft/status", v1.RaftStatusHandler(log, neoha)),
		rest.Post("/v1/raft/trytoleader", v1.RaftTryToLeaderHandler(log, neoha)),
		rest.Put("/v1/raft/disablechecksemisync", v1.RaftDisableCheckSemiSyncHandler(log, neoha)),
		rest.Put("/v1/raft/disable", v1.RaftDisableHandler(log, neoha)),

		// neoha.
		rest.Get("/v1/neoha/ping", v1.NeoHAPingHandler(log, neoha)),
	)
}
