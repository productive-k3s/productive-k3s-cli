#!/usr/bin/env bash
set -euo pipefail
TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ROOT_DIR="$(cd "${TESTS_DIR}/.." && pwd)"
ARTIFACTS_DIR="${TEST_ARTIFACTS_DIR:-${ROOT_DIR}/test-artifacts}"
source "${TESTS_DIR}/lib/json.sh"
ARTIFACT="${ARTIFACTS_DIR}/ci-workflow-contract.json"
started_at="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

make_recipe="$(cd "${ROOT_DIR}" && make -n test-cli-contract)"
printf '%s\n' "${make_recipe}" | grep -F 'bash ./tests/run-cli-contracts.sh' >/dev/null

workflow_file="${ROOT_DIR}/.github/workflows/cli-contracts.yml"
grep -F 'name: Run Go tests' "${workflow_file}" >/dev/null
grep -F 'shell: bash' "${workflow_file}" >/dev/null
grep -F 'run: make go-test' "${workflow_file}" >/dev/null
grep -F 'run: make test-cli-contract' "${workflow_file}" >/dev/null
grep -F 'live-onprem-remote-github-host' "${workflow_file}" >/dev/null
grep -F 'run: make test-live-gha-onprem-remote' "${workflow_file}" >/dev/null
grep -F 'path: |' "${workflow_file}" >/dev/null
grep -F 'productive-k3s-cli/test-artifacts' "${workflow_file}" >/dev/null

write_result "${ARTIFACT}" "ci-workflow-contract" "passed" "CI workflow uses bash explicitly, validates Go tests before CLI contracts, and includes the GitHub-host remote live validator." "${started_at}" "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
