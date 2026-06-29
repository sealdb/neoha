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
	"fmt"
	"strconv"
	"strings"
	"time"

	"database/sql"
	"github.com/sealdb/neoha/internal/base/model"

	"github.com/pkg/errors"
)

// http://dev.mysql.com/doc/refman/5.7/en/privileges-provided.html
var (
	mysqlAllPrivileges = []string{
		"ALL",
	}

	mysqlReplPrivileges = []string{
		"REPLICATION SLAVE",
		"REPLICATION CLIENT",
	}

	mysqlNormalPrivileges = []string{
		"ALTER", "ALTER ROUTINE", "CREATE", "CREATE ROUTINE",
		"CREATE TEMPORARY TABLES", "CREATE VIEW", "DELETE",
		"DROP", "EXECUTE", "EVENT", "INDEX", "INSERT",
		"LOCK TABLES", "PROCESS", "RELOAD", "SELECT",
		"SHOW DATABASES", "SHOW VIEW", "UPDATE", "TRIGGER", "REFERENCES",
		"REPLICATION SLAVE", "REPLICATION CLIENT",
	}

	// TODO: 与本文件中的 SSLTypYes or No 重复
	mysqlSSLType = []string{
		"YES", "NO",
	}

	_ MysqlHandler = &MysqlBase{
		replMode:        model.ReplModeSemiSync,
		queryTimeout:    10000, // TODO: 20000 for MGR
		mgrQueryTimeout: 40000,
	}
)

const (
	// ssl type: YES | NO
	SSLTypYes = "YES"
	SSLTypNo  = "NO"
)

// MysqlBase tuple.
type MysqlBase struct {
	MysqlHandler

	replMode        model.MysqlReplMode
	queryTimeout    int
	mgrQueryTimeout int
}

// SetReplMode used to set replication mode
func (my *MysqlBase) SetReplMode(replMode model.MysqlReplMode) {
	my.replMode = replMode
}

// GetReplMode used to get replication mode
func (my *MysqlBase) GetReplMode() *model.MysqlReplMode {
	return &my.replMode
}

// SetQueryTimeout used to set parameter queryTimeout
func (my *MysqlBase) SetQueryTimeout(timeout int) {
	my.queryTimeout = timeout
	my.mgrQueryTimeout = timeout * 2
}

// Ping has 2 affects:
// one for heath check
// other for get master_binglog the slave is syncing
func (my *MysqlBase) Ping(db *sql.DB) (*PingEntry, error) {
	pe := &PingEntry{}
	query := "SHOW SLAVE STATUS"
	rows, err := QueryWithTimeout(db, my.queryTimeout, query)
	if err != nil {
		return nil, err
	}
	if len(rows) > 0 {
		pe.Relay_Master_Log_File = rows[0]["Relay_Master_Log_File"]
	}
	return pe, nil
}

// SetReadOnly used to set mysql to readonly.
func (my *MysqlBase) SetReadOnly(db *sql.DB, readonly bool) error {
	enabled := 0
	if readonly {
		enabled = 1
	}

	cmds := []string{}
	cmds = append(cmds, fmt.Sprintf("SET GLOBAL read_only = %d", enabled))
	// Set super_read_only on the slave.
	// https://dev.mysql.com/doc/refman/5.7/en/server-system-variables.html#sysvar_super_read_only
	cmds = append(cmds, fmt.Sprintf("SET GLOBAL super_read_only = %d", enabled))
	return ExecuteSuperQueryListWithTimeout(db, my.queryTimeout, cmds)
}

// GTIDSubtract used to execute GTID_SUBTRACT
func (my *MysqlBase) GTIDSubtract(db *sql.DB, gtid1 string, gtid2 string) ([]map[string]string, error) {
	query := fmt.Sprintf("SELECT GTID_SUBTRACT('%s', '%s') RES", gtid1, gtid2)
	return QueryWithTimeout(db, my.queryTimeout, query)
}

