# Reasons Behind

The CLI exists to solve three concrete problems:

## One command surface

Users should not need to memorize separate command surfaces for Core and Infra. `pk3s` gives them one executable and one help system.

## Release-aware execution

The CLI can resolve published bundles from GitHub Releases and keep Core and Infra versions aligned through an explicit manifest.

## Safer remote-first behavior

For users consuming the project as a product, remote mode is the default because it validates the same published artifacts that others will download.

Local mode still exists, but only as an explicit development choice.
