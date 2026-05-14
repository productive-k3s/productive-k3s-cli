# Profiles

Infra workflows are profile-driven.

The CLI accepts a profile in two forms:

- an explicit file path;
- a short name that can be resolved inside the local or remote Infra bundle profile directories.

## List available profiles

```bash
pk3s profile list
```

## Validate a profile

```bash
pk3s profile validate --profile profiles/on-prem/basic.env
```

This checks the profile contract only.

## Validate a deployed scenario

```bash
pk3s validate --profile profiles/on-prem/basic.env
```

This delegates to the Infra scenario validation and may require generated cluster state.

## Plan and apply

```bash
pk3s plan --profile profiles/aws-single-node/basic.env
pk3s apply --profile profiles/aws-single-node/basic.env
```

## Notes

- A profile can still fail later if the underlying scenario requirements are not met.
- For example, a Multipass-based profile may require `multipass` on the host. That validation should come from Productive K3S Infra, not from the CLI pretending to know every scenario rule.