// GTIDDiff used to get difference between gtid1 and gtid2.
// If gtid1 < gtid2, return "0"
// For example:
// gtid1 is uuid:1-100:1000-1003:2000-2005,
// gtid2 is uuid:1-99:1000-1002:2000-2005,
// the diff is 2
func (my *MysqlBase) GTIDDiff(db *sql.DB, gtid1 string, gtid2 string) (string, error) {
	rows, err := my.GTIDSubtract(db, gtid1, gtid2)
	if err != nil {
		return "", err
	}

	res := ""
	if len(rows) > 0 {
		if rows[0]["RES"] == "" {
			res = "0"
		} else {
			diff := 0
			a := strings.Split(rows[0]["RES"], ":")
			for i := 1; i < len(a); i++ {
				v := strings.Split(a[i], "-")
				b, err := strconv.Atoi(v[0])
				if err != nil {
					return res, err
				}
				if len(v) == 1 {
					diff = diff + 1
				} else {
					e, err := strconv.Atoi(v[1])
					if err != nil {
						return res, err
					}
					diff = diff + e - b + 1
				}
			}
			res = fmt.Sprintf("%d", diff)
		}
	}
	return res, nil
}

// GetMGRGTID used to get binlog info from MGR.
func (my *MysqlBase) GetMGRGTID(db *sql.DB) (*model.GTID, error) {
	gtid := &model.GTID{}

	query := "SHOW MASTER STATUS"
	rows, err := QueryWithTimeout(db, my.queryTimeout, query)
	if err != nil {
		return gtid, err
	}
	if len(rows) > 0 {
		row := rows[0]
		gtid.Master_Log_File = row["File"]
		gtid.Read_Master_Log_Pos, _ = strconv.ParseUint(row["Position"], 10, 64)
		gtid.Executed_GTID_Set = row["Executed_Gtid_Set"]
	}

	query = "SELECT RECEIVED_TRANSACTION_SET,LAST_ERROR_MESSAGE FROM performance_schema.replication_connection_status WHERE CHANNEL_NAME='group_replication_applier'"
	rows, err = QueryWithTimeout(db, my.queryTimeout, query)
	if err != nil {
		return gtid, err
	}
	if len(rows) > 0 {
		row := rows[0]
		gtid.Retrieved_GTID_Set = row["RECEIVED_TRANSACTION_SET"]
		gtid.Last_Error_Message = row["LAST_ERROR_MESSAGE"]
	}

	if diff, err := my.GTIDDiff(db, gtid.Retrieved_GTID_Set, gtid.Executed_GTID_Set); err == nil {
		gtid.Txns_Behind_Master = diff
	}
	return gtid, nil
}

// GTIDBigger determine if the first GTID is bigger than another one.
func (my *MysqlBase) GTIDBigger(db *sql.DB, gtid1 string, gtid2 string) (bool, error) {
	rows, err := my.GTIDSubtract(db, gtid1, gtid2)
	if err != nil {
		return false, err
	}
	if len(rows) > 0 && rows[0]["RES"] != "" {
		return true, nil
	}
	return false, nil
}

// IsMGRRunningOK used to check whether local MGR is running
func (my *MysqlBase) IsMGRRunningOK(db *sql.DB) (bool, error) {
	query := "SELECT MEMBER_ID,MEMBER_STATE,SERVICE_STATE FROM performance_schema.replication_group_members gm JOIN performance_schema.replication_connection_status cs WHERE gm.CHANNEL_NAME=cs.CHANNEL_NAME AND ((SERVICE_STATE='OFF') OR (MEMBER_STATE='ERROR' AND SERVICE_STATE='ON'))"
	rows, err := QueryWithTimeout(db, my.queryTimeout, query)
	if err != nil {
		return false, err
	}
	if len(rows) > 0 {
		return false, nil
	}
	return true, nil
}

// StartMGR used to start group_replication.
func (my *MysqlBase) StartMGR(db *sql.DB) error {
	cmd := "START GROUP_REPLICATION"
	return ExecuteWithTimeout(db, my.mgrQueryTimeout, cmd)
}

// StopMGR used to stop group_replication.
func (my *MysqlBase) StopMGR(db *sql.DB) error {
	cmd := "STOP GROUP_REPLICATION"
	return ExecuteWithTimeout(db, my.queryTimeout*2, cmd)
}

