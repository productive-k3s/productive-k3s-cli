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

if [[ ! "${PK3S_CLI_VERSION}" =~ ^[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]]; then
  printf 'PK3S_CLI_VERSION must be an explicit release version, got: %s\n' "${PK3S_CLI_VERSION}" >&2
  exit 1
fi
if [[ ! "${PK3S_CLI_REPO}" =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]]; then
  printf 'PK3S_CLI_REPO must use the owner/repository form.\n' >&2
  exit 1
fi

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
need_cmd awk
need_cmd tr
need_cmd install

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
RELEASE_URL="https://github.com/${PK3S_CLI_REPO}/releases/download/${PK3S_CLI_VERSION}"
URL="${RELEASE_URL}/${ARCHIVE}"
CHECKSUMS_URL="${RELEASE_URL}/checksums.txt"
umask 077
WORK_DIR="$(mktemp -d)"
TARGET_TMP=""
trap 'rm -rf "${WORK_DIR}"; [[ -z "${TARGET_TMP}" ]] || rm -f "${TARGET_TMP}"' EXIT

download() {
  curl \
    --fail \
    --silent \
    --show-error \
    --location \
    --proto '=https' \
    --proto-redir '=https' \
    "$1" \
    -o "$2"
}

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
    return
  fi
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
    return
  fi
  printf 'Missing required command: sha256sum or shasum\n' >&2
  exit 1
}

download "${URL}" "${WORK_DIR}/${ARCHIVE}"
download "${CHECKSUMS_URL}" "${WORK_DIR}/checksums.txt"

checksum_matches="$(awk -v archive="${ARCHIVE}" '$2 == archive {print}' "${WORK_DIR}/checksums.txt")"
checksum_count="$(printf '%s\n' "${checksum_matches}" | awk 'NF {count++} END {print count + 0}')"
if [[ "${checksum_count}" -ne 1 ]]; then
  printf 'checksums.txt must contain exactly one entry for %s\n' "${ARCHIVE}" >&2
  exit 1
fi
read -r expected_digest checksum_name extra <<<"${checksum_matches}"
if [[ -n "${extra:-}" || "${checksum_name:-}" != "${ARCHIVE}" || ! "${expected_digest:-}" =~ ^[0-9A-Fa-f]{64}$ ]]; then
  printf 'Malformed checksum entry for %s\n' "${ARCHIVE}" >&2
  exit 1
fi

actual_digest="$(sha256_file "${WORK_DIR}/${ARCHIVE}" | tr 'A-F' 'a-f')"
expected_digest="$(printf '%s' "${expected_digest}" | tr 'A-F' 'a-f')"
if [[ "${actual_digest}" != "${expected_digest}" ]]; then
  printf 'Checksum verification failed for %s\n' "${ARCHIVE}" >&2
  exit 1
fi

if ! archive_entries="$(tar -tzf "${WORK_DIR}/${ARCHIVE}" 2>/dev/null)"; then
  printf 'Release archive must contain exactly one pk3s binary.\n' >&2
  exit 1
fi
archive_entry_count="$(printf '%s\n' "${archive_entries}" | awk 'NF {count++} END {print count + 0}')"
if [[ "${archive_entry_count}" -ne 1 || "${archive_entries}" != "pk3s" ]]; then
  printf 'Release archive must contain exactly one pk3s binary.\n' >&2
  exit 1
fi
tar -xzf "${WORK_DIR}/${ARCHIVE}" -C "${WORK_DIR}"
if [[ ! -f "${WORK_DIR}/pk3s" || -L "${WORK_DIR}/pk3s" ]]; then
  printf 'Release archive does not contain a regular pk3s binary.\n' >&2
  exit 1
fi

mkdir -p "${PK3S_CLI_INSTALL_DIR}"
TARGET_TMP="$(mktemp "${PK3S_CLI_INSTALL_DIR}/.pk3s.install.XXXXXX")"
install -m 0755 "${WORK_DIR}/pk3s" "${TARGET_TMP}"
mv -f "${TARGET_TMP}" "${PK3S_CLI_INSTALL_DIR}/pk3s"
TARGET_TMP=""

printf 'Installed pk3s to %s/pk3s\n' "${PK3S_CLI_INSTALL_DIR}"
