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

package ha

import (
	"context"
	"testing"
	"time"

	"github.com/sealdb/neoha/internal/base/nlog"
	"github.com/sealdb/neoha/internal/config"
	"github.com/sealdb/neoha/internal/coordination"
	dbdriver "github.com/sealdb/neoha/internal/database/driver"
	"github.com/stretchr/testify/assert"
)

type stubCoord struct {
	leader bool
}

func (c *stubCoord) Start(context.Context) error              { return nil }
func (c *stubCoord) Stop() error                                { return nil }
func (c *stubCoord) LocalID() string                            { return "n1" }
func (c *stubCoord) IsLeader() bool                             { return c.leader }
func (c *stubCoord) ClusterView(context.Context) (coordination.ClusterView, error) {
	return coordination.ClusterView{LeaderID: "leader"}, nil
}
func (c *stubCoord) Watch(context.Context) (<-chan coordination.ClusterView, error) {
	return nil, coordination.ErrNotSupported
}
func (c *stubCoord) AddMember(context.Context, string) error    { return coordination.ErrNotSupported }
func (c *stubCoord) RemoveMember(context.Context, string) error { return coordination.ErrNotSupported }
func (c *stubCoord) GetClusterConfig(context.Context) ([]byte, error) {
	return nil, coordination.ErrNotSupported
}
func (c *stubCoord) SetClusterConfig(context.Context, []byte) error {
	return coordination.ErrNotSupported
}

type stubDriver struct {
	role          dbdriver.Role
	writable      bool
	promotable    bool
	demoteCalls   int
	applyPrimary  int
	applyReplica  int
}

func (d *stubDriver) Type() dbdriver.Engine { return dbdriver.EngineMySQL }
func (d *stubDriver) Start()                {}
func (d *stubDriver) Stop()                 {}
func (d *stubDriver) SetupBootstrap(context.Context) error { return nil }
func (d *stubDriver) Promotable(context.Context) (bool, string) {
	if d.promotable {
		return true, ""
	}
	return false, "not.promotable"
}
func (d *stubDriver) ReplicationLagBytes(context.Context) (int64, error) { return 0, nil }
func (d *stubDriver) ApplyPrimary(context.Context) error {
	d.applyPrimary++
	d.writable = true
	d.role = dbdriver.RolePrimary
	return nil
}
func (d *stubDriver) ApplyReplica(context.Context, dbdriver.PrimaryRef) error {
	d.applyReplica++
	return nil
}
func (d *stubDriver) Demote(context.Context) error {
	d.demoteCalls++
	d.role = dbdriver.RoleReplica
	d.writable = false
	return nil
}
func (d *stubDriver) Status(context.Context) (dbdriver.Health, error) {
	return dbdriver.Health{
		Alive:    true,
		Writable: d.writable,
		Role:     d.role,
	}, nil
}

func TestReconcilerRunOnceNilCoord(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	r := NewReconciler(log, nil, nil, config.DefaultTagsConfig())
	assert.NoError(t, r.RunOnce(context.Background()))
}

func TestReconcilerApplyDemoteWhenNotLeader(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	drv := &stubDriver{role: dbdriver.RolePrimary, writable: true}
	r := NewReconciler(log, &stubCoord{leader: false}, drv, config.DefaultTagsConfig())

	assert.NoError(t, r.RunOnce(context.Background()))
	assert.Equal(t, 1, drv.demoteCalls)
	assert.Equal(t, 0, drv.applyPrimary)
}

func TestReconcilerNoOpWhenAlreadyReplica(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	drv := &stubDriver{role: dbdriver.RoleReplica, writable: false}
	r := NewReconciler(log, &stubCoord{leader: false}, drv, config.DefaultTagsConfig())

	assert.NoError(t, r.RunOnce(context.Background()))
	assert.Equal(t, 0, drv.demoteCalls)
}

func TestReconcilerSkipPromoteByDefault(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	drv := &stubDriver{role: dbdriver.RoleReplica, writable: false, promotable: true}
	r := NewReconciler(log, &stubCoord{leader: true}, drv, config.DefaultTagsConfig())

	assert.NoError(t, r.RunOnce(context.Background()))
	assert.Equal(t, 0, drv.applyPrimary)
}

