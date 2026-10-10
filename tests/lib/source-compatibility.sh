#!/usr/bin/env bash

pk3s_read_shell_default() {
  local file="$1"
  local variable="$2"
  sed -n "s/^: \"\${${variable}:=\([^}]*\)}\"$/\1/p" "${file}" | head -n1
}

pk3s_export_source_compatibility() {
  local core_repo_dir="$1"
  local infra_repo_dir="$2"
  local core_version infra_version infra_core_version

  core_version="$(pk3s_read_shell_default "${core_repo_dir}/scripts/component-versions.sh" PRODUCTIVE_K3S_CORE_VERSION)"
  infra_version="$(pk3s_read_shell_default "${infra_repo_dir}/scripts/productive-k3s-infra.sh" PRODUCTIVE_K3S_INFRA_ENGINE_VERSION)"
  infra_core_version="$(pk3s_read_shell_default "${infra_repo_dir}/scripts/productive-k3s-infra.sh" PRODUCTIVE_K3S_CORE_VERSION)"

  [[ "${core_version}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || {
    printf '[FAIL] could not resolve a stable Core source version from %s\n' "${core_repo_dir}" >&2
    return 1
  }
  [[ "${infra_version}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || {
    printf '[FAIL] could not resolve a stable Infra source version from %s\n' "${infra_repo_dir}" >&2
    return 1
  }
  [[ "${infra_core_version}" == "${core_version}" ]] || {
    printf '[FAIL] Infra source requires Core %s but the Core source declares %s\n' "${infra_core_version:-unknown}" "${core_version}" >&2
    return 1
  }

  export PRODUCTIVE_K3S_SOURCE_CORE_VERSION="${core_version}"
  export PRODUCTIVE_K3S_SOURCE_INFRA_VERSION="${infra_version}-${infra_core_version}"
}
