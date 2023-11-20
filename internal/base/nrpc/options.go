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

package nrpc

import (
	"github.com/sealdb/neoha/internal/base/nlog"
)

var (
	DefaultConnectionStr = ":8080"
	DefaultLog           = nlog.NewStdLog()
)

type Options struct {
	ConnectionStr string
	Log           *nlog.Log
}

type Option func(*Options)

func newOptions(opts ...Option) *Options {
	opt := &Options{}
	for _, o := range opts {
		o(opt)
	}

	if opt.Log == nil {
		panic("nrpc.log.handler.is.nil")
	}

	if len(opt.ConnectionStr) == 0 {
		opt.ConnectionStr = DefaultConnectionStr
	}
	return opt
}

// ConnectionStr:
// server connection string
func ConnectionStr(v string) Option {
	return func(o *Options) {
		o.ConnectionStr = v
	}
}

// Log:
// server log
func Log(v *nlog.Log) Option {
	return func(o *Options) {
		o.Log = v
	}
}
