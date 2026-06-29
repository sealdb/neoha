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

package postgresql

import (
	"testing"

	"github.com/sealdb/neoha/internal/config"
	dbdriver "github.com/sealdb/neoha/internal/database/driver"
	"github.com/stretchr/testify/assert"
)

func TestBuildRewindConninfo(t *testing.T) {
	conf := config.DefaultPostgresqlConfig()
	conf.Auth.Rewind.Username = "rewind"
	conf.Auth.Rewind.Password = "secret"
	pg := NewPostgresql(conf, 0, nil)

	got := pg.BuildRewindConninfo(dbdriver.PrimaryRef{Host: "10.0.0.1", Port: 5433})
	assert.Contains(t, got, "host=10.0.0.1")
	assert.Contains(t, got, "port=5433")
	assert.Contains(t, got, "user=rewind")
	assert.Contains(t, got, "password=secret")
}

func TestBuildRewindConninfoSuperuserFallback(t *testing.T) {
	conf := config.DefaultPostgresqlConfig()
	conf.Auth.Rewind = nil
	conf.Auth.SuperUser.Username = "postgres"
	conf.Auth.SuperUser.Password = "zalando"
	pg := NewPostgresql(conf, 0, nil)

	got := pg.BuildRewindConninfo(dbdriver.PrimaryRef{MemberID: "127.0.0.1:5432"})
	assert.Contains(t, got, "user=postgres")
	assert.Contains(t, got, "password=zalando")
}

func TestPgBinFromBinDir(t *testing.T) {
	conf := config.DefaultPostgresqlConfig()
	conf.BinDir = "/opt/pg/bin"
	pg := NewPostgresql(conf, 0, nil)
	assert.Equal(t, "/opt/pg/bin/pg_rewind", pg.pgBin("pg_rewind"))
}
