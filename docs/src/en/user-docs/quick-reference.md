# Quick Reference

This page is meant to be copy-paste friendly. It is not the full CLI specification; it is the shortest practical command list for day-to-day use.

## General

Check the CLI version:

```bash
pk3s version
```

Show the recursive bill of materials for the CLI and the bundles it resolves:

```bash
pk3s bom --json | jq
```

Use local sibling repositories instead of published remote bundles:

```bash
PRODUCTIVE_K3S_SOURCE=local pk3s bom --json | jq
```

## Catalog and discovery

List published profiles from the catalog:

```bash
pk3s profile list
```

Show one profile and its install inputs summary:

```bash
pk3s profile show multipass-1-server-2-agents
```

List published addons from the catalog:

```bash
pk3s addon list
pk3s addon show nginx
```

List published stacks from the catalog:

```bash
pk3s stack list
pk3s stack show cluster-health
```

## Profiles and infrastructure

Validate a published profile package:

```bash
pk3s profile validate multipass-1-server-2-agents
```

Install a local Multipass cluster from a published profile:

```bash
pk3s infra install multipass-1-server-2-agents
```

Check the current status of that profile deployment:

```bash
pk3s infra status multipass-1-server-2-agents
```

Destroy that profile deployment:

```bash
pk3s infra destroy multipass-1-server-2-agents
```

Install a profile that requires installation-specific local overrides:

```bash
pk3s infra install aws-single-node-basic --env-file ./aws.env
```

Inspect a profile before installing so you know whether `--env-file` is required:

```bash
pk3s profile show aws-single-node-basic
```

## Addons

Validate a published addon package:

```bash
pk3s addon validate nginx
```

Install an addon into a previously installed profile target:

```bash
pk3s addon install nginx --profile multipass-1-server-2-agents
```

Install an addon and expose it publicly through Traefik with a single ingress host:

```bash
pk3s addon install nginx --profile multipass-1-server-2-agents --public-host nginx.k3s.lab.internal
```

Install an addon against an explicit kubeconfig instead of a profile:

```bash
pk3s addon install nginx --kubeconfig ~/.kube/config
```

Install an addon against a registered cluster or a specific kube context:

```bash
pk3s addon install nginx --cluster local-dev
pk3s addon install nginx --cluster-context default
```

## Stacks

Install a published stack by catalog name:

```bash
pk3s stack install cluster-health --dry-run
```

Install a stack on an explicit target:

```bash
pk3s stack install cluster-health --cluster local-dev
pk3s stack install cluster-health --kubeconfig ~/.kube/config
pk3s stack install cluster-health --cluster-context default
pk3s stack install cluster-health --profile multipass-1-server-2-agents
```

Export a published stack as a self-contained installer:

```bash
pk3s stack export cluster-health --output ./cluster-health-installer
```

## Telemetry

Show the current telemetry preference:

```bash
pk3s config telemetry status
```

Enable persisted telemetry preference:

```bash
pk3s config telemetry enable
```

Disable telemetry for a single mutating command:

```bash
pk3s infra install multipass-1-server-2-agents --telemetry disable
```

## Short Notes

- `profile list`, `profile show`, `addon list`, `addon show`, `stack list`, `stack show`, `bom --json`, and `version` are read-only commands.
- `infra install`, `infra destroy`, `addon install`, and `stack install` are mutating commands.
- `addon install --profile <name>` requires profile state from a previous `infra install` or `infra status`.
- `--public-host` only works when the addon package declares support for the basic Core-managed ingress contract.