func (my *MysqlBase) MGRChangeMasterToCommands(master *model.Repl) []string {
	var args []string
	args = append(args, fmt.Sprintf("MASTER_USER = '%s'", master.Repl_User))
	args = append(args, fmt.Sprintf("MASTER_PASSWORD = '%s'", master.Repl_Password))
	changeMasterTo := "CHANGE MASTER TO\n  " + strings.Join(args, ",\n  ") + " FOR CHANNEL 'group_replication_recovery'"
	return []string{changeMasterTo}
}

// MGRChangeMasterTo stop for all channels and reset all replication filter to null.
// In NeoHA, we never set replication filter.
func (my *MysqlBase) MGRChangeMasterTo(db *sql.DB, master *model.Repl) error {
	cmds := []string{}
	cmds = append(cmds, "STOP GROUP_REPLICATION")
	cmds = append(cmds, my.MGRChangeMasterToCommands(master)...)
	cmds = append(cmds, "SET GLOBAL group_replication_bootstrap_group=OFF")
	cmds = append(cmds, "START GROUP_REPLICATION")
	return ExecuteSuperQueryListWithTimeout(db, my.mgrQueryTimeout, cmds)
}

// GetSlaveGTID gets the gtid from the default channel.
// Here, We just show the default slave channel.
func (my *MysqlBase) GetSlaveGTID(db *sql.DB) (*model.GTID, error) {
	gtid := &model.GTID{}

	query := "SHOW SLAVE STATUS"
	rows, err := QueryWithTimeout(db, my.queryTimeout, query)
	if err != nil {
		return gtid, err
	}
	if len(rows) > 0 {
		row := rows[0]
		gtid.Master_Log_File = row["Master_Log_File"]
		gtid.Read_Master_Log_Pos, _ = strconv.ParseUint(row["Read_Master_Log_Pos"], 10, 64)
		gtid.Retrieved_GTID_Set = row["Retrieved_Gtid_Set"]
		gtid.Executed_GTID_Set = row["Executed_Gtid_Set"]
		gtid.Slave_IO_Running = (row["Slave_IO_Running"] == "Yes")
		gtid.Slave_IO_Running_Str = row["Slave_IO_Running"]
		gtid.Slave_SQL_Running = (row["Slave_SQL_Running"] == "Yes")
		gtid.Slave_SQL_Running_Str = row["Slave_SQL_Running"]
		gtid.Seconds_Behind_Master = row["Seconds_Behind_Master"]
		gtid.Last_Error = row["Last_Error"]
		gtid.Last_IO_Error = row["Last_IO_Error"]
		gtid.Last_SQL_Error = row["Last_SQL_Error"]
		gtid.Slave_SQL_Running_State = row["Slave_SQL_Running_State"]
	}
	return gtid, nil
}

// GetMasterGTID used to get binlog info from master.
func (my *MysqlBase) GetMasterGTID(db *sql.DB) (*model.GTID, error) {
	gtid := &model.GTID{}

	query := "SHOW MASTER STATUS"
	rows, err := QueryWithTimeout(db, my.queryTimeout, query)
	if err != nil {
		return nil, err
	}
	if len(rows) > 0 {
		row := rows[0]
		gtid.Master_Log_File = row["File"]
		gtid.Read_Master_Log_Pos, _ = strconv.ParseUint(row["Position"], 10, 64)
		gtid.Executed_GTID_Set = row["Executed_Gtid_Set"]
		gtid.Seconds_Behind_Master = "0"
		gtid.Slave_IO_Running = true
		gtid.Slave_SQL_Running = true
	}
	return gtid, nil
}

// GetUUID used to get local uuid.
func (my *MysqlBase) GetUUID(db *sql.DB) (string, error) {
	uuid := ""
	query := "SELECT @@SERVER_UUID"
	rows, err := QueryWithTimeout(db, my.queryTimeout, query)
	if err != nil {
		return uuid, err
	}
	if len(rows) > 0 {
		row := rows[0]
		uuid = row["@@SERVER_UUID"]
	}

	return uuid, nil
}

// StartSlaveIOThread used to start the io thread.
func (my *MysqlBase) StartSlaveIOThread(db *sql.DB) error {
	cmd := "START SLAVE IO_THREAD"
	return ExecuteWithTimeout(db, my.queryTimeout, cmd)
}

