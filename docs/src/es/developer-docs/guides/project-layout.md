# Organización del proyecto

El repositorio del CLI debe seguir el mismo flujo de documentación usado por los repositorios Productive K3S.

Organización esperada:

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

Los archivos Markdown viven bajo `docs/src`.

La configuración de MkDocs vive en `docs/mkdocs.yml` y usa `docs_dir: src`.
