#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PK3S_BIN="${PRODUCTIVE_K3S_CLI_BIN:-${ROOT_DIR}/pk3s}"
WORK_DIR="$(mktemp -d "${ROOT_DIR}/.live-cli-onprem-remote-github-host.XXXXXX")"
ENV_FILE="${WORK_DIR}/onprem-remote.env"
SSH_KEY_PATH="${WORK_DIR}/id_ed25519"
CURRENT_USER="$(id -un)"
LOCALHOST_IP="127.0.0.1"

fail() {
  printf '[FAIL] %s\n' "$*" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

cleanup() {
  rm -rf "${WORK_DIR}"
}

prepare_openssh_server() {
  if ! command -v sshd >/dev/null 2>&1; then
    sudo apt-get update
    sudo apt-get install -y openssh-server
  fi

  sudo mkdir -p /run/sshd
  sudo systemctl enable ssh >/dev/null 2>&1 || true
  sudo systemctl restart ssh
}

prepare_ssh_key() {
  ssh-keygen -q -t ed25519 -N '' -f "${SSH_KEY_PATH}" >/dev/null
  install -d -m 700 "${HOME}/.ssh"
  touch "${HOME}/.ssh/authorized_keys"
  chmod 600 "${HOME}/.ssh/authorized_keys"
  if ! grep -qxF "$(cat "${SSH_KEY_PATH}.pub")" "${HOME}/.ssh/authorized_keys"; then
    printf '%s\n' "$(cat "${SSH_KEY_PATH}.pub")" >> "${HOME}/.ssh/authorized_keys"
  fi
}

wait_for_ssh() {
  local attempt
  for attempt in $(seq 1 30); do
    if ssh -o BatchMode=yes -o StrictHostKeyChecking=accept-new -o ConnectTimeout=5 -i "${SSH_KEY_PATH}" "${CURRENT_USER}@${LOCALHOST_IP}" true >/dev/null 2>&1; then
      return 0
    fi
    sleep 2
  done
  fail "ssh did not become ready on ${LOCALHOST_IP}"
}

write_env_file() {
  cat > "${ENV_FILE}" <<EOF
PK3S_INFRA_PROFILE_NAME=pk3s-cli-gha-onprem-remote
PK3S_INFRA_SCENARIO=on-prem
PK3S_INFRA_ENGINE=ansible

ONPREM_SERVER_IP=${LOCALHOST_IP}
ONPREM_AGENT_IPS=
ONPREM_SSH_USER=${CURRENT_USER}
ONPREM_SSH_PORT=22
ONPREM_SSH_KEY_PATH=${SSH_KEY_PATH}

ONPREM_CLUSTER_NAME=pk3s-cli-gha-onprem-remote
ONPREM_BASE_DOMAIN=k3s.lab.internal
ONPREM_RANCHER_HOST=rancher.k3s.lab.internal
ONPREM_REGISTRY_HOST=registry.k3s.lab.internal
ONPREM_REMOTE_DIR=/home/${CURRENT_USER}/pk3s-cli-gha-onprem-remote

PRODUCTIVE_K3S_SOURCE=remote
TELEMETRY_ENABLED=false
EOF
}

run_pk3s() {
  PRODUCTIVE_K3S_SOURCE=remote "${PK3S_BIN}" "$@"
}

need_cmd sudo
need_cmd ssh
need_cmd ssh-keygen
need_cmd systemctl
need_cmd jq
need_cmd curl
need_cmd tar
need_cmd python3
[[ -x "${PK3S_BIN}" ]] || fail "pk3s binary is not executable: ${PK3S_BIN}"

trap cleanup EXIT

prepare_openssh_server
prepare_ssh_key
wait_for_ssh
write_env_file

run_pk3s profile validate --profile "${ENV_FILE}"
run_pk3s plan --profile "${ENV_FILE}"
run_pk3s apply --profile "${ENV_FILE}"
run_pk3s status --profile "${ENV_FILE}"
run_pk3s validate --profile "${ENV_FILE}"

printf '[PASS] onprem-basic remote GitHub-host CLI validation completed\n'