// StopSlaveIOThread used to stop the op thread.
func (my *MysqlBase) StopSlaveIOThread(db *sql.DB) error {
	cmd := "STOP SLAVE IO_THREAD"
	return ExecuteWithTimeout(db, my.queryTimeout, cmd)
}

// StartSlave used to start slave.
func (my *MysqlBase) StartSlave(db *sql.DB) error {
	cmd := "START SLAVE"
	return ExecuteWithTimeout(db, my.queryTimeout, cmd)
}

// StopSlave used to stop the slave.
func (my *MysqlBase) StopSlave(db *sql.DB) error {
	cmd := "STOP SLAVE"
	return ExecuteWithTimeout(db, my.queryTimeout, cmd)
}

func (my *MysqlBase) changeMasterToCommands(master *model.Repl) []string {
	var args []string

	args = append(args, fmt.Sprintf("MASTER_HOST = '%s'", master.Master_Host))
	args = append(args, fmt.Sprintf("MASTER_PORT = %d", master.Master_Port))
	args = append(args, fmt.Sprintf("MASTER_USER = '%s'", master.Repl_User))
	args = append(args, fmt.Sprintf("MASTER_PASSWORD = '%s'", master.Repl_Password))
	args = append(args, "MASTER_AUTO_POSITION = 1")
	changeMasterTo := "CHANGE MASTER TO\n  " + strings.Join(args, ",\n  ")
	return []string{changeMasterTo}
}

// ChangeMasterTo stop for all channels and reset all replication filter to null.
// We never set replication filter.
func (my *MysqlBase) ChangeMasterTo(db *sql.DB, master *model.Repl) error {
	cmds := []string{}
	cmds = append(cmds, "STOP SLAVE")
	if master.Repl_GTID_Purged != "" {
		cmds = append(cmds, "RESET MASTER")
		cmds = append(cmds, "RESET SLAVE ALL")
		cmds = append(cmds, fmt.Sprintf("SET GLOBAL gtid_purged='%s'", master.Repl_GTID_Purged))
	}
	cmds = append(cmds, my.changeMasterToCommands(master)...)
	cmds = append(cmds, "START SLAVE")
	return ExecuteSuperQueryListWithTimeout(db, my.queryTimeout, cmds)
}

// countMGRLiveMembers counts ONLINE/RECOVERING members in the group view.
func (my *MysqlBase) countMGRLiveMembers(db *sql.DB) int {
	live, _, _ := my.mgrGroupView(db)
	return live
}

func (my *MysqlBase) mgrGroupView(db *sql.DB) (live int, total int, err error) {
	rows, err := my.GetMGRStats(db)
	if err != nil {
		return 0, 0, err
	}
	for _, row := range rows {
		total++
		state := row["MEMBER_STATE"]
		if state == model.MGRStateOnline || state == model.MGRStateRecovering {
			live++
		}
	}
	return live, total, nil
}

func (my *MysqlBase) groupReplicationLocalAddress(db *sql.DB) (string, error) {
	query := "SELECT @@GLOBAL.group_replication_local_address"
	rows, err := QueryWithTimeout(db, my.queryTimeout, query)
	if err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return "", errors.New("group_replication_local_address.not.found")
	}
	return rows[0]["@@GLOBAL.group_replication_local_address"], nil
}

// ChangeToMaster changes a slave to be master.
func (my *MysqlBase) ChangeToMaster(db *sql.DB, master *model.Repl) error {
	cmds := []string{}
	if my.replMode == model.ReplModeMGR {
		// Sole-survivor rebuild: force the group view only when this node is the
		// last live member of an existing group. Fresh clusters (empty view) use
		// normal bootstrap without force_members.
		live, total, err := my.mgrGroupView(db)
		if err != nil {
			return err
		}
		useForce := live == 1 && total >= 1
		var forceAddr string
		if useForce {
			addr, err := my.groupReplicationLocalAddress(db)
			if err != nil {
				return err
			}
			forceAddr = addr
			// force_members requires the member to still be ONLINE.
			cmds = append(cmds, fmt.Sprintf("SET GLOBAL group_replication_force_members = '%s'", forceAddr))
		}

		cmds = append(cmds, "STOP GROUP_REPLICATION")
		cmds = append(cmds, my.MGRChangeMasterToCommands(master)...)
		cmds = append(cmds,
			"SET GLOBAL group_replication_bootstrap_group=ON",
			"START GROUP_REPLICATION", //TODO: too slow
			"SET GLOBAL group_replication_bootstrap_group=OFF",
		)
		if forceAddr != "" {
			cmds = append(cmds, "SET GLOBAL group_replication_force_members = ''")
		}
	} else {
		cmds = append(cmds, "STOP SLAVE")
		cmds = append(cmds, "RESET SLAVE ALL") //"ALL" makes it forget the master host:port
	}
	return ExecuteSuperQueryListWithTimeout(db, my.queryTimeout, cmds)
}

