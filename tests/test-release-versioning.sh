#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONFIG="${ROOT_DIR}/scripts/release-config.sh"

assert_eq() {
  local actual="$1"
  local expected="$2"
  local label="$3"
  if [[ "${actual}" != "${expected}" ]]; then
    printf '[FAIL] %s: expected %s, got %s\n' "${label}" "${expected}" "${actual}" >&2
    exit 1
  fi
}

if [[ ! -f "${CONFIG}" ]]; then
  printf '[FAIL] expected release config at %s\n' "${CONFIG}" >&2
  exit 1
fi

set -a
# shellcheck disable=SC1090
source "${CONFIG}"
set +a

assert_eq "${PK3S_CLI_VERSION_DEFAULT}" "1.0.0" "default cli version"
assert_eq "${PRODUCTIVE_K3S_CORE_VERSION_DEFAULT}" "0.9.1" "default core version"
assert_eq "${PRODUCTIVE_K3S_INFRA_VERSION_DEFAULT}" "0.9.41-0.9.1" "default infra version"
assert_eq "${PRODUCTIVE_K3S_CORE_RELEASE_REPO_DEFAULT}" "jemacchi/productive-k3s-core" "default core release repo"
assert_eq "${PRODUCTIVE_K3S_INFRA_RELEASE_REPO_DEFAULT}" "jemacchi/productive-k3s-infra" "default infra release repo"

infra_core="${PRODUCTIVE_K3S_INFRA_VERSION_DEFAULT#*-}"
assert_eq "${infra_core}" "${PRODUCTIVE_K3S_CORE_VERSION_DEFAULT}" "infra/core compatibility suffix"

printf '[PASS] cli release config defaults are aligned\n'
