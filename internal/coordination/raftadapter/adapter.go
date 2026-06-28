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

// Package raftadapter wraps internal/election/raft as coordination.Coordinator.
package raftadapter

import (
	"context"
	"time"

	"github.com/sealdb/neoha/internal/coordination"
	"github.com/sealdb/neoha/internal/election/raft"
)

const watchPollInterval = 200 * time.Millisecond

// Coordinator adapts *raft.Raft to coordination.Coordinator.
type Coordinator struct {
	raft       *raft.Raft
	localID    string
	memberName string
	dbType     string
}

// New returns a Raft-backed coordinator. Raft Start/Stop remain owned by election.Election.
func New(r *raft.Raft, localID, memberName, dbType string) *Coordinator {
	return &Coordinator{
		raft:       r,
		localID:    localID,
		memberName: memberName,
		dbType:     dbType,
	}
}

func (c *Coordinator) Start(context.Context) error {
	return nil
}

func (c *Coordinator) Stop() error {
	return nil
}

func (c *Coordinator) LocalID() string {
	return c.localID
}

func (c *Coordinator) IsLeader() bool {
	return c.raft.GetState() == raft.LEADER
}

func (c *Coordinator) ClusterView(_ context.Context) (coordination.ClusterView, error) {
	peers := c.raft.GetPeers()
	members := make([]coordination.Member, 0, len(peers)+1)
	members = append(members, coordination.Member{
		ID:       c.localID,
		Name:     c.memberName,
		Database: c.dbType,
	})
	for _, p := range peers {
		members = append(members, coordination.Member{
			ID:       p,
			Database: c.dbType,
		})
	}
	host, port := c.raft.GetLeaderDatabaseEndpoint()
	return coordination.ClusterView{
		LeaderID: c.raft.GetLeader(),
		LeaderDatabase: coordination.DatabaseEndpoint{
			Host: host,
			Port: port,
		},
		Members:  members,
		Epoch:    c.raft.GetEpochID(),
		ViewID:   c.raft.GetVewiID(),
	}, nil
}

func (c *Coordinator) Watch(ctx context.Context) (<-chan coordination.ClusterView, error) {
	ch := make(chan coordination.ClusterView, 1)
	go func() {
		defer close(ch)
		ticker := time.NewTicker(watchPollInterval)
		defer ticker.Stop()
		var last coordination.ClusterView
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				view, err := c.ClusterView(ctx)
				if err != nil {
					continue
				}
				if view.LeaderID != last.LeaderID || view.Epoch != last.Epoch || view.ViewID != last.ViewID {
					last = view
					select {
					case ch <- view:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()
	return ch, nil
}

func (c *Coordinator) AddMember(_ context.Context, id string) error {
	return c.raft.AddPeer(id)
}

func (c *Coordinator) RemoveMember(_ context.Context, id string) error {
	return c.raft.RemovePeer(id)
}

func (c *Coordinator) GetClusterConfig(context.Context) ([]byte, error) {
	return nil, coordination.ErrNotSupported
}

func (c *Coordinator) SetClusterConfig(context.Context, []byte) error {
	return coordination.ErrNotSupported
}

// Compile-time check.
var _ coordination.Coordinator = (*Coordinator)(nil)
