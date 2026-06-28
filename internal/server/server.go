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
	"context"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/sealdb/neoha/internal/base/nlog"
	"github.com/sealdb/neoha/internal/base/nrpc"
	"github.com/sealdb/neoha/internal/config"
	"github.com/sealdb/neoha/internal/coordination"
	"github.com/sealdb/neoha/internal/coordination/wire"
	"github.com/sealdb/neoha/internal/database"
	"github.com/sealdb/neoha/internal/database/mysql"
	"github.com/sealdb/neoha/internal/election"
	"github.com/sealdb/neoha/internal/election/raft"
	"github.com/sealdb/neoha/internal/ha"
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
	log        *nlog.Log
	conf       *config.Config
	election   *election.Election
	coord      coordination.Coordinator
	reconciler *ha.Reconciler
	haCancel   context.CancelFunc
	initState  ServerState
	db         *database.Database
	manager    *manager.Manager
	rpc        *nrpc.Service
	rpcs       RPCS
	begin      time.Time
}

func NewServer(conf *config.Config, log *nlog.Log, state ServerState) *Server {
	s := &Server{
		log:       log,
		conf:      conf,
		initState: state,
		db:        nil,
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
	s.setupHA()
}

func (s *Server) setupHA() {
	var raftInst *raft.Raft
	if s.election != nil {
		raftInst = s.election.GetRaft()
	}
	coord, err := wire.NewCoordinator(s.conf, raftInst)
	if err != nil {
		s.log.Warning("server.coordination.setup.skipped[%v]", err)
		return
	}
	s.coord = coord
	hooks := ha.NewPrimaryHooks(s.log, s.conf.EffectivePrimaryHooks())
	if s.initState == ServerLeader {
		hooks.SetSkipPrimaryStartOnce()
	}
	opts := []ha.Option{ha.WithPrimaryHooks(hooks)}
	if s.conf.HA != nil {
		if s.conf.HA.ReconcileInterval > 0 {
			opts = append(opts, ha.WithInterval(time.Duration(s.conf.HA.ReconcileInterval)*time.Second))
		}
		if s.conf.HA.DelegateDBApply {
			opts = append(opts, ha.WithApplyPromote(true))
		}
	}
	if s.conf.Database != nil && s.conf.Database.Mysql != nil {
		opts = append(opts, ha.WithMySQLPort(s.conf.Database.Mysql.Port))
	}
	s.reconciler = ha.NewReconciler(s.log, coord, s.db.Driver(), s.conf.Tags, opts...)
}

// setupRPC used to setup rpc handlers
func (s *Server) setupRPC() {
	log := s.log
	log.Info("server.prepare.setup.RPC")
	s.rpcs.NodeRPC = s.GetNodeRPC()
	s.rpcs.ServerRPC = s.GetServerRPC()
	s.rpcs.UserRPC = s.GetUserRPC()
	if raftInst := s.election.GetRaft(); raftInst != nil {
		s.rpcs.RaftRPC = raftInst.GetRaftRPC()
		s.rpcs.HARPC = raftInst.GetHARPC()
	}
	if mysqld := s.manager.GetMysqld(); mysqld != nil {
		s.rpcs.MysqldRPC = mysqld.GetMysqldRPC()
		s.rpcs.BackupRPC = mysqld.GetBackupRPC()
	}
	if mysql := s.db.GetMysql(); mysql != nil {
		s.rpcs.MysqlRPC = mysql.GetMysqlRPC()
	}

	if err := s.rpc.RegisterService(s.rpcs.NodeRPC); err != nil {
		log.Panic("server.rpc.RegisterService.NodeRPC.error[%+v]", err)
	}
	if err := s.rpc.RegisterService(s.rpcs.ServerRPC); err != nil {
		log.Panic("server.rpc.RegisterService.ServerRPC.error[%+v]", err)
	}
	if err := s.rpc.RegisterService(s.rpcs.UserRPC); err != nil {
		log.Panic("server.rpc.RegisterService.UserRPC.error[%+v]", err)
	}
	if s.rpcs.HARPC != nil {
		if err := s.rpc.RegisterService(s.rpcs.HARPC); err != nil {
			log.Panic("server.rpc.RegisterService.HARPC.error[%+v]", err)
		}
	}
	if s.rpcs.RaftRPC != nil {
		if err := s.rpc.RegisterService(s.rpcs.RaftRPC); err != nil {
			log.Panic("server.rpc.RegisterService.RaftRPC.error[%+v]", err)
		}
	}
	if s.rpcs.MysqldRPC != nil {
		if err := s.rpc.RegisterService(s.rpcs.MysqldRPC); err != nil {
			log.Panic("server.rpc.RegisterService.MysqldRPC.error[%+v]", err)
		}
	}
	if s.rpcs.BackupRPC != nil {
		if err := s.rpc.RegisterService(s.rpcs.BackupRPC); err != nil {
			log.Panic("server.rpc.RegisterService.BackupRPC.error[%+v]", err)
		}
	}
	if s.rpcs.MysqlRPC != nil {
		if err := s.rpc.RegisterService(s.rpcs.MysqlRPC); err != nil {
			log.Panic("server.rpc.RegisterService.MysqlRPC.error[%+v]", err)
		}
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
	s.db.Start()
	s.election.Start()
	if s.coord != nil {
		if err := s.coord.Start(context.Background()); err != nil {
			s.log.Warning("server.coordination.start.error[%v]", err)
		}
	}
	if s.reconciler != nil {
		ctx, cancel := context.WithCancel(context.Background())
		s.haCancel = cancel
		go s.reconciler.Loop(ctx)
	}
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
	if s.haCancel != nil {
		s.haCancel()
	}
	if s.coord != nil {
		_ = s.coord.Stop()
	}
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
