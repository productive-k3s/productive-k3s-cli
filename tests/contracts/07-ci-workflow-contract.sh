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
grep -F 'name: Set up Go' "${workflow_file}" >/dev/null
grep -E 'uses:[[:space:]]+actions/setup-go@[0-9a-f]{40}[[:space:]]+# actions/setup-go@v5' "${workflow_file}" >/dev/null
grep -F 'go-version-file: productive-k3s-cli/go.mod' "${workflow_file}" >/dev/null
grep -F 'shell: bash' "${workflow_file}" >/dev/null
grep -F 'run: make go-test' "${workflow_file}" >/dev/null
grep -F 'run: make test-cli-contract' "${workflow_file}" >/dev/null
grep -F 'live-onprem-remote-github-host' "${workflow_file}" >/dev/null
grep -F 'run: make test-live-gha-onprem-remote' "${workflow_file}" >/dev/null
grep -F 'path: |' "${workflow_file}" >/dev/null
grep -F 'productive-k3s-cli/test-artifacts' "${workflow_file}" >/dev/null

release_workflow="${ROOT_DIR}/.github/workflows/release.yml"
grep -F "tags:" "${release_workflow}" >/dev/null
grep -F 'git merge-base --is-ancestor "$GITHUB_SHA" "origin/main"' "${release_workflow}" >/dev/null
grep -E 'uses:[[:space:]]+actions/setup-go@[0-9a-f]{40}[[:space:]]+# actions/setup-go@v5' "${release_workflow}" >/dev/null
grep -F 'run: |' "${release_workflow}" >/dev/null
grep -F 'make build-release' "${release_workflow}" >/dev/null
grep -F 'sha256sum pk3s-${GITHUB_REF_NAME}-* > checksums.txt' "${release_workflow}" >/dev/null
grep -F 'gh release create "${GITHUB_REF_NAME}"' "${release_workflow}" >/dev/null

write_result "${ARTIFACT}" "ci-workflow-contract" "passed" "CI validates contracts and remote live behavior; release tags on main publish checksum-backed multiplatform archives." "${started_at}" "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
