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
	"strings"

	"github.com/sealdb/neoha/internal/base/common"
	"github.com/sealdb/neoha/internal/base/nlog"
	"github.com/sealdb/neoha/internal/config"
)

const shellBash = "bash"

// PrimaryHooks runs configured actions when this node starts/stops serving as the writable primary.
// Examples: bind VIP, notify load balancer, custom orchestration scripts.
type PrimaryHooks struct {
	log           *nlog.Log
	cmd           common.Command
	onStart       string
	onStop        string
	active        bool
	skipStartOnce bool
}

// NewPrimaryHooks builds a hook runner from config (commands may be "nop").
func NewPrimaryHooks(log *nlog.Log, hooks config.PrimaryHooksConfig) *PrimaryHooks {
	return &PrimaryHooks{
		log:     log,
		cmd:     common.NewLinuxCommand(log),
		onStart: hooks.OnPrimaryStart,
		onStop:  hooks.OnPrimaryStop,
	}
}

// SetSkipPrimaryStartOnce skips the next OnPrimaryStart (e.g. neoha -r LEADER).
func (h *PrimaryHooks) SetSkipPrimaryStartOnce() {
	if h == nil {
		return
	}
	h.skipStartOnce = true
}

// Active reports whether the start hook has been applied on this process.
func (h *PrimaryHooks) Active() bool {
	if h == nil {
		return false
	}
	return h.active
}

// EnsurePrimary runs on_primary_start when the DB is writable and serving as primary.
func (h *PrimaryHooks) EnsurePrimary(ctx context.Context) error {
	if h == nil || !hookConfigured(h.onStart) {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if h.active {
		return nil
	}
	if h.skipStartOnce {
		h.skipStartOnce = false
		h.active = true
		h.log.Warning("primary.hooks.start.skipped[init.role.leader]")
		return nil
	}
	if err := h.run(h.onStart, "primary.hooks.start"); err != nil {
		return err
	}
	h.active = true
	return nil
}

// EnsureReplica runs on_primary_stop when the node should not expose primary accessibility.
func (h *PrimaryHooks) EnsureReplica(ctx context.Context) error {
	if h == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if !h.active && !hookConfigured(h.onStop) {
		return nil
	}
	if !h.active {
		return h.runStopIfConfigured("primary.hooks.stop.defensive")
	}
	if err := h.runStopIfConfigured("primary.hooks.stop"); err != nil {
		return err
	}
	h.active = false
	return nil
}

// RunStopIfConfigured executes on_primary_stop once (startup cleanup on followers).
func (h *PrimaryHooks) RunStopIfConfigured(ctx context.Context) error {
	if h == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return h.runStopIfConfigured("primary.hooks.stop.startup")
}

func (h *PrimaryHooks) runStopIfConfigured(label string) error {
	if !hookConfigured(h.onStop) {
		return nil
	}
	return h.run(h.onStop, label)
}

func (h *PrimaryHooks) run(command, label string) error {
	if !hookConfigured(command) {
		return nil
	}
	args := []string{"-c", command}
	if out, err := h.cmd.RunCommand(shellBash, args); err != nil {
		h.log.Error("%s[%v].out[%v].error[%+v]", label, args, out, err)
		return err
	}
	h.log.Warning("%s.done", label)
	return nil
}

func hookConfigured(command string) bool {
	c := strings.TrimSpace(command)
	return c != "" && c != "nop"
}
