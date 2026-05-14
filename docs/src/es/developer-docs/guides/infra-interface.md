# Mapeo de interfaz Infra

Productive K3S Infra expone el entrypoint público:

```bash
./productive-k3s-infra.sh <command> [args...]
```

El script raíz delega en:

```bash
scripts/productive-k3s-infra.sh
```

## Comandos basados en profiles

La interfaz actual basada en profiles incluye:

```bash
./productive-k3s-infra.sh doctor
./productive-k3s-infra.sh list-profiles
./productive-k3s-infra.sh validate --profile <file>
./productive-k3s-infra.sh plan --profile <file>
./productive-k3s-infra.sh apply --profile <file>
./productive-k3s-infra.sh destroy --profile <file>
./productive-k3s-infra.sh status --profile <file>
./productive-k3s-infra.sh bundle info --json
```

## Compatibilidad con escenarios legacy

Infra también expone comandos de compatibilidad para escenarios existentes:

```bash
./productive-k3s-infra.sh multipass [command]
./productive-k3s-infra.sh onprem [command]
./productive-k3s-infra.sh onprem-basic [command]
./productive-k3s-infra.sh aws-single-node [command]
```

## Mapeo desde el CLI

Productive K3S CLI debe tratar la interfaz basada en profiles como el contrato principal.

Ejemplos:

| Comando CLI | Delegación Infra |
| --- | --- |
| `pk3s profile list` | `productive-k3s-infra.sh list-profiles` |
| `pk3s plan --profile <file>` | `productive-k3s-infra.sh plan --profile <file>` |
| `pk3s apply --profile <file>` | `productive-k3s-infra.sh apply --profile <file>` |
| `pk3s destroy --profile <file>` | `productive-k3s-infra.sh destroy --profile <file>` |
| `pk3s status --profile <file>` | `productive-k3s-infra.sh status --profile <file>` |
