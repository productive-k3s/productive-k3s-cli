# Productive K3S CLI Tests

`productive-k3s-cli` keeps Go tests as the primary unit-test layer and shell contract tests as the orchestration contract layer.

This directory now exposes a normalized local interface aligned with `productive-k3s-core` and `productive-k3s-infra`, while keeping the root `Makefile` focused on top-level product and maintainer flows.

## Root entrypoints

```bash
make test-local-all
make test-live-remote
make test-live-catalog
make test-clean-all
make docs-build
make docs-serve
```

## Detailed targets

Run detailed checks from inside `tests/`:

```bash
make -C tests test-unit
make -C tests test-lint
make -C tests test-format
make -C tests test-spell
make -C tests test-coverage
make -C tests test-cli-contract
make -C tests test-live-gha-onprem-remote
make -C tests test-checkstatus
make -C tests test-clean
make -C tests test-clean-all
```

## Focus

- Go unit coverage for CLI parsing, telemetry/config helpers, platform gating, bundle resolution, and command dispatch/error paths
- existing shell contract tests for CLI-to-core/infra delegation
- shell lint/format checks for helper scripts
- spell checks for docs, shell scripts, and test content

## Current Coverage Baseline

Latest local `make -C tests test-coverage` run:

- total Go coverage: `80.0%`
- `internal/app`: `80.8%`
- `internal/bundles`: `77.8%`
- `internal/platform`: `100.0%`

Treat this as a maintainer baseline for new changes, not as a hard CI gate.

## Layout

```text
tests/
  bin/
  contracts/
  fixtures/
  helpers/
  lib/
  spell/
```

Generated at runtime and intentionally not tracked:

- `test-artifacts/`
- `tests/coverage/`

Unlike `core` and `infra`, this repo keeps `tests/fixtures/` because the shell contract layer consumes real bundle/profile/manifests fixtures.
