# Troubleshooting

## The CLI says the host is unsupported for Core

That usually means you are trying to run a Productive K3S Core host-installation command from a host that is not part of the current supported runtime matrix.

Examples:

- Windows host trying `pk3s install --core-only`
- macOS host trying `pk3s validate --core`

In that case, review the Core supported-platforms page:

- <https://core.productive-k3s.io/en/product/supported-platforms/>

## Remote bundle resolution fails

Check:

- network access to GitHub
- whether the expected release tag exists
- whether the release asset names still match the current published layout

## Local bundle resolution fails

`PRODUCTIVE_K3S_SOURCE=local` expects sibling repositories near the CLI checkout:

- `productive-k3s-core`
- `productive-k3s-infra`

Each one must expose its public command surface at the repository root.

## A scenario command fails after delegation

At that point the failure is usually inside Productive K3S Core or Productive K3S Infra, not in the CLI wrapper.

Typical examples:

- missing `multipass`
- missing cloud credentials
- invalid profile variables
- unsupported remote target environment
