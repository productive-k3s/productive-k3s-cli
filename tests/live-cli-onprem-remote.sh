#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PK3S_BIN="${PRODUCTIVE_K3S_CLI_BIN:-${ROOT_DIR}/pk3s}"
# shellcheck disable=SC1091
source "${ROOT_DIR}/scripts/release-config.sh"
WORK_ROOT="${PK3S_CLI_ONPREM_REMOTE_WORK_ROOT:-${HOME}}"
WORK_DIR="$(mktemp -d "${WORK_ROOT}/pk3s-live-cli-onprem-remote.XXXXXX")"
STAMP="$(date +%Y%m%d%H%M%S)"
SERVER_NAME="pk3s-cli-onprem-server-${STAMP}"
AGENT_NAME="pk3s-cli-onprem-agent-${STAMP}"
ENV_FILE="${WORK_DIR}/onprem-remote.env"
PROFILES_REPO_DIR=""
INFRA_REPO_DIR_LOCAL=""
CORE_REPO_DIR_LOCAL=""
ADDONS_REPO_DIR_LOCAL=""
SSH_KEY_PATH=""
SSH_PUBKEY=""
MULTIPASS_LAUNCH_RETRIES="${MULTIPASS_LAUNCH_RETRIES:-3}"
MULTIPASS_LAUNCH_RETRY_DELAY_SECONDS="${MULTIPASS_LAUNCH_RETRY_DELAY_SECONDS:-5}"
PROFILE_NAME="${PK3S_CLI_ONPREM_PROFILE_NAME:-on-prem-basic}"

# This local live validator targets remote SSH hosts, but it intentionally
# exercises the source/dev surface so the scenario can assemble the shared
# infra + profiles + core trees from isolated clones.
: "${PRODUCTIVE_K3S_INFRA_REPO_REF:=development}"
: "${PRODUCTIVE_K3S_CORE_REPO_REF:=development}"
: "${PRODUCTIVE_K3S_PROFILES_REPO_REF:=development}"
: "${PRODUCTIVE_K3S_ADDONS_REPO_REF:=development}"
export PRODUCTIVE_K3S_INFRA_REPO_REF
export PRODUCTIVE_K3S_CORE_REPO_REF
export PRODUCTIVE_K3S_PROFILES_REPO_REF
export PRODUCTIVE_K3S_ADDONS_REPO_REF

fail() {
  printf '[FAIL] %s\n' "$*" >&2
  exit 1
}

