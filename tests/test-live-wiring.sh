#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

assert_contains() {
  local path="$1"
  local needle="$2"
  if ! grep -Fq "${needle}" "${path}"; then
    printf '[FAIL] expected %s to contain: %s\n' "${path}" "${needle}" >&2
    exit 1
  fi
}

assert_executable() {
  local path="$1"
  if [[ ! -x "${path}" ]]; then
    printf '[FAIL] expected executable file: %s\n' "${path}" >&2
    exit 1
  fi
}

assert_contains "${ROOT_DIR}/scripts/productive-k3s-cli-dev.sh" "test-live-remote"
assert_contains "${ROOT_DIR}/scripts/productive-k3s-cli-dev.sh" "test-live-gha-onprem-remote"
assert_contains "${ROOT_DIR}/scripts/productive-k3s-cli-dev.sh" "test-clean"
assert_contains "${ROOT_DIR}/scripts/productive-k3s-cli-dev.sh" "test-checkstatus"
assert_contains "${ROOT_DIR}/Makefile" "test-live-remote"
assert_contains "${ROOT_DIR}/Makefile" "test-live-gha-onprem-remote"
assert_contains "${ROOT_DIR}/Makefile" "test-clean"
assert_contains "${ROOT_DIR}/Makefile" "test-checkstatus"
assert_contains "${ROOT_DIR}/tests/run-cli-live.sh" "multipass"
assert_contains "${ROOT_DIR}/tests/run-cli-live.sh" "onprem-basic"

assert_executable "${ROOT_DIR}/tests/run-cli-live.sh"
assert_executable "${ROOT_DIR}/tests/live-cli-multipass-remote.sh"
assert_executable "${ROOT_DIR}/tests/live-cli-onprem-remote.sh"
assert_executable "${ROOT_DIR}/tests/live-cli-onprem-remote-github-host.sh"
assert_executable "${ROOT_DIR}/tests/summarize-live-artifacts.py"
assert_executable "${ROOT_DIR}/tests/check-test-status.sh"
assert_executable "${ROOT_DIR}/tests/clean-test-state.sh"

printf '[PASS] live CLI validator wiring is present\n'
