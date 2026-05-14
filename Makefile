.PHONY: build build-release go-test docs-build docs-serve docs-up docs-down docs-clean docs-publish-check test-clean test-checkstatus test-static test-cli-contract test-cli-contract-clean test-live-remote test-live-gha-onprem-remote set-bundles-versions tag-release

SCRIPTS_DIR := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))/scripts
GO_BIN ?= go

build:
	$(SCRIPTS_DIR)/build-cli.sh build-local

build-release:
	$(SCRIPTS_DIR)/build-cli.sh build-release

go-test:
	$(GO_BIN) test ./...

test-static:
	$(SCRIPTS_DIR)/productive-k3s-cli-dev.sh test-static

docs-build:
	$(SCRIPTS_DIR)/productive-k3s-cli-dev.sh docs-build

docs-serve:
	$(SCRIPTS_DIR)/productive-k3s-cli-dev.sh docs-serve

docs-up:
	$(SCRIPTS_DIR)/productive-k3s-cli-dev.sh docs-up

docs-down:
	$(SCRIPTS_DIR)/productive-k3s-cli-dev.sh docs-down

docs-clean:
	$(SCRIPTS_DIR)/productive-k3s-cli-dev.sh docs-clean

docs-publish-check:
	bash ./tests/test-docs-publishing.sh
	bash ./tests/test-docs-structure.sh

test-clean:
	$(SCRIPTS_DIR)/productive-k3s-cli-dev.sh test-clean

test-checkstatus:
	$(SCRIPTS_DIR)/productive-k3s-cli-dev.sh test-checkstatus

test-cli-contract:
	$(MAKE) build
	bash ./tests/run-cli-contracts.sh

test-live-remote:
	$(MAKE) build
	bash ./tests/run-cli-live.sh $(SCENARIOS)

test-live-gha-onprem-remote:
	$(MAKE) build
	bash ./tests/live-cli-onprem-remote-github-host.sh

set-bundles-versions:
	$(SCRIPTS_DIR)/set-bundles-versions.sh $(CORE_VERSION) $(INFRA_VERSION)

tag-release:
	$(SCRIPTS_DIR)/tag-release.sh $(VERSION)

test-cli-contract-clean:
	rm -rf test-artifacts/*