warn() {
  printf '[WARN] %s\n' "$*" >&2
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

prepare_profiles_repo_dir() {
  [[ -n "${PROFILES_REPO_DIR}" ]] && return 0
  local profiles_source_dir="${PRODUCTIVE_K3S_PROFILES_REPO_DIR:-}"
  local profiles_repo_url="${PRODUCTIVE_K3S_PROFILES_REPO_URL:-${PRODUCTIVE_K3S_PROFILES_GIT_REMOTE_URL_DEFAULT}}"
  local profiles_repo_ref="${PRODUCTIVE_K3S_PROFILES_REPO_REF:-${PRODUCTIVE_K3S_INFRA_REPO_REF:-development}}"
  local infra_repo_dir=""

  prepare_infra_repo_dir
  infra_repo_dir="${PRODUCTIVE_K3S_INFRA_REPO_DIR:-${INFRA_REPO_DIR_LOCAL}}"
  PROFILES_REPO_DIR="${WORK_DIR}/productive-k3s-profiles"

  if [[ -n "${profiles_source_dir}" ]]; then
    [[ -d "${profiles_source_dir}/profiles" && -d "${profiles_source_dir}/scenarios" ]] || {
      fail "invalid PRODUCTIVE_K3S_PROFILES_REPO_DIR: ${profiles_source_dir}"
    }
    mkdir -p "${PROFILES_REPO_DIR}"
    cp -a "${profiles_source_dir}/." "${PROFILES_REPO_DIR}/"
  else
    git clone --depth 1 --branch "${profiles_repo_ref}" "${profiles_repo_url}" "${PROFILES_REPO_DIR}" >/dev/null 2>&1 || {
      fail "could not clone productive-k3s-profiles from ${profiles_repo_url} (${profiles_repo_ref})"
    }
  fi

  mkdir -p "${PROFILES_REPO_DIR}/ansible" "${PROFILES_REPO_DIR}/scripts" "${PROFILES_REPO_DIR}/tests"
  cp -a "${infra_repo_dir}/ansible/." "${PROFILES_REPO_DIR}/ansible/"
  cp -a "${infra_repo_dir}/scripts/." "${PROFILES_REPO_DIR}/scripts/"
  cp -a "${infra_repo_dir}/tests/." "${PROFILES_REPO_DIR}/tests/"
}

prepare_infra_repo_dir() {
  [[ -n "${PRODUCTIVE_K3S_INFRA_REPO_DIR:-}" ]] && return 0
  [[ -n "${INFRA_REPO_DIR_LOCAL}" ]] && return 0

  local infra_repo_url="${PRODUCTIVE_K3S_INFRA_REPO_URL:-${PRODUCTIVE_K3S_INFRA_GIT_REMOTE_URL_DEFAULT}}"
  local infra_repo_ref="${PRODUCTIVE_K3S_INFRA_REPO_REF:-development}"
  INFRA_REPO_DIR_LOCAL="${WORK_DIR}/productive-k3s-infra"
  git clone --depth 1 --branch "${infra_repo_ref}" "${infra_repo_url}" "${INFRA_REPO_DIR_LOCAL}" >/dev/null 2>&1 || {
    fail "could not clone productive-k3s-infra from ${infra_repo_url} (${infra_repo_ref})"
  }
}

prepare_core_repo_dir() {
  [[ -n "${PRODUCTIVE_K3S_REPO:-}" ]] && return 0
  [[ -n "${CORE_REPO_DIR_LOCAL}" ]] && return 0

  local core_repo_url="${PRODUCTIVE_K3S_CORE_REPO_URL:-${PRODUCTIVE_K3S_CORE_GIT_REMOTE_URL_DEFAULT}}"
  local core_repo_ref="${PRODUCTIVE_K3S_CORE_REPO_REF:-development}"
  CORE_REPO_DIR_LOCAL="${WORK_DIR}/productive-k3s-core"
  git clone --depth 1 --branch "${core_repo_ref}" "${core_repo_url}" "${CORE_REPO_DIR_LOCAL}" >/dev/null 2>&1 || {
    fail "could not clone productive-k3s-core from ${core_repo_url} (${core_repo_ref})"
  }
}

prepare_addons_repo_dir() {
  [[ -n "${PRODUCTIVE_K3S_ADDONS_REPO_DIR:-}" ]] && return 0
  [[ -n "${ADDONS_REPO_DIR_LOCAL}" ]] && return 0

  local addons_repo_url="${PRODUCTIVE_K3S_ADDONS_REPO_URL:-${PRODUCTIVE_K3S_ADDONS_GIT_REMOTE_URL_DEFAULT}}"
  local addons_repo_ref="${PRODUCTIVE_K3S_ADDONS_REPO_REF:-${PRODUCTIVE_K3S_CORE_REPO_REF:-development}}"
  ADDONS_REPO_DIR_LOCAL="${WORK_DIR}/productive-k3s-addons"
  git clone --depth 1 --branch "${addons_repo_ref}" "${addons_repo_url}" "${ADDONS_REPO_DIR_LOCAL}" >/dev/null 2>&1 || {
    fail "could not clone productive-k3s-addons from ${addons_repo_url} (${addons_repo_ref})"
  }
}

pick_ssh_key() {
  local candidate
  for candidate in "${HOME}/.ssh/id_ed25519" "${HOME}/.ssh/id_rsa"; do
    if [[ -f "${candidate}" && -f "${candidate}.pub" ]]; then
      SSH_KEY_PATH="${candidate}"
      SSH_PUBKEY="$(<"${candidate}.pub")"
      return 0
    fi
  done
  fail "could not find a usable SSH key pair in ~/.ssh"
}

write_cloud_init() {
  local file="$1"
  cat > "${file}" <<EOF
#cloud-config
package_update: false
package_upgrade: false
manage_etc_hosts: true
users:
  - name: ubuntu
    groups: [sudo]
    shell: /bin/bash
    sudo: "ALL=(ALL) NOPASSWD:ALL"
    lock_passwd: true
    ssh_authorized_keys:
      - ${SSH_PUBKEY}
EOF
}

launch_instance() {
  local name="$1"
  local cloud_init_file="$2"
  local attempts="${MULTIPASS_LAUNCH_RETRIES}"
  local attempt=1
  local stderr_file
  stderr_file="$(mktemp "${WORK_DIR}/multipass-launch.${name}.XXXXXX.stderr")"

  while (( attempt <= attempts )); do
    if multipass launch 24.04 --name "${name}" --cpus 4 --memory 14G --disk 70G --cloud-init "${cloud_init_file}" 2>"${stderr_file}"; then
      rm -f "${stderr_file}"
      return 0
    fi

    if grep -Fq 'Remote "" is unknown or unreachable.' "${stderr_file}" && (( attempt < attempts )); then
      warn "multipass launch hit a transient remote resolution error for ${name}; retrying (${attempt}/${attempts})"
      multipass list >/dev/null 2>&1 || true
      sleep "${MULTIPASS_LAUNCH_RETRY_DELAY_SECONDS}"
      ((attempt++))
      continue
    fi

    cat "${stderr_file}" >&2
    rm -f "${stderr_file}"
    fail "could not launch multipass instance ${name}"
  done

  cat "${stderr_file}" >&2
  rm -f "${stderr_file}"
  fail "could not launch multipass instance ${name}"
}

instance_ip() {
  local name="$1"
  multipass info --format json "${name}" | jq -r --arg name "${name}" '.info[$name].ipv4[0] // empty'
}

wait_for_ssh() {
  local ip="$1"
  local attempt
  ssh-keygen -f "${HOME}/.ssh/known_hosts" -R "${ip}" >/dev/null 2>&1 || true
  ssh-keygen -f "${HOME}/.ssh/known_hosts" -R "[${ip}]:22" >/dev/null 2>&1 || true
  for attempt in $(seq 1 60); do
    if ssh -o BatchMode=yes -o StrictHostKeyChecking=accept-new -o ConnectTimeout=5 -i "${SSH_KEY_PATH}" "ubuntu@${ip}" true >/dev/null 2>&1; then
      return 0
    fi
    sleep 5
  done
  fail "ssh did not become ready for ${ip}"
}

ssh_remote() {
  local ip="$1"
  shift
  ssh \
    -o BatchMode=yes \
    -o StrictHostKeyChecking=accept-new \
    -o ConnectTimeout=10 \
    -i "${SSH_KEY_PATH}" \
    "ubuntu@${ip}" \
    "$@"
}

assert_remote_cluster() {
  local server_ip="$1"
  ssh_remote "${server_ip}" "sudo k3s kubectl get nodes >/dev/null"
  ssh_remote "${server_ip}" "sudo k3s kubectl get namespace cattle-system >/dev/null"
  ssh_remote "${server_ip}" "sudo k3s kubectl get namespace registry >/dev/null"
}

wait_for_cloud_init() {
  local name="$1"
  local attempt
  for attempt in $(seq 1 60); do
    if multipass exec "${name}" -- cloud-init status --wait >/dev/null 2>&1; then
      return 0
    fi
    sleep 5
  done
  fail "cloud-init did not finish for ${name}"
}

run_pk3s() {
  local source_mode="remote"
  if [[ -n "${PRODUCTIVE_K3S_CORE_REPO_DIR:-}${PRODUCTIVE_K3S_CORE_REPO_URL:-}${PRODUCTIVE_K3S_CORE_REPO_REF:-}${PRODUCTIVE_K3S_INFRA_REPO_DIR:-}${PRODUCTIVE_K3S_INFRA_REPO_URL:-}${PRODUCTIVE_K3S_INFRA_REPO_REF:-}" ]]; then
    source_mode="local"
  fi
  if [[ "${source_mode}" == "local" ]]; then
    prepare_profiles_repo_dir
    prepare_core_repo_dir
    prepare_addons_repo_dir
  fi
  PRODUCTIVE_K3S_SOURCE="${source_mode}" \
    PRODUCTIVE_K3S_AUTO_APPROVE_PREFLIGHT_WARNINGS="${PRODUCTIVE_K3S_AUTO_APPROVE_PREFLIGHT_WARNINGS:-true}" \
    PRODUCTIVE_K3S_INFRA_REPO_DIR="${PRODUCTIVE_K3S_INFRA_REPO_DIR:-${INFRA_REPO_DIR_LOCAL}}" \
    PRODUCTIVE_K3S_PROFILES_REPO_DIR="${PRODUCTIVE_K3S_PROFILES_REPO_DIR:-${PROFILES_REPO_DIR}}" \
    PRODUCTIVE_K3S_REPO="${PRODUCTIVE_K3S_REPO:-${CORE_REPO_DIR_LOCAL}}" \
    PRODUCTIVE_K3S_ADDONS_REPO_DIR="${PRODUCTIVE_K3S_ADDONS_REPO_DIR:-${ADDONS_REPO_DIR_LOCAL}}" \
    "${PK3S_BIN}" "$@"
}

cleanup() {
  multipass delete "${SERVER_NAME}" "${AGENT_NAME}" >/dev/null 2>&1 || true
  multipass purge >/dev/null 2>&1 || true
  rm -rf "${WORK_DIR}"
}

need_cmd multipass
need_cmd jq
need_cmd ssh
need_cmd curl
need_cmd tar
need_cmd python3
need_cmd git
[[ -x "${PK3S_BIN}" ]] || fail "pk3s binary is not executable: ${PK3S_BIN}"
pick_ssh_key
trap cleanup EXIT

write_cloud_init "${WORK_DIR}/server.yaml"
write_cloud_init "${WORK_DIR}/agent.yaml"
launch_instance "${SERVER_NAME}" "${WORK_DIR}/server.yaml"
launch_instance "${AGENT_NAME}" "${WORK_DIR}/agent.yaml"

SERVER_IP="$(instance_ip "${SERVER_NAME}")"
AGENT_IP="$(instance_ip "${AGENT_NAME}")"
[[ -n "${SERVER_IP}" && -n "${AGENT_IP}" ]] || fail "could not determine VM IPs"

wait_for_cloud_init "${SERVER_NAME}"
wait_for_cloud_init "${AGENT_NAME}"
wait_for_ssh "${SERVER_IP}"
wait_for_ssh "${AGENT_IP}"

cat > "${ENV_FILE}" <<EOF
PK3S_INFRA_PROFILE_NAME=pk3s-cli-onprem-remote
PK3S_INFRA_SCENARIO=on-prem
PK3S_INFRA_ENGINE=ansible

ONPREM_SERVER_IP=${SERVER_IP}
ONPREM_AGENT_IPS=${AGENT_IP}
ONPREM_SSH_USER=ubuntu
ONPREM_SSH_PORT=22
ONPREM_SSH_KEY_PATH=${SSH_KEY_PATH}

ONPREM_CLUSTER_NAME=pk3s-cli-onprem-${STAMP}
ONPREM_BASE_DOMAIN=k3s.lab.internal
ONPREM_RANCHER_HOST=rancher.k3s.lab.internal
ONPREM_REGISTRY_HOST=registry.k3s.lab.internal

PRODUCTIVE_K3S_AUTO_APPROVE_PREFLIGHT_WARNINGS=true
TELEMETRY_ENABLED=false
EOF

run_pk3s profile validate --profile "${ENV_FILE}"
run_pk3s plan --profile "${ENV_FILE}"
run_pk3s apply --profile "${ENV_FILE}"
run_pk3s status --profile "${ENV_FILE}"
assert_remote_cluster "${SERVER_IP}"

printf '[PASS] onprem-basic remote CLI validation completed\n'
