# Targets De Make

El repositorio expone un conjunto pequeño de comandos de desarrollo estables:

Prerrequisitos de desarrollo para estos targets:

- un toolchain funcional de `Go`, disponible en `PATH` o seteado explícitamente con `GO_BIN`

Este requisito es específico de `productive-k3s-cli`, porque este repositorio compila y testea el binario `pk3s` desde fuente. No debe leerse como un prerrequisito transversal para desarrollar `productive-k3s-core` o `productive-k3s-infra`.

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

Los targets detallados viven en `tests/` y `docs/`:

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

Estos comandos buscan reflejar las mismas convenciones ya usadas en Productive K3S Core y Productive K3S Infra.
