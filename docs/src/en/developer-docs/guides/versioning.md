# Versioning Model

Productive K3S CLI, Productive K3S Core, and Productive K3S Infra use semantic-style release versions without a `v` prefix.

Examples:

```text
1.0.0
2.1.0
```

## Core bundle versions

Productive K3S Core bundle versions use the format:

```text
X.Y.Z
```

Example:

```text
2.1.0
```

## Infra bundle versions

Productive K3S Infra bundle versions encode both the Infra release and the Core release it is bound to.

The format is:

```text
X.Y.Z-A.B.C
```

Where:

- `X.Y.Z` is the Infra bundle version;
- `A.B.C` is the Productive K3S Core bundle version required by that Infra bundle.

Example:

```text
1.4.0-2.1.0
```

This means Infra `1.4.0` is bound to Core `2.1.0`.

## CLI compatibility mapping

Each CLI release must define the exact Core and Infra bundle versions it supports.

Example:

| CLI version | Core bundle | Infra bundle |
| --- | --- | --- |
| `1.0.0` | `2.1.0` | `1.4.0-2.1.0` |

The CLI must reject arbitrary or incompatible Core/Infra combinations.
