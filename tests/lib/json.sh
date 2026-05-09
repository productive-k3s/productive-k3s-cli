
#!/usr/bin/env bash
set -euo pipefail

write_result() {
  local file="$1"
  local name="$2"
  local status="$3"
  local message="$4"
  local started_at="$5"
  local ended_at="$6"

  mkdir -p "$(dirname "${file}")"
  python3 - "$file" "$name" "$status" "$message" "$started_at" "$ended_at" <<'PY'
import json, sys
path, name, status, message, started_at, ended_at = sys.argv[1:]
result = {
  "schema_version": "productive-k3s-cli-test-result/v1",
  "name": name,
  "status": status,
  "message": message,
  "started_at": started_at,
  "ended_at": ended_at,
}
open(path, "w", encoding="utf-8").write(json.dumps(result, indent=2, sort_keys=True) + "\n")
PY
}
