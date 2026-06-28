/*
 * Copyright 2022-2026 The NeoHA Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package mysql

import (
	"context"
	"strconv"

	"github.com/sealdb/neoha/internal/base/model"
	"github.com/sealdb/neoha/internal/base/nlog"
	"github.com/sealdb/neoha/internal/config"
	dbdriver "github.com/sealdb/neoha/internal/database/driver"
)

// Driver implements database.Driver for MySQL (semi-sync / MGR).
type Driver struct {
	m    *Mysql
	conf *config.MysqlConfig
	log  *nlog.Log
}

// NewDriver wraps an existing *Mysql as a database.Driver.
func NewDriver(m *Mysql, conf *config.MysqlConfig, log *nlog.Log) *Driver {
	return &Driver{m: m, conf: conf, log: log}
}

func (d *Driver) Type() dbdriver.Engine {
	return dbdriver.EngineMySQL
}

func (d *Driver) Start() {
	d.m.PingStart()
}

func (d *Driver) Stop() {
	d.m.PingStop()
}

// SetupBootstrap performs first-start MySQL setup (repl user, readonly, semi-sync slave).
func (d *Driver) SetupBootstrap(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	log := d.log
	log.Info("database.mysql.wait.for.work[maxwait:60s]")
	if err := d.m.WaitMysqlWorks(60 * 1000); err != nil {
		log.Error("database.mysql.WaitMysqlWorks.error[%v]", err)
		return err
	}

	gtid, _ := d.m.GetGTID()
	log.Info("database.mysql.gtid:%+v", gtid)

	if err := d.ensureReplUser(); err != nil {
		return err
	}

	log.Info("database.mysql.set.to.READONLY")
	if err := d.m.SetReadOnly(); err != nil {
		log.Error("database.mysql.SetReadOnly.error[%+v]", err)
		return err
	}

	if d.conf.ReplMode != model.ReplModeMGR {
		log.Info("database.mysql.start.slave")
		if err := d.m.StartSlave(); err != nil {
			log.Error("database.mysql.start.slave.error[%+v]", err)
			return err
		}
	}
	log.Info("server.mysql.setup.done")
	return nil
}

func (d *Driver) ensureReplUser() error {
	log := d.log
	log.Info("database.mysql.check.replication.user...")
	ret, err := d.m.CheckUserExists(d.conf.ReplUser, "%")
	if err != nil {
		log.Error("database.mysql.CheckUserExists.error[%+v]", err)
		return err
	}
	if !ret {
		log.Info("setupMysql.database.mysql.prepare.to.create.replication.user[%v]", d.conf.ReplUser)
		if err = d.m.CreateReplUserWithoutBinlog(d.conf.ReplUser, d.conf.ReplPasswd); err != nil {
			log.Error("server.mysql.create.replication.user.error[%+v]", err)
			return err
		}
	}
	return nil
}

func (d *Driver) Promotable(ctx context.Context) (bool, string) {
	if err := ctx.Err(); err != nil {
		return false, err.Error()
	}
	if d.m.Promotable() {
		return true, ""
	}
	return false, "mysql.not.promotable"
}

func (d *Driver) ReplicationLagBytes(ctx context.Context) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	gtid, err := d.m.GetGTID()
	if err != nil {
		return 0, err
	}
	if gtid.Seconds_Behind_Master != "" {
		if v, err := strconv.ParseInt(gtid.Seconds_Behind_Master, 10, 64); err == nil {
			return v, nil
		}
	}
	return 0, nil
}

func (d *Driver) ApplyPrimary(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if d.conf.ReplMode == model.ReplModeMGR {
		return d.applyPrimaryMGR(ctx)
	}
	return d.applyPrimarySemiSync(ctx)
}

func (d *Driver) applyPrimaryMGR(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := d.m.WaitApplyRelayLog(15, 2); err != nil {
		return err
	}
	if err := d.m.SetMasterGlobalSysVar(); err != nil {
		return err
	}
	if d.mgrAlreadyPrimary() {
		return d.m.SetReadOnly()
	}
	repl := d.replFromConf()
	if err := d.m.ChangeToMaster(repl); err != nil {
		return err
	}
	return d.m.SetReadOnly()
}

func (d *Driver) mgrAlreadyPrimary() bool {
	ok, _ := d.m.IsMGRRunningOK()
	if !ok {
		return false
	}
	status, err := d.m.GetLocalMGRStat()
	return err == nil && status.Role == model.MGRRolePrimary
}

func (d *Driver) IsMGRMode() bool {
	return d.conf.ReplMode == model.ReplModeMGR
}

func (d *Driver) MGRPrimaryPhase1Done(ctx context.Context) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if !d.IsMGRMode() {
		return false, nil
	}
	return d.mgrAlreadyPrimary(), nil
}

func (d *Driver) MGRClusterWritableReady(ctx context.Context) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if !d.IsMGRMode() {
		return false, nil
	}
	rows, err := d.m.GetMGRStats()
	if err != nil {
		return false, err
	}
	uuid, err := d.m.GetMGRMasterUUID()
	if err != nil {
		return false, err
	}
	cnt := 0
	masterOK := false
	for _, row := range rows {
		if row["MEMBER_STATE"] == model.MGRStateOnline || row["MEMBER_STATE"] == model.MGRStateRecovering {
			cnt++
			if row["MEMBER_ID"] == uuid {
				masterOK = true
			}
		}
	}
	const mgrQuorum = 2
	return masterOK && cnt >= mgrQuorum, nil
}

func (d *Driver) ApplyPrimaryMGRPhase2(ctx context.Context) error {
	return d.EnableReadWrite(ctx)
}

var _ dbdriver.MGRLifecycle = (*Driver)(nil)

func (d *Driver) applyPrimarySemiSync(ctx context.Context) error {
	gtid, err := d.m.GetGTID()
	if err != nil {
		return err
	}
	if err := d.m.WaitUntilAfterGTID(gtid.Retrieved_GTID_Set); err != nil {
		return err
	}
	repl := d.replFromConf()
	if err := d.m.ChangeToMaster(repl); err != nil {
		return err
	}
	if err := d.m.EnableSemiSyncMaster(); err != nil {
		return err
	}
	if err := d.m.SetMasterGlobalSysVar(); err != nil {
		return err
	}
	return d.m.SetReadWrite()
}

func (d *Driver) ApplyReplica(ctx context.Context, primary dbdriver.PrimaryRef) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	repl := d.replFromConf()
	if primary.Host != "" {
		repl.Master_Host = primary.Host
	}
	if primary.Port != 0 {
		repl.Master_Port = primary.Port
	}
	repl.Repl_GTID_Purged = d.m.GetReplGtidPurged()

	if d.conf.ReplMode != model.ReplModeMGR {
		if err := d.m.DisableSemiSyncMaster(); err != nil {
			d.log.Warning("database.mysql.DisableSemiSyncMaster.error[%v]", err)
		}
		if err := d.m.SetSlaveGlobalSysVar(); err != nil {
			d.log.Warning("database.mysql.SetSlaveGlobalSysVar.error[%v]", err)
		}
		if err := d.m.StartSlave(); err != nil {
			d.log.Warning("database.mysql.StartSlave.error[%v]", err)
		}
	}
	if err := d.m.SetReadOnly(); err != nil {
		return err
	}
	if d.conf.ReplMode == model.ReplModeMGR {
		return d.m.MGRChangeMasterTo(repl)
	}
	if err := d.m.ChangeMasterTo(repl); err != nil {
		return err
	}
	return d.m.StartSlave()
}

func (d *Driver) Demote(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return d.m.SetReadOnly()
}

func (d *Driver) Status(ctx context.Context) (dbdriver.Health, error) {
	if err := ctx.Err(); err != nil {
		return dbdriver.Health{}, err
	}
	role := dbdriver.RoleReplica
	if d.m.GetOption() == MysqlReadwrite {
		role = dbdriver.RolePrimary
	}
	return dbdriver.Health{
		Alive:     d.m.GetState() == model.MysqlAlive,
		Writable:  d.m.GetOption() == MysqlReadwrite,
		Role:      role,
		LastError: "",
		Details:   d.m,
	}, nil
}

func (d *Driver) replFromConf() *model.Repl {
	return &model.Repl{
		Master_Host:      d.conf.ReplHost,
		Master_Port:      d.conf.Port,
		Repl_User:        d.conf.ReplUser,
		Repl_Password:    d.conf.ReplPasswd,
		Repl_GTID_Purged: d.conf.ReplGtidPurged,
	}
}

func (d *Driver) ChangeToMasterRepl(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	repl := d.replFromConf()
	return d.m.ChangeToMaster(repl)
}

// EnableReadWrite opens writes after semi-sync/MGR promotion steps.
func (d *Driver) EnableReadWrite(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return d.m.SetReadWrite()
}

// Mysql returns the underlying *Mysql for legacy callers (raft, RPC). Deprecated: use Driver.
func (d *Driver) Mysql() *Mysql {
	return d.m
}
