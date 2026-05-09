
#!/usr/bin/env bash
set -euo pipefail
TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${TESTS_DIR}/lib/json.sh"
ARTIFACT="${TESTS_DIR}/artifacts/error-handling-contract.json"
CLI_BIN="${PRODUCTIVE_K3S_CLI_BIN:-${TESTS_DIR}/../productive-k3s}"
started_at="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

if [[ ! -x "${CLI_BIN}" ]]; then
  write_result "${ARTIFACT}" "error-handling-contract" "pending" "CLI binary not found; expected unsupported commands to fail with non-zero exit code and readable error." "${started_at}" "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
  exit 0
fi

set +e
output="$("${CLI_BIN}" unsupported-command 2>&1)"
code=$?
set -e

[[ "${code}" -ne 0 ]]
grep -Ei "unsupported|unknown|invalid" <<< "${output}" >/dev/null

write_result "${ARTIFACT}" "error-handling-contract" "passed" "Unsupported command fails with non-zero status and readable error." "${started_at}" "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
