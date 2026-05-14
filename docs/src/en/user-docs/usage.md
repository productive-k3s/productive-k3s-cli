# Usage

## Resolution modes

The CLI supports two bundle resolution modes:

- `local`: only use sibling repositories on disk
- `remote`: only use published GitHub Releases

If `PRODUCTIVE_K3S_SOURCE` is not set, `pk3s` uses `remote` by default.

Examples:

```bash
PRODUCTIVE_K3S_SOURCE=remote pk3s bundle core info --json
PRODUCTIVE_K3S_SOURCE=local pk3s profile list
```

## Core-oriented commands

Examples:

```bash
pk3s doctor --core
pk3s install --core-only
pk3s validate --core
pk3s backup --core
pk3s bundle core info --json
```

These commands operate on the local host through Productive K3S Core.

If you run them from an unsupported host, the CLI stops early and points you to the supported-platforms documentation instead of trying to continue blindly.

## Infra-oriented commands

Examples:

```bash
pk3s profile list
pk3s profile validate --profile profiles/multipass/1-server-2-agents.env
pk3s validate --profile profiles/multipass/1-server-2-agents.env
pk3s plan --profile profiles/multipass/1-server-2-agents.env
pk3s apply --profile profiles/multipass/1-server-2-agents.env
pk3s destroy --profile profiles/multipass/1-server-2-agents.env
pk3s status --profile profiles/multipass/1-server-2-agents.env
pk3s bundle infra info --json
```

These commands delegate to Productive K3S Infra.

- `pk3s profile validate` checks the profile contract only.
- `pk3s validate --profile ...` runs the Infra scenario validation and may require generated deployment state.

## What remote mode resolves today

The current remote baseline is:

- Productive K3S Core `0.9.1`
- Productive K3S Infra `0.9.41-0.9.1`

The CLI downloads those release bundles from GitHub Releases, verifies their checksums, extracts them into the local cache, and then runs their public entrypoints.
