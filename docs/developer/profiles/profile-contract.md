# Profile Contract

Profiles are environment files consumed by Productive K3S Infra.

They are passed to commands using:

```bash
--profile <file>
```

## Required common variables

Every profile must define:

```bash
PK3S_INFRA_PROFILE_NAME=
PK3S_INFRA_ENGINE=
PK3S_INFRA_SCENARIO=
```

## Supported engines

```text
opentofu
ansible
shell
```

## Supported scenarios

```text
multipass
onprem-basic
aws-single-node
```

The Infra CLI accepts aliases for some scenarios, such as `onprem` and `on-prem`, but internally resolves them to `onprem-basic`.

## Multipass profiles

Multipass profiles must use:

```bash
PK3S_INFRA_ENGINE=opentofu
PK3S_INFRA_SCENARIO=multipass
```

Typical required variables include cluster name, base image, base domain, remote directory, server resources, and agent resources.

## On-prem profiles

On-prem profiles must use:

```bash
PK3S_INFRA_SCENARIO=onprem-basic
```

Supported engines are:

```text
ansible
shell
```

Required variables include server IP, SSH user, and SSH key path.

## AWS single-node profiles

AWS single-node profiles must use:

```bash
PK3S_INFRA_ENGINE=opentofu
PK3S_INFRA_SCENARIO=aws-single-node
```

Required variables include AWS region, cluster name, instance type, SSH user, SSH key path, and root volume size.
