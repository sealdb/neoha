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

package election

import (
	"log"
	"neoha/base/nlog"
	"neoha/config"
	"neoha/database/database"
	"neoha/election/raft"
)

// TODO: remove
//import (
//	"neoha/election/etcd"
//	"neoha/election/raft"
//)
//
//// ElectionHandler interface.
//type ElectionHandler interface {
//}
//
//var (
//	handlers = make(map[string]ElectionHandler)
//)
//
//func init() {
//	handlers["raft"] = new(raft.Raft)
//	handlers["etcd"] = new(etcd.Etcd)
//}
//
//func getHandler(name string) ElectionHandler {
//	handler, ok := handlers[name]
//	if !ok {
//		return new(raft.Raft) // default
//	}
//	return handler
//}

type ElectionType int

const (
	ElectionRaft ElectionType = 1 << iota
	ElectionEtcd
	ElectionUnknown
)

type Election struct {
	dbType database.DBType
	log    *nlog.Log
	conf   *config.Config
	etype  ElectionType
	// TODO: refactor election handler
	raft *raft.Raft
	//etcd *etcd.Etcd
}

// NewElection creates the new election tuple.
func NewElection(conf *config.Config, state raft.State, db *database.Database, dbType database.DBType, log *nlog.Log) *Election {
	election := &Election{
		log:    log,
		conf:   conf,
		dbType: dbType,
	}

	if conf.Election.Algo == "raft" {
		election.etype = ElectionRaft
		election.raft = raft.NewRaft(conf.Endpoint, conf.Election.Raft, conf.Database.Mysql.SemiSyncTimeoutForTwoNodes,
			log, db, dbType, state)
	} else if conf.Election.Algo == "etcd" {
		election.etype = ElectionEtcd
		// election.etcd = etcd.NewEtcd()
	} else {
		log.Panic("unsupported election type")
	}

	return election
}

func (e *Election) Start() {
	switch e.etype {
	case ElectionRaft:
		if err := e.raft.Start(); err != nil {
			log.Panic("election.raft.start.error")
		}
	case ElectionEtcd:
		// TODO: start etcd
	default:
		log.Panic("unsupported election type")
	}
}

func (e *Election) Stop() {
	switch e.etype {
	case ElectionRaft:
		e.raft.Stop()
	case ElectionEtcd:
		// TODO: stop etcd
	default:
		log.Panic("unsupported election type")
	}
}

func (e *Election) GetRaft() *raft.Raft {
	return e.raft
}
