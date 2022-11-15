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

package model

const (
	RPCBackupStatus   = "BackupRPC.GetBackupStatus"
	RPCBackupDo       = "BackupRPC.DoBackup"
	RPCBackupCancel   = "BackupRPC.CancelBackup"
	RPCBackupApplyLog = "BackupRPC.DoApplyLog"
)

type BackupStats struct {
	// How many times backup have been called
	Backups uint64

	// How many times backup have failed
	BackupErrs uint64

	// How many times apply-log have been called
	AppLogs uint64

	// How many times apply-log have failed
	AppLogErrs uint64

	// How many times cannel have been taken
	Cancels uint64

	// The last error message of backup/applylog
	LastError string

	// The last backup command info  we call
	LastCMD string
}

type BackupRPCRequest struct {
	// The IP of this request
	From string

	// The Backup dir of this request
	BackupDir string

	// The SSH IP of this request
	SSHHost string

	// The SSH user of this request
	SSHUser string

	// The SSH password of this request
	SSHPasswd string

	// The SSH port(default is 22) of this request
	SSHPort int

	// The Backup IOPS throttle of this request
	IOPSLimits int

	// The xtrabackup/xbstream binary dir
	XtrabackupBinDir string
}

type BackupRPCResponse struct {
	// Return code to rpc client:
	// OK or other errors
	RetCode string
}

func NewBackupRPCRequest() *BackupRPCRequest {
	return &BackupRPCRequest{}
}

func NewBackupRPCResponse(code string) *BackupRPCResponse {
	return &BackupRPCResponse{RetCode: code}
}
