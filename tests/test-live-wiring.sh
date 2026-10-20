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

assert_not_contains() {
  local path="$1"
  local needle="$2"
  if grep -Fq "${needle}" "${path}"; then
    printf '[FAIL] expected %s not to contain: %s\n' "${path}" "${needle}" >&2
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
assert_contains "${ROOT_DIR}/scripts/productive-k3s-cli-dev.sh" "test-live-catalog"
assert_contains "${ROOT_DIR}/scripts/productive-k3s-cli-dev.sh" "test-live-export"
assert_contains "${ROOT_DIR}/scripts/productive-k3s-cli-dev.sh" "test-live-gha-onprem-remote"
assert_contains "${ROOT_DIR}/scripts/productive-k3s-cli-dev.sh" "test-clean"
assert_contains "${ROOT_DIR}/scripts/productive-k3s-cli-dev.sh" "test-checkstatus"
assert_contains "${ROOT_DIR}/tests/live-cli-catalog-multipass.sh" 'TOFU_APPLY_PARALLELISM="${TOFU_APPLY_PARALLELISM:-1}"'
assert_contains "${ROOT_DIR}/tests/live-cli-catalog-multipass.sh" 'MULTIPASS_LAUNCH_MAX_ATTEMPTS="${MULTIPASS_LAUNCH_MAX_ATTEMPTS:-5}"'
assert_contains "${ROOT_DIR}/tests/live-cli-catalog-multipass.sh" 'run_with_timeout 30 multipass delete "${instance}"'
assert_not_contains "${ROOT_DIR}/tests/live-cli-catalog-multipass.sh" 'multipass delete "${instances[@]}"'
assert_contains "${ROOT_DIR}/Makefile" "test-live-remote"
assert_contains "${ROOT_DIR}/Makefile" "test-live-catalog"
assert_contains "${ROOT_DIR}/Makefile" "test-live-export"
assert_contains "${ROOT_DIR}/tests/Makefile" "test-live-gha-onprem-remote"
assert_contains "${ROOT_DIR}/tests/Makefile" "test-live-export"
assert_contains "${ROOT_DIR}/tests/Makefile" "test-clean"
assert_contains "${ROOT_DIR}/tests/Makefile" "test-checkstatus"
assert_contains "${ROOT_DIR}/tests/run-cli-live.sh" "multipass"
assert_contains "${ROOT_DIR}/tests/run-cli-live.sh" "catalog-multipass"
assert_contains "${ROOT_DIR}/tests/run-cli-live.sh" "stack-export-ubuntu24"
assert_contains "${ROOT_DIR}/tests/run-cli-live.sh" "profile-export-multipass"
assert_contains "${ROOT_DIR}/tests/run-cli-live.sh" "onprem-basic"
assert_contains "${ROOT_DIR}/tests/lib/source-compatibility.sh" "PRODUCTIVE_K3S_SOURCE_CORE_VERSION"
assert_contains "${ROOT_DIR}/tests/lib/source-compatibility.sh" "PRODUCTIVE_K3S_SOURCE_INFRA_VERSION"
assert_contains "${ROOT_DIR}/tests/live-cli-multipass-remote.sh" "pk3s_export_source_compatibility"
assert_contains "${ROOT_DIR}/tests/live-cli-onprem-remote.sh" "pk3s_export_source_compatibility"
assert_contains "${ROOT_DIR}/tests/live-cli-onprem-remote-github-host.sh" "pk3s_export_source_compatibility"
assert_contains "${ROOT_DIR}/tests/live-cli-onprem-remote-github-host.sh" 'profiles/edge/on-prem/basic.env'
assert_contains "${ROOT_DIR}/tests/live-cli-onprem-remote-github-host.sh" 'cp "${canonical_profile}" "${ENV_FILE}"'
assert_not_contains "${ROOT_DIR}/tests/live-cli-onprem-remote-github-host.sh" 'PK3S_INFRA_PROFILE_NAME=pk3s-cli-gha-onprem-remote'
assert_not_contains "${ROOT_DIR}/tests/live-cli-onprem-remote-github-host.sh" 'PK3S_INFRA_SCENARIO=on-prem'
assert_contains "${ROOT_DIR}/tests/live-cli-onprem-remote.sh" 'profiles/edge/on-prem/basic.env'
assert_contains "${ROOT_DIR}/tests/live-cli-onprem-remote.sh" 'cp "${canonical_profile}" "${ENV_FILE}"'
assert_not_contains "${ROOT_DIR}/tests/live-cli-onprem-remote.sh" 'PK3S_INFRA_PROFILE_NAME=pk3s-cli-onprem-remote'
assert_not_contains "${ROOT_DIR}/tests/live-cli-onprem-remote.sh" 'PK3S_INFRA_SCENARIO=on-prem'

ansible_check_line="$(grep -nF 'need_cmd ansible-playbook' "${ROOT_DIR}/tests/live-cli-onprem-remote.sh" | cut -d: -f1)"
first_launch_line="$(grep -nF 'launch_instance "${SERVER_NAME}"' "${ROOT_DIR}/tests/live-cli-onprem-remote.sh" | cut -d: -f1)"
[[ -n "${ansible_check_line}" && -n "${first_launch_line}" && "${ansible_check_line}" -lt "${first_launch_line}" ]] \
  || fail "onprem live validation must require ansible-playbook before launching VMs"

assert_executable "${ROOT_DIR}/tests/run-cli-live.sh"
assert_executable "${ROOT_DIR}/tests/live-cli-catalog-multipass.sh"
assert_executable "${ROOT_DIR}/tests/live-cli-multipass-remote.sh"
assert_executable "${ROOT_DIR}/tests/live-cli-stack-export-ubuntu24.sh"
assert_executable "${ROOT_DIR}/tests/live-cli-profile-export-multipass.sh"
assert_executable "${ROOT_DIR}/tests/live-cli-onprem-remote.sh"
assert_executable "${ROOT_DIR}/tests/live-cli-onprem-remote-github-host.sh"
assert_executable "${ROOT_DIR}/tests/summarize-live-artifacts.py"
assert_executable "${ROOT_DIR}/tests/check-test-status.sh"
assert_executable "${ROOT_DIR}/tests/clean-test-state.sh"

printf '[PASS] live CLI validator wiring is present\n'
