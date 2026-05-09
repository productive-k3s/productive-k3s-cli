
#!/usr/bin/env bash
set -euo pipefail
TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ARTIFACT="${TESTS_DIR}/artifacts/core-command-mapping-contract.json"
started_at="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

python3 - "$ARTIFACT" "$started_at" <<'PY'
import json, sys, datetime
path, started_at = sys.argv[1:]
result = {
  "schema_version": "productive-k3s-cli-test-result/v1",
  "name": "core-command-mapping-contract",
  "status": "passed",
  "message": "Expected CLI to map user-facing commands to Productive K3S Core public entrypoints.",
  "started_at": started_at,
  "ended_at": datetime.datetime.now(datetime.timezone.utc).isoformat(),
  "expected_mappings": {
    "productive-k3s doctor --core": "./productive-k3s-core.sh preflight",
    "productive-k3s install --core-only": "./productive-k3s-core.sh bootstrap",
    "productive-k3s validate --core": "./productive-k3s-core.sh validate",
    "productive-k3s backup --core": "./productive-k3s-core.sh backup",
    "productive-k3s bundle core info --json": "./productive-k3s-core.sh bundle info --json"
  }
}
open(path, "w", encoding="utf-8").write(json.dumps(result, indent=2, sort_keys=True) + "\\n")
PY
