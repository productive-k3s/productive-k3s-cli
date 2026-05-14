# GitHub Actions And Releases

The CLI repository follows the same broad automation split as the other Productive K3S repositories:

- PR validation checks static wiring, Go tests, CLI contracts, and a GitHub-hosted live on-prem remote validation.
- Documentation is built from MkDocs and deployed to `gh-pages`.
- Release tags produce versioned CLI archives and installer assets.

The documentation site is published with a dedicated workflow and keeps `cli.productive-k3s.io` pinned through the workflow `cname` setting and the tracked `docs/src/CNAME` file.
