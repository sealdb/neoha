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

package mysql

import (
	"sync/atomic"

	"neoha/base/model"
)

// IncMysqlDowns used to increase the mysql down counter.
func (s *Mysql) IncMysqlDowns() {
	atomic.AddUint64(&s.stats.MysqlDowns, 1)
}

func (s *Mysql) getStats() *model.MysqlStats {
	return &model.MysqlStats{
		MysqlDowns: atomic.LoadUint64(&s.stats.MysqlDowns),
	}
}
