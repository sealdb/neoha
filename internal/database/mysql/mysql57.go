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
	"database/sql"
	"github.com/sealdb/neoha/internal/base/model"
)

var (
	_ MysqlHandler = &Mysql57{}
)

// Mysql57 tuple.
type Mysql57 struct {
	MysqlBase
}

// GetLocalMGRStat used to get MGR status.
func (my *Mysql57) GetLocalMGRStat(db *sql.DB) (*model.MGRStatus, error) {
	mgrStatus := &model.MGRStatus{
		State:       "UNKNOWN",
		Role:        "UNKNOWN",
		PrimaryUUID: "",
	}

	myid, err := my.MysqlBase.GetServerID(db)
	if err != nil {
		return mgrStatus, err
	}

	rows, err := my.MysqlBase.GetMGRStats(db)
	if err != nil {
		return mgrStatus, err
	}
	if len(rows) > 0 {
		for _, v := range rows {
			if v["MEMBER_ID"] == myid {
				mgrStatus.State = v["MEMBER_STATE"]
				break
			}
		}
		if mgrStatus.State == "" && len(rows) == 1 {
			mgrStatus.State = rows[0]["MEMBER_STATE"]
		}
	} else {
		mgrStatus.State = model.MGRStateOffline
		return mgrStatus, nil
	}

	pid, err := my.MysqlBase.GetMGRMasterUUID(db)
	if err != nil {
		return mgrStatus, err
	}
	if pid == myid {
		mgrStatus.Role = model.MGRRolePrimary
	} else {
		mgrStatus.Role = model.MGRRoleSecondary
	}
	mgrStatus.PrimaryUUID = pid

	return mgrStatus, nil
}
