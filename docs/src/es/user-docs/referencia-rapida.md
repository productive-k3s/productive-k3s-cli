# Referencia rápida

Esta página está pensada para copiar y pegar comandos. No reemplaza la especificación completa del CLI; es la lista corta más útil para el uso diario.

## General

Ver la versión del CLI:

```bash
pk3s version
```

Ver el BOM recursivo del CLI y de los bundles que resuelve:

```bash
pk3s bom --json | jq
```

Usar repos sibling locales en lugar de bundles remotos publicados:

```bash
PRODUCTIVE_K3S_SOURCE=local pk3s bom --json | jq
```

## Catálogo y discovery

Listar los profiles publicados en el catálogo:

```bash
pk3s profile list
```

Ver un profile y el resumen de inputs de instalación:

```bash
pk3s profile show multipass-1-server-2-agents
```

Listar los addons publicados en el catálogo:

```bash
pk3s addon list
```

## Profiles e infraestructura

Validar un profile publicado:

```bash
pk3s profile validate multipass-1-server-2-agents
```

Instalar un cluster local de Multipass desde un profile publicado:

```bash
pk3s infra install multipass-1-server-2-agents
```

Ver el estado actual de ese deployment:

```bash
pk3s infra status multipass-1-server-2-agents
```

Destruir ese deployment:

```bash
pk3s infra destroy multipass-1-server-2-agents
```

Instalar un profile que requiere overrides locales específicos de la instalación:

```bash
pk3s infra install aws-single-node-basic --env-file ./aws.env
```

Inspeccionar un profile antes de instalarlo para saber si `--env-file` es obligatorio:

```bash
pk3s profile show aws-single-node-basic
```

## Addons

Validar un addon publicado:

```bash
pk3s addon validate nginx
```

Instalar un addon dentro de un profile ya instalado:

```bash
pk3s addon install nginx --profile multipass-1-server-2-agents
```

Instalar un addon y publicarlo por Traefik con un único host de ingress:

```bash
pk3s addon install nginx --profile multipass-1-server-2-agents --public-host nginx.k3s.lab.internal
```

Instalar un addon contra un kubeconfig explícito en vez de un profile:

```bash
pk3s addon install nginx --kubeconfig ~/.kube/config
```

Instalar un addon contra un contexto específico:

```bash
pk3s addon install nginx --cluster-context default
```

## Telemetría

Ver la preferencia actual de telemetría:

```bash
pk3s config telemetry status
```

Habilitar la preferencia persistida de telemetría:

```bash
pk3s config telemetry enable
```

Deshabilitar telemetría para un único comando mutante:

```bash
pk3s infra install multipass-1-server-2-agents --telemetry disable
```

## Notas cortas

- `profile list`, `profile show`, `addon list`, `bom --json` y `version` son comandos de solo lectura.
- `infra install`, `infra destroy` y `addon install` son comandos mutantes.
- `addon install --profile <name>` requiere estado persistido del profile por una corrida previa de `infra install` o `infra status`.
- `--public-host` solo funciona cuando el addon declara soporte para el contrato básico de ingress administrado por Core.
