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
	"github.com/sealdb/neoha/internal/base/model"
)

type ServerRPC struct {
	server *Server
}

func (s *Server) GetServerRPC() *ServerRPC {
	return &ServerRPC{s}
}

// check the server connection whether OK
func (s *ServerRPC) Ping(req *model.ServerRPCRequest, rsp *model.ServerRPCResponse) error {
	rsp.RetCode = model.OK
	return nil
}

func (s *ServerRPC) Status(req *model.ServerRPCRequest, rsp *model.ServerRPCResponse) error {
	rsp.RetCode = model.OK
	config := &model.ConfigStatus{
		LogLevel:              s.server.conf.Log.Level,
		BackupDir:             s.server.conf.Database.Mysql.Backup.BackupDir,
		BackupIOPSLimits:      s.server.conf.Database.Mysql.Backup.BackupIOPSLimits,
		XtrabackupBinDir:      s.server.conf.Database.Mysql.Backup.XtrabackupBinDir,
		MysqldBaseDir:         s.server.conf.Database.Mysql.Backup.Basedir,
		MysqldDefaultsFile:    s.server.conf.Database.Mysql.Backup.DefaultsFile,
		MysqlAdmin:            s.server.conf.Database.Mysql.Admin,
		MysqlHost:             s.server.conf.Database.Mysql.Host,
		MysqlPort:             s.server.conf.Database.Mysql.Port,
		MysqlReplUser:         s.server.conf.Database.Mysql.ReplUser,
		MysqlPingTimeout:      s.server.conf.Database.Mysql.PingTimeout,
		RaftDataDir:           s.server.conf.Election.Raft.MetaDatadir,
		RaftHeartbeatTimeout:  s.server.conf.Election.Raft.HeartbeatTimeout,
		RaftElectionTimeout:   s.server.conf.Election.Raft.ElectionTimeout,
		RaftRPCRequestTimeout: s.server.conf.Election.Raft.RequestTimeout,
		RaftStartVipCommand:   s.server.conf.Election.Raft.LeaderStartCommand,
		RaftStopVipCommand:    s.server.conf.Election.Raft.LeaderStopCommand,
	}
	rsp.Config = config
	rsp.Stats = s.server.getStats()
	return nil
}
