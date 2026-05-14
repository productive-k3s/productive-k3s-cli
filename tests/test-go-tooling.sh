#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

assert_not_contains() {
  local path="$1"
  local needle="$2"
  if grep -Fq "${needle}" "${path}"; then
    printf '[FAIL] did not expect %s to contain: %s\n' "${path}" "${needle}" >&2
    exit 1
  fi
}

assert_contains() {
  local path="$1"
  local needle="$2"
  if ! grep -Fq "${needle}" "${path}"; then
    printf '[FAIL] expected %s to contain: %s\n' "${path}" "${needle}" >&2
    exit 1
  fi
}

assert_not_contains "${ROOT_DIR}/Makefile" "/usr/local/go/bin/go"
assert_not_contains "${ROOT_DIR}/scripts/productive-k3s-cli-dev.sh" "/usr/local/go/bin/go"
assert_not_contains "${ROOT_DIR}/scripts/build-cli.sh" "/usr/local/go/bin/go"

assert_contains "${ROOT_DIR}/Makefile" '$(GO_BIN) test ./...'
assert_contains "${ROOT_DIR}/scripts/productive-k3s-cli-dev.sh" 'GO_BIN="${GO_BIN:-$(command -v go || true)}"'
assert_contains "${ROOT_DIR}/scripts/build-cli.sh" 'GO_BIN="${GO_BIN:-$(command -v go || true)}"'

printf '[PASS] go tooling resolves go from PATH instead of a hardcoded absolute path\n'
