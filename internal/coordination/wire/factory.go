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

// Package wire constructs coordination.Coordinator implementations from config.
package wire

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/sealdb/neoha/internal/config"
	"github.com/sealdb/neoha/internal/coordination"
	"github.com/sealdb/neoha/internal/coordination/etcd"
	"github.com/sealdb/neoha/internal/coordination/raftadapter"
	"github.com/sealdb/neoha/internal/election/raft"
)

// NewCoordinator builds the coordination backend selected in config.
func NewCoordinator(conf *config.Config, r *raft.Raft) (coordination.Coordinator, error) {
	if conf == nil {
		return nil, errors.New("config is nil")
	}
	provider := conf.EffectiveCoordination().Provider
	switch provider {
	case coordination.ProviderRaft:
		if r == nil {
			return nil, errors.New("raft coordinator requires *raft.Raft")
		}
		return raftadapter.New(r, conf.Endpoint, conf.Name, conf.Database.Type), nil
	case coordination.ProviderEtcd:
		return etcd.New(conf), nil
	case coordination.ProviderConsul:
		return nil, coordination.ErrProviderNotImplemented
	case coordination.ProviderKubernetes:
		return nil, coordination.ErrProviderNotImplemented
	case coordination.ProviderZooKeeper:
		return nil, coordination.ErrProviderNotImplemented
	default:
		return nil, fmt.Errorf("unsupported coordination provider %q", provider)
	}
}
