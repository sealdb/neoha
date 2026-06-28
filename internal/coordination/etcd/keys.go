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

package etcd

import (
	"fmt"
	"strings"

	"github.com/sealdb/neoha/internal/config"
)

func clusterPrefix(conf *config.Config) string {
	ns := conf.NameSpace
	if ns == "" {
		ns = "/service/"
	}
	if !strings.HasSuffix(ns, "/") {
		ns += "/"
	}
	scope := conf.Scope
	if scope == "" {
		scope = "database"
	}
	return fmt.Sprintf("%s%s", ns, scope)
}

func leaderKey(conf *config.Config) string {
	return clusterPrefix(conf) + "/leader"
}

func memberKey(conf *config.Config, name string) string {
	return clusterPrefix(conf) + "/members/" + name
}

func memberPrefix(conf *config.Config) string {
	return clusterPrefix(conf) + "/members/"
}

func etcdEndpoints(conf *config.Config) []string {
	ec := conf.EffectiveCoordination().Etcd
	if ec == nil {
		return nil
	}
	if len(ec.Hosts) > 0 {
		return ec.Hosts
	}
	if ec.Host != "" {
		return []string{ec.Host}
	}
	return nil
}

func leaseTTL(conf *config.Config) int {
	if ec := conf.EffectiveCoordination().Etcd; ec != nil && ec.TTL > 0 {
		return ec.TTL
	}
	if conf.Bootstrap != nil && conf.Bootstrap.BootstrapPostgresql != nil &&
		conf.Bootstrap.BootstrapPostgresql.DcsConf != nil &&
		conf.Bootstrap.BootstrapPostgresql.DcsConf.TTL > 0 {
		return conf.Bootstrap.BootstrapPostgresql.DcsConf.TTL
	}
	return 30
}
