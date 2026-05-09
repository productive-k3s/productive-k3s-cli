# Execution Model

The CLI delegates to public bundle entrypoints.

```text
productive-k3s CLI
  -> resolve CLI release mapping
  -> resolve Core bundle
  -> resolve Infra bundle
  -> verify bundle info contract
  -> dispatch to bundle entrypoint
  -> return normalized result
```

## Public bundle entrypoints

Core:

```bash
./productive-k3s-core.sh <command> [args...]
```

Infra:

```bash
./productive-k3s-infra.sh <command> [args...]
```

## Execution rules

- The CLI must execute the root public wrapper when possible.
- The CLI must not call private scripts directly unless explicitly documented as part of a future contract.
- The CLI must preserve bundle exit codes unless a command explicitly normalizes them.
- The CLI must surface stderr/stdout clearly.
- The CLI must support a verbose/debug mode for troubleshooting.

## Local development mode

In development mode, the CLI may use local paths instead of downloaded bundles.

Local mode must still validate the public `bundle info --json` contract.
