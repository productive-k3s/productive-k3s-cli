# CLI Scope and Responsibilities

Productive K3S CLI is a user-facing orchestration layer for the Productive K3S ecosystem.

The CLI does not replace the public bundle scripts exposed by Productive K3S Core or Productive K3S Infra. Instead, it wraps them behind a unified command-line experience.

## Responsibilities

The CLI is responsible for:

- resolving compatible Core and Infra bundle versions;
- downloading or locating the required bundles;
- validating bundle metadata;
- exposing a consistent user-facing command structure;
- delegating execution to the bundle entrypoints;
- preserving deterministic behavior across releases.

## Non-goals

The CLI should not duplicate the installation, validation, OpenTofu, Ansible, or scenario-specific logic already maintained in the Core and Infra repositories.
