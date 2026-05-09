# Install Flow

The `install` command is the main high-level workflow exposed by the unified CLI.

## Proposed command

```bash
productive-k3s install --profile <file>
```

## Normal flow

```text
1. Load CLI release metadata.
2. Resolve bound Core and Infra bundle versions.
3. Validate that Infra version suffix matches the selected Core version.
4. Download or locate bundles.
5. Read Core bundle info.
6. Read Infra bundle info.
7. Validate both bundle contracts.
8. Run Core preflight.
9. Run Infra doctor with profile.
10. Run Infra validate with profile.
11. Run Infra apply with profile.
12. Run Core validate.
13. Print final summary.
```

## Dry-run flow

```bash
productive-k3s install --profile <file> --dry-run
```

```text
1. Load CLI release metadata.
2. Resolve bound Core and Infra bundle versions.
3. Validate that Infra version suffix matches the selected Core version.
4. Download or locate bundles.
5. Read Core bundle info.
6. Read Infra bundle info.
7. Validate both bundle contracts.
8. Run Core preflight.
9. Run Infra doctor with profile.
10. Run Infra validate with profile.
11. Run Infra plan with profile.
12. Skip mutating actions.
13. Print final summary.
```

## Error handling

The CLI should fail early when:

- bundle metadata cannot be read;
- bundle contract validation fails;
- the Infra bundle version suffix does not match the selected Core bundle version;
- a required profile is missing;
- a profile is invalid;
- a delegated command returns a non-zero exit code.
