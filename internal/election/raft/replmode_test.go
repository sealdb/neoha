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

package raft

import (
	"testing"

	"github.com/sealdb/neoha/internal/base/model"
)

// forEachMySQLReplMode runs fn as subtests for Semi-Sync and MGR.
func forEachMySQLReplMode(t *testing.T, fn func(t *testing.T, replMode model.MysqlReplMode)) {
	t.Helper()
	for _, tc := range []struct {
		name string
		mode model.MysqlReplMode
	}{
		{"SemiSync", model.ReplModeSemiSync},
		{"MGR", model.ReplModeMGR},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fn(t, tc.mode)
		})
	}
}
