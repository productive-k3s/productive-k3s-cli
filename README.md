# Productive K3S CLI

[![License: Apache-2.0](https://img.shields.io/badge/License-Apache--2.0-yellow.svg)](./LICENSE)

`productive-k3s-cli` is the unified command-line entrypoint for the Productive K3S ecosystem. It resolves Productive K3S Core and Productive K3S Infra bundles either from sibling repositories on disk or from published GitHub Releases, then delegates execution to their public entrypoints.

## Overview

This repository packages the user-facing executable `pk3s`.

The first release supports:

- local bundle resolution against sibling repositories;
- remote bundle resolution against GitHub Releases;
- command delegation to `productive-k3s-core.sh` and `productive-k3s-infra.sh`;
- native CLI builds for Linux, macOS, and Windows.

## Documentation

- Product docs site content: [docs/src/en/product](./docs/src/en/product/index.md)
- User docs site content: [docs/src/en/user-docs](./docs/src/en/user-docs/index.md)
- Developer docs site content: [docs/src/en/developer-docs](./docs/src/en/developer-docs/index.md)

## Installation

Unix-like install:

```bash
curl -fsSL https://raw.githubusercontent.com/jemacchi/productive-k3s-cli/main/install.sh | bash
```

Manual local install from source tree:

```bash
make build
./pk3s version
```

## Usage

Show the main command groups:

```bash
pk3s help
```

Force remote bundle resolution:

```bash
PRODUCTIVE_K3S_SOURCE=remote pk3s bundle core info --json
PRODUCTIVE_K3S_SOURCE=remote pk3s profile list
```

Force local sibling resolution:

```bash
PRODUCTIVE_K3S_SOURCE=local pk3s bundle infra info --json
PRODUCTIVE_K3S_SOURCE=local pk3s profile list
```

## Development

Useful commands:

```bash
make go-test
make test-static
make build
make test-cli-contract
make docs-build
make docs-publish-check
make set-bundles-versions CORE_VERSION=0.9.1 INFRA_VERSION=0.9.3-0.9.1
make tag-release VERSION=1.0.1
```

## Roadmap

- publish versioned CLI release archives and installer assets
- expand user-facing command coverage beyond the initial command contract
- document Windows PowerShell installation flow in release notes and docs

## License

This project is licensed under the Apache License 2.0.

See [LICENSE](./LICENSE).
