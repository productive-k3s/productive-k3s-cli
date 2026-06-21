#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
GO_BIN="${GO_BIN:-$(command -v go || true)}"

usage() {
  cat <<'USAGE'
Usage:
  ./scripts/productive-k3s-cli-dev.sh <command> [args...]

Development commands:
  docs-build
  docs-serve
  docs-up
  docs-down
  docs-clean
  docs-publish-check
  test-local-all
  test-clean
  test-checkstatus
  test-live-remote
  test-live-catalog
  test-live-gha-onprem-remote
  set-bundles-versions
  tag-release
USAGE
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "Missing required command: $1" >&2
    exit 1
  }
}

if (($# == 0)); then
  usage >&2
  exit 1
fi

COMMAND="$1"
shift || true

case "$COMMAND" in
  docs-build)
    exec "${REPO_DIR}/docs/build.sh" "$@"
    ;;
  docs-serve)
    exec "${REPO_DIR}/docs/serve.sh" "$@"
    ;;
  docs-up)
    exec "${REPO_DIR}/docs/serve.sh" --background "$@"
    ;;
  docs-down|docs-clean)
    exec "${REPO_DIR}/docs/clean.sh" "$@"
    ;;
  docs-publish-check)
    bash "${REPO_DIR}/tests/test-docs-publishing.sh"
    exec bash "${REPO_DIR}/tests/test-docs-structure.sh" "$@"
    ;;
  test-local-all)
    exec env GO_BIN="${GO_BIN}" make -C "${REPO_DIR}/tests" test-static-raw
    ;;
  test-clean)
    exec bash "${REPO_DIR}/tests/clean-test-state.sh" "$@"
    ;;
  test-checkstatus)
    exec bash "${REPO_DIR}/tests/check-test-status.sh" "$@"
    ;;
  test-live-remote)
    exec bash "${REPO_DIR}/tests/run-cli-live.sh" "$@"
    ;;
  test-live-catalog)
    exec bash "${REPO_DIR}/tests/run-cli-live.sh" catalog-multipass "$@"
    ;;
  test-live-gha-onprem-remote)
    exec bash "${REPO_DIR}/tests/live-cli-onprem-remote-github-host.sh" "$@"
    ;;
  set-bundles-versions)
    exec "${REPO_DIR}/scripts/set-bundles-versions.sh" "$@"
    ;;
  tag-release)
    exec "${REPO_DIR}/scripts/tag-release.sh" "$@"
    ;;
  -h|--help|help)
    usage
    ;;
  *)
    echo "Unsupported development command: ${COMMAND}" >&2
    usage >&2
    exit 1
    ;;
esac
