/*
 * Copyright 2022 The NeoHA Authors.
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
	"os"
	"path"
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
