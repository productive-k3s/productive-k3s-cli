#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  ./scripts/tag-release.sh <cli-version>

Example:
  ./scripts/tag-release.sh 1.0.1
EOF
}

err() {
  printf '%s\n' "$*" >&2
}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# shellcheck disable=SC1091
source "${SCRIPT_DIR}/release-config.sh"

VERSION="${1:-${VERSION:-}}"
if [[ -z "${VERSION}" || $# -gt 1 ]]; then
  usage >&2
  exit 1
fi

if [[ ! "${VERSION}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  err "invalid cli version: ${VERSION}"
  err "expected X.Y.Z"
  exit 1
fi

if [[ ! "${PRODUCTIVE_K3S_CORE_VERSION_DEFAULT}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  err "invalid default productive-k3s-core version: ${PRODUCTIVE_K3S_CORE_VERSION_DEFAULT}"
  exit 1
fi

if [[ ! "${PRODUCTIVE_K3S_INFRA_VERSION_DEFAULT}" =~ ^[0-9]+\.[0-9]+\.[0-9]+-[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  err "invalid default productive-k3s-infra version: ${PRODUCTIVE_K3S_INFRA_VERSION_DEFAULT}"
  exit 1
fi

infra_core="${PRODUCTIVE_K3S_INFRA_VERSION_DEFAULT#*-}"
if [[ "${infra_core}" != "${PRODUCTIVE_K3S_CORE_VERSION_DEFAULT}" ]]; then
  err "default infra version ${PRODUCTIVE_K3S_INFRA_VERSION_DEFAULT} is not bound to default core version ${PRODUCTIVE_K3S_CORE_VERSION_DEFAULT}"
  exit 1
fi

core_remote_url="${PRODUCTIVE_K3S_CORE_GIT_REMOTE_URL:-${PRODUCTIVE_K3S_CORE_GIT_REMOTE_URL_DEFAULT:-https://github.com/${PRODUCTIVE_K3S_CORE_RELEASE_REPO_DEFAULT}.git}}"
infra_remote_url="${PRODUCTIVE_K3S_INFRA_GIT_REMOTE_URL:-${PRODUCTIVE_K3S_INFRA_GIT_REMOTE_URL_DEFAULT:-https://github.com/${PRODUCTIVE_K3S_INFRA_RELEASE_REPO_DEFAULT}.git}}"

core_refs="$(git ls-remote --tags "${core_remote_url}" "refs/tags/${PRODUCTIVE_K3S_CORE_VERSION_DEFAULT}" "refs/tags/v${PRODUCTIVE_K3S_CORE_VERSION_DEFAULT}" || true)"
if [[ -z "${core_refs}" ]]; then
  err "productive-k3s-core version ${PRODUCTIVE_K3S_CORE_VERSION_DEFAULT} was not found in ${core_remote_url}"
  exit 1
fi

infra_refs="$(git ls-remote --tags "${infra_remote_url}" "refs/tags/${PRODUCTIVE_K3S_INFRA_VERSION_DEFAULT}" "refs/tags/v${PRODUCTIVE_K3S_INFRA_VERSION_DEFAULT}" || true)"
if [[ -z "${infra_refs}" ]]; then
  err "productive-k3s-infra version ${PRODUCTIVE_K3S_INFRA_VERSION_DEFAULT} was not found in ${infra_remote_url}"
  exit 1
fi

if git -C "${REPO_ROOT}" rev-parse --verify "refs/tags/${VERSION}" >/dev/null 2>&1; then
  err "tag ${VERSION} already exists locally"
  exit 1
fi

git -C "${REPO_ROOT}" tag -a "${VERSION}" -m "Release ${VERSION}"
printf 'Created tag %s\n' "${VERSION}"