// WaitUntilAfterGTID used to do 'SELECT WAIT_UNTIL_SQL_THREAD_AFTER_GTIDS' command.
// https://dev.mysql.com/doc/refman/5.7/en/gtid-functions.html
func (my *MysqlBase) WaitUntilAfterGTID(db *sql.DB, targetGTID string) error {
	query := fmt.Sprintf("SELECT WAIT_UNTIL_SQL_THREAD_AFTER_GTIDS('%s')", targetGTID)
	return Execute(db, query)
}

// GetMGRStats used to get group_replication status of all nodes.
func (my *MysqlBase) GetMGRStats(db *sql.DB) ([]map[string]string, error) {
	var rows []map[string]string
	query := "SELECT MEMBER_ID,MEMBER_HOST,MEMBER_STATE FROM performance_schema.replication_group_members"
	rows, err := QueryWithTimeout(db, my.queryTimeout, query)
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return rows, nil
}

// GetMGRMasterUUID used to get uuid of master.
func (my *MysqlBase) GetMGRMasterUUID(db *sql.DB) (string, error) {
	query := "SELECT VARIABLE_VALUE FROM performance_schema.global_status where VARIABLE_NAME='group_replication_primary_member'"
	rows, err := QueryWithTimeout(db, my.queryTimeout, query)
	if err != nil {
		return "", err
	}
	if len(rows) != 1 {
		return "", errors.New("get.group.replication.master.failed")
	}
	return rows[0]["VARIABLE_VALUE"], err
}

// GetServerID used to get uuid.
func (my *MysqlBase) GetServerID(db *sql.DB) (string, error) {
	query := "select @@server_uuid"
	rows, err := QueryWithTimeout(db, my.queryTimeout, query)
	if err != nil {
		return "", err
	}
	if len(rows) != 1 {
		return "", errors.New("get.group.replication.master.failed")
	}
	return rows[0]["@@server_uuid"], err
}

// WaitApplyRelayLog used to wait for the relay to apply completed.
func (my *MysqlBase) WaitApplyRelayLog(db *sql.DB, retry int, interval int) error {
	var err error
	for i := 0; i < retry; i++ {
		gtid, err := my.GetMGRGTID(db)
		if err == nil && gtid.Txns_Behind_Master == "0" {
			return nil
		}
		time.Sleep(time.Second * time.Duration(interval))
	}
	return errors.Errorf("wait.until.after.gtid.failed[%v]", err)
}

// GetGTIDSubtract used to do "SELECT GTID_SUBTRACT('subsetGTID','setGTID') as gtid_sub" command
func (my *MysqlBase) GetGTIDSubtract(db *sql.DB, subsetGTID string, setGTID string) (string, error) {
	query := fmt.Sprintf("SELECT GTID_SUBTRACT('%s','%s') as gtid_sub", subsetGTID, setGTID)
	rows, err := QueryWithTimeout(db, my.queryTimeout, query)
	if err != nil {
		return "", err
	}

	if len(rows) > 0 {
		row := rows[0]
		gtid_sub := row["gtid_sub"]
		return gtid_sub, nil
	}
	return "", nil
}

// SetGlobalSysVar used to set global variables.
func (my *MysqlBase) SetGlobalSysVar(db *sql.DB, varsql string) error {
	prefix := "SET GLOBAL"
	if !strings.HasPrefix(varsql, prefix) {
		return errors.Errorf("[%v].must.be.startwith:%v", varsql, prefix)
	}
	return ExecuteWithTimeout(db, my.queryTimeout, varsql)
}

