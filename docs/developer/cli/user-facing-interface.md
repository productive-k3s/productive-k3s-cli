# User-facing Interface

The Productive K3S CLI should expose a unified interface while mapping to the existing public interfaces of Core and Infra.

Proposed executable name:

```bash
productive-k3s
```

## Top-level commands

Initial command set:

```bash
productive-k3s version
productive-k3s bundle info
productive-k3s doctor
productive-k3s install
productive-k3s validate
productive-k3s plan
productive-k3s apply
productive-k3s destroy
productive-k3s status
productive-k3s list-profiles
productive-k3s backup
productive-k3s preflight
productive-k3s bootstrap
```

## Profile-driven usage

```bash
productive-k3s install --profile profiles/edge-arm.env
productive-k3s validate --profile profiles/onprem-basic.env
productive-k3s plan --profile profiles/aws-single-node.env
productive-k3s apply --profile profiles/multipass.env
productive-k3s status --profile profiles/multipass.env
```

## Bundle-oriented usage

```bash
productive-k3s bundle info
productive-k3s core bundle info --json
productive-k3s infra bundle info --json
```

## Local development usage

```bash
productive-k3s \
  --core-source ../productive-k3s-core \
  --infra-source ../productive-k3s-infra \
  doctor
```

Local development mode must preserve the same validation contract as release mode, but it may skip remote download and extraction.
