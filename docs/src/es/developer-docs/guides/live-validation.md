# Validación remota live

`pk3s` ahora tiene una capa chica de validación live enfocada en el flujo público remoto.

El objetivo es acotado: probar que el CLI puede resolver bundles publicados o un grafo source alineado explícitamente, aceptar un package de profile o un `.env` local generado, y ejecutar `productive-k3s-infra` de punta a punta.

## Entry points

Usá los targets agregados:

```bash
make test-live-remote
make test-live-catalog
```

O corré un único validador:

```bash
make -C tests test-live-remote SCENARIOS=multipass
make -C tests test-live-remote SCENARIOS=onprem-basic
```

## Escenarios cubiertos

### `multipass`

El validador:

1. usa bundles publicados por defecto, o checkouts source cuando recibe refs/directorios de repositorios
2. deriva las versiones de compatibilidad source desde los checkouts seleccionados de Core e Infra
3. valida el profile de Multipass con `pk3s profile validate`
4. corre `pk3s plan`
5. corre `pk3s apply`
6. corre `pk3s status`
7. corre `pk3s destroy`

Esto es la prueba más directa, a nivel CLI, de que tanto un grafo source coordinado como el bundle publicado de Infra sirven para el profile de Multipass.

### `catalog-multipass`

El validador:

1. fuerza resolución catalog-backed
2. valida el package publicado de Multipass con `pk3s profile validate`
3. corre `pk3s infra install`
4. corre `pk3s infra status`
5. corre `pk3s addon install`
6. corre `pk3s infra destroy`

Esto prueba la UX catalog-backed de packages contra un target real de Multipass.

### `onprem-basic`

El validador es deliberadamente externo al escenario:

1. crea dos VMs efímeras con Multipass
2. espera a que SSH esté disponible
3. genera un `.env` temporal de `onprem-basic`
4. fuerza `PRODUCTIVE_K3S_SOURCE=remote`
5. corre `pk3s profile validate`
6. corre `pk3s plan`
7. corre `pk3s apply`
8. corre `pk3s status`
9. corre `pk3s validate`
10. elimina las VMs efímeras

Esto valida el camino del CLI para un profile de estilo remoto sin tocar el host local.

## Artefactos

Cada ejecución live escribe:

- un manifest por escenario bajo `test-artifacts/cli-live-runs/`
- un log por escenario bajo `test-artifacts/cli-live-runs/`
- un summary raíz bajo `test-artifacts/<run-id>-summary.json`
- una copia de conveniencia en `test-artifacts/live-summary.json`

Los manifests incluyen:

- nombre del escenario
- resultado pass/skip/fail
- timestamps y duración
- topología esperada
- path del binario del CLI
- versiones configuradas de los bundles Core e Infra

## Notas

- Estos validadores están separados intencionalmente de `test-local-all`.
- Los harnesses source-aligned entregan las versiones de Core e Infra como un único par; CLI falla cerrado si ese par está incompleto, es inválido o no está ligado entre sí. El modo source sin un par explícito conserva los pins de compatibilidad compilados.
- El isolated de Ops alinea Core, Infra, Profiles y Addons con el ref solicitado para `live-remote`; `live-catalog` permanece como gate de artefactos publicados.
- También requieren un toolchain local funcional de `Go`, porque los targets live compilan `pk3s` antes de correr las validaciones.
- Requieren dependencias reales como `multipass`, `ssh`, `curl`, `tar` y `python3`.
- `onprem-basic` no usa `pk3s destroy`, porque ese escenario no expone un contrato público de destroy. La limpieza se hace borrando las VMs temporales.
- Usá `make -C tests test-checkstatus` para inspeccionar los artifacts del CLI.
- Usá `make -C tests test-clean` para borrar el estado local de artifacts del CLI.
