# Flujo de documentación

El repositorio del CLI sigue el mismo flujo MkDocs usado por los repositorios Productive K3S.

## Generar documentación

```bash
make docs-build
```

## Servir documentación localmente

```bash
make docs-serve
```

## Servir documentación en background

```bash
make docs-up
```

El sitio local se sirve en:

```text
http://127.0.0.1:8000
```

## Detener y limpiar artefactos de documentación

```bash
make docs-down
make docs-clean
```

Los scripts de documentación crean un virtualenv local en `docs/.venv` e instalan los requerimientos Python declarados en `docs/requirements.txt`.
