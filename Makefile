.PHONY: docs-build docs-serve docs-up docs-down docs-clean test-static test-contract test-productive-k3s-cli

SCRIPTS_DIR := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))/scripts

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

test-static:
	$(SCRIPTS_DIR)/productive-k3s-cli-dev.sh test-static

test-contract:
	$(SCRIPTS_DIR)/productive-k3s-cli-dev.sh test-contract

test-productive-k3s-cli:
	$(SCRIPTS_DIR)/productive-k3s-cli-dev.sh test-productive-k3s-cli
