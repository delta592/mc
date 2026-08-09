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

.DEFAULT_GOAL := build

##@ Build

.PHONY: all
all: build ## Build the mc binary (default target)

.PHONY: build
build: checks ## Build the mc binary to ./mc
	@echo "Building mc binary to './mc'"
	@GOOS=$(TARGET_GOOS) GOARCH=$(TARGET_GOARCH) CGO_ENABLED=0 go build -trimpath -tags $(BUILD_TAGS) --ldflags "$(LDFLAGS)" -o $(PWD)/mc

.PHONY: install
install: build ## Install mc to $(GOPATH)/bin
	@echo "Installing mc binary to '$(GOPATH)/bin/mc'"
	@mkdir -p $(GOPATH)/bin && cp -f $(PWD)/mc $(GOPATH)/bin/mc
	@echo "Installation successful. To learn more, try \"mc --help\"."

.PHONY: checks
checks: ## Verify build dependencies
	@echo "Checking dependencies"
	@env bash $(PWD)/buildscripts/checkdeps.sh

.PHONY: crosscompile
crosscompile: ## Cross-compile mc for all supported platforms
	@env bash $(PWD)/buildscripts/cross-compile.sh

.PHONY: docker
docker: build ## Build development Docker image
	@docker build -t $(TAG) . -f Dockerfile.dev

##@ Code quality

.PHONY: getdeps
getdeps: ## Install tools declared in go.mod
	@echo "Installing tools from go.mod"
	@go install tool

.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet"
	@go vet -tags $(BUILD_TAGS) ./...

.PHONY: fmt
fmt: getdeps ## Format code (gofmt, gofumpt, goimports)
	@echo "Formatting code"
	@$(GOLANGCI) fmt --config ./.golangci.yml

.PHONY: fmt-check
fmt-check: getdeps ## Check code formatting without modifying files
	@echo "Checking code formatting"
	@$(GOLANGCI) fmt --config ./.golangci.yml --diff

.PHONY: lint
lint: getdeps ## Run golangci-lint
	@echo "Running golangci-lint"
	@$(GOLANGCI) run --config ./.golangci.yml

.PHONY: lint-fix
lint-fix: getdeps ## Run golangci-lint with automatic fixes
	@echo "Running golangci-lint with fixes"
	@$(GOLANGCI) run --config ./.golangci.yml --fix

.PHONY: verifiers
verifiers: vet fmt-check lint ## Run vet, formatting, and lint checks

##@ Tests

.PHONY: test-short
test-short: ## Run unit tests (short mode, no race detector)
	@echo "Running unit tests (short)"
	@CGO_ENABLED=0 go test $(UNIT_TEST_FLAGS) -short $(TESTPKG)

.PHONY: test
test: verifiers build ## Run unit tests and integration suite
	@echo "Running unit tests"
	@CGO_ENABLED=0 go test $(UNIT_TEST_FLAGS) $(TESTPKG)
	@echo "Running integration tests"
	@MC_TEST_RUN_FULL_SUITE=true CGO_ENABLED=1 go test $(RACE_TEST_FLAGS) -v ./cmd -run Test_FullSuite

.PHONY: test-race
test-race: build ## Run unit tests with the race detector
	@echo "Running unit tests with race detector"
	@CGO_ENABLED=1 go test $(RACE_TEST_FLAGS) -v $(TESTPKG)

.PHONY: test-platform
test-platform: build test-short test-race ## Run cross-platform unit tests (CI test job)

.PHONY: test-coverage
test-coverage: ## Run unit tests and write a coverage profile
	@echo "Running tests with coverage"
	@CGO_ENABLED=0 go test $(UNIT_TEST_FLAGS) -short -coverprofile=$(COVERAGE_FILE) -covermode=atomic $(TESTPKG)
	@go tool cover -func=$(COVERAGE_FILE)
	@env bash $(PWD)/buildscripts/check-coverage.sh 80

.PHONY: test-integration
test-integration: build ## Run the full integration test suite
	@env bash $(PWD)/buildscripts/run-integration-tests.sh

