# Productive K3S CLI

Status: **WIP**

This repository contains the implementation and developer documentation for the Productive K3S CLI.

The CLI provides a unified user-facing interface over the Productive K3S ecosystem while preserving the public bundle interfaces exposed by Productive K3S Core and Productive K3S Infra.

## Documentation

The implementation-oriented specification is available under **Developer Documentation**.

## Local documentation workflow

```bash
make docs-build
make docs-serve
```

The documentation is built with MkDocs Material, following the same documentation workflow style used by the Productive K3S Core and Productive K3S Infra repositories.
