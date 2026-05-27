#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/helpers/test-common.sh"

GOFMT_BIN="$(go_fmt_bin)"
need_cmd "${GOFMT_BIN}"

cd "${REPO_DIR}"
mapfile -t gofiles < <(go_files)
if ((${#gofiles[@]} > 0)); then
  unformatted="$("${GOFMT_BIN}" -l "${gofiles[@]}")"
  if [[ -n "${unformatted}" ]]; then
    printf 'Go files need formatting:\n%s\n' "${unformatted}" >&2
    exit 1
  fi
fi

if command -v shfmt >/dev/null 2>&1; then
  mapfile -t shellfiles < <(shell_files)
  if ((${#shellfiles[@]} > 0)); then
    shfmt -d -i 2 -ci "${shellfiles[@]}"
  fi
fi
