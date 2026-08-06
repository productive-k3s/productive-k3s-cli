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
assert_eq "${PRODUCTIVE_K3S_GITHUB_OWNER_DEFAULT}" "productive-k3s" "default github owner"
assert_eq "${PRODUCTIVE_K3S_GITHUB_BASE_URL_DEFAULT}" "https://github.com/productive-k3s" "default github base url"
assert_eq "${PRODUCTIVE_K3S_GITHUB_RAW_BASE_URL_DEFAULT}" "https://raw.githubusercontent.com/productive-k3s" "default github raw base url"
assert_eq "${PRODUCTIVE_K3S_CORE_REPO_NAME_DEFAULT}" "productive-k3s-core" "default core repo name"
assert_eq "${PRODUCTIVE_K3S_INFRA_REPO_NAME_DEFAULT}" "productive-k3s-infra" "default infra repo name"
assert_eq "${PRODUCTIVE_K3S_PROFILES_REPO_NAME_DEFAULT}" "productive-k3s-profiles" "default profiles repo name"
assert_eq "${PRODUCTIVE_K3S_CLI_REPO_NAME_DEFAULT}" "productive-k3s-cli" "default cli repo name"
assert_eq "${PRODUCTIVE_K3S_CORE_VERSION_DEFAULT}" "0.9.5" "default core version"
assert_eq "${PRODUCTIVE_K3S_INFRA_VERSION_DEFAULT}" "0.9.63-0.9.5" "default infra version"
assert_eq "${PRODUCTIVE_K3S_CORE_RELEASE_REPO_DEFAULT}" "productive-k3s/productive-k3s-core" "default core release repo"
assert_eq "${PRODUCTIVE_K3S_INFRA_RELEASE_REPO_DEFAULT}" "productive-k3s/productive-k3s-infra" "default infra release repo"
assert_eq "${PRODUCTIVE_K3S_CORE_GIT_REMOTE_URL_DEFAULT}" "https://github.com/productive-k3s/productive-k3s-core.git" "default core git remote url"
assert_eq "${PRODUCTIVE_K3S_INFRA_GIT_REMOTE_URL_DEFAULT}" "https://github.com/productive-k3s/productive-k3s-infra.git" "default infra git remote url"
assert_eq "${PRODUCTIVE_K3S_PROFILES_GIT_REMOTE_URL_DEFAULT}" "https://github.com/productive-k3s/productive-k3s-profiles.git" "default profiles git remote url"
assert_eq "${PRODUCTIVE_K3S_PROFILES_MULTIPASS_PROFILE_URL_DEFAULT}" "https://raw.githubusercontent.com/productive-k3s/productive-k3s-profiles/main/profiles/local/multipass/1-server-2-agents.env" "default multipass profile url"
assert_eq "${PRODUCTIVE_K3S_CLI_REPO_DEFAULT}" "productive-k3s/productive-k3s-cli" "default cli release repo"
assert_eq "${PRODUCTIVE_K3S_CATALOG_URL_DEFAULT}" "https://catalogs.productive-k3s.io/catalogs/index.yaml" "default catalog url"

infra_core="${PRODUCTIVE_K3S_INFRA_VERSION_DEFAULT#*-}"
assert_eq "${infra_core}" "${PRODUCTIVE_K3S_CORE_VERSION_DEFAULT}" "infra/core compatibility suffix"

printf '[PASS] cli release config defaults are aligned\n'
