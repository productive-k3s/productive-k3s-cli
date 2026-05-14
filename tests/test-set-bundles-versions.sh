#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP_DIR="$(mktemp -d)"
WORKTREE="${TMP_DIR}/cli"
CORE_REMOTE="${TMP_DIR}/core-remote.git"
INFRA_REMOTE="${TMP_DIR}/infra-remote.git"
TARGET_CORE_VERSION="9.8.7"
TARGET_INFRA_VERSION="6.5.4-9.8.7"

cleanup() {
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

fail() {
  printf '[FAIL] %s\n' "$*" >&2
  exit 1
}

assert_contains_file() {
  local path="$1"
  local needle="$2"
  if ! grep -Fq "${needle}" "${path}"; then
    fail "expected ${path} to contain: ${needle}"
  fi
}

mkdir -p "${WORKTREE}"
cp -a "${ROOT_DIR}/." "${WORKTREE}"
git init --bare "${CORE_REMOTE}" >/dev/null
git init --bare "${INFRA_REMOTE}" >/dev/null

core_seed="${TMP_DIR}/core-seed"
git init "${core_seed}" >/dev/null
git -C "${core_seed}" config user.name tester
git -C "${core_seed}" config user.email tester@example.com
printf 'core\n' > "${core_seed}/README.md"
git -C "${core_seed}" add README.md
git -C "${core_seed}" commit -m "seed" >/dev/null
git -C "${core_seed}" tag "${TARGET_CORE_VERSION}"
git -C "${core_seed}" remote add origin "${CORE_REMOTE}"
git -C "${core_seed}" push --quiet origin HEAD "refs/tags/${TARGET_CORE_VERSION}"

infra_seed="${TMP_DIR}/infra-seed"
git init "${infra_seed}" >/dev/null
git -C "${infra_seed}" config user.name tester
git -C "${infra_seed}" config user.email tester@example.com
printf 'infra\n' > "${infra_seed}/README.md"
git -C "${infra_seed}" add README.md
git -C "${infra_seed}" commit -m "seed" >/dev/null
git -C "${infra_seed}" tag "${TARGET_INFRA_VERSION}"
git -C "${infra_seed}" remote add origin "${INFRA_REMOTE}"
git -C "${infra_seed}" push --quiet origin HEAD "refs/tags/${TARGET_INFRA_VERSION}"

(
  cd "${WORKTREE}"
  PRODUCTIVE_K3S_CORE_GIT_REMOTE_URL="${CORE_REMOTE}" \
  PRODUCTIVE_K3S_INFRA_GIT_REMOTE_URL="${INFRA_REMOTE}" \
    bash "./scripts/set-bundles-versions.sh" "${TARGET_CORE_VERSION}" "${TARGET_INFRA_VERSION}"
)

assert_contains_file "${WORKTREE}/scripts/release-config.sh" "PRODUCTIVE_K3S_CORE_VERSION_DEFAULT:=${TARGET_CORE_VERSION}"
assert_contains_file "${WORKTREE}/scripts/release-config.sh" "PRODUCTIVE_K3S_INFRA_VERSION_DEFAULT:=${TARGET_INFRA_VERSION}"
assert_contains_file "${WORKTREE}/internal/bundles/types.go" "CoreVersion:  \"${TARGET_CORE_VERSION}\""
assert_contains_file "${WORKTREE}/internal/bundles/types.go" "InfraVersion: \"${TARGET_INFRA_VERSION}\""
assert_contains_file "${WORKTREE}/docs/src/en/user-docs/usage.md" "Productive K3S Core \`${TARGET_CORE_VERSION}\`"
assert_contains_file "${WORKTREE}/docs/src/en/user-docs/usage.md" "Productive K3S Infra \`${TARGET_INFRA_VERSION}\`"
assert_contains_file "${WORKTREE}/tests/fixtures/manifests/cli-1.0.0.json" "\"bundle_version\": \"${TARGET_CORE_VERSION}\""
assert_contains_file "${WORKTREE}/tests/fixtures/manifests/cli-1.0.0.json" "\"bundle_version\": \"${TARGET_INFRA_VERSION}\""
assert_contains_file "${WORKTREE}/tests/fixtures/manifests/cli-1.0.0.json" "\"path\": \"tests/fixtures/bundles/core/${TARGET_CORE_VERSION}\""
assert_contains_file "${WORKTREE}/tests/fixtures/manifests/cli-1.0.0.json" "\"path\": \"tests/fixtures/bundles/infra/${TARGET_INFRA_VERSION}\""
assert_contains_file "${WORKTREE}/tests/fixtures/bundles/core/${TARGET_CORE_VERSION}/bundle-info.json" "\"bundle_version\": \"${TARGET_CORE_VERSION}\""
assert_contains_file "${WORKTREE}/tests/fixtures/bundles/infra/${TARGET_INFRA_VERSION}/bundle-info.json" "\"bundle_version\": \"${TARGET_INFRA_VERSION}\""
assert_contains_file "${WORKTREE}/tests/fixtures/bundles/infra/${TARGET_INFRA_VERSION}/bundle-info.json" "\"bundle_version\": \"${TARGET_CORE_VERSION}\""

printf '[PASS] set-bundles-versions updates cli bundle defaults consistently\n'
