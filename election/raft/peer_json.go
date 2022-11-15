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

package raft

import (
	"encoding/json"
	"io/ioutil"

	"github.com/pkg/errors"
)

func writePeersJSON(path string, peers []string, idlePeers []string) error {
	allPeers := make(map[string][]string)

	allPeers["peers"] = peers
	allPeers["idlepeers"] = idlePeers

	jsonStr, err := json.Marshal(allPeers)
	if err != nil {
		return errors.WithStack(err)
	}

	if err := ioutil.WriteFile(path, []byte(jsonStr), 0755); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func readPeersJSON(path string) ([]string, []string, error) {
	//var peers []string
	//var idlePeers []string
	allPeers := make(map[string][]string)

	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return []string{}, []string{}, errors.WithStack(err)
	}

	err = json.Unmarshal(buf, &allPeers)
	if err != nil {
		return []string{}, []string{}, errors.WithStack(err)
	}
	return allPeers["peers"], allPeers["idlepeers"], nil
}
