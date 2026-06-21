# Modelo de versionado

Productive K3S CLI, Productive K3S Core y Productive K3S Infra usan versiones de estilo semántico sin prefijo `v`.

Ejemplos:

```text
1.0.0
2.1.0
```

## Versiones de Core

Las versiones de bundles de Productive K3S Core usan el formato:

```text
X.Y.Z
```

Ejemplo:

```text
2.1.0
```

## Versiones de Infra

Las versiones de bundles de Productive K3S Infra codifican tanto la versión de Infra como la versión de Core a la que están atadas.

El formato es:

```text
X.Y.Z-A.B.C
```

Donde:

- `X.Y.Z` es la versión del bundle Infra;
- `A.B.C` es la versión del bundle Productive K3S Core requerida por ese bundle Infra.

Ejemplo:

```text
1.4.0-2.1.0
```

Esto significa que Infra `1.4.0` está atado a Core `2.1.0`.

## Mapeo de compatibilidad del CLI

Cada release del CLI debe definir exactamente qué versiones de Core e Infra soporta.

Ejemplo:

| Versión CLI | Bundle Core | Bundle Infra |
| --- | --- | --- |
| `1.0.0` | `2.1.0` | `1.4.0-2.1.0` |

El CLI debe rechazar combinaciones arbitrarias o incompatibles de Core/Infra.

## Defaults del repositorio

Los defaults versionados que usa el repositorio para resolver bundles remotos viven en `scripts/release-config.sh`:

- `PK3S_CLI_VERSION_DEFAULT`
- `PRODUCTIVE_K3S_CORE_VERSION_DEFAULT`
- `PRODUCTIVE_K3S_INFRA_VERSION_DEFAULT`
- `PRODUCTIVE_K3S_CORE_RELEASE_REPO_DEFAULT`
- `PRODUCTIVE_K3S_INFRA_RELEASE_REPO_DEFAULT`

El default de Infra siempre debe permanecer atado al default de Core. Eso implica que el sufijo de `PRODUCTIVE_K3S_INFRA_VERSION_DEFAULT` debe coincidir con `PRODUCTIVE_K3S_CORE_VERSION_DEFAULT`.

## Flujo de release para desarrollo

Cuando el CLI necesita pasar a una nueva baseline de Core/Infra:

1. ejecutar `make set-bundles-versions CORE_VERSION=A.B.C INFRA_VERSION=X.Y.Z-A.B.C`
2. ejecutar `make test-local-all`
3. crear el siguiente tag del CLI con `make tag-release VERSION=K.L.M`
4. pushear el tag semántico resultante con `git push origin K.L.M`

`set-bundles-versions` valida que ambos tags upstream de bundles ya existan antes de reescribir defaults, fixtures, docs y tests.

`tag-release` valida todo lo siguiente antes de crear el tag local del CLI:

- que la versión del CLI cumpla `X.Y.Z`
- que la versión default de Core sea válida
- que la versión default de Infra sea válida
- que la versión default de Infra esté atada a la versión default de Core
- que el tag default de Core exista en el remote configurado de `productive-k3s-core`
- que el tag default de Infra exista en el remote configurado de `productive-k3s-infra`
- que el tag del CLI no exista todavía en local
