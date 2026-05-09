# Ecosystem

The Productive K3S ecosystem is composed of repositories with separate responsibilities.

## Productive K3S Core

Productive K3S Core owns the base K3S installation and core cluster stack.

It exposes a public script interface through:

```bash
./productive-k3s-core.sh <command> [args...]
```

The current public Core interface includes operational commands such as `preflight`, `bootstrap`, `backup`, `validate`, and `bundle info --json`.

## Productive K3S Infra

Productive K3S Infra owns infrastructure scenarios, profiles, OpenTofu, Ansible, and environment-specific orchestration.

It exposes a public script interface through:

```bash
./productive-k3s-infra.sh <command> [args...]
```

It supports profile-driven commands and legacy scenario commands.

## Productive K3S CLI

The Productive K3S CLI is the unified user-facing layer.

It should not reimplement Core or Infra internals. Instead, it resolves compatible bundle versions, downloads or locates the bundles, validates their public metadata, and delegates to their public interfaces.

## Productive K3S Addons

Productive K3S Addons may contain optional addon examples and reference integrations. The CLI may support addon workflows in later iterations, but addons are not part of the first required contract.
