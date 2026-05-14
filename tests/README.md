
# Productive K3S CLI test contracts

This directory defines the initial TDD-oriented test contracts for the Productive K3S CLI.

The tests validate the CLI as the user-facing orchestration layer over:

- Productive K3S Core bundles
- Productive K3S Infra bundles
- profile-based workflows
- bundle compatibility metadata
- CI/CD-friendly JSON artifacts

The tests focus on CLI behavior and contracts, not on real cluster creation.

Live validators exist separately for remote-mode end-to-end coverage:

- `tests/live-cli-multipass-remote.sh`
- `tests/live-cli-onprem-remote.sh`

These scripts do create short-lived infrastructure:

- `multipass`: uses the published remote `productive-k3s-infra` bundle through `pk3s`
- `onprem-basic`: creates two ephemeral Multipass VMs outside the scenario, writes an `.env`, then drives `pk3s` against that profile

## Output artifacts

Every contract test writes JSON artifacts under:

```text
test-artifacts/
```

Expected generated files:

```text
test-artifacts/
├── cli-help-contract.json
├── cli-version-contract.json
├── bundle-resolution-contract.json
├── core-command-mapping-contract.json
├── infra-command-mapping-contract.json
├── profile-command-contract.json
├── error-handling-contract.json
└── summary.json
```

These files can be uploaded by GitHub Actions as workflow artifacts.

Live runs write separate manifests:

```text
test-artifacts/
├── cli-live-runs/
│   ├── <run-id>-multipass.json
│   ├── <run-id>-multipass.log
│   ├── <run-id>-onprem-basic.json
│   └── <run-id>-onprem-basic.log
├── <run-id>-summary.json
└── live-summary.json
```

## Running locally

```bash
make test-cli-contract
```

or:

```bash
./tests/run-cli-contracts.sh
```

Run the live remote validators:

```bash
make test-live-remote
```

Run only one live validator:

```bash
./tests/run-cli-live.sh multipass
./tests/run-cli-live.sh onprem-basic
```

GitHub-host PR validator:

```bash
make test-live-gha-onprem-remote
```

Check current artifact status:

```bash
make test-checkstatus
TEST_SCOPE=contract make test-checkstatus
TEST_SCOPE=live make test-checkstatus
```

Clear local test artifact state:

```bash
make test-clean
TEST_SCOPE=contract make test-clean
TEST_SCOPE=live make test-clean
```

## CLI binary resolution

By default the tests expect:

```bash
./pk3s
```

Override with:

```bash
PRODUCTIVE_K3S_CLI_BIN=/path/to/pk3s ./tests/run-cli-contracts.sh
```

If the binary is not available, tests run in contract-definition mode and emit `pending` JSON artifacts.
This makes the package useful before implementation starts.
