# Copyright 2022 The NeoHA Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

PREFIX    :=/usr/local
#export GOPROXY=https://goproxy.cn
export PATH := $(GOPATH)/bin:$(PATH)
COVERAGE_OUT := coverage.out

.PHONY:build

build: LDFLAGS   += $(shell GOPATH=${GOPATH} build/ldflags.sh)
build:
	@echo "--> Building..."
	@mkdir -p bin/
	go mod tidy
	go build -v -o bin/neoha    -ldflags '$(LDFLAGS)' neoha/main.go
	go build -v -o bin/neohactl -ldflags '$(LDFLAGS)' cmd/main.go
	@chmod 755 bin/*

clean:
	@echo "--> Cleaning..."
	@mkdir -p bin/
	@go clean
	@rm -f bin/*
	@rm -f coverage*

install:
	@echo "--> Installing..."
	@install bin/neoha bin/neohactl $(PREFIX)/sbin/

fmt:
	go fmt ./...

test:
	@echo "--> Testing..."
	@$(MAKE) testbase
	@$(MAKE) testconfig
	@$(MAKE) testcmd
	#@$(MAKE) testserver
	@$(MAKE) testneoha

testbase:
	go test -v neoha/base/common
	go test -v neoha/base/nlog
testconfig:
	go test -v neoha/config
testcmd:
	go test -v neoha/cmd/neohactl
testserver:
	go test -v neoha/server
testneoha:
	go test -v neoha/neoha

COVPKGS = base/nlog\
		  config\
		  cmd/neohactl\
          neoha

vet:
	go vet $(COVPKGS)

coverage:
	go test -v -covermode=set -coverprofile=${COVERAGE_OUT} ./...
	go tool cover -html=${COVERAGE_OUT}
