#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

if [[ -f "${REPO_ROOT}/scripts/release-config.sh" ]]; then
  # shellcheck disable=SC1091
  source "${REPO_ROOT}/scripts/release-config.sh"
fi

PK3S_CLI_VERSION="${PK3S_CLI_VERSION:-${PK3S_CLI_VERSION_DEFAULT:-1.0.0}}"
PK3S_CLI_REPO="${PK3S_CLI_REPO:-${PRODUCTIVE_K3S_CLI_REPO_DEFAULT:-productive-k3s/productive-k3s-cli}}"
PK3S_CLI_INSTALL_DIR="${PK3S_CLI_INSTALL_DIR:-$HOME/.local/bin}"

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    printf 'Missing required command: %s\n' "$1" >&2
    exit 1
  }
}

need_cmd curl
need_cmd tar
need_cmd uname
need_cmd mktemp

detect_os() {
  case "$(uname -s)" in
    Linux) printf 'linux\n' ;;
    Darwin) printf 'darwin\n' ;;
    *)
      printf 'Unsupported OS for this installer. Use the Windows release asset manually.\n' >&2
      exit 1
      ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) printf 'amd64\n' ;;
    arm64|aarch64) printf 'arm64\n' ;;
    *)
      printf 'Unsupported architecture: %s\n' "$(uname -m)" >&2
      exit 1
      ;;
  esac
}

OS="$(detect_os)"
ARCH="$(detect_arch)"
ARCHIVE="pk3s-${PK3S_CLI_VERSION}-${OS}-${ARCH}.tar.gz"
URL="https://github.com/${PK3S_CLI_REPO}/releases/download/${PK3S_CLI_VERSION}/${ARCHIVE}"
WORK_DIR="$(mktemp -d)"
trap 'rm -rf "${WORK_DIR}"' EXIT

mkdir -p "${PK3S_CLI_INSTALL_DIR}"
curl -fsSL "${URL}" -o "${WORK_DIR}/${ARCHIVE}"
tar -xzf "${WORK_DIR}/${ARCHIVE}" -C "${WORK_DIR}"
install -m 0755 "${WORK_DIR}/pk3s" "${PK3S_CLI_INSTALL_DIR}/pk3s"

printf 'Installed pk3s to %s/pk3s\n' "${PK3S_CLI_INSTALL_DIR}"
