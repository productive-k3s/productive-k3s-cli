#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
MKDOCS_BIN="${MKDOCS_BIN:-mkdocs}"
PYTHON_BIN="${PYTHON_BIN:-python3}"
DOCS_PORT="${DOCS_PORT:-8000}"

usage() {
  cat <<'EOF'
Usage:
  ./scripts/productive-k3s-cli-dev.sh <command>

Documentation commands:
  docs-build     Build the MkDocs site
  docs-serve     Serve the MkDocs site locally
  docs-up        Alias for docs-serve
  docs-down      Placeholder for parity with Core/Infra docs workflow
  docs-clean     Remove generated documentation output

Test commands:
  test-static                Run lightweight static checks
  test-contract              Run documentation contract checks
  test-productive-k3s-cli    Run all local CLI documentation checks
EOF
}

log() {
  local level="$1"
  shift
  printf '[pk3s-cli-dev] %-5s %s\n' "${level}" "$*"
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    log ERROR "missing dependency: $1"
    exit 5
  }
}

run_mkdocs() {
  need_cmd "${MKDOCS_BIN}"
  (cd "${REPO_DIR}" && "${MKDOCS_BIN}" "$@")
}

run_static_checks() {
  log INFO "Checking required documentation files"
  test -f "${REPO_DIR}/mkdocs.yml"
  test -f "${REPO_DIR}/docs/index.md"
  test -f "${REPO_DIR}/docs/developer/index.md"
  test -f "${REPO_DIR}/docs/developer/bundles/versioning.md"
  test -f "${REPO_DIR}/docs/developer/cli/command-mapping.md"
  log OK "Static checks passed"
}

run_contract_checks() {
  need_cmd "${PYTHON_BIN}"
  "${PYTHON_BIN}" - <<'PY_CONTRACT'
from pathlib import Path
root = Path.cwd()
text = (root / 'docs/developer/bundles/versioning.md').read_text(encoding='utf-8')
required = ['2.1.0', '1.4.0-2.1.0', 'X.Y.Z-A.B.C']
missing = [item for item in required if item not in text]
if missing:
    raise SystemExit(f'missing expected versioning references: {missing}')
nav = (root / 'mkdocs.yml').read_text(encoding='utf-8')
if 'Developer Documentation' not in nav:
    raise SystemExit('mkdocs.yml must expose the documentation under Developer Documentation')
print('[pk3s-cli-dev] OK    Contract checks passed')
PY_CONTRACT
}

case "${1:-help}" in
  -h|--help|help)
    usage
    ;;
  docs-build)
    run_mkdocs build --strict
    ;;
  docs-serve|docs-up)
    run_mkdocs serve --dev-addr "0.0.0.0:${DOCS_PORT}"
    ;;
  docs-down)
    log INFO "No long-running documentation container is managed by this script yet"
    ;;
  docs-clean)
    rm -rf "${REPO_DIR}/site"
    log OK "Documentation output removed"
    ;;
  test-static)
    run_static_checks
    ;;
  test-contract)
    run_contract_checks
    ;;
  test-productive-k3s-cli)
    run_static_checks
    run_contract_checks
    run_mkdocs build --strict
    ;;
  *)
    log ERROR "unsupported command: $1"
    usage >&2
    exit 2
    ;;
esac
