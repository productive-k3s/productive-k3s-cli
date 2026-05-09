# Mapeo de interfaz Core

Productive K3S Core expone el entrypoint público:

```bash
./productive-k3s-core.sh <command> [args...]
```

El script raíz delega en:

```bash
scripts/productive-k3s-core.sh
```

## Comandos públicos

La interfaz pública actual de Core incluye:

```bash
./productive-k3s-core.sh bundle info --json
./productive-k3s-core.sh preflight
./productive-k3s-core.sh preflight --strict
./productive-k3s-core.sh bootstrap
./productive-k3s-core.sh bootstrap --dry-run
./productive-k3s-core.sh backup
./productive-k3s-core.sh validate
./productive-k3s-core.sh validate --strict
```

## Mapeo desde el CLI

Productive K3S CLI debe mapear comandos de usuario hacia la interfaz de Core sin duplicar su lógica.

Ejemplos:

| Comando CLI | Delegación Core |
| --- | --- |
| `productive-k3s doctor` | `productive-k3s-core.sh preflight` más chequeos de Infra |
| `productive-k3s install` | `productive-k3s-core.sh bootstrap` más workflow opcional de Infra |
| `productive-k3s validate` | `productive-k3s-core.sh validate` más validación opcional de Infra |
| `productive-k3s bundle info` | `productive-k3s-core.sh bundle info --json` |
