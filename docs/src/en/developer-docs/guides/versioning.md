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

## Repository defaults

The checked-in defaults used for remote bundle resolution live in `scripts/release-config.sh`:

- `PK3S_CLI_VERSION_DEFAULT`
- `PRODUCTIVE_K3S_CORE_VERSION_DEFAULT`
- `PRODUCTIVE_K3S_INFRA_VERSION_DEFAULT`
- `PRODUCTIVE_K3S_CORE_RELEASE_REPO_DEFAULT`
- `PRODUCTIVE_K3S_INFRA_RELEASE_REPO_DEFAULT`

The Infra default must always stay bound to the Core default, which means the suffix of `PRODUCTIVE_K3S_INFRA_VERSION_DEFAULT` must match `PRODUCTIVE_K3S_CORE_VERSION_DEFAULT`.

## Developer release flow

When the CLI needs to move to a new Core/Infra baseline:

1. run `make set-bundles-versions CORE_VERSION=A.B.C INFRA_VERSION=X.Y.Z-A.B.C`
2. run `make test-static`
3. create the next CLI tag with `make tag-release VERSION=K.L.M`
4. push the resulting semantic tag with `git push origin K.L.M`

`set-bundles-versions` validates that both upstream bundle tags already exist before rewriting defaults, fixtures, docs, and tests.

`tag-release` validates all of the following before creating the local CLI tag:

- the CLI version matches `X.Y.Z`
- the default Core version is valid
- the default Infra version is valid
- the default Infra version is bound to the default Core version
- the default Core tag exists in the configured `productive-k3s-core` remote
- the default Infra tag exists in the configured `productive-k3s-infra` remote
- the CLI tag does not already exist locally
