#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
ARTIFACTS_DIR="${TEST_ARTIFACTS_DIR:-${REPO_DIR}/test-artifacts}"
CONTRACT_SUMMARY="${ARTIFACTS_DIR}/summary.json"
LIVE_SUMMARY="${ARTIFACTS_DIR}/live-summary.json"
LIVE_RUNS_DIR="${ARTIFACTS_DIR}/cli-live-runs"
TEST_SCOPE="${TEST_SCOPE:-}"

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    printf '[ERROR] Missing required command: %s\n' "$1" >&2
    exit 1
  }
}

print_contract_summary() {
  [[ -f "${CONTRACT_SUMMARY}" ]] || return 0
  python3 - "${CONTRACT_SUMMARY}" <<'PY'
import json, sys
path = sys.argv[1]
payload = json.load(open(path, encoding="utf-8"))
counts = payload.get("counts", {})
print("[INFO] contract summary")
print("  pass=%s skip=%s fail=%s invalid=%s total=%s" % (
    counts.get("pass", 0),
    counts.get("skip", 0),
    counts.get("fail", 0),
    counts.get("invalid", 0),
    payload.get("total", 0),
))
for result in payload.get("results", []):
    status = result.get("status", "unknown")
    name = result.get("name", "unknown")
    print(f"  [{status.upper()}] {name}")
PY
}

print_live_summary() {
  [[ -f "${LIVE_SUMMARY}" ]] || return 0
  python3 - "${LIVE_SUMMARY}" <<'PY'
import json, sys
path = sys.argv[1]
payload = json.load(open(path, encoding="utf-8"))
counts = payload.get("counts", {})
print("[INFO] live summary")
print("  pass=%s skip=%s fail=%s invalid=%s total=%s" % (
    counts.get("pass", 0),
    counts.get("skip", 0),
    counts.get("fail", 0),
    counts.get("invalid", 0),
    payload.get("total", 0),
))
for result in payload.get("results", []):
    status = result.get("result", "unknown")
    scenario = result.get("scenario", result.get("name", "unknown"))
    duration = result.get("duration_seconds", "n/a")
    print(f"  [{status.upper()}] live scenario={scenario} duration={duration}s")
PY
}

main() {
  need_cmd python3

  if [[ ! -d "${ARTIFACTS_DIR}" ]]; then
    printf '[WARN] No test artifacts found in %s\n' "${ARTIFACTS_DIR}" >&2
    exit 1
  fi

  case "${TEST_SCOPE}" in
    ""|all)
      print_contract_summary
      print_live_summary
      ;;
    contract)
      print_contract_summary
      ;;
    live)
      print_live_summary
      ;;
    *)
      printf '[ERROR] Unsupported TEST_SCOPE=%s; use contract, live, or all\n' "${TEST_SCOPE}" >&2
      exit 1
      ;;
  esac

  if [[ "${TEST_SCOPE:-all}" != "live" && ! -f "${CONTRACT_SUMMARY}" ]] && [[ "${TEST_SCOPE:-all}" != "contract" || ! -d "${LIVE_RUNS_DIR}" ]]; then
    printf '[WARN] Contract summary not found in %s\n' "${ARTIFACTS_DIR}" >&2
  fi

  if [[ "${TEST_SCOPE:-all}" != "contract" && ! -f "${LIVE_SUMMARY}" ]] && [[ "${TEST_SCOPE:-all}" != "live" || ! -f "${CONTRACT_SUMMARY}" ]]; then
    printf '[WARN] Live summary not found in %s\n' "${ARTIFACTS_DIR}" >&2
  fi

  if [[ -d "${LIVE_RUNS_DIR}" ]]; then
    local count
    count="$(find "${LIVE_RUNS_DIR}" -maxdepth 1 -type f -name '*.json' | wc -l | tr -d ' ')"
    printf '[INFO] live run manifests: %s\n' "${count}"
  fi
}

main "$@"
