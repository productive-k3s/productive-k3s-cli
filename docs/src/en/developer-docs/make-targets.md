# Make Targets

The repository exposes a small set of stable development commands:

Development prerequisites for these targets:

- a working `Go` toolchain available in `PATH` or set explicitly with `GO_BIN`

This requirement is specific to `productive-k3s-cli`, because this repository builds and tests the `pk3s` binary from source. It is not meant as a cross-repository prerequisite for `productive-k3s-core` or `productive-k3s-infra`.

```bash
make build
make build-release
make test
make go-test
make test-unit
make test-lint
make test-format
make test-spell
make test-coverage
make test-static
make test-cli-contract
make test-live-gha-onprem-remote
make docs-build
make docs-serve
make test-checkstatus
make test-clean
make set-bundles-versions CORE_VERSION=0.9.1 INFRA_VERSION=0.9.41-0.9.1
make tag-release VERSION=1.0.1
```

These commands intentionally mirror the conventions already used in Productive K3S Core and Productive K3S Infra.
