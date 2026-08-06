
#!/usr/bin/env bash
set -euo pipefail
TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ARTIFACTS_DIR="${TEST_ARTIFACTS_DIR:-${TESTS_DIR}/../test-artifacts}"
ARTIFACT="${ARTIFACTS_DIR}/core-command-mapping-contract.json"
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
    "pk3s doctor --core": "./productive-k3s-core.sh preflight",
    "pk3s install --core-only": "./productive-k3s-core.sh apply",
    "pk3s validate --core": "./productive-k3s-core.sh validate",
    "pk3s backup --core": "./productive-k3s-core.sh backup",
    "pk3s addon validate --tgz ./demo-addon.tgz": "./productive-k3s-core.sh addon validate --tgz <file>",
    "pk3s addon install --tgz ./demo-addon.tgz": "./productive-k3s-core.sh addon install --tgz <file>",
    "pk3s addon export --tgz ./demo-addon.tgz --output ./bundle": "./productive-k3s-core.sh addon export --tgz <file> --output <path>",
    "pk3s stack export --tgz ./demo-stack.tgz --output ./bundle": "./productive-k3s-core.sh stack export --tgz <file> --output <path>",
    "pk3s bundle core info --json": "./productive-k3s-core.sh bundle info --json"
  }
}
open(path, "w", encoding="utf-8").write(json.dumps(result, indent=2, sort_keys=True) + "\n")
PY
