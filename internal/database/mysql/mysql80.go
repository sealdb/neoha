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
	_ MysqlHandler = &Mysql80{}
)

// Mysql80 tuple.
type Mysql80 struct {
	MysqlBase
}

// GetMGRStats used to get group_replication status of all nodes.
func (my *Mysql80) GetMGRStats(db *sql.DB) ([]map[string]string, error) {
	var rows []map[string]string
	query := "SELECT MEMBER_ID,MEMBER_HOST,MEMBER_STATE,MEMBER_ROLE FROM performance_schema.replication_group_members"
	rows, err := QueryWithTimeout(db, my.queryTimeout, query)
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return rows, nil
}

// GetLocalMGRStat used to get MGR status.
func (my *Mysql80) GetLocalMGRStat(db *sql.DB) (*model.MGRStatus, error) {
	mgrStatus := &model.MGRStatus{
		State:       "UNKNOWN",
		Role:        "UNKNOWN",
		PrimaryUUID: "",
	}

	myid, err := my.MysqlBase.GetServerID(db)
	if err != nil {
		return mgrStatus, err
	}

	rows, err := my.GetMGRStats(db)
	if err != nil {
		return mgrStatus, err
	}
	if len(rows) > 0 {
		for _, v := range rows {
			if v["MEMBER_ID"] == myid {
				mgrStatus.State = v["MEMBER_STATE"]
				mgrStatus.Role = v["MEMBER_ROLE"]
			}
			if v["MEMBER_ROLE"] == model.MGRRolePrimary {
				mgrStatus.PrimaryUUID = v["MEMBER_ID"]
			}
		}
		if mgrStatus.State == "" && len(rows) == 1 {
			mgrStatus.State = rows[0]["MEMBER_STATE"]
		}
	} else {
		mgrStatus.State = model.MGRStateOffline
		return mgrStatus, nil
	}

	return mgrStatus, nil
}
