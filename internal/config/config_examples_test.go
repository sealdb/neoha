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

package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadExampleConfigs(t *testing.T) {
	root := filepath.Join("..", "..", "configs", "examples", "mysql")
	tests := []struct {
		path             string
		wantReplMode     string
		wantReplUser     string
		wantSemiSyncTout uint64
	}{
		{
			path:             filepath.Join(root, "semisync-node1.yaml"),
			wantReplMode:     "semi-sync",
			wantReplUser:     "repl",
			wantSemiSyncTout: 10000,
		},
		{
			path:         filepath.Join(root, "mgr-mysql80-node1.yaml"),
			wantReplMode: "group-replication",
			wantReplUser: "repl",
		},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			conf, err := LoadConfig(tt.path)
			assert.NoError(t, err)
			if err != nil {
				return
			}
			assert.Equal(t, tt.wantReplMode, string(conf.Database.Mysql.ReplMode))
			assert.Equal(t, tt.wantReplUser, conf.Database.Mysql.ReplUser)
			if tt.wantSemiSyncTout > 0 {
				assert.Equal(t, tt.wantSemiSyncTout, conf.Database.Mysql.SemiSyncTimeoutForTwoNodes)
			}
		})
	}

	jsonPath := filepath.Join(root, "semisync-node1.json")
	t.Run(jsonPath, func(t *testing.T) {
		conf, err := LoadConfig(jsonPath)
		assert.NoError(t, err)
		assert.Equal(t, "semi-sync", string(conf.Database.Mysql.ReplMode))
		assert.Equal(t, "repl", conf.Database.Mysql.ReplUser)
		assert.Equal(t, uint64(10000), conf.Database.Mysql.SemiSyncTimeoutForTwoNodes)
		assert.Equal(t, 1000, conf.Election.Raft.RequestTimeout)
	})
}
