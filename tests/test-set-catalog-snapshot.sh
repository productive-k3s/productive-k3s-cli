#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

cp -a "${ROOT_DIR}/." "${TMP_DIR}/cli"
DIGEST="$(printf 'a%.0s' {1..64})"
(
  cd "${TMP_DIR}/cli"
  bash scripts/set-catalog-snapshot.sh 0.9.65 "${DIGEST}"
)

grep -Fq 'PRODUCTIVE_K3S_CATALOG_URL_DEFAULT:=https://catalogs.productive-k3s.io/catalogs/0.9.65/index.yaml' "${TMP_DIR}/cli/scripts/release-config.sh"
grep -Fq "PRODUCTIVE_K3S_CATALOG_SHA256_DEFAULT:=${DIGEST}" "${TMP_DIR}/cli/scripts/release-config.sh"
grep -Fq 'version: 0.9.65' "${TMP_DIR}/cli/materials.lock.yaml"
grep -Fq 'pinPolicy: exact-digest' "${TMP_DIR}/cli/materials.lock.yaml"

if (cd "${TMP_DIR}/cli" && bash scripts/set-catalog-snapshot.sh invalid "${DIGEST}" >/dev/null 2>&1); then
  echo '[FAIL] invalid catalog version was accepted' >&2
  exit 1
fi

printf '[PASS] catalog snapshot pin updates release config and materials lock\n'
