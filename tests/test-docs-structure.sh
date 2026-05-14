#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MKDOCS_FILE="${ROOT_DIR}/docs/mkdocs.yml"

assert_file() {
  local path="$1"
  [[ -f "${path}" ]] || {
    printf '[FAIL] missing documentation file: %s\n' "${path}" >&2
    exit 1
  }
}

assert_contains() {
  local path="$1"
  local needle="$2"
  if ! grep -Fq -- "${needle}" "${path}"; then
    printf '[FAIL] expected %s to contain: %s\n' "${path}" "${needle}" >&2
    exit 1
  fi
}

for path in \
  "${ROOT_DIR}/docs/src/en/product/index.md" \
  "${ROOT_DIR}/docs/src/en/product/how-to-use.md" \
  "${ROOT_DIR}/docs/src/en/product/reasons-behind.md" \
  "${ROOT_DIR}/docs/src/en/product/productive-k3s-relationship.md" \
  "${ROOT_DIR}/docs/src/es/product/index.md" \
  "${ROOT_DIR}/docs/src/es/product/how-to-use.md" \
  "${ROOT_DIR}/docs/src/es/product/reasons-behind.md" \
  "${ROOT_DIR}/docs/src/es/product/productive-k3s-relationship.md" \
  "${ROOT_DIR}/docs/src/en/developer-docs/make-targets.md" \
  "${ROOT_DIR}/docs/src/en/developer-docs/github-actions.md" \
  "${ROOT_DIR}/docs/src/es/developer-docs/make-targets.md" \
  "${ROOT_DIR}/docs/src/es/developer-docs/github-actions.md" \
  "${ROOT_DIR}/docs/src/overrides/home.html" \
  "${ROOT_DIR}/docs/src/overrides/main.html" \
  "${ROOT_DIR}/docs/src/overrides/partials/footer.html"; do
  assert_file "${path}"
done

assert_contains "${MKDOCS_FILE}" '- Product:'
assert_contains "${MKDOCS_FILE}" '- User docs:'
assert_contains "${MKDOCS_FILE}" '- Developer docs:'
assert_contains "${MKDOCS_FILE}" 'Relationship with Productive K3S Core and Infra'
assert_contains "${MKDOCS_FILE}" 'GitHub Actions and releases'
assert_contains "${MKDOCS_FILE}" 'Relación con Productive K3S Core e Infra'
assert_contains "${MKDOCS_FILE}" 'GitHub Actions y releases'

printf '[PASS] docs structure matches the expected Product/User/Developer layout\n'
