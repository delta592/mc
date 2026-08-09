PWD := $(shell pwd)
GOPATH := $(shell go env GOPATH)
LDFLAGS := $(shell go run buildscripts/gen-ldflags.go)

TARGET_GOARCH ?= $(shell go env GOARCH)
TARGET_GOOS ?= $(shell go env GOOS)

VERSION ?= $(shell git describe --tags)
TAG ?= "delta592/mc:$(VERSION)"

GOLANGCI := go tool golangci-lint

BUILD_TAGS ?= kqueue
TESTPKG ?= ./...
TEST_TIMEOUT ?= 20m
UNIT_TEST_FLAGS := -tags $(BUILD_TAGS) -count=1 -timeout $(TEST_TIMEOUT)
RACE_TEST_FLAGS := -race -tags $(BUILD_TAGS) -count=1 -timeout $(TEST_TIMEOUT)
COVERAGE_FILE ?= coverage.out

.PHONY: all build checks getdeps crosscompile docker \
	vet fmt fmt-check lint lint-fix verifiers \
	test test-short test-race test-coverage test-integration \
	verify check ci install clean help \
	hotfix-vars hotfix hotfix-push docker-hotfix docker-hotfix-push

.DEFAULT_GOAL := build

##@ Build

all: build ## Build the mc binary (default target)

build: checks ## Build the mc binary to ./mc
	@echo "Building mc binary to './mc'"
	@GOOS=$(TARGET_GOOS) GOARCH=$(TARGET_GOARCH) CGO_ENABLED=0 go build -trimpath -tags $(BUILD_TAGS) --ldflags "$(LDFLAGS)" -o $(PWD)/mc

install: build ## Install mc to $(GOPATH)/bin
	@echo "Installing mc binary to '$(GOPATH)/bin/mc'"
	@mkdir -p $(GOPATH)/bin && cp -f $(PWD)/mc $(GOPATH)/bin/mc
	@echo "Installation successful. To learn more, try \"mc --help\"."

checks: ## Verify build dependencies
	@echo "Checking dependencies"
	@env bash $(PWD)/buildscripts/checkdeps.sh

crosscompile: ## Cross-compile mc for all supported platforms
	@env bash $(PWD)/buildscripts/cross-compile.sh

docker: build ## Build development Docker image
	@docker build -t $(TAG) . -f Dockerfile.dev

##@ Code quality

getdeps: ## Install tools declared in go.mod
	@echo "Installing tools from go.mod"
	@go install tool

vet: ## Run go vet
	@echo "Running go vet"
	@go vet -tags $(BUILD_TAGS) ./...

fmt: getdeps ## Format code (gofmt, gofumpt, goimports)
	@echo "Formatting code"
	@$(GOLANGCI) fmt --config ./.golangci.yml

fmt-check: getdeps ## Check code formatting without modifying files
	@echo "Checking code formatting"
	@$(GOLANGCI) fmt --config ./.golangci.yml --diff

lint: getdeps ## Run golangci-lint
	@echo "Running golangci-lint"
	@$(GOLANGCI) run --config ./.golangci.yml

lint-fix: getdeps ## Run golangci-lint with automatic fixes
	@echo "Running golangci-lint with fixes"
	@$(GOLANGCI) run --config ./.golangci.yml --fix

verifiers: vet fmt-check lint ## Run vet, formatting, and lint checks

##@ Tests

test-short: ## Run unit tests (short mode, no race detector)
	@echo "Running unit tests (short)"
	@CGO_ENABLED=0 go test $(UNIT_TEST_FLAGS) -short $(TESTPKG)

test: verifiers build ## Run unit tests and integration suite
	@echo "Running unit tests"
	@CGO_ENABLED=0 go test $(UNIT_TEST_FLAGS) $(TESTPKG)
	@echo "Running integration tests"
	@MC_TEST_RUN_FULL_SUITE=true CGO_ENABLED=1 go test $(RACE_TEST_FLAGS) -v $(TESTPKG) -run Test_FullSuite

test-race: build ## Run unit tests with the race detector
	@echo "Running unit tests with race detector"
	@CGO_ENABLED=1 go test $(RACE_TEST_FLAGS) -v $(TESTPKG)

test-coverage: ## Run unit tests and write a coverage profile
	@echo "Running tests with coverage"
	@CGO_ENABLED=0 go test $(UNIT_TEST_FLAGS) -short -coverprofile=$(COVERAGE_FILE) -covermode=atomic $(TESTPKG)
	@go tool cover -func=$(COVERAGE_FILE)

test-integration: build ## Run the full integration test suite
	@env bash $(PWD)/buildscripts/run-integration-tests.sh

verify: build ## Build with race detector and run integration tests
	@echo "Verifying race-enabled build"
	@CGO_ENABLED=1 go build -race -tags $(BUILD_TAGS) -trimpath --ldflags "$(LDFLAGS)" -o $(PWD)/mc
	@$(MAKE) test-integration

check: verifiers test-short test-race build ## Run local validation checks (no MinIO server required)

ci: verifiers test-short test-race test-coverage build ## Run CI checks (no MinIO server required)

##@ Release

hotfix-vars:
	$(eval LDFLAGS := $(shell MC_RELEASE="RELEASE" MC_HOTFIX="hotfix.$(shell git rev-parse --short HEAD)" go run buildscripts/gen-ldflags.go))
	$(eval VERSION := $(shell git describe --tags --abbrev=0).hotfix.$(shell git rev-parse --short HEAD))
	$(eval TAG := "delta592/mc:$(VERSION)")

hotfix: hotfix-vars install ## Build mc binary with hotfix tags
	@mv -f ./mc ./mc.$(VERSION)
	@minisign -qQSm ./mc.$(VERSION) -s "${CRED_DIR}/minisign.key" < "${CRED_DIR}/minisign-passphrase"
	@sha256sum < ./mc.$(VERSION) | sed 's, -,mc.$(VERSION),g' > mc.$(VERSION).sha256sum

hotfix-push: hotfix
	@scp -q -r mc.$(VERSION)* minio@dl-0.min.io:~/releases/client/mc/hotfixes/$(TARGET_GOOS)-$(TARGET_GOARCH)/archive/
	@scp -q -r mc.$(VERSION)* minio@dl-1.min.io:~/releases/client/mc/hotfixes/$(TARGET_GOOS)-$(TARGET_GOARCH)/archive/
	@echo "Published new hotfix binaries at https://dl.min.io/client/mc/hotfixes/$(TARGET_GOOS)-$(TARGET_GOARCH)/archive/mc.$(VERSION)"

docker-hotfix-push: docker-hotfix
	@docker push -q $(TAG) && echo "Published new container $(TAG)"

docker-hotfix: hotfix-push checks ## Build mc docker container with hotfix tags
	@echo "Building mc docker image '$(TAG)'"
	@docker build -q --no-cache -t $(TAG) --build-arg RELEASE=$(VERSION) . -f Dockerfile.hotfix

clean: ## Remove build artifacts and test binaries
	@echo "Cleaning up all the generated files"
	@find . -name '*.test' | xargs rm -fv
	@find . -name '*~' | xargs rm -fv
	@rm -rvf mc build release $(COVERAGE_FILE)

##@ Help

help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} \
		/^[a-zA-Z0-9][a-zA-Z0-9_-]*:.*?##/ { printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2 } \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' $(MAKEFILE_LIST)
