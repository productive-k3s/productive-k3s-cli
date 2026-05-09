# 0001 - Initial CLI Contract

Status: Proposed

## Context

Productive K3S Core and Productive K3S Infra already expose public shell interfaces. The Productive K3S CLI should unify the user experience without duplicating the internal logic of either repository.

## Decision

The CLI will be implemented as a Go-based orchestration layer that resolves, validates, and dispatches to versioned Core and Infra bundles.

The CLI release will bind exactly one Core bundle version and one Infra bundle version.

Core versions use semantic versions such as `2.1.0`.

Infra versions use a composed version format such as `1.4.0-2.1.0`, where the suffix declares the Core bundle version used by Infra actions.

The CLI will validate bundle metadata using the `productive-k3s-cli-bundle-info/v1` contract.

## Consequences

- The CLI remains small and focused.
- Core and Infra remain independently testable.
- Release compatibility is explicit.
- The Infra-to-Core dependency is visible in the Infra bundle version.
- The user-facing interface becomes stable even if internal scripts evolve.
- Arbitrary bundle combinations are intentionally not supported in release mode.
