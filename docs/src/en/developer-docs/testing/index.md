# Testing

The CLI has two test layers today:

- contract validators under `tests/contracts/`, which stay fast and write JSON artifacts under `test-artifacts/`
- live remote validators under `tests/run-cli-live.sh`, which execute `pk3s` end to end against published Infra bundles

For the live layer, the supported scenarios are:

- `multipass`: uses the published remote Multipass profile URL directly through `pk3s`
- `onprem-basic`: creates two ephemeral Multipass VMs outside the Infra scenario, generates a temporary `.env`, and then exercises the public `pk3s` profile workflow against that file

Typical commands:

```bash
make test
make test-unit
make test-lint
make test-format
make test-spell
make test-coverage
make test-static
make test-live-remote
./tests/run-cli-live.sh multipass
./tests/run-cli-live.sh onprem-basic
```

Live manifests are written under `test-artifacts/cli-live-runs/`, and each invocation also emits:

- `test-artifacts/<run-id>-summary.json`
- `test-artifacts/live-summary.json`

The helper targets match the Infra/Core repos:

```bash
make test-checkstatus
make test-clean
TEST_SCOPE=live make test-checkstatus
TEST_SCOPE=contract make test-clean
```

## Local prerequisites

The CLI uses Go as the primary unit-test layer and the normalized local test interface now depends on:

- a working Go toolchain
- `shellcheck` for shell helper linting
- `shfmt` for shell formatting checks
- `codespell` for spell checks

If you install tools without root, keep `~/.local/bin` in `PATH`:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

User-local install commands used during development on Ubuntu:

```bash
mkdir -p "$HOME/.local/bin" "$HOME/.local/share"
curl -fsSLO https://github.com/koalaman/shellcheck/releases/download/v0.11.0/shellcheck-v0.11.0.linux.x86_64.tar.xz
tar -xJf shellcheck-v0.11.0.linux.x86_64.tar.xz
install shellcheck-v0.11.0/shellcheck "$HOME/.local/bin/shellcheck"

curl -fsSLo "$HOME/.local/bin/shfmt" https://github.com/mvdan/sh/releases/download/v3.13.1/shfmt_v3.13.1_linux_amd64
chmod +x "$HOME/.local/bin/shfmt"

python3 -m pip install --user codespell
```

On this machine, `/usr/local/go/bin/go` is preferred over the broken snap-provided `go` binary.

## Current local coverage baseline

The current maintainer baseline from the latest local `make test-coverage` run is:

- total Go coverage: `80.0%`
- `internal/app`: `80.8%`
- `internal/bundles`: `77.8%`
- `internal/platform`: `100.0%`

This baseline is intended to guide future additions and refactors. It is not yet enforced as a CI threshold.
