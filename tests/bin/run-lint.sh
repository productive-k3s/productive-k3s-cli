#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/helpers/test-common.sh"

need_cmd "${GO_BIN}"

cd "${REPO_DIR}"
"${GO_BIN}" vet ./...

if command -v shellcheck >/dev/null 2>&1; then
  mapfile -t files < <(shell_files)
  if ((${#files[@]} > 0)); then
    shellcheck -x -e SC1091 "${files[@]}"
  fi
fi
