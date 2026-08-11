#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PK3S_BIN="${PRODUCTIVE_K3S_CLI_BIN:-${ROOT_DIR}/pk3s}"
INFRA_REPO_DIR="${PRODUCTIVE_K3S_INFRA_REPO_DIR:-${ROOT_DIR}/../productive-k3s-infra}"
PROFILES_REPO_DIR="${PRODUCTIVE_K3S_PROFILES_REPO_DIR:-${ROOT_DIR}/../productive-k3s-profiles}"
SOURCE_PROFILE_PATH="${PK3S_CLI_MULTIPASS_PROFILE_SOURCE_PATH:-${PROFILES_REPO_DIR}/profiles/local/multipass/1-server-2-agents.env}"
WORK_DIR="$(mktemp -d "${ROOT_DIR}/.live-cli-profile-export.XXXXXX")"
SEED_EXPORT_DIR="${WORK_DIR}/seed-profile-export"
PROFILE_TGZ_PATH="${SEED_EXPORT_DIR}/profile.tgz"
INSTALLER_TGZ_PATH="${WORK_DIR}/multipass-installer.tgz"
INSTALLER_DIR="${WORK_DIR}/installer"
STATE_DIR="${WORK_DIR}/state"
PROFILE_ENV_PATH="${WORK_DIR}/multipass.env"
CLUSTER_NAME="pk3s-cli-export-mp-$(date +%Y%m%d-%H%M%S)-$$"
MULTIPASS_INSTANCE_REMOVAL_TIMEOUT_SECONDS="${MULTIPASS_INSTANCE_REMOVAL_TIMEOUT_SECONDS:-180}"
MULTIPASS_INSTANCE_REMOVAL_POLL_SECONDS="${MULTIPASS_INSTANCE_REMOVAL_POLL_SECONDS:-5}"

