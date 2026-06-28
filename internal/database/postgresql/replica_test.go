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

package postgresql

import (
	"testing"

	"github.com/sealdb/neoha/internal/config"
	dbdriver "github.com/sealdb/neoha/internal/database/driver"
	"github.com/stretchr/testify/assert"
)

func TestBuildPrimaryConninfo(t *testing.T) {
	conf := config.DefaultPostgresqlConfig()
	conf.Auth.Repl.Username = "repl"
	conf.Auth.Repl.Password = "secret"
	pg := NewPostgresql(conf, 1000, nil)

	conninfo := pg.BuildPrimaryConninfo(dbdriver.PrimaryRef{Host: "10.0.0.5", Port: 5433})
	assert.Contains(t, conninfo, "host=10.0.0.5")
	assert.Contains(t, conninfo, "port=5433")
	assert.Contains(t, conninfo, "user=repl")
	assert.Contains(t, conninfo, "password=secret")
}

func TestPrimarySlotName(t *testing.T) {
	conf := config.DefaultPostgresqlConfig()
	conf.PrimarySlotName = "custom_slot"
	pg := NewPostgresql(conf, 1000, nil)
	assert.Equal(t, "custom_slot", pg.primarySlotName("n1"))

	conf.PrimarySlotName = ""
	pg = NewPostgresql(conf, 1000, nil)
	assert.Equal(t, "n1", pg.primarySlotName("n1"))
	assert.Equal(t, "pgrepl", pg.primarySlotName(""))
}
