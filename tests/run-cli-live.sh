#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ARTIFACTS_DIR="${TEST_ARTIFACTS_DIR:-${ROOT_DIR}/test-artifacts}"
RUNS_DIR="${ARTIFACTS_DIR}/cli-live-runs"
RUN_TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
RUN_ID="${RUN_TIMESTAMP}-live-$$"
SCENARIOS=("$@")

if (($# == 0)); then
  SCENARIOS=("multipass" "onprem-basic")
fi

json_escape() {
  printf '%s' "$1" | sed \
    -e 's/\\/\\\\/g' \
    -e 's/"/\\"/g' \
    -e ':a;N;$!ba;s/\n/\\n/g' \
    -e 's/\r/\\r/g' \
    -e 's/\t/\\t/g'
}

json_array_from_values() {
  if (($# == 0)); then
    printf '[]'
    return 0
  fi

  local first=1
  local value
  printf '['
  for value in "$@"; do
    if (( first == 0 )); then
      printf ', '
    fi
    first=0
    printf '"%s"' "$(json_escape "${value}")"
  done
  printf ']'
}

execution_kind() {
  if [[ "${GITHUB_ACTIONS:-}" == "true" ]]; then
    printf 'ci'
  else
    printf 'manual'
  fi
}

scenario_script() {
  case "$1" in
    multipass)
      printf '%s/tests/live-cli-multipass-remote.sh\n' "${ROOT_DIR}"
      ;;
    onprem-basic)
      printf '%s/tests/live-cli-onprem-remote.sh\n' "${ROOT_DIR}"
      ;;
    *)
      return 1
      ;;
  esac
}

scenario_environment() {
  case "$1" in
    multipass)
      printf 'vm\n'
      ;;
    onprem-basic)
      printf 'on-prem\n'
      ;;
    *)
      printf 'unknown\n'
      ;;
  esac
}

scenario_topology() {
  case "$1" in
    multipass)
      printf 'three-node\n'
      ;;
    onprem-basic)
      printf 'server-agent\n'
      ;;
    *)
      printf 'unknown\n'
      ;;
  esac
}

scenario_node_count() {
  case "$1" in
    multipass)
      printf '3\n'
      ;;
    onprem-basic)
      printf '2\n'
      ;;
    *)
      printf 'unknown\n'
      ;;
  esac
}

write_run_manifest() {
  local scenario="$1"
  local result="$2"
  local started_at="$3"
  local finished_at="$4"
  local duration_seconds="$5"
  local output_file="$6"
  local log_file="$7"
  local environment
  local topology
  local node_count_expected

  environment="$(scenario_environment "${scenario}")"
  topology="$(scenario_topology "${scenario}")"
  node_count_expected="$(scenario_node_count "${scenario}")"

  mkdir -p "${RUNS_DIR}"
  {
    printf '{\n'
    printf '  "schema_version": "productive-k3s-cli-live-result/v1",\n'
    printf '  "repository": "productive-k3s-cli",\n'
    printf '  "run_id": "%s",\n' "$(json_escape "${RUN_ID}-${scenario}")"
    printf '  "scenario": "%s",\n' "$(json_escape "${scenario}")"
    printf '  "execution_kind": "%s",\n' "$(json_escape "$(execution_kind)")"
    printf '  "test_level": "live",\n'
    printf '  "result": "%s",\n' "$(json_escape "${result}")"
    printf '  "started_at": "%s",\n' "$(json_escape "${started_at}")"
    printf '  "finished_at": "%s",\n' "$(json_escape "${finished_at}")"
    printf '  "duration_seconds": %s,\n' "${duration_seconds}"
    printf '  "pk3s": {\n'
    printf '    "binary": "%s",\n' "$(json_escape "${PRODUCTIVE_K3S_CLI_BIN:-${ROOT_DIR}/pk3s}")"
    printf '    "source": "remote",\n'
    printf '    "core_version": "%s",\n' "$(json_escape "${PRODUCTIVE_K3S_CORE_VERSION_DEFAULT:-unknown}")"
    printf '    "infra_version": "%s"\n' "$(json_escape "${PRODUCTIVE_K3S_INFRA_VERSION_DEFAULT:-unknown}")"
    printf '  },\n'
    printf '  "installation": {\n'
    printf '    "environment": "%s",\n' "$(json_escape "${environment}")"
    printf '    "topology": "%s",\n' "$(json_escape "${topology}")"
    printf '    "node_count_expected": "%s"\n' "$(json_escape "${node_count_expected}")"
    printf '  },\n'
    printf '  "artifacts": {\n'
    printf '    "log_path": "%s"\n' "$(json_escape "${log_file}")"
    printf '  }\n'
    printf '}\n'
  } > "${output_file}"
}

passes=()
skips=()
fails=()

mkdir -p "${ARTIFACTS_DIR}" "${RUNS_DIR}"
set -a
# shellcheck disable=SC1090
source "${ROOT_DIR}/scripts/release-config.sh"
set +a

for scenario in "${SCENARIOS[@]}"; do
  script_path="$(scenario_script "${scenario}")" || {
    printf '[FAIL] unsupported live scenario: %s\n' "${scenario}" >&2
    exit 2
  }
  log_file="${RUNS_DIR}/${RUN_ID}-${scenario}.log"
  manifest_file="${RUNS_DIR}/${RUN_ID}-${scenario}.json"
  started_at="$(date -Iseconds)"
  started_epoch="$(date +%s)"

  printf '\n==> [live] %s\n' "${scenario}"
  set +e
  script -qefc "bash \"${script_path}\"" /dev/null | tr -d '\000' | tee "${log_file}"
  rc=${PIPESTATUS[0]}
  set -e

  finished_at="$(date -Iseconds)"
  finished_epoch="$(date +%s)"
  duration_seconds="$((finished_epoch - started_epoch))"

  if [[ "${rc}" == "0" ]]; then
    passes+=("${scenario}")
    write_run_manifest "${scenario}" "pass" "${started_at}" "${finished_at}" "${duration_seconds}" "${manifest_file}" "${log_file}"
    printf '[PASS] %s\n' "${scenario}"
  else
    if grep -q '^\[SKIP\]' "${log_file}"; then
      skips+=("${scenario}")
      write_run_manifest "${scenario}" "skip" "${started_at}" "${finished_at}" "${duration_seconds}" "${manifest_file}" "${log_file}"
      printf '[SKIP] %s\n' "${scenario}"
    else
      fails+=("${scenario}")
      write_run_manifest "${scenario}" "fail" "${started_at}" "${finished_at}" "${duration_seconds}" "${manifest_file}" "${log_file}"
      printf '[FAIL] %s\n' "${scenario}" >&2
    fi
  fi
done

python3 "${ROOT_DIR}/tests/summarize-live-artifacts.py" "${RUNS_DIR}" > "${ARTIFACTS_DIR}/${RUN_ID}-summary.json"
cp "${ARTIFACTS_DIR}/${RUN_ID}-summary.json" "${ARTIFACTS_DIR}/live-summary.json"

printf '\nLive summary\n'
printf '  pass: %s\n' "${passes[*]:-none}"
printf '  skip: %s\n' "${skips[*]:-none}"
printf '  fail: %s\n' "${fails[*]:-none}"

if (( ${#fails[@]} > 0 )); then
  exit 1
fi
