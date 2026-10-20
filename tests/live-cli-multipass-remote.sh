#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PK3S_BIN="${PRODUCTIVE_K3S_CLI_BIN:-${ROOT_DIR}/pk3s}"
# shellcheck disable=SC1091
source "${ROOT_DIR}/scripts/release-config.sh"
# shellcheck disable=SC1091
source "${ROOT_DIR}/tests/lib/source-compatibility.sh"
# shellcheck disable=SC1091
source "${ROOT_DIR}/tests/lib/git-clone.sh"
PROFILE_NAME="${PK3S_CLI_MULTIPASS_PROFILE_NAME:-multipass-1-server-2-agents}"
CLUSTER_PREFIX="${PK3S_CLI_MULTIPASS_CLUSTER_PREFIX:-productive-k3s-mp}"
WORK_DIR="$(mktemp -d "${ROOT_DIR}/.live-cli-multipass.XXXXXX")"
PROFILES_REPO_DIR=""
INFRA_REPO_DIR_LOCAL=""
CORE_REPO_DIR_LOCAL=""
ADDONS_REPO_DIR_LOCAL=""
MULTIPASS_LAUNCH_MAX_ATTEMPTS="${MULTIPASS_LAUNCH_MAX_ATTEMPTS:-5}"
MULTIPASS_LAUNCH_RETRY_DELAY_SECONDS="${MULTIPASS_LAUNCH_RETRY_DELAY_SECONDS:-5}"
TOFU_APPLY_PARALLELISM="${TOFU_APPLY_PARALLELISM:-1}"