// ResetMaster used to reset master.
func (my *MysqlBase) ResetMaster(db *sql.DB) error {
	cmds := "RESET MASTER"
	return ExecuteWithTimeout(db, my.queryTimeout, cmds)
}

// ResetSlave used to reset slave.
func (my *MysqlBase) ResetSlave(db *sql.DB) error {
	cmds := "RESET SLAVE"
	return ExecuteWithTimeout(db, my.queryTimeout, cmds)
}

// ResetSlaveAll used to reset slave.
func (my *MysqlBase) ResetSlaveAll(db *sql.DB) error {
	cmds := []string{"STOP SLAVE",
		"RESET SLAVE ALL"} //"ALL" makes it forget the master host:port
	return ExecuteSuperQueryListWithTimeout(db, my.queryTimeout, cmds)
}

// PurgeBinlogsTo used to purge binlog.
func (my *MysqlBase) PurgeBinlogsTo(db *sql.DB, binlog string) error {
	cmds := fmt.Sprintf("PURGE BINARY LOGS TO '%s'", binlog)
	return ExecuteWithTimeout(db, my.queryTimeout, cmds)
}

// EnableSemiSyncMaster used to enable the semi-sync on master.
func (my *MysqlBase) EnableSemiSyncMaster(db *sql.DB) error {
	cmds := "SET GLOBAL rpl_semi_sync_master_enabled=ON"
	return ExecuteWithTimeout(db, my.queryTimeout, cmds)
}

// SetSemiWaitSlaveCount used set rpl_semi_sync_master_wait_for_slave_count
func (my *MysqlBase) SetSemiWaitSlaveCount(db *sql.DB, count int) error {
	cmds := fmt.Sprintf("SET GLOBAL rpl_semi_sync_master_wait_for_slave_count = %d", count)
	return ExecuteWithTimeout(db, my.queryTimeout, cmds)
}

// DisableSemiSyncMaster used to disable the semi-sync from master.
func (my *MysqlBase) DisableSemiSyncMaster(db *sql.DB) error {
	cmds := "SET GLOBAL rpl_semi_sync_master_enabled=OFF"
	return ExecuteWithTimeout(db, my.queryTimeout, cmds)
}

// SetSemiSyncMasterTimeout used to set semi-sync master timeout
func (my *MysqlBase) SetSemiSyncMasterTimeout(db *sql.DB, timeout uint64) error {
	cmds := fmt.Sprintf("SET GLOBAL rpl_semi_sync_master_timeout=%d", timeout)
	return ExecuteWithTimeout(db, my.queryTimeout, cmds)
}

// CheckUserExists used to check the user exists or not.
func (my *MysqlBase) CheckUserExists(db *sql.DB, user string, host string) (bool, error) {
	query := fmt.Sprintf("SELECT User FROM mysql.user WHERE User = '%s' and Host = '%s'", user, host)
	rows, err := Query(db, query)
	if err != nil {
		return false, err
	}
	if len(rows) > 0 {
		return true, nil
	}
	return false, nil
}

// GetUser used to get the mysql user list
func (my *MysqlBase) GetUser(db *sql.DB) ([]model.MysqlUser, error) {
	query := fmt.Sprintf("SELECT User, Host, Super_priv FROM mysql.user")
	rows, err := Query(db, query)
	if err != nil {
		return nil, err
	}

	var Users = make([]model.MysqlUser, len(rows))
	for i, v := range rows {
		Users[i].User = v["User"]
		Users[i].Host = v["Host"]
		Users[i].SuperPriv = v["Super_priv"]
	}
	return Users, nil
}

// CreateUser use to create new user.
// see http://dev.mysql.com/doc/refman/5.7/en/string-literals.html
func (my *MysqlBase) CreateUser(db *sql.DB, user string, host string, passwd string, ssltype string) error {
	query := fmt.Sprintf("CREATE USER `%s`@`%s` IDENTIFIED BY '%s'", user, host, passwd)
	if strings.ToUpper(ssltype) == SSLTypYes {
		query = fmt.Sprintf("%s REQUIRE X509", query)
	}
	return Execute(db, query)
}

// DropUser used to drop the user.
func (my *MysqlBase) DropUser(db *sql.DB, user string, host string) error {
	query := fmt.Sprintf("DROP USER `%s`@`%s`", user, host)
	return Execute(db, query)
}

