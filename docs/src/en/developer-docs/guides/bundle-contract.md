# Bundle Contract

The CLI works with release bundles produced by Productive K3S Core and Productive K3S Infra.

Each bundle must expose a stable public entrypoint at the repository root:

- `productive-k3s-core.sh`
- `productive-k3s-infra.sh`

Those root scripts delegate to the implementation scripts inside `scripts/`.

## Bundle metadata

Both bundles expose metadata through:

```bash
./productive-k3s-core.sh bundle info --json
./productive-k3s-infra.sh bundle info --json
```

The CLI must use this metadata to validate the bundle before executing user-facing workflows.

## Compatibility contract

The bundle metadata uses the compatibility contract:

```text
productive-k3s-cli-bundle-info/v1
```

The CLI should reject bundles that do not expose the expected contract.
