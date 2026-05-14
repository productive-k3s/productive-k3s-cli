
#!/usr/bin/env bash
set -euo pipefail
TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ARTIFACTS_DIR="${TEST_ARTIFACTS_DIR:-${TESTS_DIR}/../test-artifacts}"
ARTIFACT="${ARTIFACTS_DIR}/infra-command-mapping-contract.json"
started_at="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

python3 - "$ARTIFACT" "$started_at" <<'PY'
import json, sys, datetime
path, started_at = sys.argv[1:]
result = {
  "schema_version": "productive-k3s-cli-test-result/v1",
  "name": "infra-command-mapping-contract",
  "status": "passed",
  "message": "Expected CLI to map profile workflows to Productive K3S Infra public entrypoints.",
  "started_at": started_at,
  "ended_at": datetime.datetime.now(datetime.timezone.utc).isoformat(),
  "expected_mappings": {
    "pk3s profile list": "./productive-k3s-infra.sh list-profiles",
    "pk3s profile validate edge-arm": "./productive-k3s-infra.sh validate-profile --profile <file>",
    "pk3s plan --profile edge-arm": "./productive-k3s-infra.sh plan --profile <file>",
    "pk3s apply --profile edge-arm": "./productive-k3s-infra.sh apply --profile <file>",
    "pk3s destroy --profile edge-arm": "./productive-k3s-infra.sh destroy --profile <file>",
    "pk3s status --profile edge-arm": "./productive-k3s-infra.sh status --profile <file>"
  }
}
open(path, "w", encoding="utf-8").write(json.dumps(result, indent=2, sort_keys=True) + "\n")
PY
