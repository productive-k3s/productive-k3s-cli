# Command Mapping

This page documents how the unified CLI maps to the current public bundle interfaces.

## Core mapping

| Unified CLI command | Core bundle command |
| --- | --- |
| `productive-k3s preflight` | `./productive-k3s-core.sh preflight` |
| `productive-k3s preflight --strict` | `./productive-k3s-core.sh preflight --strict` |
| `productive-k3s bootstrap` | `./productive-k3s-core.sh bootstrap` |
| `productive-k3s bootstrap --dry-run` | `./productive-k3s-core.sh bootstrap --dry-run` |
| `productive-k3s backup` | `./productive-k3s-core.sh backup` |
| `productive-k3s validate` | `./productive-k3s-core.sh validate` |
| `productive-k3s validate --strict` | `./productive-k3s-core.sh validate --strict` |
| `productive-k3s core bundle info --json` | `./productive-k3s-core.sh bundle info --json` |

## Infra mapping

| Unified CLI command | Infra bundle command |
| --- | --- |
| `productive-k3s doctor` | `./productive-k3s-infra.sh doctor` |
| `productive-k3s list-profiles` | `./productive-k3s-infra.sh list-profiles` |
| `productive-k3s validate --profile <file>` | `./productive-k3s-infra.sh validate --profile <file>` |
| `productive-k3s plan --profile <file>` | `./productive-k3s-infra.sh plan --profile <file>` |
| `productive-k3s apply --profile <file>` | `./productive-k3s-infra.sh apply --profile <file>` |
| `productive-k3s destroy --profile <file>` | `./productive-k3s-infra.sh destroy --profile <file>` |
| `productive-k3s status --profile <file>` | `./productive-k3s-infra.sh status --profile <file>` |
| `productive-k3s infra bundle info --json` | `./productive-k3s-infra.sh bundle info --json` |

## Legacy Infra scenario mapping

The Infra bundle keeps legacy scenario commands for compatibility.

| Unified CLI command | Infra legacy command |
| --- | --- |
| `productive-k3s multipass up` | `./productive-k3s-infra.sh multipass up` |
| `productive-k3s onprem up` | `./productive-k3s-infra.sh onprem up` |
| `productive-k3s aws-single-node up` | `./productive-k3s-infra.sh aws-single-node up` |

The first CLI implementation may expose these commands directly or keep them under an explicit compatibility namespace. The preferred long-term interface is profile-driven.

## Install mapping

The first implementation of `install` should be conservative.

Suggested behavior:

```text
productive-k3s install --profile <file>
  -> resolve bundles
  -> run Core preflight
  -> run Infra doctor --profile <file>
  -> run Infra validate --profile <file>
  -> run Infra apply --profile <file>
  -> run Core validate
```

For `--dry-run`, `install` should map Infra `apply` to Infra `plan` and avoid mutating operations.
