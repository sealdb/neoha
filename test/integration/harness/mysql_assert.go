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

package harness

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// MGRMemberRole returns MEMBER_ROLE for the local node (PRIMARY/SECONDARY). Read-only assertion helper.
func (b *MySQL80) MGRMemberRole(node *Node) (string, error) {
	dsn := fmt.Sprintf("root@tcp(127.0.0.1:%d)/", node.Port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return "", err
	}
	defer db.Close()

	var role string
	err = db.QueryRow(`
		SELECT MEMBER_ROLE FROM performance_schema.replication_group_members
		WHERE MEMBER_ID = @@server_uuid`).Scan(&role)
	return role, err
}

// OnlineMGRMembers counts ONLINE members visible from node. Read-only assertion helper.
func (b *MySQL80) OnlineMGRMembers(node *Node) (int, error) {
	dsn := fmt.Sprintf("root@tcp(127.0.0.1:%d)/", node.Port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	var cnt int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM performance_schema.replication_group_members
		WHERE MEMBER_STATE='ONLINE'`).Scan(&cnt)
	return cnt, err
}

// FindMGRPrimaryNode returns the cluster node whose local mysqld reports PRIMARY.
func (b *MySQL80) FindMGRPrimaryNode(nodes []*Node) (*Node, error) {
	for _, node := range nodes {
		role, err := b.MGRMemberRole(node)
		if err == nil && role == "PRIMARY" {
			return node, nil
		}
	}
	return nil, fmt.Errorf("no MGR PRIMARY among %d nodes", len(nodes))
}

func (b *MySQL80) WaitMGRPrimaryOnAny(ctx context.Context, nodes []*Node) (*Node, error) {
	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		for _, node := range nodes {
			role, err := b.MGRMemberRole(node)
			if err == nil && role == "PRIMARY" {
				return node, nil
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
}

// WaitMGROnlineMembers waits until at least want ONLINE MGR members are visible from node.
func (b *MySQL80) WaitMGROnlineMembers(ctx context.Context, node *Node, want int) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		cnt, err := b.OnlineMGRMembers(node)
		if err == nil && cnt >= want {
			return nil
		}
		time.Sleep(250 * time.Millisecond)
	}
}

// WaitMGROnlineMembersBelow waits until fewer than maxExclusive ONLINE members are visible from node.
func (b *MySQL80) WaitMGROnlineMembersBelow(ctx context.Context, node *Node, maxExclusive int) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		cnt, err := b.OnlineMGRMembers(node)
		if err == nil && cnt < maxExclusive {
			return nil
		}
		time.Sleep(250 * time.Millisecond)
	}
}

// ReplicaStatus holds read-only fields from SHOW REPLICA STATUS.
type ReplicaStatus struct {
	MasterHost      string
	MasterPort      int
	SlaveIORunning  string
	SlaveSQLRunning string
}

// ReplicaStatus returns replication thread state for node. Read-only assertion helper.
func (b *MySQL80) ReplicaStatus(node *Node) (*ReplicaStatus, error) {
	dsn := fmt.Sprintf("root@tcp(127.0.0.1:%d)/", node.Port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query("SHOW SLAVE STATUS")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, fmt.Errorf("no replica status on port %d", node.Port)
	}
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	vals := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	if err := rows.Scan(ptrs...); err != nil {
		return nil, err
	}
	byCol := make(map[string]any, len(cols))
	for i, c := range cols {
		byCol[c] = vals[i]
	}
	return &ReplicaStatus{
		MasterHost:      firstField(byCol, "Master_Host", "Source_Host"),
		MasterPort:      firstIntField(byCol, "Master_Port", "Source_Port"),
		SlaveIORunning:  firstField(byCol, "Slave_IO_Running", "Replica_IO_Running"),
		SlaveSQLRunning: firstField(byCol, "Slave_SQL_Running", "Replica_SQL_Running"),
	}, nil
}

func firstField(byCol map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := byCol[k]; ok {
			s := stringField(v)
			if s != "" {
				return s
			}
		}
	}
	return ""
}

func firstIntField(byCol map[string]any, keys ...string) int {
	for _, k := range keys {
		if v, ok := byCol[k]; ok {
			if n := intField(v); n != 0 {
				return n
			}
		}
	}
	return 0
}

func intField(v any) int {
	if v == nil {
		return 0
	}
	switch x := v.(type) {
	case int64:
		return int(x)
	case int32:
		return int(x)
	case int:
		return x
	case []byte:
		var n int
		fmt.Sscanf(string(x), "%d", &n)
		return n
	default:
		var n int
		fmt.Sscanf(fmt.Sprint(x), "%d", &n)
		return n
	}
}

func stringField(v any) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case []byte:
		return string(x)
	case string:
		return x
	default:
		return fmt.Sprint(x)
	}
}

// MySQLReadOnly returns @@read_only for node.
func (b *MySQL80) MySQLReadOnly(node *Node) (bool, error) {
	dsn := fmt.Sprintf("root@tcp(127.0.0.1:%d)/", node.Port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return false, err
	}
	defer db.Close()

	var ro int
	err = db.QueryRow("SELECT @@read_only").Scan(&ro)
	return ro == 1, err
}

// WaitReplicaRunning waits until IO and SQL replica threads are Yes on node.
func (b *MySQL80) WaitReplicaRunning(ctx context.Context, node *Node) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		st, err := b.ReplicaStatus(node)
		if err == nil && st.SlaveIORunning == "Yes" && st.SlaveSQLRunning == "Yes" {
			return nil
		}
		time.Sleep(250 * time.Millisecond)
	}
}

// WaitMySQLWritable waits until @@read_only=0 on node.
func (b *MySQL80) WaitMySQLWritable(ctx context.Context, node *Node) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		ro, err := b.MySQLReadOnly(node)
		if err == nil && !ro {
			return nil
		}
		time.Sleep(250 * time.Millisecond)
	}
}

// FindWritableNode returns the first node with @@read_only=0.
func (b *MySQL80) FindWritableNode(nodes []*Node) (*Node, error) {
	for _, node := range nodes {
		ro, err := b.MySQLReadOnly(node)
		if err == nil && !ro {
			return node, nil
		}
	}
	return nil, fmt.Errorf("no writable node among %d nodes", len(nodes))
}

// WaitReplicaConnectedTo waits until node replicates from masterPort with both threads running.
func (b *MySQL80) WaitReplicaConnectedTo(ctx context.Context, node *Node, masterPort int) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		st, err := b.ReplicaStatus(node)
		if err == nil &&
			st.MasterHost == "127.0.0.1" &&
			st.MasterPort == masterPort &&
			st.SlaveIORunning == "Yes" &&
			st.SlaveSQLRunning == "Yes" {
			return nil
		}
		time.Sleep(250 * time.Millisecond)
	}
}
