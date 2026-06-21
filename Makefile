.PHONY: build build-release go-test docs-build docs-serve test-local-all test-live-remote test-live-catalog set-bundles-versions tag-release

SCRIPTS_DIR := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))/scripts
ifneq ("$(wildcard /usr/local/go/bin/go)","")
GO_BIN ?= /usr/local/go/bin/go
else
GO_BIN ?= go
endif

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

test-live-remote:
	GO_BIN="$(GO_BIN)" $(MAKE) -C ./tests test-live-remote SCENARIOS="$(SCENARIOS)"

test-live-catalog:
	GO_BIN="$(GO_BIN)" $(MAKE) -C ./tests test-live-catalog

set-bundles-versions:
	$(SCRIPTS_DIR)/set-bundles-versions.sh $(CORE_VERSION) $(INFRA_VERSION)

tag-release:
	$(SCRIPTS_DIR)/tag-release.sh $(VERSION)
