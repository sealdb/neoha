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

package server

import (
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/sealdb/neoha/internal/base/nlog"
	"github.com/sealdb/neoha/internal/base/nrpc"
	"github.com/sealdb/neoha/internal/config"
	"github.com/sealdb/neoha/internal/database"
	"github.com/sealdb/neoha/internal/database/mysql"
	"github.com/sealdb/neoha/internal/election"
	"github.com/sealdb/neoha/internal/election/raft"
	"github.com/sealdb/neoha/internal/manager"
	"github.com/sealdb/neoha/internal/manager/mysqld"
)

type RPCS struct {
	NodeRPC   *NodeRPC
	ServerRPC *ServerRPC
	UserRPC   *UserRPC
	HARPC     *raft.HARPC
	RaftRPC   *raft.RaftRPC
	MysqldRPC *mysqld.MysqldRPC
	BackupRPC *mysqld.BackupRPC
	MysqlRPC  *mysql.MysqlRPC
}

// ServerState used for server state.
type ServerState int

const (
	ServerLeader ServerState = 1 << iota
	ServerFollower
	ServerIdle
	ServerLearner
	ServerUnknown
)

type Server struct {
	log      *nlog.Log
	conf     *config.Config
	election *election.Election
	db       *database.Database
	manager  *manager.Manager
	rpc      *nrpc.Service
	rpcs     RPCS
	begin    time.Time
}

func NewServer(conf *config.Config, log *nlog.Log, state ServerState) *Server {
	s := &Server{
		log:  log,
		conf: conf,
		db:   nil,
	}

	dbType := database.MySQL
	if conf.Database.Type == "postgresql" {
		dbType = database.PostgreSQL
	}

	s.db = database.NewDatabase(conf.Database, dbType, conf.Election.Raft.ElectionTimeout, log)
	s.manager = manager.NewNanager(conf.Database, dbType, log)
	s.election = election.NewElection(conf, getRaftInitState(state), s.db, dbType, log)

	rpc, err := nrpc.NewService(nrpc.Log(log),
		nrpc.ConnectionStr(conf.Endpoint))
	if err != nil {
		log.Panic("server.rpc.NewService.error[%v]", err)
	}
	s.rpc = rpc
	return s
}

func getRaftInitState(state ServerState) raft.State {
	switch state {
	case ServerLeader:
		return raft.LEADER
	case ServerFollower:
		return raft.FOLLOWER
	case ServerIdle:
		return raft.IDLE
	case ServerLearner:
		return raft.LEARNER
	default:
		return raft.UNKNOWN
	}
}

func (s *Server) Init() {
	s.manager.SetupManager()
	s.db.SetupDB()
	s.setupRPC()
}

// setupRPC used to setup rpc handlers
func (s *Server) setupRPC() {
	log := s.log
	log.Info("server.prepare.setup.RPC")
	s.rpcs.NodeRPC = s.GetNodeRPC()
	s.rpcs.ServerRPC = s.GetServerRPC()
	s.rpcs.UserRPC = s.GetUserRPC()
	s.rpcs.RaftRPC = s.election.GetRaft().GetRaftRPC()
	s.rpcs.HARPC = s.election.GetRaft().GetHARPC()
	s.rpcs.MysqldRPC = s.manager.GetMysqld().GetMysqldRPC()
	s.rpcs.BackupRPC = s.manager.GetMysqld().GetBackupRPC()
	s.rpcs.MysqlRPC = s.db.GetMysql().GetMysqlRPC()

	if err := s.rpc.RegisterService(s.rpcs.NodeRPC); err != nil {
		log.Panic("server.rpc.RegisterService.NodeRPC.error[%+v]", err)
	}
	if err := s.rpc.RegisterService(s.rpcs.ServerRPC); err != nil {
		log.Panic("server.rpc.RegisterService.ServerRPC.error[%+v]", err)
	}
	if err := s.rpc.RegisterService(s.rpcs.UserRPC); err != nil {
		log.Panic("server.rpc.RegisterService.UserRPC.error[%+v]", err)
	}
	if err := s.rpc.RegisterService(s.rpcs.HARPC); err != nil {
		log.Panic("server.rpc.RegisterService.HARPC.error[%+v]", err)
	}
	if err := s.rpc.RegisterService(s.rpcs.RaftRPC); err != nil {
		log.Panic("server.rpc.RegisterService.RaftRPC.error[%+v]", err)
	}
	if err := s.rpc.RegisterService(s.rpcs.MysqldRPC); err != nil {
		log.Panic("server.rpc.RegisterService.MysqldRPC.error[%+v]", err)
	}
	if err := s.rpc.RegisterService(s.rpcs.BackupRPC); err != nil {
		log.Panic("server.rpc.RegisterService.BackupRPC.error[%+v]", err)
	}
	if err := s.rpc.RegisterService(s.rpcs.MysqlRPC); err != nil {
		log.Panic("server.rpc.RegisterService.MysqlRPC.error[%+v]", err)
	}
	log.Info("server.RPC.setup.done")
}

// Start server
func (s *Server) Start() {
	log := s.log
	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 4096)
			buf = buf[:runtime.Stack(buf, false)]
			log.Error("server.got.panic[%v:%s]", r, buf)
		}
	}()

	s.manager.Start()
	s.db.GetMysql().PingStart()
	s.election.Start()
	if err := s.rpc.Start(); err != nil {
		log.Panic("server.rpc.start.error[%+v]", err)
	}
	s.updateUptime()

	log.Info("server.start.success...")
}

func (s *Server) updateUptime() {
	s.begin = time.Now()
}

func (s *Server) Shutdown() {
	s.log.Info("server.prepare.to.shutdown")
	s.rpc.Stop()
	s.election.Stop()
	s.db.Stop()
	s.manager.Stop()
	s.log.Info("server.shutdown.done")
}

// waits for os signal
func (s *Server) Wait() {
	ossig := make(chan os.Signal, 1)
	signal.Notify(ossig,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGHUP)
	s.log.Info("server.signal:%+v", <-ossig)
	s.Shutdown()
}

func (s *Server) Address() string {
	return s.conf.Endpoint
}

func (s *Server) GetElection() *election.Election {
	return s.election
}

// PeerAddress returns the peer address.
func (s *Server) PeerAddress() string {
	return s.conf.PeerAddress
}

// MySQLAdmin returns the mysql admin user.
func (s *Server) MySQLAdmin() string {
	return s.conf.Database.Mysql.Admin
}

// MySQLPasswd returns the mysql admin password.
func (s *Server) MySQLPasswd() string {
	return s.conf.Database.Mysql.Passwd
}
