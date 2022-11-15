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

type MYSQLD_STATUS string

const (
	MYSQLD_BACKUPNONE     MYSQLD_STATUS = "NONE"
	MYSQLD_BACKUPING      MYSQLD_STATUS = "BACKUPING"
	MYSQLD_BACKUPCANCELED MYSQLD_STATUS = "CANCELED"
	MYSQLD_APPLYLOGGING   MYSQLD_STATUS = "APPLYLOGGING"
	MYSQLD_SHUTDOWNING    MYSQLD_STATUS = "SHUTDOWNING"
	MYSQLD_ISRUNNING      MYSQLD_STATUS = "RUNNING"
	MYSQLD_NOTRUNNING     MYSQLD_STATUS = "NOTRUNNING"
	MYSQLD_UNKNOW         MYSQLD_STATUS = "UNKNOW"
)

const (
	RPCMysqldStatus       = "MysqldRPC.Status"
	RPCMysqldStartMonitor = "MysqldRPC.StartMonitor"
	RPCMysqldStopMonitor  = "MysqldRPC.StopMonitor"
	RPCMysqldStart        = "MysqldRPC.Start"
	RPCMysqldShutDown     = "MysqldRPC.ShutDown"
	RPCMysqldKill         = "MysqldRPC.Kill"
	RPCMysqldIsRuning     = "MysqldRPC.IsRunning"
)

// mysqld
type MysqldRPCRequest struct {
	// The IP of this request
	From string
}

type MysqldRPCResponse struct {
	// Return code to rpc client:
	// OK or other errors
	RetCode string
}

func NewMysqldRPCRequest() *MysqldRPCRequest {
	return &MysqldRPCRequest{}
}

func NewMysqldRPCResponse(code string) *MysqldRPCResponse {
	return &MysqldRPCResponse{RetCode: code}
}

// status
type MysqldStats struct {
	// How many times the mysqld have been started by NeoHA
	MysqldStarts uint64

	// How many times the mysqld have been stopped by NeoHA
	MysqldStops uint64

	// How many times the monitor have been started by NeoHA
	MonitorStarts uint64

	// How many times the monitor have been stopped by NeoHA
	MonitorStops uint64
}

type MysqldStatusRPCRequest struct {
	// The IP of this request
	From string
}

type MysqldStatusRPCResponse struct {
	// Monitor Info
	MonitorInfo string

	// Mysqld Info
	MysqldInfo string

	// Backup Info
	BackupInfo string

	// Mysqld Stats
	MysqldStats *MysqldStats

	// Backup Stats
	BackupStats *BackupStats

	// Backup Status: BACKUPING/ or others
	BackupStatus MYSQLD_STATUS

	// Return code to rpc client:
	// OK or other errors
	RetCode string
}

func NewMysqldStatusRPCRequest() *MysqldStatusRPCRequest {
	return &MysqldStatusRPCRequest{}
}

func NewMysqldStatusRPCResponse(code string) *MysqldStatusRPCResponse {
	return &MysqldStatusRPCResponse{RetCode: code}
}
