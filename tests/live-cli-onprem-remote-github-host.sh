#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PK3S_BIN="${PRODUCTIVE_K3S_CLI_BIN:-${ROOT_DIR}/pk3s}"
WORK_DIR="$(mktemp -d "${ROOT_DIR}/.live-cli-onprem-remote-github-host.XXXXXX")"
ARTIFACT_DIR="${ROOT_DIR}/test-artifacts/live-onprem-remote-github-host"
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

ssh_remote() {
  ssh \
    -o BatchMode=yes \
    -o StrictHostKeyChecking=accept-new \
    -o ConnectTimeout=10 \
    -i "${SSH_KEY_PATH}" \
    "${CURRENT_USER}@${LOCALHOST_IP}" \
    "$@"
}

write_remote_capture() {
  local name="$1"
  shift
  local output_file="${ARTIFACT_DIR}/${name}.log"

  {
    printf '[CMD] %s\n' "$*"
    ssh_remote "$@"
  } >"${output_file}" 2>&1 || true
}

dump_cluster_diagnostics() {
  mkdir -p "${ARTIFACT_DIR}"

  write_remote_capture "system-df" "df -h"
  write_remote_capture "system-free" "free -m"
  write_remote_capture "system-k3s-service" "sudo systemctl status k3s --no-pager"
  write_remote_capture "cluster-nodes" "sudo k3s kubectl get nodes -o wide"
  write_remote_capture "cluster-pods-all" "sudo k3s kubectl get pods -A -o wide"
  write_remote_capture "cluster-events" "sudo k3s kubectl get events -A --sort-by=.lastTimestamp"
  write_remote_capture "registry-deploy" "sudo k3s kubectl describe deploy/registry -n registry"
  write_remote_capture "registry-pods" "sudo k3s kubectl get pods -n registry -o wide"
  write_remote_capture "registry-pods-describe" "sudo k3s kubectl describe pods -n registry"
  write_remote_capture "rancher-pods" "sudo k3s kubectl get pods -n cattle-system -o wide"

  local pod_names
  pod_names="$(ssh_remote "sudo k3s kubectl get pods -n registry -o jsonpath='{range .items[*]}{.metadata.name}{\"\\n\"}{end}'" 2>/dev/null || true)"
  if [[ -n "${pod_names}" ]]; then
    while IFS= read -r pod_name; do
      [[ -n "${pod_name}" ]] || continue
      write_remote_capture "registry-${pod_name}" "sudo k3s kubectl logs -n registry ${pod_name} --all-containers=true --tail=200"
      write_remote_capture "registry-${pod_name}-previous" "sudo k3s kubectl logs -n registry ${pod_name} --all-containers=true --previous --tail=200"
    done <<< "${pod_names}"
  fi
}

run_step() {
  local step_name="$1"
  shift
  local output_file="${ARTIFACT_DIR}/${step_name}.log"

  mkdir -p "${ARTIFACT_DIR}"
  printf '[INFO] Running step: %s\n' "${step_name}"

  if "$@" > >(tee "${output_file}") 2> >(tee -a "${output_file}" >&2); then
    return 0
  fi

  printf '[FAIL] Step failed: %s\n' "${step_name}" >&2
  dump_cluster_diagnostics
  return 1
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
  local source_mode="remote"
  if [[ -n "${PRODUCTIVE_K3S_CORE_REPO_DIR:-}${PRODUCTIVE_K3S_CORE_REPO_URL:-}${PRODUCTIVE_K3S_CORE_REPO_REF:-}${PRODUCTIVE_K3S_INFRA_REPO_DIR:-}${PRODUCTIVE_K3S_INFRA_REPO_URL:-}${PRODUCTIVE_K3S_INFRA_REPO_REF:-}" ]]; then
    source_mode="local"
  fi
  PRODUCTIVE_K3S_SOURCE="${source_mode}" "${PK3S_BIN}" "$@"
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

mkdir -p "${ARTIFACT_DIR}"

prepare_openssh_server
prepare_ssh_key
wait_for_ssh
write_env_file

cp "${ENV_FILE}" "${ARTIFACT_DIR}/onprem-remote.env"

run_step "profile-validate" run_pk3s profile validate --profile "${ENV_FILE}"
run_step "plan" run_pk3s plan --profile "${ENV_FILE}"
run_step "apply" run_pk3s apply --profile "${ENV_FILE}"
run_step "status" run_pk3s status --profile "${ENV_FILE}"
run_step "validate" run_pk3s validate --profile "${ENV_FILE}"

printf '[PASS] onprem-basic remote GitHub-host CLI validation completed\n'
