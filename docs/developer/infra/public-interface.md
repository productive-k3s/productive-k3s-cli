# Productive K3S Infra Public Interface

The Productive K3S Infra bundle exposes a root wrapper:

```bash
./productive-k3s-infra.sh <command> [args...]
```

The root wrapper delegates to:

```bash
./scripts/productive-k3s-infra.sh <command> [args...]
```

## Supported commands

```bash
./productive-k3s-infra.sh help
./productive-k3s-infra.sh version
./productive-k3s-infra.sh bundle info --json
./productive-k3s-infra.sh doctor
./productive-k3s-infra.sh list-profiles
./productive-k3s-infra.sh validate --profile <file>
./productive-k3s-infra.sh plan --profile <file>
./productive-k3s-infra.sh apply --profile <file>
./productive-k3s-infra.sh destroy --profile <file>
./productive-k3s-infra.sh status --profile <file>
```

## Supported global flags

```bash
--profile <file>
--debug
--yes
--dry-run
--json
```

## Legacy compatibility commands

```bash
./productive-k3s-infra.sh multipass [command]
./productive-k3s-infra.sh onprem [command]
./productive-k3s-infra.sh onprem-basic [command]
./productive-k3s-infra.sh aws-single-node [command]
```

## Release-bound Core dependency

Infra releases are bound to a specific Productive K3S Core version.

In release mode, Infra exports and enforces the Core version expected by that Infra bundle. The CLI must not override this relationship with an incompatible Core bundle.

The Infra bundle version expresses this relationship directly using the format:

```text
X.Y.Z-A.B.C
```

Example:

```text
1.4.0-2.1.0
```

This tells the CLI that Infra `1.4.0` expects Core `2.1.0`.
