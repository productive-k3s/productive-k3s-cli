# Core Interface Mapping

Productive K3S Core exposes the public entrypoint:

```bash
./productive-k3s-core.sh <command> [args...]
```

The root script delegates to:

```bash
scripts/productive-k3s-core.sh
```

## Public commands

The current Core public interface includes:

```bash
./productive-k3s-core.sh bundle info --json
./productive-k3s-core.sh preflight
./productive-k3s-core.sh preflight --strict
./productive-k3s-core.sh bootstrap
./productive-k3s-core.sh bootstrap --dry-run
./productive-k3s-core.sh backup
./productive-k3s-core.sh validate
./productive-k3s-core.sh validate --strict
```

## CLI mapping

The Productive K3S CLI should map user-facing commands to the Core interface without duplicating Core logic.

Example mappings:

| CLI command | Core delegation |
| --- | --- |
| `productive-k3s doctor` | `productive-k3s-core.sh preflight` plus Infra checks |
| `productive-k3s install` | `productive-k3s-core.sh bootstrap` plus optional Infra workflow |
| `productive-k3s validate` | `productive-k3s-core.sh validate` plus optional Infra validation |
| `productive-k3s bundle info` | `productive-k3s-core.sh bundle info --json` |
