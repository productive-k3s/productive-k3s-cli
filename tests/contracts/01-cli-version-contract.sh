
#!/usr/bin/env bash
set -euo pipefail
TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ARTIFACTS_DIR="${TEST_ARTIFACTS_DIR:-${TESTS_DIR}/../test-artifacts}"
source "${TESTS_DIR}/lib/json.sh"
ARTIFACT="${ARTIFACTS_DIR}/cli-version-contract.json"
CLI_BIN="${PRODUCTIVE_K3S_CLI_BIN:-${TESTS_DIR}/../pk3s}"
started_at="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

if [[ ! -x "${CLI_BIN}" ]]; then
  write_result "${ARTIFACT}" "cli-version-contract" "pending" "CLI binary not found; expected version output to use semver without v prefix." "${started_at}" "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
  exit 0
fi

version="$("${CLI_BIN}" version)"
[[ "${version}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]

write_result "${ARTIFACT}" "cli-version-contract" "passed" "CLI version follows semver without v prefix." "${started_at}" "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
