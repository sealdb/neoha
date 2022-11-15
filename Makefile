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
export LDFLAGS :=
COVERAGE_OUT := coverage.out
COVERAGE_LOG := coverage.log

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
	@$(MAKE) testdatabase
	@$(MAKE) testmanager
	@$(MAKE) testelection
	@$(MAKE) testserver
	@$(MAKE) testcmd

testbase:
	go test -v neoha/base/common
	go test -v neoha/base/nlog
	go test -v neoha/base/nrpc
testconfig:
	go test -v neoha/config
testdatabase:
	go test -v neoha/database/mysql
	# go test -v neoha/database/postgresql
testmanager:
	go test -v neoha/manager/mysqld
	# gp test -v neoha/manager/postmaster
testelection:
	go test -v neoha/election/raft
	# go test -v neoha/election/etcd
testserver:
	go test -v neoha/server
testcmd:
	go test -v neoha/cmd/neohactl
	go test -v neoha/api/v1

PKGS = 	base/common \
		base/nlog \
		base/nrpc \
		config \
		database/mysql \
		manager/mysqld \
		election/raft \
		server \
		cmd/neohactl \
		api/v1
		#database/postgresql \
		#manager/postmaster \
		#election/etcd \

vet:
	go vet $(PKGS)

coverage:
	#echo > ${COVERAGE_LOG}
	go test -v -covermode=set -coverprofile=${COVERAGE_OUT} ./... #| tee -a ${COVERAGE_LOG}
	#grep "FAIL:" ${COVERAGE_LOG} ; if [[ $? -eq 0 ]]; then exit 1 ; fi
	go tool cover -html=${COVERAGE_OUT}
