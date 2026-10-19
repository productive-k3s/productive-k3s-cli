#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

VERSION="9.9.9"
case "$(uname -s)" in
  Linux) OS=linux ;;
  Darwin) OS=darwin ;;
  *) printf 'unsupported test OS\n' >&2; exit 1 ;;
esac
case "$(uname -m)" in
  x86_64|amd64) ARCH=amd64 ;;
  arm64|aarch64) ARCH=arm64 ;;
  *) printf 'unsupported test architecture\n' >&2; exit 1 ;;
esac
ARCHIVE="pk3s-${VERSION}-${OS}-${ARCH}.tar.gz"
FIXTURES="${TMP_DIR}/fixtures"
FAKE_BIN="${TMP_DIR}/bin"
INSTALL_DIR="${TMP_DIR}/install"
mkdir -p "${FIXTURES}" "${FAKE_BIN}" "${INSTALL_DIR}" "${TMP_DIR}/payload"
printf '#!/usr/bin/env bash\nprintf "verified installer fixture\\n"\n' >"${TMP_DIR}/payload/pk3s"
chmod 0755 "${TMP_DIR}/payload/pk3s"
tar -czf "${FIXTURES}/${ARCHIVE}" -C "${TMP_DIR}/payload" pk3s

cat >"${FAKE_BIN}/curl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >>"${CURL_ARGS_LOG}"
url=""
output=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    -o)
      output="$2"
      shift 2
      ;;
    https://*)
      url="$1"
      shift
      ;;
    *)
      shift
      ;;
  esac
done
cp "${FIXTURE_DIR}/$(basename "${url}")" "${output}"
EOF
chmod 0755 "${FAKE_BIN}/curl"

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

write_valid_checksum() {
  printf '%s  %s\n' "$(sha256_file "${FIXTURES}/${ARCHIVE}")" "${ARCHIVE}" >"${FIXTURES}/checksums.txt"
}

run_installer() {
  CURL_ARGS_LOG="${TMP_DIR}/curl-args.log" \
  FIXTURE_DIR="${FIXTURES}" \
  PATH="${FAKE_BIN}:${PATH}" \
  PK3S_CLI_VERSION="${VERSION}" \
  PK3S_CLI_REPO="productive-k3s/productive-k3s-cli" \
  PK3S_CLI_INSTALL_DIR="${INSTALL_DIR}" \
    bash "${ROOT_DIR}/internal/install/install.sh"
}

expect_failure_preserves_binary() {
  local expected="$1"
  printf 'existing binary\n' >"${INSTALL_DIR}/pk3s"
  if run_installer >"${TMP_DIR}/stdout" 2>"${TMP_DIR}/stderr"; then
    printf 'installer unexpectedly succeeded: %s\n' "${expected}" >&2
    exit 1
  fi
  grep -F "${expected}" "${TMP_DIR}/stderr" >/dev/null
  grep -Fx 'existing binary' "${INSTALL_DIR}/pk3s" >/dev/null
}

write_valid_checksum
run_installer >/dev/null
grep -F 'verified installer fixture' "${INSTALL_DIR}/pk3s" >/dev/null
grep -F -- "--proto =https" "${TMP_DIR}/curl-args.log" >/dev/null
grep -F -- "--proto-redir =https" "${TMP_DIR}/curl-args.log" >/dev/null

printf '%064d  other.tar.gz\n' 0 >"${FIXTURES}/checksums.txt"
expect_failure_preserves_binary "exactly one entry"

write_valid_checksum
cat "${FIXTURES}/checksums.txt" >>"${FIXTURES}/checksums.txt.duplicate"
cat "${FIXTURES}/checksums.txt" >>"${FIXTURES}/checksums.txt.duplicate"
mv "${FIXTURES}/checksums.txt.duplicate" "${FIXTURES}/checksums.txt"
expect_failure_preserves_binary "exactly one entry"

printf 'not-a-digest  %s\n' "${ARCHIVE}" >"${FIXTURES}/checksums.txt"
expect_failure_preserves_binary "Malformed checksum entry"

printf '%064d  %s\n' 0 "${ARCHIVE}" >"${FIXTURES}/checksums.txt"
expect_failure_preserves_binary "Checksum verification failed"

printf 'not a gzip archive\n' >"${FIXTURES}/${ARCHIVE}"
write_valid_checksum
expect_failure_preserves_binary "Release archive must contain exactly one pk3s binary"

printf 'installer security tests passed\n'
