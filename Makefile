
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

NAME ?= auger
PKG ?= github.com/etcd-io/$(NAME)
GO_VERSION ?= 1.23.1
GOOS ?= linux
GOARCH ?= amd64
TEMP_DIR := $(shell mktemp -d)
GOFILES = $(shell find . -name \*.go)
CGO_ENABLED ?= 0

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: fmt
fmt: ## Run go fmt against code.
	@echo "Verifying gofmt, failures can be fixed with ./scripts/fix.sh"
	@!(gofmt -l -s -d ${GOFILES} | grep '[a-z]')

	@echo "Verifying goimports, failures can be fixed with ./scripts/fix.sh"
	@!(go run golang.org/x/tools/cmd/goimports@latest -l -d ${GOFILES} | grep '[a-z]')

##@ Lint / Verify
.PHONY: verify
verify:
	golangci-lint run --config tools/.golangci.yaml ./...

##@ Build

# Local development build
.PHONY: build
build: ## local build 
	@mkdir -p build
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) go build -o build/$(NAME)
	@echo build/$(NAME) built!

# Local development test
# `go test` automatically manages the build, so no need to depend on the build target here in make
.PHONY: test
test: ## Run go test
	@echo Vetting
	go vet ./...
	@echo Testing
	go test ./...

pkg/scheme/scheme.go: ./hack/gen_scheme.sh go.mod
	go mod vendor
	-rm ./pkg/scheme/scheme.go
	./hack/gen_scheme.sh > ./pkg/scheme/scheme.go

.PHONY: generate
generate: pkg/scheme/scheme.go ## Generate code
