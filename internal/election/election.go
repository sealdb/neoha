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

package election

import (
	"log"
	"github.com/sealdb/neoha/internal/base/nlog"
	"github.com/sealdb/neoha/internal/config"
	"github.com/sealdb/neoha/internal/database"
	"github.com/sealdb/neoha/internal/election/raft"
)

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

	if conf.EffectiveCoordination().Provider == "raft" {
		election.etype = ElectionRaft
		election.raft = raft.NewRaft(conf, log, db, dbType, state)
	} else if conf.EffectiveCoordination().Provider == "etcd" {
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
		// Leader election runs in coordination.Coordinator (server.setupHA / server.Start).
	default:
		log.Panic("unsupported election type")
	}
}

func (e *Election) Stop() {
	switch e.etype {
	case ElectionRaft:
		e.raft.Stop()
	case ElectionEtcd:
		// Coordinator Stop is handled by server.Shutdown.
	default:
		log.Panic("unsupported election type")
	}
}

// Provider returns the active coordination provider name.
func (e *Election) Provider() string {
	if e.conf == nil {
		return ""
	}
	return e.conf.EffectiveCoordination().Provider
}

func (e *Election) GetRaft() *raft.Raft {
	return e.raft
}
