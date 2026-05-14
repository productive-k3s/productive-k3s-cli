# Uso

Modos de resolución:

- `local`
- `remote`

Si `PRODUCTIVE_K3S_SOURCE` no está seteada, `pk3s` usa `remote` por defecto.

Ejemplos:

```bash
PRODUCTIVE_K3S_SOURCE=remote pk3s bundle core info --json
PRODUCTIVE_K3S_SOURCE=local pk3s profile list
```

Comandos orientados a Infra:

```bash
pk3s profile list
pk3s profile validate --profile profiles/multipass/1-server-2-agents.env
pk3s validate --profile profiles/multipass/1-server-2-agents.env
pk3s plan --profile profiles/multipass/1-server-2-agents.env
pk3s apply --profile profiles/multipass/1-server-2-agents.env
pk3s destroy --profile profiles/multipass/1-server-2-agents.env
pk3s status --profile profiles/multipass/1-server-2-agents.env
```

- `pk3s profile validate` valida sólo el contrato del profile.
- `pk3s validate --profile ...` ejecuta la validación del escenario de Infra y puede requerir estado generado del deployment.
