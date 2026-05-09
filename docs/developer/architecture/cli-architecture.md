# CLI Architecture

The Productive K3S CLI is planned as a Go application.

The CLI acts as:

- a unified UX layer;
- a bundle resolver;
- a compatibility gate;
- a command dispatcher;
- a workflow orchestrator;
- a diagnostics entrypoint.

The CLI must not embed the implementation details of Productive K3S Core or Productive K3S Infra.

## Main responsibilities

1. Resolve the CLI release metadata.
2. Resolve the exact Core and Infra bundle versions bound to that CLI release.
3. Download remote bundles or use local bundles in development mode.
4. Verify bundle metadata.
5. Execute supported public bundle commands.
6. Normalize output and exit codes where needed.
7. Provide a stable user-facing interface.

## Implementation boundaries

The CLI owns:

- command-line parsing;
- bundle cache management;
- version mapping;
- download and extraction;
- command mapping;
- high-level workflows.

Core and Infra own:

- cluster installation internals;
- host preflight internals;
- backup internals;
- validation internals;
- OpenTofu, Ansible, and shell scenario execution;
- profile interpretation.
