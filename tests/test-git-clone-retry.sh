#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck disable=SC1091
source "${ROOT_DIR}/tests/lib/git-clone.sh"

WORK_DIR="$(mktemp -d)"
trap 'rm -rf "${WORK_DIR}"' EXIT
FAKE_BIN="${WORK_DIR}/bin"
COUNTER_FILE="${WORK_DIR}/counter"
DESTINATION="${WORK_DIR}/checkout"
mkdir -p "${FAKE_BIN}"

cat >"${FAKE_BIN}/git" <<'FAKE'
#!/usr/bin/env bash
set -euo pipefail
counter_file="${PK3S_TEST_GIT_COUNTER_FILE:?}"
count=0
[[ ! -f "${counter_file}" ]] || count="$(cat "${counter_file}")"
count=$((count + 1))
printf '%s\n' "${count}" >"${counter_file}"
destination="${@: -1}"
mkdir -p "${destination}"
if (( count <= PK3S_TEST_GIT_FAIL_UNTIL )); then
  printf 'simulated clone failure %d\n' "${count}" >&2
  printf 'partial\n' >"${destination}/partial"
  exit 1
fi
[[ ! -e "${destination}/partial" ]] || exit 3
printf 'ok\n' >"${destination}/success"
FAKE
chmod +x "${FAKE_BIN}/git"

export PATH="${FAKE_BIN}:${PATH}"
export PK3S_TEST_GIT_COUNTER_FILE="${COUNTER_FILE}"
export PK3S_GIT_CLONE_MAX_ATTEMPTS=3
export PK3S_GIT_CLONE_RETRY_DELAY_SECONDS=0
export PK3S_TEST_GIT_FAIL_UNTIL=2

pk3s_git_clone_with_retry https://example.invalid/repo.git development "${DESTINATION}" demo 2>"${WORK_DIR}/retry.log"
[[ "$(cat "${COUNTER_FILE}")" == "3" ]]
[[ -f "${DESTINATION}/success" ]]
[[ ! -e "${DESTINATION}/partial" ]]
grep -Fq 'attempt 1/3' "${WORK_DIR}/retry.log"
grep -Fq 'attempt 2/3' "${WORK_DIR}/retry.log"

printf '0\n' >"${COUNTER_FILE}"
export PK3S_TEST_GIT_FAIL_UNTIL=9
if pk3s_git_clone_with_retry https://example.invalid/repo.git development "${DESTINATION}" demo 2>"${WORK_DIR}/failure.log"; then
  printf '[FAIL] expected exhausted clone retries to fail\n' >&2
  exit 1
fi
[[ ! -e "${DESTINATION}" ]]
grep -Fq 'after 3 attempts' "${WORK_DIR}/failure.log"
grep -Fq 'simulated clone failure 3' "${WORK_DIR}/failure.log"

printf '[PASS] git clone retries clean partial checkouts and preserve final diagnostics\n'
