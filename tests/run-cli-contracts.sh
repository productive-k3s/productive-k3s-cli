
#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TESTS_DIR="${ROOT_DIR}/tests"
ARTIFACTS_DIR="${TESTS_DIR}/artifacts"

mkdir -p "${ARTIFACTS_DIR}"

status=0

for test_script in "${TESTS_DIR}"/contracts/*.sh; do
  echo "[contracts] running ${test_script}"
  if ! bash "${test_script}"; then
    status=1
  fi
done

python3 "${TESTS_DIR}/summarize-artifacts.py" "${ARTIFACTS_DIR}" > "${ARTIFACTS_DIR}/summary.json"

echo "[contracts] summary: ${ARTIFACTS_DIR}/summary.json"
exit "${status}"
