#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${1:-}"
DIGEST="${2:-}"

[[ "${VERSION}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || {
  echo "usage: $0 <catalog-version> <sha256>" >&2
  exit 2
}
[[ "${DIGEST}" =~ ^[0-9a-f]{64}$ ]] || {
  echo "catalog digest must be a lowercase SHA-256" >&2
  exit 2
}

URL="https://catalogs.productive-k3s.io/catalogs/${VERSION}/index.yaml"
python3 - "${ROOT_DIR}/scripts/release-config.sh" "${ROOT_DIR}/internal/bundles/defaults.go" "${ROOT_DIR}/materials.lock.yaml" "${ROOT_DIR}/tests/test-release-versioning.sh" "${VERSION}" "${URL}" "${DIGEST}" <<'PY'
import re
import sys
from pathlib import Path

config_path = Path(sys.argv[1])
defaults_path = Path(sys.argv[2])
materials_path = Path(sys.argv[3])
release_test_path = Path(sys.argv[4])
version, url, digest = sys.argv[5:]

config = config_path.read_text(encoding="utf-8")
config = re.sub(
    r'PRODUCTIVE_K3S_CATALOG_URL_DEFAULT:=.*}',
    f'PRODUCTIVE_K3S_CATALOG_URL_DEFAULT:={url}}}',
    config,
)
config = re.sub(
    r'PRODUCTIVE_K3S_CATALOG_SHA256_DEFAULT:=.*}',
    f'PRODUCTIVE_K3S_CATALOG_SHA256_DEFAULT:={digest}}}',
    config,
)
config_path.write_text(config, encoding="utf-8")

defaults = defaults_path.read_text(encoding="utf-8")
defaults, url_count = re.subn(
    r'envOrDefault\("PRODUCTIVE_K3S_CATALOG_URL_DEFAULT", "[^"]+"\)',
    f'envOrDefault("PRODUCTIVE_K3S_CATALOG_URL_DEFAULT", "{url}")',
    defaults,
)
defaults, digest_count = re.subn(
    r'return "[0-9a-f]{64}"(?=\n\})',
    f'return "{digest}"',
    defaults,
)
if url_count != 1 or digest_count != 1:
    raise SystemExit("expected exactly one compiled catalog URL and digest default")
defaults_path.write_text(defaults, encoding="utf-8")

materials = materials_path.read_text(encoding="utf-8")
replacement = (
    '    - {id: public-catalog, type: product-artifact, '
    f'name: productive-k3s-catalog, version: {version}, source: "{url}", '
    f'pinPolicy: exact-digest, digest: "sha256:{digest}", required: true}}'
)
materials, count = re.subn(r'^    - \{id: public-catalog,.*$', replacement, materials, flags=re.MULTILINE)
if count != 1:
    raise SystemExit("expected exactly one public-catalog material")
materials_path.write_text(materials, encoding="utf-8")

release_test = release_test_path.read_text(encoding="utf-8")
release_test, url_count = re.subn(
    r'assert_eq "\$\{PRODUCTIVE_K3S_CATALOG_URL_DEFAULT\}" "[^"]+" "default catalog url"',
    f'assert_eq "${{PRODUCTIVE_K3S_CATALOG_URL_DEFAULT}}" "{url}" "default catalog url"',
    release_test,
)
release_test, digest_count = re.subn(
    r'assert_eq "\$\{PRODUCTIVE_K3S_CATALOG_SHA256_DEFAULT\}" "[^"]*" "[^"]*catalog digest"',
    f'assert_eq "${{PRODUCTIVE_K3S_CATALOG_SHA256_DEFAULT}}" "{digest}" "default catalog digest"',
    release_test,
)
if url_count != 1 or digest_count != 1:
    raise SystemExit("expected exactly one catalog URL and digest release assertion")
release_test_path.write_text(release_test, encoding="utf-8")
PY

printf 'Pinned catalog snapshot version=%s sha256=%s\n' "${VERSION}" "${DIGEST}"
