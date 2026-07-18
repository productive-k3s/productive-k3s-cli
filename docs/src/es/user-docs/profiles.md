# Perfiles

Los flujos de Infra usan perfiles, y el CLI es la forma más simple de consumir esos caminos curados de despliegue.

Ejemplos:

```bash
pk3s profile list
pk3s profile validate --profile profiles/on-prem/basic.env
pk3s validate --profile profiles/on-prem/basic.env
pk3s plan --profile profiles/multipass/1-server-2-agents.env
```

- `pk3s profile validate` chequea sólo el contrato del `.env`.
- `pk3s validate --profile ...` delega la validación del escenario a Infra y puede requerir estado generado del clúster.
