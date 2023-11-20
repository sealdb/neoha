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
	"fmt"

	"github.com/sealdb/neoha/internal/base/model"
	"github.com/sealdb/neoha/internal/election/raft"
)

type UserRPC struct {
	server *Server
}

func (s *Server) GetUserRPC() *UserRPC {
	return &UserRPC{s}
}

// CreateNormalUser used to create a normal user.
func (u *UserRPC) CreateNormalUser(req *model.MysqlUserRPCRequest, rsp *model.MysqlUserRPCResponse) error {
	log := u.server.log
	rsp.RetCode = model.OK
	state := u.server.election.GetRaft().GetState()

	log.Warning("server.create.normaluser[%+v]...", req)
	if state != raft.LEADER {
		rsp.RetCode = fmt.Sprintf("nonleader.can.not.createuser")
		return nil
	}

	// check
	ok, err := u.server.db.GetMysql().CheckUserExists(req.User, req.Host)
	if err != nil {
		rsp.RetCode = err.Error()
		log.Error("rpc[%v].create.normaluser[%v]@[%v].with.error[%v]", state.String(), req.User, req.Host, err)
		return nil
	}

	if ok {
		msg := fmt.Sprintf("normaluser[%v]@[%v].exists.when.create", req.User, req.Host)
		rsp.RetCode = msg
		u.server.log.Error("%v", msg)
		return nil
	}

	// create
	if err := u.server.db.GetMysql().CreateUser(req.User, req.Host, req.Passwd, req.SSL); err != nil {
		rsp.RetCode = err.Error()
		log.Error("rpc[%v].create.normaluser[%v]@[%v].error[%v]", state.String(), req.User, req.Host, err)
		return nil
	}

	// grants
	if err := u.server.db.GetMysql().GrantNormalPrivileges(req.User, req.Host); err != nil {
		rsp.RetCode = err.Error()
		log.Error("rpc[%v].create.normaluser[%v]@[%v].error[%v]", state.String(), req.User, req.Host, err)
		return nil
	}
	return nil
}

// CreateSuperUser used to create a admin user with all grants.
func (u *UserRPC) CreateSuperUser(req *model.MysqlUserRPCRequest, rsp *model.MysqlUserRPCResponse) error {
	log := u.server.log
	rsp.RetCode = model.OK
	state := u.server.election.GetRaft().GetState()

	log.Warning("server.create.superuser[%+v]...", req)
	if state != raft.LEADER {
		rsp.RetCode = fmt.Sprintf("nonleader.can.not.createuser")
		return nil
	}

	// check
	ok, err := u.server.db.GetMysql().CheckUserExists(req.User, req.Host)
	if err != nil {
		rsp.RetCode = err.Error()
		log.Error("rpc[%v].create.superuser[%v]@[%v].with.error[%v]", state.String(), req.User, req.Host, err)
		return nil
	}

	if ok {
		msg := fmt.Sprintf("superuser[%v]@[%v].exists.when.create", req.User, req.Host)
		rsp.RetCode = msg
		u.server.log.Error("%v", msg)
		return nil
	}

	// create & grants
	if err := u.server.db.GetMysql().GrantAllPrivileges(req.User, req.Host, req.Passwd, req.SSL); err != nil {
		rsp.RetCode = err.Error()
		log.Error("rpc[%v].create.user[%v].error[%v]", state.String(), req.User, err)
		return nil
	}

	return nil
}

// CreateUserWithPrivileges creates user with privileges.
// This is used to create normal user.
func (u *UserRPC) CreateUserWithPrivileges(req *model.MysqlUserRPCRequest, rsp *model.MysqlUserRPCResponse) error {
	log := u.server.log
	rsp.RetCode = model.OK
	state := u.server.election.GetRaft().GetState()

	log.Warning("server.create.user[%+v].with.privileges...", req)
	if state != raft.LEADER {
		rsp.RetCode = fmt.Sprintf("nonleader.can.not.createuser")
		return nil
	}

	// check
	ok, err := u.server.db.GetMysql().CheckUserExists(req.User, req.Host)
	if err != nil {
		rsp.RetCode = err.Error()
		log.Error("rpc[%v].create.user[%v]@[%v].with.priv.error[%v]", state.String(), req.User, req.Host, err)
		return nil
	}

	if ok {
		msg := fmt.Sprintf("user[%v]@[%v].exists.when.create.with.priv", req.User, req.Host)
		rsp.RetCode = msg
		u.server.log.Error("%v", msg)
		return nil
	}

	// creates
	if err := u.server.db.GetMysql().CreateUserWithPrivileges(req.User, req.Passwd, req.Database, req.Table, req.Host, req.Privileges, req.SSL); err != nil {
		rsp.RetCode = err.Error()
		log.Error("rpc[%v].create.user[%v].with.priv.error[%v]", state.String(), req.User, err)
		return nil
	}
	return nil
}

// change password
func (u *UserRPC) ChangePasword(req *model.MysqlUserRPCRequest, rsp *model.MysqlUserRPCResponse) error {
	log := u.server.log
	rsp.RetCode = model.OK

	log.Warning("server.change.password[%+v]...", req)
	state := u.server.election.GetRaft().GetState()
	if state != raft.LEADER {
		rsp.RetCode = fmt.Sprintf("nonleader.can.not.changepassword")
		return nil
	}

	// change
	if err := u.server.db.GetMysql().ChangeUserPasswd(req.User, req.Host, req.Passwd); err != nil {
		rsp.RetCode = err.Error()
		log.Error("rpc[%v].change.pwd.[%v].error[%v]", state.String(), req.User, err)
		return nil
	}
	return nil
}

// drop user
func (u *UserRPC) DropUser(req *model.MysqlUserRPCRequest, rsp *model.MysqlUserRPCResponse) error {
	log := u.server.log
	rsp.RetCode = model.OK

	log.Warning("server.drop.user[%+v]...", req)
	state := u.server.election.GetRaft().GetState()
	if state != raft.LEADER {
		rsp.RetCode = fmt.Sprintf("nonleader.can.not.dropuser")
		return nil
	}

	// drop
	if err := u.server.db.GetMysql().DropUser(req.User, req.Host); err != nil {
		rsp.RetCode = err.Error()
		log.Error("[%v].drop.user.[%v]@[%v].error[%v]", state.String(), req.User, req.Host, err)
		return nil
	}
	return nil
}

// GetUser get mysql user list
func (u *UserRPC) GetUser(req *model.MysqlUserRPCRequest, rsp *model.MysqlUserRPCResponse) error {
	var err error

	log := u.server.log
	rsp.RetCode = model.OK

	log.Warning("server.get.mysql.user...")
	state := u.server.election.GetRaft().GetState()
	if state != raft.LEADER {
		rsp.RetCode = fmt.Sprintf("nonleader.can.not.get.mysql.user")
		return nil
	}

	rsp.Users, err = u.server.db.GetMysql().GetUser()
	if err != nil {
		rsp.RetCode = err.Error()
		log.Error("[%v].get.user.error[%v]", state.String(), err)
		return nil
	}

	return nil
}
