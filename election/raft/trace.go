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

package raft

import (
	"fmt"
)

// log wrapper for raft
func (r *Raft) logMsg(format string, v ...interface{}) string {
	return fmt.Sprintf("%v[ID:%v, V:%v, E:%v].%v", r.state.String(), r.getID(), r.getViewID(), r.getEpochID(), fmt.Sprintf(format, v...))
}

// DEBUG level log.
func (r *Raft) DEBUG(format string, v ...interface{}) {
	r.log.Debug("%v", r.logMsg(format, v...))
}

// INFO level log.
func (r *Raft) INFO(format string, v ...interface{}) {
	r.log.Info("%v", r.logMsg(format, v...))
}

// WARNING level log.
func (r *Raft) WARNING(format string, v ...interface{}) {
	r.log.Warning("%v", r.logMsg(format, v...))
}

// ERROR level log.
func (r *Raft) ERROR(format string, v ...interface{}) {
	r.log.Error("%v", r.logMsg(format, v...))
}

// PANIC level log.
func (r *Raft) PANIC(format string, v ...interface{}) {
	r.log.Panic("%v", r.logMsg(format, v...))
}
