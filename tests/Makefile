SHELL := /bin/bash

DEV_WRAPPER := ../scripts/productive-k3s-cli-dev.sh
ifneq ("$(wildcard /usr/local/go/bin/go)","")
GO_BIN ?= /usr/local/go/bin/go
else
GO_BIN ?= go
endif

.PHONY: test-local-all test-unit test-lint test-format test-spell test-coverage test-static-raw docs-publish-check test-cli-contract test-live-remote test-live-catalog test-live-gha-onprem-remote test-checkstatus test-clean

test-local-all:
	GO_BIN="$(GO_BIN)" bash $(DEV_WRAPPER) test-local-all

test-unit:
	bash ./bin/run-go-tests.sh

test-lint:
	bash ./bin/run-lint.sh

test-format:
	bash ./bin/run-format-check.sh

test-spell:
	bash ./bin/run-spellcheck.sh

test-coverage:
	bash ./bin/run-coverage.sh

docs-publish-check:
	bash ./test-docs-publishing.sh
	bash ./test-docs-structure.sh

test-static-raw:
	GO_BIN="$(GO_BIN)" bash ./bin/run-go-tests.sh
	bash ./test-go-tooling.sh
	$(MAKE) docs-publish-check
	bash ./test-release-versioning.sh
	bash ./test-set-bundles-versions.sh
	bash ./test-tag-release.sh
	bash ./test-live-artifacts.sh
	bash ./test-live-wiring.sh
	bash ./run-cli-contracts.sh

test-cli-contract:
	$(MAKE) -C .. build
	bash ./run-cli-contracts.sh

test-live-remote:
	$(MAKE) -C .. build
	bash ./run-cli-live.sh $(SCENARIOS)

test-live-catalog:
	$(MAKE) -C .. build
	bash ./run-cli-live.sh catalog-multipass

test-live-gha-onprem-remote:
	$(MAKE) -C .. build
	bash ./live-cli-onprem-remote-github-host.sh

test-checkstatus:
	bash $(DEV_WRAPPER) test-checkstatus

test-clean:
	bash $(DEV_WRAPPER) test-clean
