
#!/usr/bin/env python3
import json
import sys
from pathlib import Path
from datetime import datetime, timezone

artifacts_dir = Path(sys.argv[1])
results = []

for path in sorted(artifacts_dir.glob("*-contract.json")):
    try:
        results.append(json.loads(path.read_text(encoding="utf-8")))
    except Exception as exc:
        results.append({
            "schema_version": "productive-k3s-cli-test-result/v1",
            "name": path.name,
            "status": "invalid",
            "message": str(exc),
        })

counts = {}
for result in results:
    status = result.get("status", "unknown")
    if status == "passed":
        status = "pass"
    counts[status] = counts.get(status, 0) + 1

summary = {
    "schema_version": "productive-k3s-cli-test-summary/v1",
    "generated_at": datetime.now(timezone.utc).isoformat(),
    "total": len(results),
    "counts": counts,
    "results": results,
}

print(json.dumps(summary, indent=2, sort_keys=True))
