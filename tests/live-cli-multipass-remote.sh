#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PK3S_BIN="${PRODUCTIVE_K3S_CLI_BIN:-${ROOT_DIR}/pk3s}"
PROFILE_URL="${PK3S_CLI_MULTIPASS_PROFILE_URL:-https://raw.githubusercontent.com/jemacchi/productive-k3s-profiles/main/profiles/multipass/1-server-2-agents.env}"
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
  PRODUCTIVE_K3S_SOURCE="${source_mode}" "${PK3S_BIN}" "$@"
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
}

need_cmd multipass
need_cmd jq
need_cmd curl
need_cmd tar
need_cmd python3
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
