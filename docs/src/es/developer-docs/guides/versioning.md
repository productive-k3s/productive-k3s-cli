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