.PHONY: build-race
build-race: build ## Build mc with the race detector enabled
	@echo "Verifying race-enabled build"
	@CGO_ENABLED=1 go build -race -tags $(BUILD_TAGS) -trimpath --ldflags "$(LDFLAGS)" -o $(PWD)/mc

.PHONY: verify
verify: build-race ## Build with race detector and run integration tests
	@$(MAKE) test-integration

.PHONY: ci-integration
ci-integration: build-race ## Run full CI integration suite (MinIO, go tests, functional tests)
	@MC_TEST_RUN_FUNCTIONAL=true \
	MC_TEST_INSTALL_CA=true \
	MC_TEST_ENABLE_HTTPS=true \
	MC_TEST_SKIP_INSECURE=true \
	MC_TEST_SKIP_BUILD=true \
	MC_TEST_SERVER_ENDPOINT=localhost:9000 \
	MC_TEST_ACCESS_KEY=minioadmin \
	MC_TEST_SECRET_KEY=minioadmin \
	SERVER_ENDPOINT=localhost:9000 \
	ACCESS_KEY=minioadmin \
	SECRET_KEY=minioadmin \
	ENABLE_HTTPS=1 \
	env bash $(PWD)/buildscripts/run-integration-tests.sh

.PHONY: check
check: verifiers test-short test-race build ## Run local validation checks (no MinIO server required)

.PHONY: ci
ci: verifiers test-short test-race test-coverage build ## Run CI checks (no MinIO server required)

##@ Release

.PHONY: hotfix-vars
hotfix-vars:
	$(eval LDFLAGS := $(shell MC_RELEASE="RELEASE" MC_HOTFIX="hotfix.$(shell git rev-parse --short HEAD)" go run buildscripts/gen-ldflags.go))
	$(eval VERSION := $(shell git describe --tags --abbrev=0).hotfix.$(shell git rev-parse --short HEAD))
	$(eval TAG := "delta592/mc:$(VERSION)")

.PHONY: hotfix
hotfix: hotfix-vars install ## Build mc binary with hotfix tags
	@mv -f ./mc ./mc.$(VERSION)
	@minisign -qQSm ./mc.$(VERSION) -s "${CRED_DIR}/minisign.key" < "${CRED_DIR}/minisign-passphrase"
	@sha256sum < ./mc.$(VERSION) | sed 's, -,mc.$(VERSION),g' > mc.$(VERSION).sha256sum

.PHONY: hotfix-push
hotfix-push: hotfix
	@scp -q -r mc.$(VERSION)* minio@dl-0.min.io:~/releases/client/mc/hotfixes/$(TARGET_GOOS)-$(TARGET_GOARCH)/archive/
	@scp -q -r mc.$(VERSION)* minio@dl-1.min.io:~/releases/client/mc/hotfixes/$(TARGET_GOOS)-$(TARGET_GOARCH)/archive/
	@echo "Published new hotfix binaries at https://dl.min.io/client/mc/hotfixes/$(TARGET_GOOS)-$(TARGET_GOARCH)/archive/mc.$(VERSION)"

.PHONY: docker-hotfix-push
docker-hotfix-push: docker-hotfix
	@docker push -q $(TAG) && echo "Published new container $(TAG)"

.PHONY: docker-hotfix
docker-hotfix: hotfix-push checks ## Build mc docker container with hotfix tags
	@echo "Building mc docker image '$(TAG)'"
	@docker build -q --no-cache -t $(TAG) --build-arg RELEASE=$(VERSION) . -f Dockerfile.hotfix

.PHONY: clean
clean: ## Remove build artifacts and test binaries
	@echo "Cleaning up all the generated files"
	@find . -name '*.test' | xargs rm -fv
	@find . -name '*~' | xargs rm -fv
	@rm -rvf mc build release $(COVERAGE_FILE)

##@ Help

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} \
		/^[a-zA-Z0-9][a-zA-Z0-9_-]*:.*?##/ { printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2 } \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' $(MAKEFILE_LIST)
