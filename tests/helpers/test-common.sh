#!/usr/bin/env bash
set -euo pipefail

TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPO_DIR="$(cd "${TESTS_DIR}/.." && pwd)"
COVERAGE_DIR="${TESTS_DIR}/coverage"
# shellcheck disable=SC2034
SPELL_ALLOWLIST="${TESTS_DIR}/spell/allowlist.txt"
GO_BIN="${GO_BIN:-}"
if [[ -z "${GO_BIN}" ]]; then
  if [[ -x /usr/local/go/bin/go ]]; then
    GO_BIN="/usr/local/go/bin/go"
  else
    GO_BIN="go"
  fi
fi

mkdir -p "${COVERAGE_DIR}" "${TESTS_DIR}/artifacts"

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    printf 'Missing required command: %s\n' "$1" >&2
    exit 127
  }
}

go_fmt_bin() {
  if [[ -n "${GOFMT_BIN:-}" ]]; then
    printf '%s\n' "${GOFMT_BIN}"
    return 0
  fi
  if [[ -n "${GO_BIN:-}" ]]; then
    local candidate
    candidate="$(cd "$(dirname "${GO_BIN}")" && pwd)/gofmt"
    if [[ -x "${candidate}" ]]; then
      printf '%s\n' "${candidate}"
      return 0
    fi
  fi
  command -v gofmt
}

shell_files() {
  (
    cd "${REPO_DIR}"
    rg --files tests/bin tests/helpers -g '*.sh'
  )
}

spell_files() {
  (
    cd "${REPO_DIR}"
    rg --files README.md scripts tests docs/src/en internal -g '*.md' -g '*.sh' -g '*.go' \
      -g '!tests/artifacts/**' \
      -g '!tests/coverage/**' \
      -g '!tests/bin/run-spellcheck.sh'
  )
}

go_files() {
  (
    cd "${REPO_DIR}"
    rg --files . -g '*.go'
  )
}
