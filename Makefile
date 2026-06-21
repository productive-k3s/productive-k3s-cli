.PHONY: build build-release go-test docs-build docs-serve test-local-all test-cli-contract test-live-remote test-live-catalog test-live-gha-onprem-remote set-bundles-versions tag-release

SCRIPTS_DIR := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))/scripts
GO_BIN ?= go

build:
	GO_BIN=$(GO_BIN) $(SCRIPTS_DIR)/build-cli.sh build-local

build-release:
	GO_BIN=$(GO_BIN) $(SCRIPTS_DIR)/build-cli.sh build-release

go-test:
	$(GO_BIN) test ./...

docs-build:
	$(MAKE) -C ./docs docs-build

docs-serve:
	$(MAKE) -C ./docs docs-serve

test-local-all:
	GO_BIN="$(GO_BIN)" $(MAKE) -C ./tests test-local-all

test-cli-contract:
	GO_BIN="$(GO_BIN)" $(MAKE) build
	bash ./tests/run-cli-contracts.sh

test-live-remote:
	GO_BIN="$(GO_BIN)" $(MAKE) -C ./tests test-live-remote SCENARIOS="$(SCENARIOS)"

test-live-catalog:
	GO_BIN="$(GO_BIN)" $(MAKE) -C ./tests test-live-catalog

test-live-gha-onprem-remote:
	GO_BIN="$(GO_BIN)" $(MAKE) -C ./tests test-live-gha-onprem-remote

set-bundles-versions:
	$(SCRIPTS_DIR)/set-bundles-versions.sh $(CORE_VERSION) $(INFRA_VERSION)

tag-release:
	$(SCRIPTS_DIR)/tag-release.sh $(VERSION)