fail() {
  printf '[FAIL] %s\n' "$*" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

assert_server_namespace() {
  local namespace="$1"
  multipass exec "${CLUSTER_NAME}-server" -- bash -lc "sudo k3s kubectl get namespace '${namespace}' >/dev/null 2>&1" \
    || fail "expected namespace '${namespace}' was not created on ${CLUSTER_NAME}-server"
}

list_matching_instances() {
  local prefix="$1"
  multipass list --format json 2>/dev/null | jq -r --arg prefix "${prefix}" '.list[]?.name | select(startswith($prefix))'
}

wait_for_instance_removal() {
  local prefix="$1"
  local deadline=$((SECONDS + MULTIPASS_INSTANCE_REMOVAL_TIMEOUT_SECONDS))
  local matches=""
  while (( SECONDS < deadline )); do
    matches="$(list_matching_instances "${prefix}" || true)"
    if [[ -z "${matches}" ]]; then
      return 0
    fi
    sleep "${MULTIPASS_INSTANCE_REMOVAL_POLL_SECONDS}"
  done
  matches="$(list_matching_instances "${prefix}" || true)"
  [[ -z "${matches}" ]]
}

force_delete_instances_by_prefix() {
  local prefix="$1"
  local matches=""
  matches="$(list_matching_instances "${prefix}" || true)"
  if [[ -z "${matches}" ]]; then
    return 0
  fi
  # shellcheck disable=SC2206
  local names=( ${matches} )
  multipass delete "${names[@]}" >/dev/null 2>&1 || true
  multipass purge >/dev/null 2>&1 || true
}

cleanup() {
  if [[ -d "${INSTALLER_DIR}/bundle" ]]; then
    (
      cd "${INSTALLER_DIR}/bundle"
      PK3S_PROFILE_STATE_DIR="${STATE_DIR}" bash ./productive-k3s-infra.sh profile destroy --tgz ./profile.tgz >/dev/null 2>&1 || true
    )
  fi
  force_delete_instances_by_prefix "${CLUSTER_NAME}" || true
  wait_for_instance_removal "${CLUSTER_NAME}" || true
  rm -rf "${WORK_DIR}"
}
trap cleanup EXIT

[[ -x "${PK3S_BIN}" ]] || fail "pk3s binary is not executable: ${PK3S_BIN}"
[[ -f "${INFRA_REPO_DIR}/productive-k3s-infra.sh" ]] || fail "productive-k3s-infra checkout not found: ${INFRA_REPO_DIR}"
[[ -f "${SOURCE_PROFILE_PATH}" ]] || fail "multipass source profile not found: ${SOURCE_PROFILE_PATH}"

need_cmd jq
need_cmd tar
need_cmd multipass

# Keep this live validation isolated from the CLI repo's checked-in remote bundle defaults.
unset PRODUCTIVE_K3S_VERSION
unset PRODUCTIVE_K3S_RELEASE_REPO
unset PRODUCTIVE_K3S_CORE_VERSION_DEFAULT
unset PRODUCTIVE_K3S_RELEASE_REPO_DEFAULT
unset PRODUCTIVE_K3S_INFRA_VERSION_DEFAULT

cat > "${PROFILE_ENV_PATH}" <<EOF
TF_VAR_cluster_name=${CLUSTER_NAME}
PRODUCTIVE_K3S_SOURCE=remote
EOF

force_delete_instances_by_prefix "${CLUSTER_NAME}" || true
wait_for_instance_removal "${CLUSTER_NAME}" || true

PRODUCTIVE_K3S_PROFILES_REPO_DIR="${PROFILES_REPO_DIR}" \
TELEMETRY_ENABLED=false \
  bash "${INFRA_REPO_DIR}/productive-k3s-infra.sh" export --profile "${SOURCE_PROFILE_PATH}" --output "${SEED_EXPORT_DIR}"

[[ -f "${PROFILE_TGZ_PATH}" ]] || fail "seed multipass profile.tgz was not created from current source"

PRODUCTIVE_K3S_SOURCE=local \
PRODUCTIVE_K3S_INFRA_REPO_DIR="${INFRA_REPO_DIR}" \
PRODUCTIVE_K3S_PROFILES_REPO_DIR="${PROFILES_REPO_DIR}" \
TELEMETRY_ENABLED=false \
  "${PK3S_BIN}" profile export --tgz "${PROFILE_TGZ_PATH}" --env-file "${PROFILE_ENV_PATH}" --output "${INSTALLER_TGZ_PATH}"

mkdir -p "${INSTALLER_DIR}"
tar -xzf "${INSTALLER_TGZ_PATH}" -C "${INSTALLER_DIR}"

[[ -f "${INSTALLER_DIR}/bundle/install.sh" ]] || fail "exported profile installer is missing install.sh"
[[ -f "${INSTALLER_DIR}/bundle/profile.tgz" ]] || fail "exported profile installer is missing profile.tgz"

(
  cd "${INSTALLER_DIR}/bundle"
  PK3S_PROFILE_STATE_DIR="${STATE_DIR}" \
  PRODUCTIVE_K3S_AUTO_APPROVE_PREFLIGHT_WARNINGS=true \
    bash ./install.sh
)

(
  cd "${INSTALLER_DIR}/bundle"
  PK3S_PROFILE_STATE_DIR="${STATE_DIR}" \
    bash ./productive-k3s-infra.sh profile status --tgz ./profile.tgz
)

running_instances="$(list_matching_instances "${CLUSTER_NAME}" || true)"
instance_count="$(printf '%s\n' "${running_instances}" | sed '/^$/d' | wc -l | tr -d ' ')"
[[ "${instance_count}" == "3" ]] || fail "expected 3 multipass instances for exported profile installer, got ${instance_count}"
assert_server_namespace cert-manager
assert_server_namespace longhorn-system
assert_server_namespace cattle-system
assert_server_namespace registry

printf '[PASS] cli profile export live validation completed\n'
