/*
 * Copyright 2022-2026 The NeoHA Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package harness

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeepWorkDir(t *testing.T) {
	t.Setenv(EnvTeardownWorkDir, "")
	t.Setenv(EnvKeepWorkDir, "")
	assert.True(t, KeepWorkDir())

	t.Setenv(EnvTeardownWorkDir, "1")
	assert.False(t, KeepWorkDir())

	t.Setenv(EnvTeardownWorkDir, "")
	t.Setenv(EnvKeepWorkDir, "0")
	assert.False(t, KeepWorkDir())
}

func TestNeoHABinPathFromRepoBin(t *testing.T) {
	root, err := repoRootFromCWD()
	if err != nil {
		t.Skip("repo root not found from cwd")
	}
	bin := filepath.Join(root, "bin", "neoha")
	if _, err := os.Stat(bin); err != nil {
		t.Skipf("%s not built", bin)
	}
	t.Setenv(EnvNeoHABin, "")
	s := &IntegrationSettings{}
	assert.Equal(t, bin, s.NeoHABinPath())
}
