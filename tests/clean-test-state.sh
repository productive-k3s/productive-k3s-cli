#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
ARTIFACTS_DIR="${TEST_ARTIFACTS_DIR:-${REPO_DIR}/test-artifacts}"
TEST_SCOPE="${TEST_SCOPE:-}"

remove_if_exists() {
  local path="$1"
  [[ -e "${path}" ]] || return 0
  rm -rf "${path}"
}

case "${TEST_SCOPE}" in
  ""|all)
    remove_if_exists "${ARTIFACTS_DIR}"
    printf '[INFO] Cleared local test state from %s\n' "${ARTIFACTS_DIR}"
    ;;
  contract)
    remove_if_exists "${ARTIFACTS_DIR}/summary.json"
    find "${ARTIFACTS_DIR}" -maxdepth 1 -type f -name '*-contract.json' -delete 2>/dev/null || true
    printf '[INFO] Cleared contract test state from %s\n' "${ARTIFACTS_DIR}"
    ;;
  live)
    remove_if_exists "${ARTIFACTS_DIR}/live-summary.json"
    find "${ARTIFACTS_DIR}" -maxdepth 1 -type f -name '*-summary.json' ! -name 'summary.json' -delete 2>/dev/null || true
    remove_if_exists "${ARTIFACTS_DIR}/cli-live-runs"
    printf '[INFO] Cleared live test state from %s\n' "${ARTIFACTS_DIR}"
    ;;
  *)
    printf '[ERROR] Unsupported TEST_SCOPE=%s; use contract, live, or all\n' "${TEST_SCOPE}" >&2
    exit 1
    ;;
esac

if [[ -d "${ARTIFACTS_DIR}" ]] && [[ -z "$(find "${ARTIFACTS_DIR}" -mindepth 1 -print -quit)" ]]; then
  rmdir "${ARTIFACTS_DIR}" 2>/dev/null || true
fi
