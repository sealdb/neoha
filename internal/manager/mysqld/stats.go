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

package mysqld

import (
	"sync/atomic"

	"github.com/sealdb/neoha/internal/base/model"
)

// IncBackups used to increase the backup counter.
func (s *Backup) IncBackups() {
	atomic.AddUint64(&s.stats.Backups, 1)
}

// IncBackupErrs used to increase the backup error counter.
func (s *Backup) IncBackupErrs() {
	atomic.AddUint64(&s.stats.BackupErrs, 1)
}

// IncCancels used to increase the backup cancel counter.
func (s *Backup) IncCancels() {
	atomic.AddUint64(&s.stats.Cancels, 1)
}

// IncApplyLogs used to increase the apply counter.
func (s *Backup) IncApplyLogs() {
	atomic.AddUint64(&s.stats.AppLogs, 1)
}

// IncApplyLogErrs used to increase the apply error counter.
func (s *Backup) IncApplyLogErrs() {
	atomic.AddUint64(&s.stats.AppLogErrs, 1)
}

func (s *Backup) getStats() *model.BackupStats {
	return &model.BackupStats{
		Backups:    atomic.LoadUint64(&s.stats.Backups),
		BackupErrs: atomic.LoadUint64(&s.stats.BackupErrs),
		AppLogs:    atomic.LoadUint64(&s.stats.AppLogs),
		AppLogErrs: atomic.LoadUint64(&s.stats.AppLogErrs),
		Cancels:    atomic.LoadUint64(&s.stats.Cancels),
	}
}

// IncMysqldStarts used to increase the mysql start counter.
func (s *Mysqld) IncMysqldStarts() {
	atomic.AddUint64(&s.stats.MysqldStarts, 1)
}

// IncMysqldStops used to increase the mysql stop counter.
func (s *Mysqld) IncMysqldStops() {
	atomic.AddUint64(&s.stats.MysqldStops, 1)
}

// IncMonitorStarts used to increase the monitor start counter.
func (s *Mysqld) IncMonitorStarts() {
	atomic.AddUint64(&s.stats.MonitorStarts, 1)
}

// IncMonitorStops used to increase the monitor stop counter.
func (s *Mysqld) IncMonitorStops() {
	atomic.AddUint64(&s.stats.MonitorStops, 1)
}

func (s *Mysqld) getStats() *model.MysqldStats {
	return &model.MysqldStats{
		MysqldStarts:  atomic.LoadUint64(&s.stats.MysqldStarts),
		MysqldStops:   atomic.LoadUint64(&s.stats.MysqldStops),
		MonitorStarts: atomic.LoadUint64(&s.stats.MonitorStarts),
		MonitorStops:  atomic.LoadUint64(&s.stats.MonitorStops),
	}
}
