#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP_DIR="$(mktemp -d)"
WORKTREE="${TMP_DIR}/cli"
CORE_REMOTE="${TMP_DIR}/core-remote.git"
INFRA_REMOTE="${TMP_DIR}/infra-remote.git"
TAG_NAME="1.2.3"

cleanup() {
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

fail() {
  printf '[FAIL] %s\n' "$*" >&2
  exit 1
}

assert_contains() {
  local haystack="$1"
  local needle="$2"
  if [[ "${haystack}" != *"${needle}"* ]]; then
    fail "expected output to contain: ${needle}"
  fi
}

mkdir -p "${WORKTREE}"
if git -C "${ROOT_DIR}" rev-parse --git-dir >/dev/null 2>&1; then
  git clone --quiet "${ROOT_DIR}" "${WORKTREE}" || fail "could not clone cli repo into temporary worktree"
  (
    cd "${ROOT_DIR}"
    tar --exclude=.git -cf - .
  ) | (
    cd "${WORKTREE}"
    tar -xf -
  )
else
  cp -a "${ROOT_DIR}/." "${WORKTREE}"
fi
git init --bare "${CORE_REMOTE}" >/dev/null
git init --bare "${INFRA_REMOTE}" >/dev/null
git -C "${WORKTREE}" tag -d "${TAG_NAME}" >/dev/null 2>&1 || true

if PRODUCTIVE_K3S_CORE_GIT_REMOTE_URL="${CORE_REMOTE}" \
  PRODUCTIVE_K3S_INFRA_GIT_REMOTE_URL="${INFRA_REMOTE}" \
  bash "${WORKTREE}/scripts/tag-release.sh" >/dev/null 2>&1; then
  fail "missing VERSION unexpectedly succeeded"
fi

if PRODUCTIVE_K3S_CORE_GIT_REMOTE_URL="${CORE_REMOTE}" \
  PRODUCTIVE_K3S_INFRA_GIT_REMOTE_URL="${INFRA_REMOTE}" \
  bash "${WORKTREE}/scripts/tag-release.sh" 1.2 >/dev/null 2>&1; then
  fail "invalid VERSION unexpectedly succeeded"
fi

if PRODUCTIVE_K3S_CORE_GIT_REMOTE_URL="${CORE_REMOTE}" \
  PRODUCTIVE_K3S_INFRA_GIT_REMOTE_URL="${INFRA_REMOTE}" \
  bash "${WORKTREE}/scripts/tag-release.sh" "${TAG_NAME}" >/dev/null 2>&1; then
  fail "tag creation succeeded without upstream bundle tags"
fi

core_seed="${TMP_DIR}/core-seed"
git init "${core_seed}" >/dev/null
git -C "${core_seed}" config user.name tester
git -C "${core_seed}" config user.email tester@example.com
printf 'core\n' > "${core_seed}/README.md"
git -C "${core_seed}" add README.md
git -C "${core_seed}" commit -m "seed" >/dev/null
git -C "${core_seed}" tag 0.9.5
git -C "${core_seed}" remote add origin "${CORE_REMOTE}"
git -C "${core_seed}" push --quiet origin HEAD refs/tags/0.9.5

infra_seed="${TMP_DIR}/infra-seed"
git init "${infra_seed}" >/dev/null
git -C "${infra_seed}" config user.name tester
git -C "${infra_seed}" config user.email tester@example.com
printf 'infra\n' > "${infra_seed}/README.md"
git -C "${infra_seed}" add README.md
git -C "${infra_seed}" commit -m "seed" >/dev/null
git -C "${infra_seed}" tag 0.9.63-0.9.5
git -C "${infra_seed}" remote add origin "${INFRA_REMOTE}"
git -C "${infra_seed}" push --quiet origin HEAD refs/tags/0.9.63-0.9.5

output="$(
  cd "${WORKTREE}" && \
  PRODUCTIVE_K3S_CORE_GIT_REMOTE_URL="${CORE_REMOTE}" \
  PRODUCTIVE_K3S_INFRA_GIT_REMOTE_URL="${INFRA_REMOTE}" \
  bash "./scripts/tag-release.sh" "${TAG_NAME}"
)"
assert_contains "${output}" "Created tag ${TAG_NAME}"
git -C "${WORKTREE}" rev-parse --verify "${TAG_NAME}^{tag}" >/dev/null 2>&1 || fail "expected local tag ${TAG_NAME}"

if (
  cd "${WORKTREE}" && \
  PRODUCTIVE_K3S_CORE_GIT_REMOTE_URL="${CORE_REMOTE}" \
  PRODUCTIVE_K3S_INFRA_GIT_REMOTE_URL="${INFRA_REMOTE}" \
  bash "./scripts/tag-release.sh" "${TAG_NAME}"
) >/dev/null 2>&1; then
  fail "duplicate local tag unexpectedly succeeded"
fi

printf '[PASS] cli tag-release validates upstream bundle tags and creates semantic tags\n'
