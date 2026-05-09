# Productive K3S Core Public Interface

The Productive K3S Core bundle exposes a root wrapper:

```bash
./productive-k3s-core.sh <command> [args...]
```

The root wrapper delegates to:

```bash
./scripts/productive-k3s-core.sh <command> [args...]
```

## Supported operational commands

```bash
./productive-k3s-core.sh bundle info --json
./productive-k3s-core.sh preflight
./productive-k3s-core.sh preflight --strict
./productive-k3s-core.sh bootstrap
./productive-k3s-core.sh bootstrap --dry-run
./productive-k3s-core.sh backup
./productive-k3s-core.sh validate
./productive-k3s-core.sh validate --strict
./productive-k3s-core.sh help
```

## Default behavior

If no command is provided, Core defaults to `bootstrap`.

If the first argument is an option, Core also defaults to `bootstrap` for release-installer compatibility.

## Bundle metadata

Core exposes metadata through:

```bash
./productive-k3s-core.sh bundle info --json
```

The CLI must use this command to validate the selected Core bundle before dispatching operational commands.

## Make target compatibility

The Core repository exposes Make targets that map to the public wrapper and developer script. The Productive K3S CLI repository should follow the same naming style for documentation and tests, including targets such as:

```bash
make docs-build
make docs-serve
make docs-up
make docs-down
make docs-clean
```
