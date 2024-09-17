
# Copyright 2022 The Kubernetes Authors.
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

GOOS ?= linux
GOARCH ?= amd64
GOFILES = $(shell find . -name \*.go)
CGO_ENABLED ?= 0

.PHONY: fmt
fmt:
	@echo "Verifying gofmt, failures can be fixed with ./scripts/fix.sh"
	@!(gofmt -l -s -d ${GOFILES} | grep '[a-z]')

	@echo "Verifying goimports, failures can be fixed with ./scripts/fix.sh"
	@!(go tool goimports -l -d ${GOFILES} | grep '[a-z]')

.PHONY: verify
verify:
	go tool golangci-lint run --config tools/.golangci.yaml ./...

# Local development build
build:
	@mkdir -p build
	@echo "building for $(GOOS)/$(GOARCH)"
	@GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) go build \
		-ldflags "-X github.com/etcd-io/auger/cmd.appVersion=$(shell git describe --tags --dirty --always) \
			-X github.com/etcd-io/auger/cmd.gitCommit=$(shell git rev-parse --short HEAD) \
			-X github.com/etcd-io/auger/cmd.buildDate=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')" \
		-o build/auger
	@echo build/auger built!
	@GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) go build \
		-ldflags "-X github.com/etcd-io/auger/augerctl/command.appVersion=$(shell git describe --tags --dirty --always) \
			-X github.com/etcd-io/auger/augerctl/command.gitCommit=$(shell git rev-parse --short HEAD) \
			-X github.com/etcd-io/auger/augerctl/command.buildDate=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')" \
		-o build/augerctl augerctl/main.go
	@echo build/augerctl built!


# Local development test
# `go test` automatically manages the build, so no need to depend on the build target here in make
test: build
	@echo Vetting
	go vet ./...
	@echo Testing
	go test ./...

clean:
	rm -rf build

pkg/scheme/scheme.go: ./hack/gen_scheme.sh go.mod
	go mod vendor
	-rm ./pkg/scheme/scheme.go
	./hack/gen_scheme.sh > ./pkg/scheme/scheme.go

.PHONY: generate
generate: pkg/scheme/scheme.go

.PHONY: build test clean

test-e2e: build
	@echo "Running e2e tests"
	go test ./tests/e2e/...