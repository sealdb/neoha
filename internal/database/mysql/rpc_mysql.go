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

package mysql

import (
	"github.com/sealdb/neoha/internal/base/model"
)

// MysqlRPC tuple.
type MysqlRPC struct {
	mysql *Mysql
}

// GetMysqlRPC returns the MysqlRPC.
func (m *Mysql) GetMysqlRPC() *MysqlRPC {
	return &MysqlRPC{m}
}

// SetGlobalSysVar used to set global vars.
func (m *MysqlRPC) SetGlobalSysVar(req *model.MysqlVarRPCRequest, rsp *model.MysqlVarRPCResponse) error {
	rsp.RetCode = model.OK
	if err := m.mysql.SetGlobalSysVar(req.VarSql); err != nil {
		rsp.RetCode = err.Error()
		return nil
	}
	return nil
}

// ResetMaster used to reset master.
func (m *MysqlRPC) ResetMaster(req *model.MysqlRPCRequest, rsp *model.MysqlRPCResponse) error {
	rsp.RetCode = model.OK
	if err := m.mysql.ResetMaster(); err != nil {
		rsp.RetCode = err.Error()
		return nil
	}
	return nil
}

// ChangeToMaster used to do 'reset slave all' command.
func (m *MysqlRPC) ChangeToMaster(req *model.MysqlRPCRequest, rsp *model.MysqlRPCResponse) error {
	rsp.RetCode = model.OK
	repl := m.mysql.GetRepl()
	if err := m.mysql.ChangeToMaster(&repl); err != nil {
		rsp.RetCode = err.Error()
		return nil
	}
	return nil
}

// ResetSlaveAll used to reset slave.
func (m *MysqlRPC) ResetSlaveAll(req *model.MysqlRPCRequest, rsp *model.MysqlRPCResponse) error {
	rsp.RetCode = model.OK
	repl := m.mysql.GetRepl()
	if err := m.mysql.ChangeToMaster(&repl); err != nil {
		rsp.RetCode = err.Error()
		return nil
	}
	return nil
}

// StopSlave used to stop the slave.
func (m *MysqlRPC) StopSlave(req *model.MysqlRPCRequest, rsp *model.MysqlRPCResponse) error {
	rsp.RetCode = model.OK
	if err := m.mysql.StopSlave(); err != nil {
		rsp.RetCode = err.Error()
		return nil
	}
	return nil
}

// StartSlave used to start the slave.
func (m *MysqlRPC) StartSlave(req *model.MysqlRPCRequest, rsp *model.MysqlRPCResponse) error {
	rsp.RetCode = model.OK
	if err := m.mysql.StartSlave(); err != nil {
		rsp.RetCode = err.Error()
		return nil
	}
	return nil
}

// ResetSlave used to reset slave.
func (m *MysqlRPC) ResetSlave(req *model.MysqlRPCRequest, rsp *model.MysqlRPCResponse) error {
	rsp.RetCode = model.OK
	if err := m.mysql.ResetSlave(); err != nil {
		rsp.RetCode = err.Error()
		return nil
	}
	return nil
}

// IsWorking used to check the mysql works or not.
func (m *MysqlRPC) IsWorking(req *model.MysqlRPCRequest, rsp *model.MysqlRPCResponse) error {
	if m.mysql.GetState() == model.MysqlAlive {
		rsp.RetCode = model.OK
	} else {
		rsp.RetCode = model.ErrorMySQLDown
	}
	return nil
}

// Status returns the mysql GTID info.
func (m *MysqlRPC) Status(req *model.MysqlStatusRPCRequest, rsp *model.MysqlStatusRPCResponse) error {
	var err error

	rsp.RetCode = model.OK
	rsp.ReplMode = m.mysql.GetReplMode()
	if rsp.GTID, err = m.mysql.GetGTID(); err != nil {
		rsp.RetCode = err.Error()
		return nil
	}
	rsp.Status = string(m.mysql.GetState())
	rsp.Options = string(m.mysql.GetOption())
	rsp.Stats = m.mysql.getStats()
	mgrStatus, err := m.mysql.GetLocalMGRStat()
	if err != nil {
		rsp.MGRStatus = &model.MGRStatus{
			Role:  "UNKNOWN",
			State: "UNKNOWN",
		}
	} else {
		rsp.MGRStatus = mgrStatus
	}
	return nil
}

// GTIDSubstract returns the mysql GTID subtract info.
func (m *MysqlRPC) GTIDSubtract(req *model.MysqlGTIDSubtractRPCRequest, rsp *model.MysqlGTIDSubtractRPCResponse) error {
	var err error

	rsp.RetCode = model.OK
	if rsp.Subtract, err = m.mysql.GetGTIDSubtract(req.SubsetGTID, req.SetGTID); err != nil {
		rsp.RetCode = err.Error()
		return nil
	}
	return nil
}

// SetState used to set the mysql state.
func (m *MysqlRPC) SetState(req *model.MysqlSetStateRPCRequest, rsp *model.MysqlSetStateRPCResponse) error {
	rsp.RetCode = model.OK
	m.mysql.setState(req.State)
	return nil
}
