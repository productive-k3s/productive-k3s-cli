#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
GO_BIN="${GO_BIN:-$(command -v go || true)}"
DIST_DIR="${REPO_DIR}/dist"
BIN_NAME="pk3s"

# shellcheck disable=SC1091
source "${SCRIPT_DIR}/release-config.sh"

usage() {
  cat <<'EOF'
Usage:
  ./scripts/build-cli.sh <command>

Commands:
  build-local
  build-release
EOF
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    printf 'Missing required command: %s\n' "$1" >&2
    exit 1
  }
}

need_cmd "$GO_BIN"

resolve_version() {
  if [[ -n "${PK3S_CLI_VERSION:-}" ]]; then
    printf '%s\n' "${PK3S_CLI_VERSION}"
    return
  fi
  if version="$(git -C "${REPO_DIR}" describe --tags --exact-match 2>/dev/null)"; then
    printf '%s\n' "${version}"
    return
  fi
  printf '%s\n' "${PK3S_CLI_VERSION_DEFAULT}"
}

VERSION="$(resolve_version)"

go_build() {
  local output_path="$1"
  shift || true
  (
    cd "${REPO_DIR}"
    PATH="$(dirname "${GO_BIN}"):${PATH}" \
      "${GO_BIN}" build \
      -ldflags "-X github.com/productive-k3s/productive-k3s-cli/internal/app.Version=${VERSION}" \
      -o "${output_path}" \
      "$@"
  )
}

build_local() {
  go_build "${REPO_DIR}/${BIN_NAME}" .
}

archive_for() {
  local goos="$1"
  local goarch="$2"
  local ext=""
  if [[ "${goos}" == "windows" ]]; then
    ext=".exe"
  fi

  local target_dir="${DIST_DIR}/${goos}-${goarch}"
  local binary_path="${target_dir}/${BIN_NAME}${ext}"
  mkdir -p "${target_dir}"

  (
    export GOOS="${goos}" GOARCH="${goarch}"
    go_build "${binary_path}" .
  )

  local archive_base="pk3s-${VERSION}-${goos}-${goarch}"
  if [[ "${goos}" == "windows" ]]; then
    (
      cd "${target_dir}"
      zip -q "${DIST_DIR}/${archive_base}.zip" "${BIN_NAME}${ext}"
    )
  else
    tar -czf "${DIST_DIR}/${archive_base}.tar.gz" -C "${target_dir}" "${BIN_NAME}${ext}"
  fi
}

build_release() {
  need_cmd tar
  need_cmd zip
  rm -rf "${DIST_DIR}"
  mkdir -p "${DIST_DIR}"
  archive_for linux amd64
  archive_for linux arm64
  archive_for darwin amd64
  archive_for darwin arm64
  archive_for windows amd64
}

if (($# == 0)); then
  usage >&2
  exit 1
fi

case "$1" in
  build-local)
    build_local
    ;;
  build-release)
    build_release
    ;;
  -h|--help|help)
    usage
    ;;
  *)
    printf 'Unsupported build command: %s\n' "$1" >&2
    usage >&2
    exit 1
    ;;
esac
