# Testing

The CLI has two test layers today:

- contract validators under `tests/contracts/`, which stay fast and write JSON artifacts under `test-artifacts/`
- live remote validators under `tests/run-cli-live.sh`, which execute `pk3s` end to end against published Infra bundles

For the live layer, the supported scenarios are:

- `multipass`: uses the published remote Multipass profile URL directly through `pk3s`
- `onprem-basic`: creates two ephemeral Multipass VMs outside the Infra scenario, generates a temporary `.env`, and then exercises the public `pk3s` profile workflow against that file

Typical commands:

```bash
make test-static
make test-live-remote
./tests/run-cli-live.sh multipass
./tests/run-cli-live.sh onprem-basic
```

Live manifests are written under `test-artifacts/cli-live-runs/`, and each invocation also emits:

- `test-artifacts/<run-id>-summary.json`
- `test-artifacts/live-summary.json`

The helper targets match the Infra/Core repos:

```bash
make test-checkstatus
make test-clean
TEST_SCOPE=live make test-checkstatus
TEST_SCOPE=contract make test-clean
```
