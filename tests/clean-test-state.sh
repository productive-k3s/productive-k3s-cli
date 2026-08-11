#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
ARTIFACTS_DIR="${TEST_ARTIFACTS_DIR:-${REPO_DIR}/test-artifacts}"
TEST_SCOPE="${TEST_SCOPE:-}"
CLUSTER_PREFIX="${PK3S_CLI_MULTIPASS_CLUSTER_PREFIX:-productive-k3s-mp}"

cleanup_live_workdirs() {
  find "${REPO_DIR}" -maxdepth 1 -mindepth 1 -type d \
    \( -name '.live-cli-multipass.*' -o -name '.live-cli-onprem-remote.*' \) \
    -exec rm -rf {} + 2>/dev/null || true
}

cleanup_multipass_instances() {
  command -v multipass >/dev/null 2>&1 || return 0

  local instances=()
  local name
  while IFS= read -r name; do
    [[ -n "${name}" ]] || continue
    case "${name}" in
      "${CLUSTER_PREFIX}-server"|\
      "${CLUSTER_PREFIX}-agent-1"|\
      "${CLUSTER_PREFIX}-agent-2"|\
      pk3s-cli-onprem-server-*|\
      pk3s-cli-onprem-agent-*)
        instances+=("${name}")
        ;;
    esac
  done < <(multipass list --format csv 2>/dev/null | tail -n +2 | cut -d, -f1)

  if ((${#instances[@]} == 0)); then
    return 0
  fi

  multipass delete "${instances[@]}" >/dev/null 2>&1 || true
  multipass purge >/dev/null 2>&1 || true
  printf '[INFO] Cleared multipass test instances: %s\n' "${instances[*]}"
}

remove_if_exists() {
  local path="$1"
  [[ -e "${path}" ]] || return 0
  rm -rf "${path}"
}

case "${TEST_SCOPE}" in
  ""|all)
    remove_if_exists "${ARTIFACTS_DIR}"
    cleanup_live_workdirs
    cleanup_multipass_instances
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
    cleanup_live_workdirs
    cleanup_multipass_instances
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
