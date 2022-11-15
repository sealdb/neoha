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
	"testing"

	"neoha/base/nlog"
	"neoha/config"
	"neoha/database/database"
	"neoha/election/raft"

	"github.com/stretchr/testify/assert"
)

func TestElectionStart(t *testing.T) {

}

func TestElectionStop(t *testing.T) {

}

func TestElectionGetRaft(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	conf := config.DefaultConfig()
	db := database.NewDatabase(conf.Database, database.MySQL, 10000, log)
	s := NewElection(conf, raft.FOLLOWER, db, database.MySQL, log)
	r := s.GetRaft()
	assert.NotNil(t, r)
}
