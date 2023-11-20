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

package common

import (
	"math/rand"
	"os"
	"path"
	"time"
)

type (
	// File type enum.
	FileType string
)

const (
	// YamlType enum.
	YamlType FileType = ".yaml"
	// YmlType enum.
	YmlType FileType = ".yml"
	// JsonType enum.
	JsonType FileType = ".json"
)

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil // exists
	}
	if os.IsNotExist(err) {
		return false, nil // not exist
	}
	return false, err // not sure
}

func GetFileType(filepath string) FileType {
	fileType := path.Ext(path.Base(filepath))
	return FileType(fileType)
}

func RandomTimeout(min int) *time.Timer {
	var max int
	if min <= 5 {
		max = min * 2
	} else if min <= 20 {
		max = min + min/2
	} else {
		max = min + 10
	}
	rand := rand.New(rand.NewSource(time.Now().UnixNano()))
	d, delta := min, (max - min)
	if delta > 0 {
		d += rand.Intn(int(delta))
	}
	return time.NewTimer(time.Duration(d) * time.Millisecond)
}

func RandomPort(min int, max int) int {
	rand := rand.New(rand.NewSource(time.Now().UnixNano()))
	d, delta := min, (max - min)
	if delta > 0 {
		d += rand.Intn(int(delta))
	}
	return d
}

func NormalTimeout(d int) *time.Timer {
	return time.NewTimer(time.Duration(d) * time.Millisecond)
}

func NormalTimerRelaese(t *time.Timer) {
	if t == nil {
		return
	}

	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
}

func NormalTicker(d int) *time.Ticker {
	return time.NewTicker(time.Duration(d) * time.Millisecond)
}