// CreateReplUserWithoutBinlog create replication accounts without writing binlog.
func (my *MysqlBase) CreateReplUserWithoutBinlog(db *sql.DB, user string, passwd string) error {
	queryList := []string{
		"SET sql_log_bin=0",
		"SET GLOBAL read_only=off",
		fmt.Sprintf("CREATE USER `%s` IDENTIFIED BY '%s'", user, passwd),
		fmt.Sprintf("GRANT %s ON *.* TO `%s`", strings.Join(mysqlReplPrivileges, ","), user),
		"SET GLOBAL super_read_only=on",
		"SET sql_log_bin=1",
	}
	return ExecuteSuperQueryList(db, queryList)
}

// ChangeUserPasswd used to change the user password.
func (my *MysqlBase) ChangeUserPasswd(db *sql.DB, user string, host string, passwd string) error {
	query := fmt.Sprintf("ALTER USER `%s`@`%s` IDENTIFIED BY '%s'", user, host, passwd)
	return Execute(db, query)
}

// GrantNormalPrivileges used to grants normal privileges.
func (my *MysqlBase) GrantNormalPrivileges(db *sql.DB, user string, host string) error {
	query := fmt.Sprintf("GRANT %s ON *.* TO `%s`@`%s`", strings.Join(mysqlNormalPrivileges, ","), user, host)
	return my.grantPrivileges(db, query)
}

// CreateUserWithPrivileges for create normal user.
func (my *MysqlBase) CreateUserWithPrivileges(db *sql.DB, user, passwd, database, table, host, privs string, ssl string) error {
	// build normal privs map
	var query string
	normal := make(map[string]string)
	for _, priv := range mysqlNormalPrivileges {
		normal[priv] = priv
	}

	// check privs
	privs = strings.TrimSuffix(privs, ",")
	privsList := strings.Split(privs, ",")
	for _, priv := range privsList {
		priv = strings.ToUpper(strings.TrimSpace(priv))
		if _, ok := normal[priv]; !ok {
			return errors.Errorf("can't create user[%v] with privileges[%v]", user, priv)
		}
	}

	// check ssl_type
	upperSSL := strings.ToUpper(ssl)
	if upperSSL != SSLTypYes && upperSSL != SSLTypNo {
		return errors.Errorf("wrong ssl_type[%v], it should be 'YES' or 'NO'", ssl)
	}

	if err := my.CreateUser(db, user, host, passwd, upperSSL); err != nil {
		return errors.Errorf("create user[%v] with privileges[%v] require ssl_type[%v] failed: [%v]", user, privs, ssl, err)
	}

	query = fmt.Sprintf("GRANT %s ON %s.%s TO `%s`@`%s`", privs, database, table, user, host)
	return my.grantPrivileges(db, query)
}

// GrantReplicationPrivileges used to grant repli privis.
func (my *MysqlBase) GrantReplicationPrivileges(db *sql.DB, user string) error {
	query := fmt.Sprintf("GRANT %s ON *.* TO `%s`", strings.Join(mysqlReplPrivileges, ","), user)
	return my.grantPrivileges(db, query)
}

// GrantAllPrivileges used to grant all privis.
func (my *MysqlBase) GrantAllPrivileges(db *sql.DB, user string, host string, passwd string, ssl string) error {
	var query string

	// check ssl_type
	upperSSL := strings.ToUpper(ssl)
	if upperSSL != SSLTypYes && upperSSL != SSLTypNo {
		return errors.Errorf("wrong ssl_type[%v], it should be 'YES' or 'NO'", ssl)
	}

	if err := my.CreateUser(db, user, host, passwd, upperSSL); err != nil {
		return errors.Errorf("create user[%v]@[%v] with all privileges require ssl_type[%v] failed: [%v]", user, host, ssl, err)
	}

	query = fmt.Sprintf("GRANT %s ON *.* TO `%s`@`%s` WITH GRANT OPTION", strings.Join(mysqlAllPrivileges, ","), user, host)
	return my.grantPrivileges(db, query)
}

func (my *MysqlBase) grantPrivileges(db *sql.DB, query string) error {
	return Execute(db, query)
}
