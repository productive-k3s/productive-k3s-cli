#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PK3S_BIN="${PRODUCTIVE_K3S_CLI_BIN:-${ROOT_DIR}/pk3s}"
# shellcheck disable=SC1091
source "${ROOT_DIR}/scripts/release-config.sh"
PROFILE_URL="${PK3S_CLI_MULTIPASS_PROFILE_URL:-${PRODUCTIVE_K3S_PROFILES_MULTIPASS_PROFILE_URL_DEFAULT}}"
CLUSTER_PREFIX="${PK3S_CLI_MULTIPASS_CLUSTER_PREFIX:-productive-k3s-mp}"
WORK_DIR="$(mktemp -d "${ROOT_DIR}/.live-cli-multipass.XXXXXX")"
PROFILES_REPO_DIR=""

fail() {
  printf '[FAIL] %s\n' "$*" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

prepare_profiles_repo_dir() {
  [[ -n "${PRODUCTIVE_K3S_PROFILES_REPO_DIR:-}" ]] && return 0
  [[ -n "${PROFILES_REPO_DIR}" ]] && return 0

  local profiles_repo_url="${PRODUCTIVE_K3S_PROFILES_REPO_URL:-${PRODUCTIVE_K3S_PROFILES_GIT_REMOTE_URL_DEFAULT}}"
  local profiles_repo_ref="${PRODUCTIVE_K3S_PROFILES_REPO_REF:-${PRODUCTIVE_K3S_INFRA_REPO_REF:-development}}"
  PROFILES_REPO_DIR="${WORK_DIR}/productive-k3s-profiles"
  git clone --depth 1 --branch "${profiles_repo_ref}" "${profiles_repo_url}" "${PROFILES_REPO_DIR}" >/dev/null 2>&1 || {
    fail "could not clone productive-k3s-profiles from ${profiles_repo_url} (${profiles_repo_ref})"
  }
}

run_pk3s() {
  local source_mode="remote"
  if [[ -n "${PRODUCTIVE_K3S_CORE_REPO_DIR:-}${PRODUCTIVE_K3S_CORE_REPO_URL:-}${PRODUCTIVE_K3S_CORE_REPO_REF:-}${PRODUCTIVE_K3S_INFRA_REPO_DIR:-}${PRODUCTIVE_K3S_INFRA_REPO_URL:-}${PRODUCTIVE_K3S_INFRA_REPO_REF:-}" ]]; then
    source_mode="local"
  fi
  if [[ "${source_mode}" == "local" ]]; then
    prepare_profiles_repo_dir
  fi
  PRODUCTIVE_K3S_SOURCE="${source_mode}" \
    PRODUCTIVE_K3S_PROFILES_REPO_DIR="${PRODUCTIVE_K3S_PROFILES_REPO_DIR:-${PROFILES_REPO_DIR}}" \
    "${PK3S_BIN}" "$@"
}

fallback_cleanup() {
  multipass delete \
    "${CLUSTER_PREFIX}-server" \
    "${CLUSTER_PREFIX}-agent-1" \
    "${CLUSTER_PREFIX}-agent-2" >/dev/null 2>&1 || true
  multipass purge >/dev/null 2>&1 || true
}

cleanup() {
  run_pk3s destroy --profile "${PROFILE_URL}" >/dev/null 2>&1 || true
  fallback_cleanup
  rm -rf "${WORK_DIR}"
}

need_cmd multipass
need_cmd jq
need_cmd curl
need_cmd tar
need_cmd python3
need_cmd git
[[ -x "${PK3S_BIN}" ]] || fail "pk3s binary is not executable: ${PK3S_BIN}"

trap cleanup EXIT
cleanup

run_pk3s profile validate --profile "${PROFILE_URL}"
run_pk3s plan --profile "${PROFILE_URL}"
run_pk3s apply --profile "${PROFILE_URL}"
run_pk3s status --profile "${PROFILE_URL}"
run_pk3s destroy --profile "${PROFILE_URL}"

trap - EXIT
fallback_cleanup
printf '[PASS] multipass remote CLI validation completed\n'
