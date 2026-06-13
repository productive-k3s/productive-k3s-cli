# Uso

Si buscas una lista corta de comandos para copiar y pegar, usa [Referencia rápida](referencia-rapida.md).

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

## Controles de telemetría

El CLI puede persistir tu preferencia de telemetría y también overridearla por comando.

Ejemplos:

```bash
pk3s config telemetry status
pk3s config telemetry enable
pk3s config observability set pk3s_live_xxxxx
pk3s config observability status
pk3s plan --profile profiles/multipass/1-server-2-agents.env --telemetry disable
pk3s install --core-only --telemetry enable
```

Cuando el CLI resuelve la telemetría como habilitada, propaga esa decisión y los IDs de correlación hacia las ejecuciones delegadas de Infra o Core.
