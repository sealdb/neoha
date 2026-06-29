/*
 * Copyright 2022-2026 The NeoHA Authors.
 *
 * See the AUTHORS file for a list of contributors.
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

package etcd

import (
	"context"
	"testing"

	"github.com/sealdb/neoha/internal/config"
	"github.com/sealdb/neoha/internal/coordination"
	"github.com/stretchr/testify/assert"
)

func TestEtcdCoordinatorStubNoEndpoints(t *testing.T) {
	conf := config.DefaultConfig()
	conf.Endpoint = "127.0.0.1:8080"
	conf.Name = "n1"
	c := New(conf)
	assert.Equal(t, "127.0.0.1:8080", c.LocalID())
	assert.False(t, c.IsLeader())
	assert.ErrorIs(t, c.Start(context.Background()), coordination.ErrProviderNotImplemented)
	_, err := c.ClusterView(context.Background())
	assert.ErrorIs(t, err, coordination.ErrNotSupported)
}
