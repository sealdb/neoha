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

package neohactl

import (
	"os"

	"github.com/sealdb/neoha/internal/config"
)

var defaultConfig = config.DefaultConfig()

func createConfig() error {
	path := "/tmp/test.cli.config.json"
	err := config.WriteConfig(path, defaultConfig)
	if err != nil {
		return err
	}

	flag := os.O_RDWR | os.O_TRUNC | os.O_CREATE
	f, err := os.OpenFile("./config.path", flag, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(path)
	return err
}

func removeConfig() error {
	os.Remove("/tmp/test.cli.config.json")
	return nil
}
