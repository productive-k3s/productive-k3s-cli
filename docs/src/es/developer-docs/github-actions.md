# GitHub Actions Y Releases

El repositorio del CLI sigue la misma separación general de automatización que el resto de los repositorios Productive K3S:

- La validación de PR chequea wiring estático, tests de Go, contratos del CLI y una validación live remota on-prem sobre GitHub-hosted.
- La documentación se construye con MkDocs y se publica en `gh-pages`.
- Los tags de release generan archives versionados del CLI e instaladores.

El sitio de documentación se publica con un workflow dedicado y mantiene `cli.productive-k3s.io` fijo tanto por el setting `cname` del workflow como por el archivo versionado `docs/src/CNAME`.
