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

// PostgresqlHandler interface.
type PostgresqlHandler interface {
}

var (
	postgresqlHandlers = make(map[string]PostgresqlHandler)
)

func init() {
	postgresqlHandlers["postgresql14"] = new(Postgresql14)
	postgresqlHandlers["postgresql14"] = new(Postgresql15)
}

func getPostgresqlHandler(name string) PostgresqlHandler {
	handler, ok := postgresqlHandlers[name]
	if !ok {
		return new(Postgresql14) // default
	}
	return handler
}
