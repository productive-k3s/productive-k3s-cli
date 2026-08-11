# Validación remota live

`pk3s` ahora tiene una capa chica de validación live enfocada en el flujo público remoto.

El objetivo es acotado: probar que el CLI puede resolver bundles remotos publicados, aceptar tanto una URL de profile remota como un `.env` local generado, y ejecutar de punta a punta el release correspondiente de `productive-k3s-infra`.

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

1. fuerza `PRODUCTIVE_K3S_SOURCE=remote`
2. valida la URL publicada del profile de Multipass con `pk3s profile validate`
3. corre `pk3s plan`
4. corre `pk3s apply`
5. corre `pk3s status`
6. corre `pk3s destroy`

Esto es la prueba más directa, a nivel CLI, de que el bundle remoto publicado de Infra sirve para el profile integrado de Multipass.

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
- También requieren un toolchain local funcional de `Go`, porque los targets live compilan `pk3s` antes de correr las validaciones.
- Requieren dependencias reales como `multipass`, `ssh`, `curl`, `tar` y `python3`.
- `onprem-basic` no usa `pk3s destroy`, porque ese escenario no expone un contrato público de destroy. La limpieza se hace borrando las VMs temporales.
- Usá `make -C tests test-checkstatus` para inspeccionar los artifacts del CLI.
- Usá `make -C tests test-clean` para borrar el estado local de artifacts del CLI.
