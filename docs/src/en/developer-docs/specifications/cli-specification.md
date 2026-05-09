# CLI Specification

## Versioning

Versions do not use a `v` prefix.

Examples:

- `1.0.0`
- `2.1.0`

## Infra versioning

Infra bundles follow:

```text
X.Y.Z-A.B.C
```

Where:

- `X.Y.Z` is the Infra bundle version
- `A.B.C` is the Productive K3S Core bundle dependency

Example:

```text
1.4.0-2.1.0
```

## Public contracts

### Productive K3S Core

```bash
./productive-k3s-core.sh preflight
./productive-k3s-core.sh bootstrap
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
