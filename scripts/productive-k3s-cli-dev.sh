#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

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
  test-clean
  test-checkstatus
  test-static
  test-live-remote
  test-live-gha-onprem-remote
  set-bundles-versions
  tag-release
USAGE
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
  test-clean)
    exec bash "${REPO_DIR}/tests/clean-test-state.sh" "$@"
    ;;
  test-checkstatus)
    exec bash "${REPO_DIR}/tests/check-test-status.sh" "$@"
    ;;
  test-static)
    PATH=/usr/local/go/bin:$PATH /usr/local/go/bin/go test ./...
    bash "${REPO_DIR}/tests/test-release-versioning.sh"
    bash "${REPO_DIR}/tests/test-set-bundles-versions.sh"
    bash "${REPO_DIR}/tests/test-tag-release.sh"
    bash "${REPO_DIR}/tests/test-live-artifacts.sh"
    bash "${REPO_DIR}/tests/test-live-wiring.sh"
    exec bash "${REPO_DIR}/tests/run-cli-contracts.sh"
    ;;
  test-live-remote)
    exec bash "${REPO_DIR}/tests/run-cli-live.sh" "$@"
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
