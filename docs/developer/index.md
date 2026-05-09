# Developer Documentation

This section defines the first implementation-oriented specification for the Productive K3S CLI.

The CLI is intended to provide a unified user-facing interface over the Productive K3S ecosystem while preserving the existing public bundle interfaces exposed by Productive K3S Core and Productive K3S Infra.

## Goals

- Provide one consistent CLI entrypoint for users.
- Keep Productive K3S Core and Productive K3S Infra as independent versioned bundles.
- Map CLI commands to existing public bundle interfaces.
- Keep bundle combinations deterministic and release-bound.
- Document the contract clearly enough for implementation and maintenance.
- Use MkDocs for project documentation.

## Non-goals

- Replace Helm, OpenTofu, Ansible, K3S, or k3sup.
- Move infrastructure logic into the CLI.
- Allow arbitrary bundle version combinations in release mode.
- Make the CLI a generic orchestration framework.
