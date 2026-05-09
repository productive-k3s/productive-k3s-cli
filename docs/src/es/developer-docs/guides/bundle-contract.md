# Contrato de bundles

El CLI trabaja con bundles de release producidos por Productive K3S Core y Productive K3S Infra.

Cada bundle debe exponer un entrypoint público estable en la raíz del repositorio:

- `productive-k3s-core.sh`
- `productive-k3s-infra.sh`

Esos scripts raíz delegan en los scripts de implementación dentro de `scripts/`.

## Metadata de bundle

Ambos bundles exponen metadata mediante:

```bash
./productive-k3s-core.sh bundle info --json
./productive-k3s-infra.sh bundle info --json
```

El CLI debe usar esta metadata para validar el bundle antes de ejecutar workflows orientados al usuario.

## Contrato de compatibilidad

La metadata de bundle usa el contrato de compatibilidad:

```text
productive-k3s-cli-bundle-info/v1
```

El CLI debe rechazar bundles que no expongan el contrato esperado.
