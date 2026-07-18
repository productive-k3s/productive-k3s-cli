# Pruebas

El CLI tiene hoy dos capas de testing:

- validadores de contrato bajo `tests/contracts/`, rápidos y con artefactos JSON en `test-artifacts/`
- validadores live remotos bajo `tests/run-cli-live.sh`, que ejecutan `pk3s` end to end contra bundles publicados de Infra

Para la capa live, los escenarios soportados son:

- `multipass`: usa directamente el profile remoto publicado de Multipass a través de `pk3s`
- `onprem-basic`: crea dos VMs efímeras de Multipass fuera del scenario de Infra, genera un `.env` temporal y luego ejercita el workflow público de `pk3s` contra ese archivo

Comandos típicos:

```bash
make test-local-all
make test-live-remote
make test-live-catalog
make -C tests test-unit
make -C tests test-lint
make -C tests test-format
make -C tests test-spell
make -C tests test-coverage
make -C tests test-live-remote SCENARIOS=multipass
make -C tests test-live-remote SCENARIOS=onprem-basic
```

Los manifests live se escriben bajo `test-artifacts/cli-live-runs/`, y cada ejecución también emite:

- `test-artifacts/<run-id>-summary.json`
- `test-artifacts/live-summary.json`

Los targets auxiliares siguen el mismo estilo que Infra/Core:

```bash
make -C tests test-checkstatus
make -C tests test-clean
TEST_SCOPE=live make -C tests test-checkstatus
TEST_SCOPE=contract make -C tests test-clean
```

## Prerrequisitos locales

El CLI usa Go como capa primaria de tests unitarios y la interfaz local normalizada depende de:

- una toolchain funcional de Go
- `shellcheck` para linting de helpers shell
- `shfmt` para checks de formato shell
- `codespell` para control ortográfico

Si instalás herramientas sin root, mantené `~/.local/bin` en `PATH`:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

## Baseline actual de cobertura local

La baseline actual desde la última corrida local de `make -C tests test-coverage` es:

- cobertura total Go: `80.0%`
- `internal/app`: `80.8%`
- `internal/bundles`: `77.8%`
- `internal/platform`: `100.0%`

Esta baseline sirve como guía para futuras mejoras y refactors. Todavía no se aplica como umbral duro en CI.
