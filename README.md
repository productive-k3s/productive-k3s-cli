# Productive K3S CLI

[![License: Apache-2.0](https://img.shields.io/badge/License-Apache--2.0-yellow.svg)](./LICENSE)

`productive-k3s-cli` is the simplest and recommended unified interface for using Productive K3S.

It resolves Productive K3S Core and Productive K3S Infra bundles either from sibling repositories on disk or from published GitHub Releases, then delegates execution to their public command surfaces.

## Overview

This repository packages the user-facing executable `pk3s`.

The first release supports:

- local bundle resolution against sibling repositories;
- remote bundle resolution against GitHub Releases;
- command delegation to `productive-k3s-core.sh` and `productive-k3s-infra.sh`;
- native CLI builds for Linux, macOS, and Windows.

`pk3s` does not replace `core` or `infra`. It keeps them independently usable while making the normal path much easier.

## Documentation

- Product docs site content: [docs/src/en/product](./docs/src/en/product/index.md)
- User docs site content: [docs/src/en/user-docs](./docs/src/en/user-docs/index.md)
- Developer docs site content: [docs/src/en/developer-docs](./docs/src/en/developer-docs/index.md)

## Installation

Unix-like install:

```bash
curl -fsSL https://raw.githubusercontent.com/productive-k3s/productive-k3s-cli/main/install.sh | bash
```

Manual local install from source tree:

```bash
make build
./pk3s version
./pk3s bom --json
```

## Usage

Show the main command groups:

```bash
pk3s help
pk3s bom --json
```

Force remote bundle resolution:

```bash
PRODUCTIVE_K3S_SOURCE=remote pk3s bundle core info --json
PRODUCTIVE_K3S_SOURCE=remote pk3s profile list
```

`pk3s profile list` is catalog-backed. It does not delegate to the Infra bundle, which keeps the remote Infra release surface focused on packaged deployment commands.

Catalog-backed package usage:

```bash
pk3s addon list
pk3s profile show aws-single-node-basic
pk3s infra install aws-single-node-basic --env-file ./aws.env
pk3s addon install nginx --profile aws-single-node-basic
pk3s addon install nginx --profile multipass-1-server-2-agents --public-host nginx-01.k3s.lab.internal
```

The embedded `profile.env` inside a distributed `profile.tgz` is treated as a base/defaults file. Using a packaged profile without local overrides only makes sense for self-contained targets such as local host-driven scenarios. For real installations, especially cloud and on-prem targets, pass installation-specific values from the invoking machine through `--env-file`.

When the catalog declares that a profile requires local overrides, `pk3s` now fails early before runtime if `--env-file` is missing. Use `pk3s profile show <name>` to inspect the install inputs summary exposed by the catalog.

`pk3s` only prepares command-level telemetry for mutating workflows such as `install`, `profile install`, `infra install`, `infra apply`, `infra destroy`, `apply`, `destroy`, and `addon install`. Read-only commands such as `help`, `version`, `bom --json`, `bundle info --json`, `profile list`, `profile show`, `profile validate`, `infra plan`, `infra status`, `addon list`, and `addon validate` do not prompt for telemetry or emit command-level events.

For add-ons, `--public-host` is intentionally narrow. It only covers the basic Core-managed ingress case for add-ons that declare that support in metadata. Richer ingress behavior remains an add-on concern rather than a generic `pk3s` or Core feature.

## When to use CLI

- when you want the simplest and recommended Productive K3S experience
- when you want one coherent interface across base installation and deployment workflows
- when you want to move between local development and published releases without changing the user-facing workflow

## When to use Core or Infra directly

- use `core` directly when you want the most explicit Kubernetes base-installation contract
- use `infra` directly when you want the most explicit deployment-layer control

Force local sibling resolution:

```bash
PRODUCTIVE_K3S_SOURCE=local pk3s bundle infra info --json
PRODUCTIVE_K3S_SOURCE=local pk3s profile list
```

## Development

Development prerequisites for this repository:

- a working `Go` toolchain available in `PATH` or through `GO_BIN`
- standard Unix tooling used by the repository scripts and docs flow

`Go` is a development prerequisite for `productive-k3s-cli` itself because local builds, Go tests, and live remote validation build the `pk3s` executable from source.
It is not a general development prerequisite for `productive-k3s-core` or `productive-k3s-infra`.

If your environment exposes more than one `go` binary, you can force the toolchain explicitly:

```bash
GO_BIN=/usr/local/go/bin/go make build
```

Useful commands:

```bash
make build
make test-local-all
make test-live-remote
make test-live-catalog
make test-live-export
make test-clean-all
make docs-build
make set-bundles-versions CORE_VERSION=0.9.5 INFRA_VERSION=0.9.63-0.9.5
make tag-release VERSION=1.0.1
```

## Roadmap

- publish versioned CLI release archives and installer assets
- expand user-facing command coverage beyond the initial command contract
- document Windows PowerShell installation flow in release notes and docs

## License

This project is licensed under the Apache License 2.0.

See [LICENSE](./LICENSE).