func TestReconcilerApplyPromoteWhenEnabled(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	drv := &stubDriver{role: dbdriver.RoleReplica, writable: false, promotable: true}
	r := NewReconciler(log, &stubCoord{leader: true}, drv, config.DefaultTagsConfig(), WithApplyPromote(true))

	assert.NoError(t, r.RunOnce(context.Background()))
	assert.Equal(t, 1, drv.applyPrimary)
}

func TestReconcilerLoopStopsOnCancel(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	drv := &stubDriver{role: dbdriver.RoleReplica, writable: false}
	r := NewReconciler(log, &stubCoord{}, drv, config.DefaultTagsConfig(), WithInterval(10*time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		r.Loop(ctx)
		close(done)
	}()

	time.Sleep(30 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("reconcile loop did not stop")
	}
}

type stubMGRDriver struct {
	stubDriver
	mgrMode       bool
	phase1Done    bool
	clusterReady  bool
	phase2Calls   int
}

func (d *stubMGRDriver) IsMGRMode() bool { return d.mgrMode }
func (d *stubMGRDriver) MGRPrimaryPhase1Done(context.Context) (bool, error) {
	return d.phase1Done, nil
}
func (d *stubMGRDriver) MGRClusterWritableReady(context.Context) (bool, error) {
	return d.clusterReady, nil
}
func (d *stubMGRDriver) ApplyPrimaryMGRPhase2(context.Context) error {
	d.phase2Calls++
	d.writable = true
	d.role = dbdriver.RolePrimary
	return nil
}
func (d *stubMGRDriver) MGRReplicaJoined(context.Context) (bool, error) {
	return false, nil
}

func TestReconcilerApplyReplicaWhenFollower(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	drv := &stubDriver{role: dbdriver.RoleReplica, writable: false}
	r := NewReconciler(log, &stubCoord{leader: false}, drv, config.DefaultTagsConfig(),
		WithApplyPromote(true),
	)
	desired := DesiredState{
		DBRole: dbdriver.RoleReplica,
		Primary: dbdriver.PrimaryRef{
			MemberID: "127.0.0.1:8081",
			Host:     "127.0.0.1",
			Port:     13316,
		},
		Reason: "not.coord.leader",
	}
	assert.NoError(t, r.applyReplicaIfNeeded(context.Background(), desired))
	assert.Equal(t, 1, drv.applyReplica)
}

func TestReconcilerReapplyMGRReplicaWhenNotJoined(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	drv := &stubMGRDriver{
		stubDriver: stubDriver{role: dbdriver.RoleReplica, writable: false},
		mgrMode:    true,
	}
	r := NewReconciler(log, &stubCoord{leader: false}, drv, config.DefaultTagsConfig(), WithApplyPromote(true))
	r.lastReplicaPrimary = "127.0.0.1:13308"

	desired := DesiredState{
		DBRole: dbdriver.RoleReplica,
		Primary: dbdriver.PrimaryRef{
			MemberID: "127.0.0.1:18103",
			Host:     "127.0.0.1",
			Port:     13308,
		},
		Reason: "not.coord.leader",
	}
	assert.NoError(t, r.applyReplicaIfNeeded(context.Background(), desired))
	assert.Equal(t, 1, drv.applyReplica)
	assert.Equal(t, "127.0.0.1:13308", r.lastReplicaPrimary)
}

func TestEnrichPrimaryRef(t *testing.T) {
	ref := enrichPrimaryRef(dbdriver.PrimaryRef{MemberID: "10.0.0.2:8080"}, 3306)
	assert.Equal(t, "10.0.0.2", ref.Host)
	assert.Equal(t, 3306, ref.Port)

	preset := enrichPrimaryRef(dbdriver.PrimaryRef{
		MemberID: "127.0.0.1:8081",
		Host:     "127.0.0.1",
		Port:     13316,
	}, 13317)
	assert.Equal(t, 13316, preset.Port)
}

func TestReconcilerMGRPhase2(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	drv := &stubMGRDriver{
		stubDriver: stubDriver{
			role:       dbdriver.RoleReplica,
			writable:   false,
			promotable: true,
		},
		mgrMode:      true,
		phase1Done:   true,
		clusterReady: true,
	}
	r := NewReconciler(log, &stubCoord{leader: true}, drv, config.DefaultTagsConfig(), WithApplyPromote(true))

	assert.NoError(t, r.RunOnce(context.Background()))
	assert.Equal(t, 1, drv.phase2Calls)
	assert.True(t, drv.writable)
}
