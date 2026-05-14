
#!/usr/bin/env bash
set -euo pipefail
TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ARTIFACTS_DIR="${TEST_ARTIFACTS_DIR:-${TESTS_DIR}/../test-artifacts}"
source "${TESTS_DIR}/lib/json.sh"
ARTIFACT="${ARTIFACTS_DIR}/bundle-resolution-contract.json"
MANIFEST="${TESTS_DIR}/fixtures/manifests/cli-1.0.0.json"
started_at="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

python3 - "$MANIFEST" <<'PY'
import json, re, sys
m=json.load(open(sys.argv[1], encoding="utf-8"))
assert re.match(r"^[0-9]+\.[0-9]+\.[0-9]+$", m["cli_version"])
assert re.match(r"^[0-9]+\.[0-9]+\.[0-9]+$", m["core"]["bundle_version"])
assert re.match(r"^[0-9]+\.[0-9]+\.[0-9]+-[0-9]+\.[0-9]+\.[0-9]+$", m["infra"]["bundle_version"])
infra_core = m["infra"]["bundle_version"].split("-",1)[1]
assert infra_core == m["core"]["bundle_version"]
PY

write_result "${ARTIFACT}" "bundle-resolution-contract" "passed" "CLI manifest resolves one core bundle and one infra bundle with matching dependency." "${started_at}" "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
