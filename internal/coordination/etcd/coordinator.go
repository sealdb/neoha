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

// Package etcd provides an etcd-backed coordination.Coordinator (Patroni-style leader lease).
package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/sealdb/neoha/internal/config"
	"github.com/sealdb/neoha/internal/coordination"
)

// Coordinator implements coordination.Coordinator using an etcd leader key + lease.
type Coordinator struct {
	conf       *config.Config
	localID    string
	memberName string
	dbType     string

	client   *clientv3.Client
	leaseID  clientv3.LeaseID
	isLeader bool
	leaderID string

	mu     sync.RWMutex
	cancel context.CancelFunc
}

type memberRecord struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Database string `json:"database"`
	DBHost   string `json:"db_host,omitempty"`
	DBPort   int    `json:"db_port,omitempty"`
}

// New builds an etcd coordinator from runtime config.
func New(conf *config.Config) *Coordinator {
	if conf == nil {
		return &Coordinator{}
	}
	dbType := conf.Database.Type
	if dbType == "" {
		dbType = "mysql"
	}
	return &Coordinator{
		conf:       conf,
		localID:    conf.Endpoint,
		memberName: conf.Name,
		dbType:     dbType,
	}
}

func (c *Coordinator) Start(ctx context.Context) error {
	endpoints := etcdEndpoints(c.conf)
	if len(endpoints) == 0 {
		return coordination.ErrProviderNotImplemented
	}
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return err
	}
	c.client = cli
	runCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	if err := c.registerMember(runCtx); err != nil {
		_ = cli.Close()
		c.client = nil
		cancel()
		return err
	}
	go c.run(runCtx)
	return nil
}

func (c *Coordinator) Stop() error {
	if c.cancel != nil {
		c.cancel()
	}
	if c.client != nil {
		err := c.client.Close()
		c.client = nil
		return err
	}
	return nil
}

func (c *Coordinator) LocalID() string {
	return c.localID
}

func (c *Coordinator) IsLeader() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isLeader
}

func (c *Coordinator) ClusterView(ctx context.Context) (coordination.ClusterView, error) {
	if c.client == nil {
		return coordination.ClusterView{}, coordination.ErrNotSupported
	}
	c.mu.RLock()
	localLeader := c.leaderID
	c.mu.RUnlock()

	key := leaderKey(c.conf)
	resp, err := c.client.Get(ctx, key)
	if err != nil {
		return coordination.ClusterView{}, err
	}
	leaderID := localLeader
	if len(resp.Kvs) > 0 {
		leaderID = string(resp.Kvs[0].Value)
	}

	members, err := c.listMembers(ctx)
	if err != nil {
		return coordination.ClusterView{}, err
	}
	return coordination.ClusterView{
		LeaderID: leaderID,
		LeaderDatabase: c.leaderDatabaseEndpoint(members, leaderID),
		Members:  members,
	}, nil
}

func (c *Coordinator) Watch(ctx context.Context) (<-chan coordination.ClusterView, error) {
	if c.client == nil {
		return nil, coordination.ErrNotSupported
	}
	ch := make(chan coordination.ClusterView, 1)
	go func() {
		defer close(ch)
		ticker := time.NewTicker(500 * time.Millisecond)
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
				if view.LeaderID != last.LeaderID {
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

func (c *Coordinator) AddMember(ctx context.Context, id string) error {
	if c.client == nil {
		return coordination.ErrNotSupported
	}
	rec := memberRecord{ID: id, Name: id, Database: c.dbType}
	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	_, err = c.client.Put(ctx, memberKey(c.conf, id), string(data))
	return err
}

func (c *Coordinator) RemoveMember(ctx context.Context, id string) error {
	if c.client == nil {
		return coordination.ErrNotSupported
	}
	_, err := c.client.Delete(ctx, memberKey(c.conf, id))
	return err
}

func (c *Coordinator) GetClusterConfig(context.Context) ([]byte, error) {
	return nil, coordination.ErrNotSupported
}

func (c *Coordinator) SetClusterConfig(context.Context, []byte) error {
	return coordination.ErrNotSupported
}

func (c *Coordinator) registerMember(ctx context.Context) error {
	host, port := c.memberDatabaseEndpoint()
	rec := memberRecord{
		ID:       c.localID,
		Name:     c.memberName,
		Database: c.dbType,
		DBHost:   host,
		DBPort:   port,
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	_, err = c.client.Put(ctx, memberKey(c.conf, c.memberName), string(data))
	return err
}

func (c *Coordinator) listMembers(ctx context.Context) ([]coordination.Member, error) {
	resp, err := c.client.Get(ctx, memberPrefix(c.conf), clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	out := make([]coordination.Member, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var rec memberRecord
		if err := json.Unmarshal(kv.Value, &rec); err != nil {
			continue
		}
		out = append(out, coordination.Member{
			ID:       rec.ID,
			Name:     rec.Name,
			Database: rec.Database,
			Meta: map[string]string{
				"db_host": rec.DBHost,
				"db_port": fmt.Sprintf("%d", rec.DBPort),
			},
		})
	}
	return out, nil
}

func (c *Coordinator) run(ctx context.Context) {
	ttl := leaseTTL(c.conf)
	ticker := time.NewTicker(time.Duration(ttl/3+1) * time.Second)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		c.tryAcquireLeader(ctx, ttl)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (c *Coordinator) tryAcquireLeader(ctx context.Context, ttl int) {
	if c.client == nil {
		return
	}
	c.mu.RLock()
	leader := c.isLeader
	lease := c.leaseID
	c.mu.RUnlock()
	if leader && lease != 0 {
		_, err := c.client.KeepAliveOnce(ctx, lease)
		if err == nil {
			return
		}
		c.mu.Lock()
		c.isLeader = false
		c.leaseID = 0
		c.mu.Unlock()
	}

	leaseResp, err := c.client.Grant(ctx, int64(ttl))
	if err != nil {
		return
	}
	key := leaderKey(c.conf)
	txn := c.client.Txn(ctx).
		If(clientv3.Compare(clientv3.CreateRevision(key), "=", 0)).
		Then(clientv3.OpPut(key, c.localID, clientv3.WithLease(leaseResp.ID))).
		Else(clientv3.OpGet(key))
	resp, err := txn.Commit()
	if err != nil {
		return
	}
	if resp.Succeeded {
		c.mu.Lock()
		c.isLeader = true
		c.leaderID = c.localID
		c.leaseID = leaseResp.ID
		c.mu.Unlock()
		go c.client.KeepAlive(ctx, leaseResp.ID)
		return
	}
	leaderVal := ""
	if len(resp.Responses) > 0 {
		if get := resp.Responses[0].GetResponseRange(); get != nil && len(get.Kvs) > 0 {
			leaderVal = string(get.Kvs[0].Value)
		}
	}
	c.mu.Lock()
	c.isLeader = false
	c.leaderID = leaderVal
	c.leaseID = 0
	c.mu.Unlock()
}

var _ coordination.Coordinator = (*Coordinator)(nil)
