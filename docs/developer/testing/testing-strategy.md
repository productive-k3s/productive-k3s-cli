# Testing Strategy

The CLI should be tested as a dispatcher, resolver, and compatibility gate.

## Test layers

### Static tests

Validate formatting, linting, and basic repository hygiene.

### Contract tests

Validate:

- bundle metadata parsing;
- Core semantic version parsing;
- Infra composed version parsing using `X.Y.Z-A.B.C`;
- Infra suffix compatibility with the selected Core version;
- command mapping;
- error handling;
- local source behavior;
- JSON output stability.

### Smoke tests

Validate that the CLI can:

- resolve local bundle paths;
- call Core `bundle info --json`;
- call Infra `bundle info --json`;
- call help/version commands;
- run dry-run workflows without mutating infrastructure.

### Integration tests

Validate selected workflows against real or simulated environments.

Examples:

```bash
productive-k3s install --profile profiles/multipass.env --dry-run
productive-k3s validate --profile profiles/onprem-basic.env
productive-k3s plan --profile profiles/aws-single-node.env
```

## Documentation tests

The repository should expose documentation checks through Make targets:

```bash
make docs-build
make test-static
make test-contract
make test-productive-k3s-cli
```
