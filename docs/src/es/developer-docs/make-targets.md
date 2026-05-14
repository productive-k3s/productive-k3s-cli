# Targets De Make

El repositorio expone un conjunto pequeño de comandos de desarrollo estables:

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

Estos comandos buscan reflejar las mismas convenciones ya usadas en Productive K3S Core y Productive K3S Infra.
