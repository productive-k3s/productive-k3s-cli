# Project Layout

The CLI repository should follow the same documentation workflow used by the Productive K3S repositories.

Expected top-level layout:

```text
.
├── Makefile
├── scripts/
│   └── productive-k3s-cli-dev.sh
└── docs/
    ├── build.sh
    ├── serve.sh
    ├── clean.sh
    ├── requirements.txt
    ├── mkdocs.yml
    └── src/
        ├── index.md
        ├── assets/
        ├── en/
        └── es/
```

Documentation Markdown files live under `docs/src`.

The MkDocs configuration lives under `docs/mkdocs.yml` and uses `docs_dir: src`.
