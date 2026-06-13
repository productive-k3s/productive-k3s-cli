
#!/usr/bin/env bash
set -euo pipefail
TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ARTIFACTS_DIR="${TEST_ARTIFACTS_DIR:-${TESTS_DIR}/../test-artifacts}"
source "${TESTS_DIR}/lib/json.sh"
ARTIFACT="${ARTIFACTS_DIR}/cli-help-contract.json"
CLI_BIN="${PRODUCTIVE_K3S_CLI_BIN:-${TESTS_DIR}/../pk3s}"
started_at="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

if [[ ! -x "${CLI_BIN}" ]]; then
  write_result "${ARTIFACT}" "cli-help-contract" "pending" "CLI binary not found; expected help command to expose user-facing command groups." "${started_at}" "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
  exit 0
fi

output="$("${CLI_BIN}" help || true)"
for expected in "install" "doctor" "validate" "bundle" "profile" "infra" "addon" "version"; do
  grep -q "${expected}" <<< "${output}"
done

write_result "${ARTIFACT}" "cli-help-contract" "passed" "Help output exposes expected command groups." "${started_at}" "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
