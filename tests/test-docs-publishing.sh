#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORKFLOW_FILE="${ROOT_DIR}/.github/workflows/docs.yml"
MKDOCS_FILE="${ROOT_DIR}/docs/mkdocs.yml"
CNAME_FILE="${ROOT_DIR}/docs/src/CNAME"

assert_contains() {
  local path="$1"
  local needle="$2"
  if ! grep -Fq "${needle}" "${path}"; then
    printf '[FAIL] expected %s to contain: %s\n' "${path}" "${needle}" >&2
    exit 1
  fi
}

[[ -f "${WORKFLOW_FILE}" ]] || {
  printf '[FAIL] missing docs workflow: %s\n' "${WORKFLOW_FILE}" >&2
  exit 1
}

assert_contains "${WORKFLOW_FILE}" 'name: Documentation'
assert_contains "${WORKFLOW_FILE}" 'uses: actions/setup-python@v5'
assert_contains "${WORKFLOW_FILE}" 'uses: peaceiris/actions-gh-pages@v4'
assert_contains "${WORKFLOW_FILE}" 'publish_dir: ./docs/site'
assert_contains "${WORKFLOW_FILE}" 'publish_branch: gh-pages'
assert_contains "${WORKFLOW_FILE}" 'cname: cli.productive-k3s.io'
assert_contains "${MKDOCS_FILE}" 'site_url: https://cli.productive-k3s.io/'

[[ -f "${CNAME_FILE}" ]] || {
  printf '[FAIL] missing tracked CNAME file: %s\n' "${CNAME_FILE}" >&2
  exit 1
}

grep -Fx 'cli.productive-k3s.io' "${CNAME_FILE}" >/dev/null || {
  printf '[FAIL] expected %s to equal cli.productive-k3s.io\n' "${CNAME_FILE}" >&2
  exit 1
}

printf '[PASS] docs publishing workflow and CNAME wiring are present\n'
