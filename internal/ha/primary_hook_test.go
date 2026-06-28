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

package ha

import (
	"context"
	"testing"

	"github.com/sealdb/neoha/internal/base/nlog"
	"github.com/sealdb/neoha/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestPrimaryHooksNop(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	h := NewPrimaryHooks(log, config.PrimaryHooksConfig{OnPrimaryStart: "nop", OnPrimaryStop: "nop"})
	assert.NoError(t, h.EnsurePrimary(context.Background()))
	assert.False(t, h.Active())
}

func TestPrimaryHooksSkipStartOnce(t *testing.T) {
	log := nlog.NewStdLog(nlog.Level(nlog.PANIC))
	h := NewPrimaryHooks(log, config.PrimaryHooksConfig{OnPrimaryStart: "true", OnPrimaryStop: "nop"})
	h.SetSkipPrimaryStartOnce()
	assert.NoError(t, h.EnsurePrimary(context.Background()))
	assert.True(t, h.Active())
}

func TestEffectivePrimaryHooksLegacyFallback(t *testing.T) {
	conf := config.DefaultConfig()
	conf.HA.PrimaryHooks = &config.PrimaryHooksConfig{}
	conf.Election.Raft.LeaderStartCommand = "start-cmd"
	conf.Election.Raft.LeaderStopCommand = "stop-cmd"
	hooks := conf.EffectivePrimaryHooks()
	assert.Equal(t, "start-cmd", hooks.OnPrimaryStart)
	assert.Equal(t, "stop-cmd", hooks.OnPrimaryStop)
}
