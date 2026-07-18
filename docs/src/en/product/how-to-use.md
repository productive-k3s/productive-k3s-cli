# How To Use

For end users, the expected flow is simple:

1. Install `pk3s`.
2. Choose a profile or scenario.
3. Run the command you need, such as `plan`, `apply`, `status`, or `destroy`.
4. Let the CLI fetch the matching Productive K3S Core and Infra bundles when running in remote mode.

The CLI is designed to feel like a normal platform command:

```bash
pk3s help
pk3s profile list
pk3s plan --profile https://raw.githubusercontent.com/productive-k3s/productive-k3s-profiles/main/profiles/multipass/1-server-2-agents.env
pk3s apply --profile https://raw.githubusercontent.com/productive-k3s/productive-k3s-profiles/main/profiles/multipass/1-server-2-agents.env
```

When a command targets Productive K3S Core directly, the CLI delegates into the Core public contract. When a command targets infrastructure or profiles, the CLI delegates into the Infra public contract.
