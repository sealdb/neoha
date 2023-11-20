# Copyright 2022-2026 The NeoHA Authors.
#
# See the AUTHORS file for a list of contributors.
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

.PHONY: build clean install fmt test testbase testconfig testdatabase testmanager testelection testserver testcmd test-integration vet coverage

build: LDFLAGS   += $(shell GOPATH=${GOPATH} build/ldflags.sh)
build:
	@echo "--> Building..."
	@mkdir -p bin/
	go mod tidy
	go build -v -o bin/neoha    -ldflags '$(LDFLAGS)' ./cmd/neoha
	go build -v -o bin/neohactl -ldflags '$(LDFLAGS)' ./cmd/neohactl
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
	go test -v ./internal/base/common/...
	go test -v ./internal/base/nlog/...
	go test -v ./internal/base/nrpc/...
testconfig:
	go test -v ./internal/config/...
testdatabase:
	go test -v ./internal/database/mysql/...
	# go test -v ./internal/database/postgresql/...
testmanager:
	go test -v ./internal/manager/mysqld/...
	# go test -v ./internal/manager/postmaster/...
testelection:
	go test -v ./internal/election/raft/...
	# go test -v ./internal/election/etcd/...
testserver:
	go test -v ./internal/server/...
testcmd:
	go test -v ./internal/neohactl/...
	go test -v ./api/v1/...

test-integration:
	@echo "--> Integration testing..."
	@mkdir -p bin
	@echo "--> Pre-building test binary (avoids silent compile)..."
	go test -tags=integration -c -o bin/neoha-it.test ./test/integration
	NEOHA_IT_MYSQL_BASE=$${NEOHA_IT_MYSQL_BASE:-/home/wslu/work/mysql/mysql80-debug} \
		bin/neoha-it.test -test.v -test.timeout=10m -test.count=1

PKGS =	./internal/base/common/... \
	./internal/base/nlog/... \
	./internal/base/nrpc/... \
	./internal/config/... \
	./internal/database/mysql/... \
	./internal/manager/mysqld/... \
	./internal/election/raft/... \
	./internal/server/... \
	./internal/neohactl/... \
	./api/v1/...
		#./internal/database/postgresql/... \
		#./internal/manager/postmaster/... \
		#./internal/election/etcd/... \

vet:
	go vet $(PKGS)

coverage:
	#echo > ${COVERAGE_LOG}
	go test -v -covermode=set -coverprofile=${COVERAGE_OUT} ./... #| tee -a ${COVERAGE_LOG}
	#grep "FAIL:" ${COVERAGE_LOG} ; if [[ $? -eq 0 ]]; then exit 1 ; fi
	go tool cover -html=${COVERAGE_OUT}
