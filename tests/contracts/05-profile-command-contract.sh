
#!/usr/bin/env bash
set -euo pipefail
TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${TESTS_DIR}/lib/json.sh"
ARTIFACT="${TESTS_DIR}/artifacts/profile-command-contract.json"
PROFILE="${TESTS_DIR}/fixtures/profiles/edge-arm.env"
started_at="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

python3 - "$PROFILE" <<'PY'
import sys
required = {"PK3S_INFRA_PROFILE_NAME", "PK3S_INFRA_ENGINE", "PK3S_INFRA_SCENARIO"}
values = {}
for line in open(sys.argv[1], encoding="utf-8"):
    line=line.strip()
    if not line or line.startswith("#") or "=" not in line:
        continue
    k,v=line.split("=",1)
    values[k]=v
missing = required - values.keys()
assert not missing, missing
assert values["PK3S_INFRA_ENGINE"] in {"opentofu", "ansible", "shell"}
PY

write_result "${ARTIFACT}" "profile-command-contract" "passed" "Profile fixture contains required infra contract variables." "${started_at}" "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
