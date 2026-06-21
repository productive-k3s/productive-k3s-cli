#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PK3S_BIN="${PRODUCTIVE_K3S_CLI_BIN:-${ROOT_DIR}/pk3s}"
# shellcheck disable=SC1091
source "${ROOT_DIR}/scripts/release-config.sh"
CATALOG_URLS="${PK3S_CATALOG_URLS:-${PRODUCTIVE_K3S_CATALOG_URL_DEFAULT}}"
PROFILE_NAME="${PK3S_CLI_CATALOG_PROFILE_NAME:-multipass-1-server-2-agents}"
ADDON_NAME="${PK3S_CLI_CATALOG_ADDON_NAME:-nginx}"
CLUSTER_PREFIX="${PK3S_CLI_MULTIPASS_CLUSTER_PREFIX:-productive-k3s-mp}"

fail() {
  printf '[FAIL] %s\n' "$*" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

run_pk3s() {
  local source_mode="remote"
  if [[ -n "${PRODUCTIVE_K3S_CORE_REPO_DIR:-}${PRODUCTIVE_K3S_CORE_REPO_URL:-}${PRODUCTIVE_K3S_CORE_REPO_REF:-}${PRODUCTIVE_K3S_INFRA_REPO_DIR:-}${PRODUCTIVE_K3S_INFRA_REPO_URL:-}${PRODUCTIVE_K3S_INFRA_REPO_REF:-}" ]]; then
    source_mode="local"
  fi
  PK3S_CATALOG_URLS="${CATALOG_URLS}" PRODUCTIVE_K3S_SOURCE="${source_mode}" "${PK3S_BIN}" "$@"
}

run_with_timeout() {
  local seconds="$1"
  shift

  if command -v timeout >/dev/null 2>&1; then
    timeout "${seconds}" "$@"
    return $?
  fi

  "$@" &
  local cmd_pid=$!
  local waited=0
  while kill -0 "${cmd_pid}" >/dev/null 2>&1; do
    if (( waited >= seconds )); then
      kill "${cmd_pid}" >/dev/null 2>&1 || true
      wait "${cmd_pid}" >/dev/null 2>&1 || true
      return 124
    fi
    sleep 1
    waited=$((waited + 1))
  done

  wait "${cmd_pid}"
}

instance_exists() {
  local name="$1"
  multipass info "${name}" >/dev/null 2>&1
}

fallback_cleanup() {
  local instances=(
    "${CLUSTER_PREFIX}-server"
    "${CLUSTER_PREFIX}-agent-1"
    "${CLUSTER_PREFIX}-agent-2"
  )
  local instance

  run_with_timeout 30 multipass delete "${instances[@]}" >/dev/null 2>&1 || true

  for instance in "${instances[@]}"; do
    local attempt=0
    while instance_exists "${instance}" && (( attempt < 30 )); do
      sleep 1
      attempt=$((attempt + 1))
    done
  done

  run_with_timeout 30 multipass purge >/dev/null 2>&1 || true
}

cleanup() {
  run_pk3s infra destroy "${PROFILE_NAME}" >/dev/null 2>&1 || true
  fallback_cleanup
}

need_cmd multipass
need_cmd jq
need_cmd curl
need_cmd tar
need_cmd python3
[[ -x "${PK3S_BIN}" ]] || fail "pk3s binary is not executable: ${PK3S_BIN}"

trap cleanup EXIT
cleanup

run_pk3s profile validate "${PROFILE_NAME}"
run_pk3s infra install "${PROFILE_NAME}"
run_pk3s infra status "${PROFILE_NAME}"
run_pk3s addon validate "${ADDON_NAME}"
run_pk3s addon install "${ADDON_NAME}" --profile "${PROFILE_NAME}"
run_pk3s infra destroy "${PROFILE_NAME}"

trap - EXIT
fallback_cleanup
printf '[PASS] catalog-backed multipass package installation completed\n'
