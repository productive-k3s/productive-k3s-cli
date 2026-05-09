# Bundle Contract

Both Productive K3S Core and Productive K3S Infra expose bundle metadata using the same contract name:

```text
productive-k3s-cli-bundle-info/v1
```

## Core bundle info

The Core bundle exposes:

```bash
./productive-k3s-core.sh bundle info --json
```

Expected JSON shape:

```json
{
  "schema_version": "1",
  "bundle_name": "productive-k3s-core",
  "bundle_type": "productive-k3s-core",
  "bundle_version": "2.1.0",
  "cli_entrypoint": "productive-k3s-core.sh",
  "platform": "any",
  "api_compatibility": {
    "contract": "productive-k3s-cli-bundle-info/v1"
  }
}
```

## Infra bundle info

The Infra bundle exposes:

```bash
./productive-k3s-infra.sh bundle info --json
```

Expected JSON shape:

```json
{
  "schema_version": "1",
  "bundle_name": "productive-k3s-infra",
  "bundle_type": "productive-k3s-infra",
  "bundle_version": "1.4.0-2.1.0",
  "cli_entrypoint": "productive-k3s-infra.sh",
  "platform": "any",
  "api_compatibility": {
    "contract": "productive-k3s-cli-bundle-info/v1"
  }
}
```

## Validation requirements

The CLI must validate:

- `schema_version` equals `1`;
- `bundle_type` matches the expected bundle;
- `cli_entrypoint` exists after extraction;
- `api_compatibility.contract` equals `productive-k3s-cli-bundle-info/v1`;
- Core bundle version matches the CLI release mapping;
- Infra bundle version matches the CLI release mapping;
- Infra bundle version suffix matches the selected Core bundle version.

## Release metadata

The CLI should carry release metadata equivalent to:

```yaml
cli_version: 1.0.0
bundles:
  core:
    name: productive-k3s-core
    version: 2.1.0
  infra:
    name: productive-k3s-infra
    version: 1.4.0-2.1.0
```
