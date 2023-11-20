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

package neohactl

import (
	"fmt"
	"testing"

	"github.com/sealdb/neoha/internal/base/common"

	"github.com/stretchr/testify/assert"
)

func TestCLIInitCommand(t *testing.T) {
	err := createConfig()
	ErrorOK(err)
	defer removeConfig()
	ip, err := common.GetLocalIP()
	assert.Nil(t, err)

	cmd := NewInitCommand()
	_, err = executeCommand(cmd, "init", "--address", ip, "--port", "8080", "--repluser", "repl", "--replpwd", "repl", "--vip", ip)
	assert.Nil(t, err)

	conf, err := GetConfig()
	assert.Nil(t, err)

	want := fmt.Sprintf("%s:8080", ip)
	got := conf.Endpoint
	assert.Equal(t, want, got)
}
