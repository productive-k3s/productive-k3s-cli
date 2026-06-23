# Cómo Usar

Para usuarios finales, el flujo esperado es simple:

1. Instalar `pk3s`.
2. Elegir un profile o escenario.
3. Ejecutar el comando necesario, por ejemplo `plan`, `apply`, `status` o `destroy`.
4. Dejar que el CLI descargue los bundles correspondientes de Productive K3S Core e Infra cuando se ejecuta en modo remoto.

El CLI está pensado para sentirse como un comando normal de plataforma:

```bash
pk3s help
pk3s profile list
pk3s plan --profile https://raw.githubusercontent.com/productive-k3s/productive-k3s-profiles/main/profiles/multipass/1-server-2-agents.env
pk3s apply --profile https://raw.githubusercontent.com/productive-k3s/productive-k3s-profiles/main/profiles/multipass/1-server-2-agents.env
```

Cuando un comando apunta directamente a Productive K3S Core, el CLI delega hacia el entrypoint público de Core. Cuando apunta a infraestructura o profiles, delega hacia el entrypoint público de Infra.
