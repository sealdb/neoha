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

package mysqld

import (
	"fmt"
	"time"

	"neoha/base/model"
)

// MysqldRPC tuple.
type MysqldRPC struct {
	mysqld *Mysqld
}

// GetMysqldRPC returns MysqldRPC tuple.
func (m *Mysqld) GetMysqldRPC() *MysqldRPC {
	return &MysqldRPC{m}
}

// StartMonitor used to start the monitor.
func (m *MysqldRPC) StartMonitor(req *model.MysqldRPCRequest, rsp *model.MysqldRPCResponse) error {
	rsp.RetCode = model.OK
	m.mysqld.MonitorStart()
	return nil
}

// StopMonitor used to stop the monitor.
func (m *MysqldRPC) StopMonitor(req *model.MysqldRPCRequest, rsp *model.MysqldRPCResponse) error {
	rsp.RetCode = model.OK
	m.mysqld.MonitorStop()
	return nil
}

// Start used to start the mysql server.
func (m *MysqldRPC) Start(req *model.MysqldRPCRequest, rsp *model.MysqldRPCResponse) error {
	rsp.RetCode = model.OK
	if err := m.mysqld.StartMysqld(); err != nil {
		m.mysqld.log.Error("rpc.mysqld.start.error[%v]", err)
		rsp.RetCode = err.Error()
		return nil
	}
	return nil
}

// ShutDown used to shutdown the mysql server.
func (m *MysqldRPC) ShutDown(req *model.MysqldRPCRequest, rsp *model.MysqldRPCResponse) error {
	rsp.RetCode = model.OK
	if err := m.mysqld.StopMysqld(); err != nil {
		m.mysqld.log.Error("rpc.mysqld.shutdown.error[%v]", err)
		rsp.RetCode = err.Error()
		return nil
	}
	return nil
}

// Kill used to kill the mysqld.
func (m *MysqldRPC) Kill(req *model.MysqldRPCRequest, rsp *model.MysqldRPCResponse) error {
	rsp.RetCode = model.OK
	if err := m.mysqld.KillMysqld(); err != nil {
		m.mysqld.log.Error("rpc.mysqld.kill.error[%v]", err)
		rsp.RetCode = err.Error()
		return nil
	}
	return nil
}

// IsRunning used to check the mysqld is running or not.
func (m *MysqldRPC) IsRunning(req *model.MysqldRPCRequest, rsp *model.MysqldRPCResponse) error {
	rsp.RetCode = model.OK
	if !m.mysqld.isMysqldRunning() {
		rsp.RetCode = model.ErrorMysqldNotRunning
	}
	return nil
}

// Status used to get all the status of the backup.
func (m *MysqldRPC) Status(req *model.MysqldStatusRPCRequest, rsp *model.MysqldStatusRPCResponse) error {
	rsp.RetCode = model.OK
	rsp.MonitorInfo = m.mysqld.getMonitorInfo()
	rsp.MysqldInfo = m.mysqld.getMysqldInfo()

	backupStatus := m.mysqld.backup.getStatus()
	backupInfo := string(m.mysqld.backup.getStatus())
	if backupStatus == model.MYSQLD_BACKUPING {
		backupInfo = fmt.Sprintf("State:[%v], Time:[%s]",
			backupStatus,
			time.Since(m.mysqld.backup.getBackupStart()),
		)
	}
	rsp.BackupInfo = backupInfo
	rsp.BackupStats = m.mysqld.backup.getStats()
	rsp.BackupStatus = backupStatus
	rsp.MysqldStats = m.mysqld.getStats()
	return nil
}
