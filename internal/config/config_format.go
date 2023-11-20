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

package config

import (
	"bytes"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/sealdb/neoha/internal/base/common"
	"gopkg.in/yaml.v3"
)

func isSupportedConfigFormat(format common.FileType) bool {
	switch format {
	case common.JsonType, common.YamlType, common.YmlType:
		return true
	default:
		return false
	}
}

// DetectConfigFormat picks YAML or JSON from the file extension, or by inspecting content.
func DetectConfigFormat(filepath string, data []byte) (common.FileType, error) {
	format := common.GetFileType(filepath)
	if isSupportedConfigFormat(format) {
		return format, nil
	}

	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return "", errors.Errorf("config file [%s] is empty", filepath)
	}

	switch trimmed[0] {
	case '{', '[':
		return common.JsonType, nil
	default:
		return common.YamlType, nil
	}
}

// ParseConfig unmarshals config bytes in YAML or JSON format onto defaults.
func ParseConfig(data []byte, format common.FileType) (*Config, error) {
	conf := DefaultConfig()
	var err error

	switch format {
	case common.JsonType:
		err = json.Unmarshal(data, conf)
	case common.YamlType, common.YmlType:
		err = yaml.Unmarshal(data, conf)
	default:
		return nil, errors.Errorf("unsupported config format [%s]", format)
	}
	if err != nil {
		return nil, errors.Wrapf(err, "parse config as %s", format)
	}
	return conf, nil
}

// MarshalConfig encodes config to YAML or JSON bytes.
func MarshalConfig(conf *Config, format common.FileType) ([]byte, error) {
	switch format {
	case common.JsonType:
		return json.MarshalIndent(conf, "", "\t")
	case common.YamlType, common.YmlType:
		return yaml.Marshal(conf)
	default:
		return nil, errors.Errorf("unsupported config format [%s]", format)
	}
}
