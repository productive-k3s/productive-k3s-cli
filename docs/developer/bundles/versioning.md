# Versioning

The Productive K3S CLI release is a compatibility contract.

A CLI version maps to exactly one Productive K3S Core bundle version and exactly one Productive K3S Infra bundle version.

## Version formats

### CLI versions

The CLI uses standard semantic versions, for example:

```text
1.0.0
1.1.0
1.2.0
```

### Core bundle versions

Productive K3S Core bundle versions also use semantic versions, for example:

```text
2.1.0
2.2.0
2.3.0
```

### Infra bundle versions

Productive K3S Infra bundle versions encode both the Infra bundle version and the Core bundle version used by Infra actions.

The format is:

```text
X.Y.Z-A.B.C
```

Where:

- `X.Y.Z` is the Productive K3S Infra bundle version.
- `A.B.C` is the Productive K3S Core bundle version bound into that Infra release.

Example:

```text
1.4.0-2.1.0
```

This means:

- Infra bundle version: `1.4.0`
- Core bundle dependency used by Infra actions: `2.1.0`

## Release-bound mapping

Example CLI release metadata:

```yaml
cli_version: 1.0.0
core_bundle_version: 2.1.0
infra_bundle_version: 1.4.0-2.1.0
```

The CLI must validate that the Infra bundle dependency suffix matches the Core bundle version selected by the CLI release.

For example, this mapping is valid:

```yaml
core_bundle_version: 2.1.0
infra_bundle_version: 1.4.0-2.1.0
```

This mapping must be rejected:

```yaml
core_bundle_version: 2.2.0
infra_bundle_version: 1.4.0-2.1.0
```

## Rules

- Users must not freely mix Core and Infra versions in release mode.
- The CLI must reject incompatible user-provided bundle versions.
- The CLI must validate the bundle metadata returned by both bundles.
- The CLI must treat the Infra version suffix as an explicit Core dependency declaration.
