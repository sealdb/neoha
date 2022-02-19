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

package nlog

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// assert fails the test if the condition is false.
func Assert(tb testing.TB, condition bool, msg string, v ...interface{}) {
	if !condition {
		tb.FailNow()
	}
}

func TestGetLog(t *testing.T) {
	GetLog().Debug("DEBUG")
	log := NewStdLog()
	log.SetLevel("INFO")
	GetLog().Debug("DEBUG")
	GetLog().Info("INFO")
	GetLog().Warning("WARNING")
	GetLog().Error("ERROR")
}

func TestSysLog(t *testing.T) {
	tests := []struct {
		name  string
		level string
	}{
		{name: "DEFAULT", level: "DEFAULT"},
		{name: "DEBUG", level: "DEBUG"},
		{name: "INFO", level: "INFO"},
		{name: "WARNING", level: "WARNING"},
		{name: "ERROR", level: "ERROR"},
	}
	for _, tt := range tests {
		//tt := tt // for parallel test
		t.Run(tt.name, func(t *testing.T) {
			//t.Parallel() // enable parallel test
			log := NewSysLog()
			if tt.level != "DEFAULT" {
				log.SetLevel(tt.level)
			}
			log.Debug("DEBUG")
			log.Info("INFO")
			log.Warning("WARNING")
			log.Error("ERROR")
			log.Close()
		})
	}
}

func TestStdLog(t *testing.T) {
	tests := []struct {
		name  string
		level string
	}{
		{name: "DEFAULT", level: "DEFAULT"},
		{name: "DEBUG", level: "DEBUG"},
		{name: "INFO", level: "INFO"},
		{name: "WARNING", level: "WARNING"},
		{name: "ERROR", level: "ERROR"},
	}
	for _, tt := range tests {
		//tt := tt // for parallel test
		t.Run(tt.name, func(t *testing.T) {
			//t.Parallel() // enable parallel test
			log := NewStdLog()
			log.Println("........DEFAULT........")
			log.SetLevel(tt.level)
			log.Debug("DEBUG")
			log.Info("INFO")
			log.Warning("WARNING")
			log.Error("ERROR")
			log.Close()
		})
	}
}

func TestLogLevel(t *testing.T) {
	log := NewStdLog()
	tests := []struct {
		name  string
		level string
		want  LogLevel
	}{
		{name: "DEFAULT", level: "DEFAULT", want: DEBUG},
		{name: "INFO", level: "INFO", want: INFO},
		{name: "DEBUGX", level: "DEBUGX", want: INFO},
		{name: "DEBUG", level: "DEBUG", want: DEBUG},
		{name: "WARNING", level: "WARNING", want: WARNING},
		{name: "ERROR", level: "ERROR", want: ERROR},
		{name: "PANIC", level: "PANIC", want: PANIC},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log.SetLevel(tt.level)
			got := log.opts.Level
			assert.Equal(t, tt.want, got, "want[%v]!=got[%v]", tt.want, got)
		})
	}
}
