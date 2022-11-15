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
	"neoha/base/model"
)

// BackupRPC tuple.
type BackupRPC struct {
	mysqld *Mysqld
}

// GetBackupRPC returns BackupRPC tuple.
func (m *Mysqld) GetBackupRPC() *BackupRPC {
	return &BackupRPC{m}
}

// DoBackup used to execute the xtrabackup command.
func (b *BackupRPC) DoBackup(req *model.BackupRPCRequest, rsp *model.BackupRPCResponse) error {
	rsp.RetCode = model.OK
	err := b.mysqld.backup.Backup(req)
	if err != nil {
		rsp.RetCode = err.Error()
		return nil
	}
	return nil
}

// DoApplyLog used to execute the apply log command.
func (b *BackupRPC) DoApplyLog(req *model.BackupRPCRequest, rsp *model.BackupRPCResponse) error {
	rsp.RetCode = model.OK
	err := b.mysqld.backup.ApplyLog(req)
	if err != nil {
		rsp.RetCode = err.Error()
		return nil
	}
	return nil
}

// CancelBackup used to cancel the job of backup.
func (b *BackupRPC) CancelBackup(req *model.BackupRPCRequest, rsp *model.BackupRPCResponse) error {
	rsp.RetCode = model.OK
	err := b.mysqld.backup.Cancel()
	if err != nil {
		rsp.RetCode = err.Error()
		return nil
	}
	return nil
}
