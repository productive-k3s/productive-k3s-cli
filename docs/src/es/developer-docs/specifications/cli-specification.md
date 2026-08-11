# Especificación del CLI

## Versionado

Las versiones no usan prefijo `v`.

Ejemplos:

- `1.0.0`
- `2.1.0`

## Versionado de Infra

Los bundles de Infra siguen el formato:

```text
X.Y.Z-A.B.C
```

Donde:

- `X.Y.Z` es la versión del bundle de Infra
- `A.B.C` es la dependencia de Productive K3S Core

Ejemplo:

```text
1.4.0-2.1.0
```

## Contratos públicos

### Productive K3S Core

```bash
./productive-k3s-core.sh preflight
./productive-k3s-core.sh apply
./productive-k3s-core.sh validate
./productive-k3s-core.sh backup
./productive-k3s-core.sh bundle info --json
```

### Productive K3S Infra

```bash
./productive-k3s-infra.sh doctor
./productive-k3s-infra.sh list-profiles
./productive-k3s-infra.sh validate --profile profile.env
./productive-k3s-infra.sh plan --profile profile.env
./productive-k3s-infra.sh apply --profile profile.env
```
