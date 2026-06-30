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
	"net"
	"strconv"
	"strings"

	"github.com/sealdb/neoha/internal/coordination"
)

func (c *Coordinator) memberDatabaseEndpoint() (string, int) {
	if c.conf == nil || c.conf.Database == nil {
		return "", 0
	}
	switch c.dbType {
	case "postgresql", "postgres":
		if pg := c.conf.Database.Postgresql; pg != nil {
			host, portStr := splitHostPort(pg.ConnectAddress, "5432")
			port, _ := strconv.Atoi(portStr)
			return host, port
		}
	default:
		if my := c.conf.Database.Mysql; my != nil {
			host := my.ReplHost
			if host == "" {
				host = my.Host
			}
			return host, my.Port
		}
	}
	return "", 0
}

func (c *Coordinator) leaderDatabaseEndpoint(members []coordination.Member, leaderID string) coordination.DatabaseEndpoint {
	for _, m := range members {
		if m.ID != leaderID && m.Name != leaderID {
			continue
		}
		host := m.Meta["db_host"]
		port, _ := strconv.Atoi(m.Meta["db_port"])
		if host != "" && port > 0 {
			return coordination.DatabaseEndpoint{Host: host, Port: port}
		}
	}
	return coordination.DatabaseEndpoint{}
}

func splitHostPort(addr, defaultPort string) (string, string) {
	if addr == "" {
		return "127.0.0.1", defaultPort
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr, defaultPort
	}
	if port == "" {
		port = defaultPort
	}
	return host, port
}

func parseEndpoints(host string, hosts []string) []string {
	if len(hosts) > 0 {
		return hosts
	}
	if host == "" {
		return nil
	}
	if strings.Contains(host, ",") {
		return strings.Split(host, ",")
	}
	return []string{host}
}
