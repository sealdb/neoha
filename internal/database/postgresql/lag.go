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

package postgresql

import (
	"context"
	"database/sql"
)

// ReplicationLagBytes returns replay lag on a standby (receive LSN minus replay LSN).
func (p *Postgresql) ReplicationLagBytes(ctx context.Context) (int64, error) {
	inRecovery, err := p.IsInRecovery(ctx)
	if err != nil {
		return 0, err
	}
	if !inRecovery {
		return 0, nil
	}
	db, err := p.getDB()
	if err != nil {
		return 0, err
	}
	var lag sql.NullInt64
	err = db.QueryRowContext(ctx, `
		SELECT pg_wal_lsn_diff(
			pg_last_wal_receive_lsn(),
			COALESCE(pg_last_wal_replay_lsn(), '0/0')
		)`).Scan(&lag)
	if err != nil {
		return 0, err
	}
	if !lag.Valid {
		return 0, nil
	}
	if lag.Int64 < 0 {
		return 0, nil
	}
	return lag.Int64, nil
}
