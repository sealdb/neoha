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
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPeersJson(t *testing.T) {
	path := "/tmp/test.peersjson"
	peers := []string{":0101", ":0202"}
	idlePeers := []string{":0303", ":0404"}

	{
		err := writePeersJSON(path, peers, idlePeers)
		assert.Nil(t, err)
		os.Remove(path)
	}

	// read error
	{
		_, _, err := readPeersJSON(path)
		want := fmt.Sprintf("open %s: no such file or directory", path)
		got := err.Error()
		assert.Equal(t, want, got)
	}

	// write json
	{
		err := writePeersJSON(path, peers, idlePeers)
		assert.Nil(t, err)
	}

	// read json OK
	{
		ps, ips, err := readPeersJSON(path)
		assert.Nil(t, err)
		assert.Equal(t, peers, ps)
		assert.Equal(t, idlePeers, ips)
	}

	// json broken
	{
		f, err := os.OpenFile(path, os.O_RDWR, 0644)
		assert.Nil(t, err)
		defer f.Close()

		_, err = f.WriteString("inject")
		assert.Nil(t, err)
	}

	// read error
	{
		_, _, err := readPeersJSON(path)
		want := "invalid character 'i' looking for beginning of value"
		got := err.Error()
		assert.Equal(t, want, got)
	}
}
