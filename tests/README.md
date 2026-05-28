# Productive K3S CLI Tests

`productive-k3s-cli` keeps Go tests as the primary unit-test layer and shell contract tests as the orchestration contract layer.

This directory now exposes a normalized local interface aligned with `productive-k3s-core` and `productive-k3s-infra`.

## Commands

```bash
make test
make test-unit
make test-lint
make test-format
make test-spell
make test-coverage
```

## Focus

- Go unit coverage for CLI parsing, telemetry/config helpers, platform gating, bundle resolution, and command dispatch/error paths
- existing shell contract tests for CLI-to-core/infra delegation
- shell lint/format checks for helper scripts
- spell checks for docs, shell scripts, and test content

## Current Coverage Baseline

Latest local `make test-coverage` run:

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

- `tests/artifacts/`
- `tests/coverage/`

Unlike `core` and `infra`, this repo keeps `tests/fixtures/` because the shell contract layer consumes real bundle/profile/manifests fixtures.