fail() {
  printf '[FAIL] %s\n' "$*" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

prepare_profiles_repo_dir() {
  [[ -n "${PROFILES_REPO_DIR}" ]] && return 0
  local profiles_source_dir="${PRODUCTIVE_K3S_PROFILES_REPO_DIR:-}"
  local profiles_repo_url="${PRODUCTIVE_K3S_PROFILES_REPO_URL:-${PRODUCTIVE_K3S_PROFILES_GIT_REMOTE_URL_DEFAULT}}"
  local profiles_repo_ref="${PRODUCTIVE_K3S_PROFILES_REPO_REF:-${PRODUCTIVE_K3S_INFRA_REPO_REF:-development}}"
  local infra_repo_dir=""

  prepare_infra_repo_dir
  infra_repo_dir="${PRODUCTIVE_K3S_INFRA_REPO_DIR:-${INFRA_REPO_DIR_LOCAL}}"
  PROFILES_REPO_DIR="${WORK_DIR}/productive-k3s-profiles"

  if [[ -n "${profiles_source_dir}" ]]; then
    [[ -d "${profiles_source_dir}/profiles" && -d "${profiles_source_dir}/scenarios" ]] || {
      fail "invalid PRODUCTIVE_K3S_PROFILES_REPO_DIR: ${profiles_source_dir}"
    }
    mkdir -p "${PROFILES_REPO_DIR}"
    cp -a "${profiles_source_dir}/." "${PROFILES_REPO_DIR}/"
  else
    pk3s_git_clone_with_retry "${profiles_repo_url}" "${profiles_repo_ref}" "${PROFILES_REPO_DIR}" productive-k3s-profiles || {
      fail "could not clone productive-k3s-profiles from ${profiles_repo_url} (${profiles_repo_ref})"
    }
  fi

  mkdir -p "${PROFILES_REPO_DIR}/ansible" "${PROFILES_REPO_DIR}/scripts" "${PROFILES_REPO_DIR}/tests"
  cp -a "${infra_repo_dir}/ansible/." "${PROFILES_REPO_DIR}/ansible/"
  cp -a "${infra_repo_dir}/scripts/." "${PROFILES_REPO_DIR}/scripts/"
  cp -a "${infra_repo_dir}/tests/." "${PROFILES_REPO_DIR}/tests/"
}

prepare_infra_repo_dir() {
  [[ -n "${PRODUCTIVE_K3S_INFRA_REPO_DIR:-}" ]] && return 0
  [[ -n "${INFRA_REPO_DIR_LOCAL}" ]] && return 0

  local infra_repo_url="${PRODUCTIVE_K3S_INFRA_REPO_URL:-${PRODUCTIVE_K3S_INFRA_GIT_REMOTE_URL_DEFAULT}}"
  local infra_repo_ref="${PRODUCTIVE_K3S_INFRA_REPO_REF:-development}"
  INFRA_REPO_DIR_LOCAL="${WORK_DIR}/productive-k3s-infra"
  pk3s_git_clone_with_retry "${infra_repo_url}" "${infra_repo_ref}" "${INFRA_REPO_DIR_LOCAL}" productive-k3s-infra || {
    fail "could not clone productive-k3s-infra from ${infra_repo_url} (${infra_repo_ref})"
  }
}

prepare_core_repo_dir() {
  [[ -n "${PRODUCTIVE_K3S_REPO:-}" ]] && return 0
  [[ -n "${CORE_REPO_DIR_LOCAL}" ]] && return 0

  local core_repo_url="${PRODUCTIVE_K3S_CORE_REPO_URL:-${PRODUCTIVE_K3S_CORE_GIT_REMOTE_URL_DEFAULT}}"
  local core_repo_ref="${PRODUCTIVE_K3S_CORE_REPO_REF:-development}"
  CORE_REPO_DIR_LOCAL="${WORK_DIR}/productive-k3s-core"
  pk3s_git_clone_with_retry "${core_repo_url}" "${core_repo_ref}" "${CORE_REPO_DIR_LOCAL}" productive-k3s-core || {
    fail "could not clone productive-k3s-core from ${core_repo_url} (${core_repo_ref})"
  }
}

prepare_addons_repo_dir() {
  [[ -n "${PRODUCTIVE_K3S_ADDONS_REPO_DIR:-}" ]] && return 0
  [[ -n "${ADDONS_REPO_DIR_LOCAL}" ]] && return 0

  local addons_repo_url="${PRODUCTIVE_K3S_ADDONS_REPO_URL:-${PRODUCTIVE_K3S_ADDONS_GIT_REMOTE_URL_DEFAULT}}"
  local addons_repo_ref="${PRODUCTIVE_K3S_ADDONS_REPO_REF:-${PRODUCTIVE_K3S_CORE_REPO_REF:-development}}"
  ADDONS_REPO_DIR_LOCAL="${WORK_DIR}/productive-k3s-addons"
  pk3s_git_clone_with_retry "${addons_repo_url}" "${addons_repo_ref}" "${ADDONS_REPO_DIR_LOCAL}" productive-k3s-addons || {
    fail "could not clone productive-k3s-addons from ${addons_repo_url} (${addons_repo_ref})"
  }
}

run_pk3s() {
  local source_mode="remote"
  if [[ -n "${PRODUCTIVE_K3S_CORE_REPO_DIR:-}${PRODUCTIVE_K3S_CORE_REPO_URL:-}${PRODUCTIVE_K3S_CORE_REPO_REF:-}${PRODUCTIVE_K3S_INFRA_REPO_DIR:-}${PRODUCTIVE_K3S_INFRA_REPO_URL:-}${PRODUCTIVE_K3S_INFRA_REPO_REF:-}" ]]; then
    source_mode="local"
  fi
  if [[ "${source_mode}" == "local" ]]; then
    prepare_profiles_repo_dir
    prepare_core_repo_dir
    prepare_addons_repo_dir
    pk3s_export_source_compatibility \
      "${PRODUCTIVE_K3S_REPO:-${CORE_REPO_DIR_LOCAL}}" \
      "${PRODUCTIVE_K3S_INFRA_REPO_DIR:-${INFRA_REPO_DIR_LOCAL}}"
  fi
  PRODUCTIVE_K3S_SOURCE="${source_mode}" \
    PRODUCTIVE_K3S_AUTO_APPROVE_PREFLIGHT_WARNINGS="${PRODUCTIVE_K3S_AUTO_APPROVE_PREFLIGHT_WARNINGS:-true}" \
    MULTIPASS_LAUNCH_MAX_ATTEMPTS="${MULTIPASS_LAUNCH_MAX_ATTEMPTS}" \
    MULTIPASS_LAUNCH_RETRY_DELAY_SECONDS="${MULTIPASS_LAUNCH_RETRY_DELAY_SECONDS}" \
    TOFU_APPLY_PARALLELISM="${TOFU_APPLY_PARALLELISM}" \
    PRODUCTIVE_K3S_INFRA_REPO_DIR="${PRODUCTIVE_K3S_INFRA_REPO_DIR:-${INFRA_REPO_DIR_LOCAL}}" \
    PRODUCTIVE_K3S_PROFILES_REPO_DIR="${PRODUCTIVE_K3S_PROFILES_REPO_DIR:-${PROFILES_REPO_DIR}}" \
    PRODUCTIVE_K3S_REPO="${PRODUCTIVE_K3S_REPO:-${CORE_REPO_DIR_LOCAL}}" \
    PRODUCTIVE_K3S_ADDONS_REPO_DIR="${PRODUCTIVE_K3S_ADDONS_REPO_DIR:-${ADDONS_REPO_DIR_LOCAL}}" \
    "${PK3S_BIN}" "$@"
}

run_pk3s_with_timeout() {
  local seconds="$1"
  shift

  run_pk3s "$@" &
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

fallback_cleanup() {
  multipass delete \
    "${CLUSTER_PREFIX}-server" \
    "${CLUSTER_PREFIX}-agent-1" \
    "${CLUSTER_PREFIX}-agent-2" >/dev/null 2>&1 || true
  multipass purge >/dev/null 2>&1 || true
}

initial_cleanup() {
  run_pk3s_with_timeout 300 infra destroy --yes "${PROFILE_NAME}" >/dev/null 2>&1 || true
  fallback_cleanup
}

cleanup() {
  initial_cleanup
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
initial_cleanup

run_pk3s profile validate "${PROFILE_NAME}"
run_pk3s infra plan "${PROFILE_NAME}"
run_pk3s infra apply "${PROFILE_NAME}"
run_pk3s infra status "${PROFILE_NAME}"
run_pk3s_with_timeout 300 infra destroy --yes "${PROFILE_NAME}"

trap - EXIT
fallback_cleanup
rm -rf "${WORK_DIR}"
printf '[PASS] multipass remote CLI validation completed\n'
