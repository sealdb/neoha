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
	"os"
	"path/filepath"
	"testing"

	"github.com/sealdb/neoha/internal/base/common"
	"github.com/stretchr/testify/assert"
)

func TestDetectConfigFormat(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		data     string
		want     common.FileType
		wantErr  bool
	}{
		{name: "yaml ext", path: "/etc/neoha/db.yaml", data: "scope: neoha\n", want: common.YamlType},
		{name: "yml ext", path: "/etc/neoha/db.yml", data: "scope: neoha\n", want: common.YmlType},
		{name: "json ext", path: "/etc/neoha/db.json", data: `{"scope":"neoha"}`, want: common.JsonType},
		{name: "sniff json", path: "/etc/neoha/db.conf", data: `{"scope":"neoha"}`, want: common.JsonType},
		{name: "sniff yaml", path: "/etc/neoha/db.conf", data: "scope: neoha\n", want: common.YamlType},
		{name: "empty", path: "/etc/neoha/db.conf", data: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DetectConfigFormat(tt.path, []byte(tt.data))
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLoadConfigYAMLAndJSONEquivalent(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "neoha.yaml")
	jsonPath := filepath.Join(dir, "neoha.json")

	conf := DefaultConfig()
	conf.Scope = "neoha-json-yaml"
	conf.Database.Mysql.ReplMode = "semi-sync"
	conf.Election.Raft.RequestTimeout = 1500

	assert.NoError(t, WriteConfig(yamlPath, conf))
	assert.NoError(t, WriteConfig(jsonPath, conf))

	fromYAML, err := LoadConfig(yamlPath)
	assert.NoError(t, err)
	fromJSON, err := LoadConfig(jsonPath)
	assert.NoError(t, err)

	assert.Equal(t, fromYAML, fromJSON)
	assert.Equal(t, "neoha-json-yaml", fromJSON.Scope)
	assert.Equal(t, 1500, fromJSON.Election.Raft.RequestTimeout)
}

func TestLoadConfigSniffJSONWithoutExtension(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "neoha.conf")

	conf := DefaultConfig()
	conf.Scope = "sniff-json"
	assert.NoError(t, WriteConfig(filepath.Join(dir, "seed.json"), conf))

	data, err := os.ReadFile(filepath.Join(dir, "seed.json"))
	assert.NoError(t, err)
	assert.NoError(t, os.WriteFile(path, data, 0o644))

	got, err := LoadConfig(path)
	assert.NoError(t, err)
	assert.Equal(t, "sniff-json", got.Scope)
}

func TestParseConfigInvalidJSON(t *testing.T) {
	_, err := ParseConfig([]byte(`{"scope":`), common.JsonType)
	assert.Error(t, err)
}
