
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
GOOS ?= linux
GOARCH ?= amd64
GOFILES = $(shell find . -name \*.go)
CGO_ENABLED ?= 0

.PHONY: fmt
fmt:
	@echo "Verifying gofmt, failures can be fixed with ./scripts/fix.sh"
	@!(gofmt -l -s -d ${GOFILES} | grep '[a-z]')

	@echo "Verifying goimports, failures can be fixed with ./scripts/fix.sh"
	@!(go run golang.org/x/tools/cmd/goimports@latest -l -d ${GOFILES} | grep '[a-z]')

.PHONY: verify
verify:
	golangci-lint run --config tools/.golangci.yaml ./...

# Local development build
build:
	@mkdir -p build
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) go build -o build/$(NAME)
	@echo build/$(NAME) built!

# Local development test
# `go test` automatically manages the build, so no need to depend on the build target here in make
test:
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

pkg/old/scheme/scheme.go: ./hack/gen_old_scheme.sh pkg/old/apis
	-rm ./pkg/old/scheme/scheme.go
	./hack/gen_old_scheme.sh > ./pkg/old/scheme/scheme.go

pkg/old/apis: ./hack/clone_old_apis.sh go.mod
	-rm -rf ./pkg/old/apis/*
	./hack/clone_old_apis.sh $(shell cat go.mod | grep 'k8s.io/api v0.' | awk '{print $$2}' | awk -F. '{print $$2}')
	./scripts/fix.sh

.PHONY: generate
generate: pkg/scheme/scheme.go

.PHONY: build test clean
