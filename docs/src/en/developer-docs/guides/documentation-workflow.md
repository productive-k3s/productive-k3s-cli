# Documentation Workflow

The CLI repository follows the same MkDocs workflow used by the Productive K3S repositories.

## Build documentation

```bash
make docs-build
```

## Serve documentation locally

```bash
make docs-serve
```

## Serve documentation in the background

```bash
make docs-up
```

The local site is served at:

```text
http://127.0.0.1:8000
```

## Stop and clean documentation artifacts

```bash
make docs-down
make docs-clean
```

The documentation scripts create a local Python virtual environment under `docs/.venv` and install the Python requirements declared in `docs/requirements.txt`.
