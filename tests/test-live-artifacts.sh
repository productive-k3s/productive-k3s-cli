#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP_DIR="$(mktemp -d)"
RUNS_DIR="${TMP_DIR}/live-runs"
SUMMARY_FILE="${TMP_DIR}/summary.json"

cleanup() {
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

mkdir -p "${RUNS_DIR}"

cat > "${RUNS_DIR}/pass.json" <<'EOF'
{
  "schema_version": "productive-k3s-cli-live-result/v1",
  "scenario": "multipass",
  "result": "pass"
}
EOF

cat > "${RUNS_DIR}/fail.json" <<'EOF'
{
  "schema_version": "productive-k3s-cli-live-result/v1",
  "scenario": "onprem-basic",
  "result": "fail"
}
EOF

python3 "${ROOT_DIR}/tests/summarize-live-artifacts.py" "${RUNS_DIR}" > "${SUMMARY_FILE}"

grep -Fq '"schema_version": "productive-k3s-cli-live-summary/v1"' "${SUMMARY_FILE}" || {
  printf '[FAIL] live summary schema version missing\n' >&2
  exit 1
}

grep -Fq '"pass": 1' "${SUMMARY_FILE}" || {
  printf '[FAIL] pass count missing from live summary\n' >&2
  exit 1
}

grep -Fq '"fail": 1' "${SUMMARY_FILE}" || {
  printf '[FAIL] fail count missing from live summary\n' >&2
  exit 1
}

printf '[PASS] live artifact summary tool aggregates run manifests\n'
