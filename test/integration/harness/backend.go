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

// Package harness orchestrates databases and NeoHA nodes for integration tests.
// Backend is intentionally database-agnostic so PostgreSQL support can plug in later.
package harness

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

const (
	EnvMySQLBase = "NEOHA_IT_MYSQL_BASE"
	EnvWorkDir   = "NEOHA_IT_WORKDIR"
)

// Backend manages lifecycle of a database engine (MySQL, PostgreSQL, …).
type Backend interface {
	Name() string
	InitNode(ctx context.Context, node *Node) error
	StartNode(ctx context.Context, node *Node) error
	StopNode(ctx context.Context, node *Node) error
	Ready(ctx context.Context, node *Node) error
}

// Node is one database (+ optional NeoHA agent) in a test cluster.
type Node struct {
	Name     string
	Port     int
	GRPort   int // group_replication local address port (MySQL MGR)
	DataDir  string
	Config   string // path to my.cnf or postgresql.conf
	Socket   string
	WorkDir  string
	cmd      *os.Process
	backend  Backend
}

// Cluster groups nodes sharing a work directory.
type Cluster struct {
	Name    string
	WorkDir string
	Backend Backend
	Nodes   []*Node
}

// NewCluster creates a cluster under workDir/name.
func NewCluster(name, workDir string, backend Backend) *Cluster {
	if workDir == "" {
		workDir = os.TempDir()
	}
	return &Cluster{
		Name:    name,
		WorkDir: filepath.Join(workDir, name),
		Backend: backend,
	}
}

// AddNode registers a node; ports are chosen by the caller.
func (c *Cluster) AddNode(name string, port, grPort int) *Node {
	node := &Node{
		Name:    name,
		Port:    port,
		GRPort:  grPort,
		WorkDir: filepath.Join(c.WorkDir, name),
		backend: c.Backend,
	}
	c.Nodes = append(c.Nodes, node)
	return node
}

// Setup initializes datadirs and configs for all nodes.
func (c *Cluster) Setup(ctx context.Context) error {
	return c.SetupFresh(ctx, true)
}

// SetupFresh initializes nodes; when clean is true, stale workdir/processes are removed first.
func (c *Cluster) SetupFresh(ctx context.Context, clean bool) error {
	if clean {
		KillProcessesOnWorkDir(c.WorkDir)
		_ = os.RemoveAll(c.WorkDir)
	}
	if err := os.MkdirAll(c.WorkDir, 0o755); err != nil {
		return err
	}
	for _, node := range c.Nodes {
		if err := node.backend.InitNode(ctx, node); err != nil {
			return fmt.Errorf("init node %s: %w", node.Name, err)
		}
	}
	return nil
}

// StartAll starts every node and waits until ready.
func (c *Cluster) StartAll(ctx context.Context) error {
	for _, node := range c.Nodes {
		if err := node.backend.StartNode(ctx, node); err != nil {
			return fmt.Errorf("start node %s: %w", node.Name, err)
		}
		if err := node.backend.Ready(ctx, node); err != nil {
			return fmt.Errorf("ready node %s: %w", node.Name, err)
		}
	}
	return nil
}

// StopAll stops all nodes (best effort).
func (c *Cluster) StopAll(ctx context.Context) {
	for i := len(c.Nodes) - 1; i >= 0; i-- {
		_ = c.Nodes[i].backend.StopNode(ctx, c.Nodes[i])
	}
}

// Teardown stops nodes and removes the work directory.
func (c *Cluster) Teardown(ctx context.Context) error {
	c.StopAll(ctx)
	return os.RemoveAll(c.WorkDir)
}

// MySQLBaseFromEnv returns the MySQL installation root or empty if unset.
func MySQLBaseFromEnv() string {
	return os.Getenv(EnvMySQLBase)
}

// WorkDirFromEnv returns the integration test work directory.
func WorkDirFromEnv() string {
	if dir := os.Getenv(EnvWorkDir); dir != "" {
		return dir
	}
	return filepath.Join(os.TempDir(), "neoha-it")
}
