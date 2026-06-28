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
	"net"
	"strconv"

	dbdriver "github.com/sealdb/neoha/internal/database/driver"
)

// enrichPrimaryRef fills Host/Port from NeoHA member endpoint (host:port) when unset.
// mysqlPort is a fallback when all members share the same database port (single-host dev).
func enrichPrimaryRef(ref dbdriver.PrimaryRef, mysqlPort int) dbdriver.PrimaryRef {
	if ref.Host != "" && ref.Port > 0 {
		return ref
	}
	if ref.Host != "" || ref.MemberID == "" {
		return ref
	}
	host, portStr, err := net.SplitHostPort(ref.MemberID)
	if err != nil {
		return ref
	}
	ref.Host = host
	if portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil && mysqlPort <= 0 {
			ref.Port = p
		}
	}
	if mysqlPort > 0 {
		ref.Port = mysqlPort
	}
	return ref
}
