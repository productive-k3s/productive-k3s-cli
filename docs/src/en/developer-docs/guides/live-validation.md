# Live Remote Validation

`pk3s` now has a small live validation layer focused on the public remote workflow.

The goal is narrow: prove that the CLI can resolve published remote bundles, accept either a remote profile URL or a generated local `.env`, and drive the corresponding `productive-k3s-infra` release end to end.

## Entry points

Use the aggregate targets:

```bash
make test-live-remote
make test-live-catalog
```

Or run a single validator:

```bash
make -C tests test-live-remote SCENARIOS=multipass
make -C tests test-live-remote SCENARIOS=onprem-basic
```

## Covered scenarios

### `multipass`

The validator:

1. forces `PRODUCTIVE_K3S_SOURCE=remote`
2. validates the published Multipass profile URL through `pk3s profile validate`
3. runs `pk3s plan`
4. runs `pk3s apply`
5. runs `pk3s status`
6. runs `pk3s destroy`

This is the closest CLI-level proof that the remote published Infra bundle is usable for the built-in Multipass profile.

### `catalog-multipass`

The validator:

1. forces catalog-backed resolution
2. validates the published Multipass profile package through `pk3s profile validate`
3. runs `pk3s infra install`
4. runs `pk3s infra status`
5. runs `pk3s addon install`
6. runs `pk3s infra destroy`

This proves the catalog-backed package UX against a real local Multipass target.

### `onprem-basic`

The validator is intentionally external to the scenario:

1. create two ephemeral Multipass VMs
2. wait for SSH reachability
3. generate a temporary `onprem-basic` `.env`
4. force `PRODUCTIVE_K3S_SOURCE=remote`
5. run `pk3s profile validate`
6. run `pk3s plan`
7. run `pk3s apply`
8. run `pk3s status`
9. run `pk3s validate`
10. delete the ephemeral VMs

This validates the CLI path for a remote-style profile without touching the local host.

## Artifacts

Every live invocation writes:

- one manifest per scenario under `test-artifacts/cli-live-runs/`
- one log per scenario under `test-artifacts/cli-live-runs/`
- one root summary under `test-artifacts/<run-id>-summary.json`
- one convenience copy at `test-artifacts/live-summary.json`

The manifests include:

- scenario name
- pass/skip/fail result
- timestamps and duration
- expected topology
- CLI binary path
- configured Core and Infra bundle versions

## Notes

- These validators are intentionally separate from `test-local-all`.
- They also require a working local `Go` toolchain because the live targets build `pk3s` before running the validators.
- They require real dependencies such as `multipass`, `ssh`, `curl`, `tar`, and `python3`.
- `onprem-basic` does not use `pk3s destroy`, because that scenario does not expose a public destroy contract. Cleanup is done by deleting the temporary VMs.
- Use `make -C tests test-checkstatus` to inspect recorded CLI artifacts.
- Use `make -C tests test-clean` to remove local CLI test artifact state.
