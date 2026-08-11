# Usage

## Resolution modes

If you want a short copy-paste oriented command list instead of this explanatory page, use [Quick reference](quick-reference.md).

The normal recommendation is:

- use `remote` when you are consuming Productive K3S as a product;
- use `local` when you are developing against sibling repositories on disk.

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

- Productive K3S Core `0.9.5`
- Productive K3S Infra `0.9.64-0.9.5`

The CLI downloads those release bundles from GitHub Releases, verifies their checksums, extracts them into the local cache, and then runs their public command surfaces.

That is why `pk3s` is the recommended user path: it keeps the experience simple while still making the underlying component versions explicit.

## Telemetry controls

The CLI can persist your telemetry preference and also override it per command.

Examples:

```bash
pk3s config telemetry status
pk3s config telemetry enable
pk3s config observability set pk3s_live_xxxxx
pk3s config observability status
pk3s plan --profile profiles/multipass/1-server-2-agents.env --telemetry disable
pk3s install --core-only --telemetry enable
```

When the CLI resolves telemetry to enabled, it propagates the decision plus correlation IDs into delegated Infra or Core runs.
