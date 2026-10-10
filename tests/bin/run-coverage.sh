#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/helpers/test-common.sh"

need_cmd "${GO_BIN}"

cd "${REPO_DIR}"
mkdir -p "${COVERAGE_DIR}"
"${GO_BIN}" test ./... -coverprofile="${COVERAGE_DIR}/coverage.out"
"${GO_BIN}" tool cover -func="${COVERAGE_DIR}/coverage.out" | tee "${COVERAGE_DIR}/coverage.txt"

coverage="$(awk '/^total:/ {gsub(/%/, "", $3); print $3}' "${COVERAGE_DIR}/coverage.txt")"
minimum="${PK3S_COVERAGE_MIN:-80}"
printf 'Coverage: %s%%; required: %s%%\n' "${coverage}" "${minimum}"
awk -v actual="${coverage}" -v required="${minimum}" 'BEGIN { exit actual + 0 >= required + 0 ? 0 : 1 }' || {
  printf 'Coverage %s%% is below required %s%%\n' "${coverage}" "${minimum}" >&2
  exit 1
}
