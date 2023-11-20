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
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestWriteConfig must run before TestLoadConfig
func TestWriteConfig(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "yaml", args: args{path: "/tmp/test.config.yaml"}, wantErr: false},
		{name: "yml", args: args{path: "/tmp/test.config.yml"}, wantErr: false},
		{name: "json", args: args{path: "/tmp/test.config.json"}, wantErr: false},
		{name: "txt", args: args{path: "/tmp/test.config.txt"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Remove(tt.args.path)
			conf := DefaultConfig()
			conf.Scope = "neoha"
			err := WriteConfig(tt.args.path, conf)
			assert.Equal(t, err != nil, tt.wantErr)
			if tt.wantErr {
				os.Create(tt.args.path) // create txt file for TestLoadConfig
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// same as TestWriteConfig
		{name: "yaml", args: args{path: "/tmp/test.config.yaml"}, wantErr: false},
		{name: "yml", args: args{path: "/tmp/test.config.yml"}, wantErr: false},
		{name: "json", args: args{path: "/tmp/test.config.json"}, wantErr: false},
		{name: "txt", args: args{path: "/tmp/test.config.txt"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := DefaultConfig()
			want.Scope = "neoha" // same as TestWriteConfig
			got, err := LoadConfig(tt.args.path)
			assert.Equal(t, err != nil, tt.wantErr)
			if tt.wantErr == false {
				assert.Equal(t, want, got)
			}
			os.Remove(tt.args.path)
		})
	}
}
