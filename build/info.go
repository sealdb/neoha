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

package build

import (
	"fmt"
	"runtime"
)

var (
	tag      = "unknown" // tag of this build
	git      string      // git hash
	time     string      // build time
	platform = fmt.Sprintf("%s %s", runtime.GOOS, runtime.GOARCH)
)

type Info struct {
	Tag       string
	Time      string
	Git       string
	GoVersion string
	Platform  string
}

func GetInfo() Info {
	return Info{
		GoVersion: runtime.Version(),
		Tag:       tag,
		Time:      time,
		Git:       git,
		Platform:  platform,
	}
}
