# Make Targets

The repository exposes a small set of stable development commands:

Development prerequisites for these targets:

- a working `Go` toolchain available in `PATH` or set explicitly with `GO_BIN`

This requirement is specific to `productive-k3s-cli`, because this repository builds and tests the `pk3s` binary from source. It is not meant as a cross-repository prerequisite for `productive-k3s-core` or `productive-k3s-infra`.

```bash
make build
make build-release
make go-test
make test-local-all
make test-live-remote
make test-live-catalog
make test-clean-all
make docs-build
make docs-serve
make set-bundles-versions CORE_VERSION=0.9.1 INFRA_VERSION=0.9.41-0.9.1
make tag-release VERSION=1.0.1
```

Detailed targets live under `tests/` and `docs/`:

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
make -C docs docs-up
make -C docs docs-down
make -C docs docs-clean
```

These commands intentionally mirror the conventions already used in Productive K3S Core and Productive K3S Infra.
