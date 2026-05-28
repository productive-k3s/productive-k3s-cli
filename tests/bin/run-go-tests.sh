#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/helpers/test-common.sh"

need_cmd "${GO_BIN}"

cd "${REPO_DIR}"
exec "${GO_BIN}" test ./...
