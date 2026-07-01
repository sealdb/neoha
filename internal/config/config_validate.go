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

package config

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/sealdb/neoha/internal/base/model"
)

// Normalize merges coordination and election blocks and fills defaults after parse.
func (c *Config) Normalize() error {
	if c == nil {
		return errors.New("config is nil")
	}
	if c.Election == nil {
		c.Election = DefaultElectionConfig()
	}
	if c.Coordination == nil {
		c.Coordination = DefaultCoordinationConfig()
	}
	if c.Database == nil {
		c.Database = DefaultDatabaseConfig()
	}
	if c.Coordination.Provider == "" && c.Election.Algo != "" {
		c.Coordination.Provider = c.Election.Algo
	}
	if c.Election.Algo == "" && c.Coordination.Provider != "" {
		c.Election.Algo = c.Coordination.Provider
	}
	if c.Coordination.Raft == nil {
		c.Coordination.Raft = c.Election.Raft
	}
	if c.Election.Raft == nil {
		c.Election.Raft = c.Coordination.Raft
	}
	if c.Coordination.Etcd == nil {
		c.Coordination.Etcd = c.Election.Etcd
	}
	if c.Election.Etcd == nil {
		c.Election.Etcd = c.Coordination.Etcd
	}
	if strings.TrimSpace(c.Database.Type) == "" {
		c.Database.Type = "mysql"
	}
	if c.HA == nil {
		c.HA = DefaultHAConfig()
	}
	if c.HA.PrimaryHooks == nil {
		c.HA.PrimaryHooks = DefaultPrimaryHooksConfig()
	}
	return nil
}

// Validate checks semantic constraints. See docs/config-design.md.
func (c *Config) Validate() error {
	if err := c.Normalize(); err != nil {
		return err
	}
	coord := c.EffectiveCoordination()
	provider := strings.ToLower(strings.TrimSpace(coord.Provider))
	if provider == "" {
		return errors.New("coordination provider is required (coordination.provider or election.algorithm)")
	}
	switch provider {
	case "raft":
		if strings.TrimSpace(c.Endpoint) == "" {
			return errors.New("endpoint is required when coordination.provider is raft")
		}
		if coord.Raft == nil {
			return errors.New("coordination.raft section is required when provider is raft")
		}
	case "etcd":
		if coord.Etcd == nil || (coord.Etcd.Host == "" && len(coord.Etcd.Hosts) == 0) {
			return errors.New("coordination.etcd host or hosts is required when provider is etcd")
		}
		inheritEtcdDCSFromBootstrap(c, coord.Etcd)
	case "consul", "kubernetes", "zookeeper":
		return errors.Errorf("coordination provider %q is not implemented yet", provider)
	default:
		return errors.Errorf("unsupported coordination provider %q", provider)
	}

	dbType := strings.ToLower(strings.TrimSpace(c.Database.Type))
	switch dbType {
	case "mysql", "":
		if c.Database.Mysql == nil {
			return errors.New("database.mysql section is required when database.type is mysql")
		}
		mode := c.Database.Mysql.ReplMode
		if mode != model.ReplModeSemiSync && mode != model.ReplModeMGR && mode != "" {
			return errors.Errorf("unsupported mysql replication-mode %q", mode)
		}
	case "postgresql", "postgres":
		if c.Database.Postgresql == nil {
			return errors.New("database.postgresql section is required when database.type is postgresql")
		}
		inheritPostgreSQLFailoverFromBootstrap(c)
	default:
		return errors.Errorf("unsupported database.type %q", dbType)
	}
	return nil
}

func bootstrapDCS(c *Config) *DcsConfig {
	if c == nil || c.Bootstrap == nil || c.Bootstrap.BootstrapPostgresql == nil {
		return nil
	}
	return c.Bootstrap.BootstrapPostgresql.DcsConf
}

func inheritEtcdDCSFromBootstrap(c *Config, etcd *EtcdConfig) {
	if etcd == nil {
		return
	}
	dcs := bootstrapDCS(c)
	if dcs == nil {
		return
	}
	if etcd.TTL <= 0 && dcs.TTL > 0 {
		etcd.TTL = dcs.TTL
	}
	if etcd.LoopWait <= 0 && dcs.LoopWait > 0 {
		etcd.LoopWait = dcs.LoopWait
	}
	if etcd.RetryTimeout <= 0 && dcs.RetryTimeout > 0 {
		etcd.RetryTimeout = dcs.RetryTimeout
	}
}

func inheritPostgreSQLFailoverFromBootstrap(c *Config) {
	pg := c.Database.Postgresql
	dcs := bootstrapDCS(c)
	if pg == nil || dcs == nil {
		return
	}
	if pg.MaximumLagOnFailover == 0 && dcs.MaximumLagOnFailover > 0 {
		pg.MaximumLagOnFailover = int64(dcs.MaximumLagOnFailover)
	}
	if !pg.UseSlots && dcs.UseSlots {
		pg.UseSlots = dcs.UseSlots
	}
	if !pg.UsePGRewind && dcs.UsePGRewind {
		pg.UsePGRewind = dcs.UsePGRewind
	}
	if !pg.SynchronousMode && dcs.SynchronousMode {
		pg.SynchronousMode = dcs.SynchronousMode
	}
}
