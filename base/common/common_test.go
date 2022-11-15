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

package common

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPathExists(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{name: "FileExists", args: args{path: "/etc/hosts"}, want: true, wantErr: false},
		{name: "FileNotExist", args: args{path: "/etc/neoha-test"}, want: false, wantErr: false},
		{name: "DirExists", args: args{path: "/etc/"}, want: true, wantErr: false},
		{name: "DirNotExist", args: args{path: "/etc/neoha-test/"}, want: false, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PathExists(tt.args.path)
			assert.Equal(t, err != nil, tt.wantErr)
			if got != tt.want {
				t.Errorf("PathExists() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetFileType(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want FileType
	}{
		{name: "FileTypeIsYaml", args: args{path: "/tmp/neoha-test.yaml"}, want: YamlType},
		{name: "FileTypeIsYml", args: args{path: "/tmp/neoha-test.yml"}, want: YmlType},
		{name: "FileTypeIsJson", args: args{path: "/tmp/neoha-test.json"}, want: JsonType},
		{name: "FileTypeIsUnknown", args: args{path: "/tmp/neoha-test.conf"}, want: ".conf"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetFileType(tt.args.path)
			if got != tt.want {
				t.Errorf("PathExists() got = %v, want %v", got, tt.want)
			}
		})
	}
}
