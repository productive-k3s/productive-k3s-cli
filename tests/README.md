
# Productive K3S CLI test contracts

This directory defines the initial TDD-oriented test contracts for the Productive K3S CLI.

The tests validate the CLI as the user-facing orchestration layer over:

- Productive K3S Core bundles
- Productive K3S Infra bundles
- profile-based workflows
- bundle compatibility metadata
- CI/CD-friendly JSON artifacts

The tests focus on CLI behavior and contracts, not on real cluster creation.

## Output artifacts

Every contract test writes JSON artifacts under:

```text
tests/artifacts/
```

Expected generated files:

```text
tests/artifacts/
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

## Running locally

```bash
make test-cli-contract
```

or:

```bash
./tests/run-cli-contracts.sh
```

## CLI binary resolution

By default the tests expect:

```bash
./productive-k3s
```

Override with:

```bash
PRODUCTIVE_K3S_CLI_BIN=/path/to/productive-k3s ./tests/run-cli-contracts.sh
```

If the binary is not available, tests run in contract-definition mode and emit `pending` JSON artifacts.
This makes the package useful before implementation starts.
