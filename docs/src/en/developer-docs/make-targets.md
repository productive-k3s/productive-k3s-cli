# Make Targets

The repository exposes a small set of stable development commands:

```bash
make build
make build-release
make go-test
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
