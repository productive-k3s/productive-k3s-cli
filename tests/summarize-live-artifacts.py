#!/usr/bin/env python3
import json
import sys
from datetime import datetime, timezone
from pathlib import Path


artifacts_dir = Path(sys.argv[1])
results = []

for path in sorted(artifacts_dir.glob("*.json")):
    try:
        payload = json.loads(path.read_text(encoding="utf-8"))
    except Exception as exc:
        payload = {
            "schema_version": "productive-k3s-cli-live-result/v1",
            "name": path.name,
            "result": "invalid",
            "message": str(exc),
        }
    results.append(payload)

counts = {}
for result in results:
    status = result.get("result", "unknown")
    counts[status] = counts.get(status, 0) + 1

summary = {
    "schema_version": "productive-k3s-cli-live-summary/v1",
    "generated_at": datetime.now(timezone.utc).isoformat(),
    "total": len(results),
    "counts": counts,
    "results": results,
}

print(json.dumps(summary, indent=2, sort_keys=True))
