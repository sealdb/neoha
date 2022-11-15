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

package nlog

var (
	defaultName  = " "
	defaultLevel = DEBUG
)

// Options used for the options of the xlog.
type Options struct {
	Name  string
	Level LogLevel
}

// Option func.
type Option func(*Options)

func newOptions(opts ...Option) *Options {
	opt := &Options{}
	for _, o := range opts {
		o(opt)
	}

	if len(opt.Name) == 0 {
		opt.Name = defaultName
	}

	if opt.Level == 0 {
		opt.Level = defaultLevel
	}
	return opt
}

// Name used to set the name.
func Name(v string) Option {
	return func(o *Options) {
		o.Name = v
	}
}

// Level used to set the log level.
func Level(v LogLevel) Option {
	return func(o *Options) {
		o.Level = v
	}
}
