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
	"fmt"

	"neoha/base/model"
)

func (m *Mysql) setState(state model.MysqlState) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.state = state
}

func (m *Mysql) getState() model.MysqlState {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.state
}

func (m *Mysql) setOption(o Option) {
	m.option = o
}

func (m *Mysql) getOption() Option {
	return m.option
}

func (m *Mysql) getConnStr() string {
	return fmt.Sprintf("%s:%d", m.conf.Host, m.conf.Port)
}
