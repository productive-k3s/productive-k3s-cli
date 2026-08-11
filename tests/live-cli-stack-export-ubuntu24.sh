#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PK3S_BIN="${PRODUCTIVE_K3S_CLI_BIN:-${ROOT_DIR}/pk3s}"
CORE_REPO_DIR="${PRODUCTIVE_K3S_CORE_REPO_DIR:-${ROOT_DIR}/../productive-k3s-core}"
WORK_DIR="$(mktemp -d "${ROOT_DIR}/.live-cli-stack-export.XXXXXX")"
CERT_MANAGER_TGZ_URL="${PK3S_CLI_CERT_MANAGER_TGZ_URL:-https://downloads.productive-k3s.io/addons/cert-manager-0.1.0.tgz}"
STACK_NAME="${PK3S_CLI_STACK_NAME:-cert-manager-lite}"
STACK_TGZ_PATH="${WORK_DIR}/${STACK_NAME}.tgz"
INSTALLER_TGZ_PATH="${WORK_DIR}/${STACK_NAME}-installer.tgz"
STACK_PKG_DIR="${WORK_DIR}/stack-pkg"
REMOTE_INSTALLER_TGZ_PATH="${REMOTE_INSTALLER_TGZ_PATH:-/tmp/pk3s-cli-stack-installer.tgz}"
REMOTE_INSTALLER_ROOT="${REMOTE_INSTALLER_ROOT:-/tmp/pk3s-cli-stack-installer}"
REMOTE_USER="${STACK_TEST_REMOTE_USER:-ubuntu}"
REMOTE_DIR="${STACK_TEST_REMOTE_DIR:-/home/${REMOTE_USER}/productive-k3s-core}"
REMOTE_ADDONS_DIR="${STACK_TEST_REMOTE_ADDONS_DIR:-/home/${REMOTE_USER}/productive-k3s-addons}"
VM_NAME="${STACK_TEST_VM_NAME:-pk3s-cli-stack-export-$(date +%Y%m%d-%H%M%S)}"
INNER_ARTIFACTS_DIR="${WORK_DIR}/artifacts"
VM_CREATED="n"

cleanup() {
  if [[ "${VM_CREATED}" == "y" ]]; then
    multipass delete "${VM_NAME}" >/dev/null 2>&1 || true
    multipass purge >/dev/null 2>&1 || true
  fi
  rm -rf "${WORK_DIR}"
}
trap cleanup EXIT

fail() {
  printf '[FAIL] %s\n' "$*" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

[[ -x "${PK3S_BIN}" ]] || fail "pk3s binary is not executable: ${PK3S_BIN}"
[[ -f "${CORE_REPO_DIR}/productive-k3s-core.sh" ]] || fail "productive-k3s-core checkout not found: ${CORE_REPO_DIR}"

need_cmd curl
need_cmd multipass

run_in_vm() {
  multipass exec "${VM_NAME}" -- bash -lc "$1" </dev/null
}

assert_in_vm() {
  local cmd="$1"
  local label="$2"
  if ! run_in_vm "$cmd" >/dev/null 2>&1; then
    fail "${label}"
  fi
}

full_answers() {
  cat <<'EOF'
y
y
y
y
home.arpa
2



y

admin





y
EOF
}

exported_stack_install_command() {
  local answers escaped_answers
  answers="$(full_answers)"
  escaped_answers="$(printf '%q' "${answers}")"
  printf "installer_dir=\"\$(find '%s' -mindepth 1 -maxdepth 1 -type d | head -1)\" && test -n \"\${installer_dir}\" && cd \"\${installer_dir}\" && bootstrap_answers_file=\"\$(mktemp)\" && printf '%%s' %s > \"\${bootstrap_answers_file}\" && export PRODUCTIVE_K3S_DISTRO='k3s' PRODUCTIVE_K3S_ENGINE='native' PRODUCTIVE_K3S_AUTO_APPROVE_PREFLIGHT_WARNINGS=true && ./install.sh < \"\${bootstrap_answers_file}\"; stack_rc=\$?; rm -f \"\${bootstrap_answers_file}\"; exit \"\${stack_rc}\"" \
    "${REMOTE_INSTALLER_ROOT}" \
    "${escaped_answers}"
}

mkdir -p "${STACK_PKG_DIR}/addons" "${INNER_ARTIFACTS_DIR}"
curl -fsSL "${CERT_MANAGER_TGZ_URL}" -o "${STACK_PKG_DIR}/addons/cert-manager-0.1.0.tgz"
cat > "${STACK_PKG_DIR}/stack.yaml" <<EOF
apiVersion: addons.productive-k3s.io/v1
kind: Stack
metadata:
  name: ${STACK_NAME}
  version: 1.0.0
spec:
  resolution:
    mode: bundled
  addons:
    - name: cert-manager
      source: addons/cert-manager-0.1.0.tgz
EOF
tar -czf "${STACK_TGZ_PATH}" -C "${STACK_PKG_DIR}" .

PRODUCTIVE_K3S_SOURCE=local \
PRODUCTIVE_K3S_CORE_REPO_DIR="${CORE_REPO_DIR}" \
TELEMETRY_ENABLED=false \
  "${PK3S_BIN}" stack export --tgz "${STACK_TGZ_PATH}" --output "${INSTALLER_TGZ_PATH}"

[[ -f "${INSTALLER_TGZ_PATH}" ]] || fail "stack installer archive was not created"

TEST_ARTIFACTS_DIR="${INNER_ARTIFACTS_DIR}" \
PRODUCTIVE_K3S_DISTRO=k3s \
PRODUCTIVE_K3S_ENGINE=native \
  bash "${CORE_REPO_DIR}/tests/test-in-vm.sh" \
    --platform ubuntu \
    --image 24.04 \
    --profile core \
    --name "${VM_NAME}" \
    --keep-vm
VM_CREATED="y"

run_in_vm "rm -rf '${REMOTE_ADDONS_DIR}' '${REMOTE_DIR}'"
assert_in_vm "test ! -d '${REMOTE_ADDONS_DIR}'" "remote addons checkout was not removed"
assert_in_vm "test ! -d '${REMOTE_DIR}'" "remote core checkout was not removed"

multipass transfer "${INSTALLER_TGZ_PATH}" "${VM_NAME}:${REMOTE_INSTALLER_TGZ_PATH}" >/dev/null
assert_in_vm "test -f '${REMOTE_INSTALLER_TGZ_PATH}'" "exported installer archive was not copied into the VM"

run_in_vm "rm -rf '${REMOTE_INSTALLER_ROOT}' && mkdir -p '${REMOTE_INSTALLER_ROOT}' && tar -xzf '${REMOTE_INSTALLER_TGZ_PATH}' -C '${REMOTE_INSTALLER_ROOT}'"
assert_in_vm "find '${REMOTE_INSTALLER_ROOT}' -mindepth 1 -maxdepth 1 -type d | grep -q ." "exported installer archive did not unpack a bundle directory"

run_in_vm "$(exported_stack_install_command)"

assert_in_vm "sudo k3s kubectl get namespace cert-manager >/dev/null 2>&1" "cert-manager namespace was not created from the exported installer"

printf '[PASS] cli stack export live validation completed\n'
