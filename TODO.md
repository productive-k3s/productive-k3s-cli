# TODO

Simple, versioned backlog for `productive-k3s-cli` only.

Format:
- `Title`: short action-oriented label
- `Description`: one sentence, max 250 chars, easy to scan in reviews

Rules:
- Keep only repo-local responsibilities here.
- Do not track work owned by other repositories.
- Cross-repo dependencies can be mentioned only as context, never as the main ownership of an item.

## Catalog and Package UX

- `Finish Stack Catalog Flows`
  `Complete catalog-backed stack discovery and install UX so published stack artifacts can be resolved and handed off cleanly through the CLI.`

- `Review Addon and Stack Command Symmetry`
  `Check that addon, stack, and profile commands expose a consistent naming, flag, and help model for local and remote package workflows.`

- `Improve Catalog Error Messages`
  `Make catalog resolution failures easier to understand by distinguishing download, validation, missing entry, and unsupported artifact cases.`

## Testing and Contracts

- `Expand Live Catalog Coverage`
  `Broaden live wiring coverage for catalog-backed profile, addon, and stack flows without relying on sibling source repositories.`

- `Refresh Bundle Fixtures`
  `Keep bundle-info fixtures and test manifests aligned with current Core and Infra release contracts to avoid stale contract coverage.`

- `Harden Remote Delegation Tests`
  `Add more focused tests around remote bundle resolution, delegation arguments, and version pin behavior across supported commands.`

## Documentation and Release

- `Document Stack Artifact Usage`
  `Update user and developer docs so stack installation examples reflect published stack artifacts and current catalog-backed workflows.`

- `Clarify Catalog Dependency Model`
  `Explain clearly which commands are catalog-backed, which delegate to Core or Infra, and where source repositories are no longer required.`
