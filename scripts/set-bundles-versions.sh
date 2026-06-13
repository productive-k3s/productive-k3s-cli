#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  ./scripts/set-bundles-versions.sh <core-version> <infra-version>

Example:
  ./scripts/set-bundles-versions.sh 0.9.1 0.9.3-0.9.1
EOF
}

err() {
  printf '%s\n' "$*" >&2
}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# shellcheck disable=SC1091
source "${SCRIPT_DIR}/release-config.sh"

CORE_VERSION="${1:-${CORE_VERSION:-}}"
INFRA_VERSION="${2:-${INFRA_VERSION:-}}"
if [[ -z "${CORE_VERSION}" || -z "${INFRA_VERSION}" || $# -gt 2 ]]; then
  usage >&2
  exit 1
fi

if [[ ! "${CORE_VERSION}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  err "invalid productive-k3s-core version: ${CORE_VERSION}"
  err "expected X.Y.Z"
  exit 1
fi

if [[ ! "${INFRA_VERSION}" =~ ^[0-9]+\.[0-9]+\.[0-9]+-[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  err "invalid productive-k3s-infra version: ${INFRA_VERSION}"
  err "expected X.Y.Z-A.B.C"
  exit 1
fi

infra_core="${INFRA_VERSION#*-}"
if [[ "${infra_core}" != "${CORE_VERSION}" ]]; then
  err "infra version ${INFRA_VERSION} is not bound to core version ${CORE_VERSION}"
  exit 1
fi

core_remote_url="${PRODUCTIVE_K3S_CORE_GIT_REMOTE_URL:-https://github.com/${PRODUCTIVE_K3S_CORE_RELEASE_REPO_DEFAULT}.git}"
infra_remote_url="${PRODUCTIVE_K3S_INFRA_GIT_REMOTE_URL:-https://github.com/${PRODUCTIVE_K3S_INFRA_RELEASE_REPO_DEFAULT}.git}"

core_refs="$(git ls-remote --tags "${core_remote_url}" "refs/tags/${CORE_VERSION}" "refs/tags/v${CORE_VERSION}" || true)"
if [[ -z "${core_refs}" ]]; then
  err "productive-k3s-core version ${CORE_VERSION} was not found in ${core_remote_url}"
  exit 1
fi

infra_refs="$(git ls-remote --tags "${infra_remote_url}" "refs/tags/${INFRA_VERSION}" "refs/tags/v${INFRA_VERSION}" || true)"
if [[ -z "${infra_refs}" ]]; then
  err "productive-k3s-infra version ${INFRA_VERSION} was not found in ${infra_remote_url}"
  exit 1
fi

replace_in_file() {
  local path="$1"
  local pattern="$2"
  local replacement="$3"
  if [[ ! -f "${path}" ]]; then
    err "missing file: ${path}"
    exit 1
  fi
  perl -0pi -e "s~${pattern}~${replacement}~gm" "${path}"
}

core_fixture_dir="$(find "${REPO_ROOT}/tests/fixtures/bundles/core" -mindepth 1 -maxdepth 1 -type d | sort | head -n1)"
infra_fixture_dir="$(find "${REPO_ROOT}/tests/fixtures/bundles/infra" -mindepth 1 -maxdepth 1 -type d | sort | head -n1)"
if [[ -z "${core_fixture_dir}" || -z "${infra_fixture_dir}" ]]; then
  err "expected bundle fixture directories under tests/fixtures/bundles"
  exit 1
fi

current_core_fixture="$(basename "${core_fixture_dir}")"
current_infra_fixture="$(basename "${infra_fixture_dir}")"
target_core_fixture_dir="${REPO_ROOT}/tests/fixtures/bundles/core/${CORE_VERSION}"
target_infra_fixture_dir="${REPO_ROOT}/tests/fixtures/bundles/infra/${INFRA_VERSION}"

if [[ "${core_fixture_dir}" != "${target_core_fixture_dir}" ]]; then
  mv "${core_fixture_dir}" "${target_core_fixture_dir}"
fi
if [[ "${infra_fixture_dir}" != "${target_infra_fixture_dir}" ]]; then
  mv "${infra_fixture_dir}" "${target_infra_fixture_dir}"
fi

replace_in_file "${REPO_ROOT}/scripts/release-config.sh" 'PRODUCTIVE_K3S_CORE_VERSION_DEFAULT:=\K[0-9]+\.[0-9]+\.[0-9]+' "${CORE_VERSION}"
replace_in_file "${REPO_ROOT}/scripts/release-config.sh" 'PRODUCTIVE_K3S_INFRA_VERSION_DEFAULT:=\K[0-9]+\.[0-9]+\.[0-9]+-[0-9]+\.[0-9]+\.[0-9]+' "${INFRA_VERSION}"
replace_in_file "${REPO_ROOT}/internal/bundles/types.go" 'CoreVersion:\s+"\K[0-9]+\.[0-9]+\.[0-9]+' "${CORE_VERSION}"
replace_in_file "${REPO_ROOT}/internal/bundles/types.go" 'InfraVersion:\s+"\K[0-9]+\.[0-9]+\.[0-9]+-[0-9]+\.[0-9]+\.[0-9]+' "${INFRA_VERSION}"
replace_in_file "${REPO_ROOT}/internal/bundles/helpers_test.go" 'productive-k3s-core-[0-9]+\.[0-9]+\.[0-9]+\.tar\.gz' "productive-k3s-core-${CORE_VERSION}.tar.gz"
replace_in_file "${REPO_ROOT}/internal/bundles/helpers_test.go" 'productive-k3s-infra-[0-9]+\.[0-9]+\.[0-9]+-[0-9]+\.[0-9]+\.[0-9]+\.tar\.gz' "productive-k3s-infra-${INFRA_VERSION}.tar.gz"
replace_in_file "${REPO_ROOT}/internal/bundles/bundles_test.go" 'productive-k3s-infra-\K[0-9]+\.[0-9]+\.[0-9]+-[0-9]+\.[0-9]+\.[0-9]+' "${INFRA_VERSION}"
replace_in_file "${REPO_ROOT}/internal/bundles/bundles_test.go" 'PK3S_INFRA_RELEASE_TAG=\K[0-9]+\.[0-9]+\.[0-9]+-[0-9]+\.[0-9]+\.[0-9]+' "${INFRA_VERSION}"
replace_in_file "${REPO_ROOT}/internal/bundles/bundles_test.go" '"infra_version":"\K[0-9]+\.[0-9]+\.[0-9]+-[0-9]+\.[0-9]+\.[0-9]+' "${INFRA_VERSION}"
replace_in_file "${REPO_ROOT}/internal/bundles/bundles_test.go" 'productive-k3s-core-\K[0-9]+\.[0-9]+\.[0-9]+' "${CORE_VERSION}"
replace_in_file "${REPO_ROOT}/internal/bundles/bundles_test.go" '"core_version":"\K[0-9]+\.[0-9]+\.[0-9]+' "${CORE_VERSION}"
replace_in_file "${REPO_ROOT}/internal/bundles/bundles_test.go" 'Version:\s+"\K[0-9]+\.[0-9]+\.[0-9]+(?=")' "${CORE_VERSION}"
replace_in_file "${REPO_ROOT}/internal/bundles/bundles_test.go" 'Version:\s+"\K[0-9]+\.[0-9]+\.[0-9]+-[0-9]+\.[0-9]+\.[0-9]+(?=")' "${INFRA_VERSION}"
replace_in_file "${REPO_ROOT}/docs/src/en/user-docs/usage.md" 'Productive K3S Core `[0-9]+\.[0-9]+\.[0-9]+`' "Productive K3S Core \`${CORE_VERSION}\`"
replace_in_file "${REPO_ROOT}/docs/src/en/user-docs/usage.md" 'Productive K3S Infra `[0-9]+\.[0-9]+\.[0-9]+-[0-9]+\.[0-9]+\.[0-9]+`' "Productive K3S Infra \`${INFRA_VERSION}\`"
replace_in_file "${REPO_ROOT}/docs/src/es/user-docs/usage.md" 'Productive K3S Core `[0-9]+\.[0-9]+\.[0-9]+`' "Productive K3S Core \`${CORE_VERSION}\`"
replace_in_file "${REPO_ROOT}/docs/src/es/user-docs/usage.md" 'Productive K3S Infra `[0-9]+\.[0-9]+\.[0-9]+-[0-9]+\.[0-9]+\.[0-9]+`' "Productive K3S Infra \`${INFRA_VERSION}\`"
replace_in_file "${REPO_ROOT}/README.md" 'make set-bundles-versions CORE_VERSION=[0-9]+\.[0-9]+\.[0-9]+ INFRA_VERSION=[0-9]+\.[0-9]+\.[0-9]+-[0-9]+\.[0-9]+\.[0-9]+' "make set-bundles-versions CORE_VERSION=${CORE_VERSION} INFRA_VERSION=${INFRA_VERSION}"
replace_in_file "${REPO_ROOT}/tests/test-release-versioning.sh" '"\K[0-9]+\.[0-9]+\.[0-9]+(?=" "default core version")' "${CORE_VERSION}"
replace_in_file "${REPO_ROOT}/tests/test-release-versioning.sh" '"\K[0-9]+\.[0-9]+\.[0-9]+-[0-9]+\.[0-9]+\.[0-9]+(?=" "default infra version")' "${INFRA_VERSION}"
replace_in_file "${REPO_ROOT}/tests/test-tag-release.sh" 'git -C "\$\{core_seed\}" tag \K[0-9]+\.[0-9]+\.[0-9]+' "${CORE_VERSION}"
replace_in_file "${REPO_ROOT}/tests/test-tag-release.sh" 'refs/tags/[0-9]+\.[0-9]+\.[0-9]+' "refs/tags/${CORE_VERSION}"
replace_in_file "${REPO_ROOT}/tests/test-tag-release.sh" 'git -C "\$\{infra_seed\}" tag \K[0-9]+\.[0-9]+\.[0-9]+-[0-9]+\.[0-9]+\.[0-9]+' "${INFRA_VERSION}"
replace_in_file "${REPO_ROOT}/tests/test-tag-release.sh" 'push --quiet origin HEAD refs/tags/\K[0-9]+\.[0-9]+\.[0-9]+-[0-9]+\.[0-9]+\.[0-9]+' "${INFRA_VERSION}"
replace_in_file "${REPO_ROOT}/internal/app/app_test.go" '"name":"productive-k3s-core","version":"[0-9]+\.[0-9]+\.[0-9]+"' "\"name\":\"productive-k3s-core\",\"version\":\"${CORE_VERSION}\""
replace_in_file "${REPO_ROOT}/internal/app/app_test.go" '"name":"productive-k3s-infra","version":"[0-9]+\.[0-9]+\.[0-9]+-[0-9]+\.[0-9]+\.[0-9]+"' "\"name\":\"productive-k3s-infra\",\"version\":\"${INFRA_VERSION}\""
replace_in_file "${REPO_ROOT}/internal/app/app_test.go" 'coreCLI\["version"\] != "[0-9]+\.[0-9]+\.[0-9]+"' "coreCLI[\"version\"] != \"${CORE_VERSION}\""
replace_in_file "${REPO_ROOT}/internal/app/app_test.go" 'infraCLI\["version"\] != "[0-9]+\.[0-9]+\.[0-9]+-[0-9]+\.[0-9]+\.[0-9]+"' "infraCLI[\"version\"] != \"${INFRA_VERSION}\""
replace_in_file "${REPO_ROOT}/internal/app/app_test.go" '"bundle_name":"productive-k3s-core","bundle_type":"productive-k3s-core","bundle_version":"[0-9]+\.[0-9]+\.[0-9]+"' "\"bundle_name\":\"productive-k3s-core\",\"bundle_type\":\"productive-k3s-core\",\"bundle_version\":\"${CORE_VERSION}\""
replace_in_file "${REPO_ROOT}/internal/app/app_test.go" '"bundle_name":"productive-k3s-infra","bundle_type":"productive-k3s-infra","bundle_version":"[0-9]+\.[0-9]+\.[0-9]+-[0-9]+\.[0-9]+\.[0-9]+"' "\"bundle_name\":\"productive-k3s-infra\",\"bundle_type\":\"productive-k3s-infra\",\"bundle_version\":\"${INFRA_VERSION}\""
replace_in_file "${REPO_ROOT}/internal/app/app_test.go" 'coreBundle\["bundle_version"\] != "[0-9]+\.[0-9]+\.[0-9]+"' "coreBundle[\"bundle_version\"] != \"${CORE_VERSION}\""
replace_in_file "${REPO_ROOT}/internal/app/app_more_test.go" 'version: \K[0-9]+\.[0-9]+\.[0-9]+-[0-9]+\.[0-9]+\.[0-9]+' "${INFRA_VERSION}"
replace_in_file "${REPO_ROOT}/internal/app/app_more_test.go" '/infra/multipass-1-server-2-agents-\K[0-9]+\.[0-9]+\.[0-9]+-[0-9]+\.[0-9]+\.[0-9]+(?=\.tgz)' "${INFRA_VERSION}"
replace_in_file "${REPO_ROOT}/internal/app/app_more_test.go" '/infra/aws-single-node-basic-\K[0-9]+\.[0-9]+\.[0-9]+-[0-9]+\.[0-9]+\.[0-9]+(?=\.tgz)' "${INFRA_VERSION}"
replace_in_file "${REPO_ROOT}/internal/app/app_more_test.go" 'multipass-1-server-2-agents\\t\K[0-9]+\.[0-9]+\.[0-9]+-[0-9]+\.[0-9]+\.[0-9]+(?=\\tlocal)' "${INFRA_VERSION}"
replace_in_file "${REPO_ROOT}/internal/app/app_more_test.go" 'aws-single-node-basic\\t\K[0-9]+\.[0-9]+\.[0-9]+-[0-9]+\.[0-9]+\.[0-9]+(?=\\tcloud\\tneeds-env)' "${INFRA_VERSION}"

replace_in_file "${REPO_ROOT}/tests/fixtures/manifests/cli-1.0.0.json" '"bundle_version": "\K[^"]+(?=")' "${CORE_VERSION}"
replace_in_file "${REPO_ROOT}/tests/fixtures/manifests/cli-1.0.0.json" '"bundle_version": "'${CORE_VERSION}'"\n    "source": "local",\n    "path": "tests/fixtures/bundles/core/\K[^"]+(?=")' "${CORE_VERSION}"
replace_in_file "${REPO_ROOT}/tests/fixtures/manifests/cli-1.0.0.json" '"bundle_name": "productive-k3s-infra",\n    "bundle_version": "\K[^"]+(?=")' "${INFRA_VERSION}"
replace_in_file "${REPO_ROOT}/tests/fixtures/manifests/cli-1.0.0.json" 'tests/fixtures/bundles/core/\K[^"]+(?=")' "${CORE_VERSION}"
replace_in_file "${REPO_ROOT}/tests/fixtures/manifests/cli-1.0.0.json" 'tests/fixtures/bundles/infra/\K[^"]+(?=")' "${INFRA_VERSION}"

replace_in_file "${target_core_fixture_dir}/bundle-info.json" '"bundle_version": "\K[^"]+(?=")' "${CORE_VERSION}"
replace_in_file "${target_infra_fixture_dir}/bundle-info.json" '"bundle_version": "\K[^"]+(?=")' "${INFRA_VERSION}"
replace_in_file "${target_infra_fixture_dir}/bundle-info.json" '"core_dependency": \{\n    "bundle_name": "productive-k3s-core",\n    "bundle_version": "\K[^"]+(?=")' "${CORE_VERSION}"

printf 'Updated bundle defaults to core=%s infra=%s\n' "${CORE_VERSION}" "${INFRA_VERSION}"
