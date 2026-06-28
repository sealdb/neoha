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

package raftadapter_test

import (
	"context"
	"testing"

	"github.com/sealdb/neoha/internal/coordination"
	"github.com/sealdb/neoha/internal/coordination/raftadapter"
	"github.com/sealdb/neoha/internal/election/raft"
	"github.com/stretchr/testify/assert"
)

func TestRaftAdapterClusterView(t *testing.T) {
	r := &raft.Raft{}
	// Use zero-value raft only for method presence; full tests live in election/raft.
	c := raftadapter.New(r, "10.0.0.1:8080", "node1", "mysql")
	assert.NotNil(t, c)
	assert.Equal(t, "10.0.0.1:8080", c.LocalID())

	_, err := c.GetClusterConfig(context.Background())
	assert.ErrorIs(t, err, coordination.ErrNotSupported)
}
