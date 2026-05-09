# Infra Interface Mapping

Productive K3S Infra exposes the public entrypoint:

```bash
./productive-k3s-infra.sh <command> [args...]
```

The root script delegates to:

```bash
scripts/productive-k3s-infra.sh
```

## Profile-driven commands

The current profile-driven interface includes:

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

## Legacy scenario compatibility

Infra also exposes compatibility commands for existing scenarios:

```bash
./productive-k3s-infra.sh multipass [command]
./productive-k3s-infra.sh onprem [command]
./productive-k3s-infra.sh onprem-basic [command]
./productive-k3s-infra.sh aws-single-node [command]
```

## CLI mapping

The Productive K3S CLI should treat the profile-driven interface as the primary contract.

Example mappings:

| CLI command | Infra delegation |
| --- | --- |
| `productive-k3s profile list` | `productive-k3s-infra.sh list-profiles` |
| `productive-k3s plan --profile <file>` | `productive-k3s-infra.sh plan --profile <file>` |
| `productive-k3s apply --profile <file>` | `productive-k3s-infra.sh apply --profile <file>` |
| `productive-k3s destroy --profile <file>` | `productive-k3s-infra.sh destroy --profile <file>` |
| `productive-k3s status --profile <file>` | `productive-k3s-infra.sh status --profile <file>` |
