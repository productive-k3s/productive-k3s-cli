#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PK3S_BIN="${PRODUCTIVE_K3S_CLI_BIN:-${ROOT_DIR}/pk3s}"
WORK_DIR="$(mktemp -d "${ROOT_DIR}/.live-cli-onprem-remote.XXXXXX")"
STAMP="$(date +%Y%m%d%H%M%S)"
SERVER_NAME="pk3s-cli-onprem-server-${STAMP}"
AGENT_NAME="pk3s-cli-onprem-agent-${STAMP}"
ENV_FILE="${WORK_DIR}/onprem-remote.env"
SSH_KEY_PATH=""
SSH_PUBKEY=""
MULTIPASS_LAUNCH_RETRIES="${MULTIPASS_LAUNCH_RETRIES:-3}"
MULTIPASS_LAUNCH_RETRY_DELAY_SECONDS="${MULTIPASS_LAUNCH_RETRY_DELAY_SECONDS:-5}"

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
  for attempt in $(seq 1 60); do
    if ssh -o BatchMode=yes -o StrictHostKeyChecking=accept-new -o ConnectTimeout=5 -i "${SSH_KEY_PATH}" "ubuntu@${ip}" true >/dev/null 2>&1; then
      return 0
    fi
    sleep 5
  done
  fail "ssh did not become ready for ${ip}"
}

run_pk3s() {
  PRODUCTIVE_K3S_SOURCE=remote "${PK3S_BIN}" "$@"
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

PRODUCTIVE_K3S_SOURCE=remote
TELEMETRY_ENABLED=false
EOF

run_pk3s profile validate --profile "${ENV_FILE}"
run_pk3s plan --profile "${ENV_FILE}"
run_pk3s apply --profile "${ENV_FILE}"
run_pk3s status --profile "${ENV_FILE}"
run_pk3s validate --profile "${ENV_FILE}"

printf '[PASS] onprem-basic remote CLI validation completed\n'
