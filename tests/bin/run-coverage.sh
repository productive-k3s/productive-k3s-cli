#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/helpers/test-common.sh"

need_cmd "${GO_BIN}"

cd "${REPO_DIR}"
mkdir -p "${COVERAGE_DIR}"
"${GO_BIN}" test ./... -coverprofile="${COVERAGE_DIR}/coverage.out"
"${GO_BIN}" tool cover -func="${COVERAGE_DIR}/coverage.out" | tee "${COVERAGE_DIR}/coverage.txt"
