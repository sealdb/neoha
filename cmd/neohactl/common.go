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

package neohactl

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"neoha/base/model"
	"neoha/base/nlog"
	"neoha/config"

	"github.com/spf13/cobra"
)

var (
	configPathFile = "config.path"
	log            = nlog.NewStdLog(nlog.Level(nlog.INFO))
)

func ErrorOK(err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: unexpected error: %s\033[39m\n\n", filepath.Base(file), line, err.Error())
		log.Panic("%v", err)
	}
}

func RspOK(ret string) {
	if ret != model.OK {
		log.Panic("rsp[%v] != [OK]", ret)
	}
}

func GetConfig() (*config.Config, error) {
	fullPath := configPathFile

	// try to search in current dir
	// and try to in Abs dir if failed
	if _, err := os.Stat(fullPath); err != nil {
		dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			return nil, err
		}
		fullPath = fmt.Sprintf("%s/%s", dir, configPathFile)
	}

	data, err := ioutil.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}

	conf, err := config.LoadConfig(strings.TrimSpace(string(data)))
	if err != nil {
		return nil, err
	}

	return conf, nil
}

func SaveConfig(conf *config.Config) error {
	fullPath := configPathFile

	// try to search in current dir
	// and try to in Abs dir if failed
	if _, err := os.Stat(fullPath); err != nil {
		dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			return err
		}
		fullPath = fmt.Sprintf("%s/%s", dir, configPathFile)
	}

	data, err := ioutil.ReadFile(fullPath)
	if err != nil {
		return err
	}

	if err := config.WriteConfig(strings.TrimSpace(string(data)), conf); err != nil {
		return err
	}

	return nil
}

func executeCommand(root *cobra.Command, args ...string) (output string, err error) {
	buf := new(bytes.Buffer)
	root.SetOutput(buf)
	root.SetArgs(args)

	_, err = root.ExecuteC()
	return buf.String(), err
}
