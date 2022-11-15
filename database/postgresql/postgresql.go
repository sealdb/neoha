/*
 * Copyright 2022-2025 The NeoHA Authors.
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

package postgresql

import (
	"time"

	"neoha/base/model"
	"neoha/base/nlog"
	"neoha/config"
)

// Postgresql tuple.
type Postgresql struct {
	//db                *sql.DB
	conf              *config.PostgresqlConfig
	log               *nlog.Log
	postgresqlHandler PostgresqlHandler
	pingTicker        *time.Ticker
	stats             model.MysqlStats
	downs             int
}

// NewPostgresql creates the new Postgresql.
func NewPostgresql(conf *config.PostgresqlConfig, queryTimeout int, log *nlog.Log) *Postgresql {
	postgresql := &Postgresql{
		//db:           nil,
		log:               log,
		conf:              conf,
		postgresqlHandler: getPostgresqlHandler(conf.Version),
		//pingTicker:   common.NormalTicker(conf.PingTimeout),
	}
	//postgresql.postgresqlHandler.SetQueryTimeout(queryTimeout)
	return postgresql
}
