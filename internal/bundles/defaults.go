package bundles

import (
	"fmt"
	"os"
	"strings"
)

func envOrDefault(name, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func GitHubOwnerDefault() string {
	return envOrDefault("PRODUCTIVE_K3S_GITHUB_OWNER_DEFAULT", "productive-k3s")
}

func GitHubBaseURLDefault() string {
	return envOrDefault("PRODUCTIVE_K3S_GITHUB_BASE_URL_DEFAULT", fmt.Sprintf("https://github.com/%s", GitHubOwnerDefault()))
}

func GitHubRawBaseURLDefault() string {
	return envOrDefault("PRODUCTIVE_K3S_GITHUB_RAW_BASE_URL_DEFAULT", fmt.Sprintf("https://raw.githubusercontent.com/%s", GitHubOwnerDefault()))
}

func RepoNameDefault(kind string) (string, error) {
	switch kind {
	case "core":
		return envOrDefault("PRODUCTIVE_K3S_CORE_REPO_NAME_DEFAULT", "productive-k3s-core"), nil
	case "infra":
		return envOrDefault("PRODUCTIVE_K3S_INFRA_REPO_NAME_DEFAULT", "productive-k3s-infra"), nil
	case "profiles":
		return envOrDefault("PRODUCTIVE_K3S_PROFILES_REPO_NAME_DEFAULT", "productive-k3s-profiles"), nil
	case "cli":
		return envOrDefault("PRODUCTIVE_K3S_CLI_REPO_NAME_DEFAULT", "productive-k3s-cli"), nil
	default:
		return "", fmt.Errorf("unsupported bundle kind: %s", kind)
	}
}

func ReleaseRepoDefault(kind string) (string, error) {
	switch kind {
	case "core":
		repoName, _ := RepoNameDefault(kind)
		return envOrDefault("PRODUCTIVE_K3S_CORE_RELEASE_REPO_DEFAULT", fmt.Sprintf("%s/%s", GitHubOwnerDefault(), repoName)), nil
	case "infra":
		repoName, _ := RepoNameDefault(kind)
		return envOrDefault("PRODUCTIVE_K3S_INFRA_RELEASE_REPO_DEFAULT", fmt.Sprintf("%s/%s", GitHubOwnerDefault(), repoName)), nil
	default:
		return "", fmt.Errorf("unsupported bundle kind: %s", kind)
	}
}

func ProfilesGitRemoteURLDefault() string {
	repoName, _ := RepoNameDefault("profiles")
	return envOrDefault("PRODUCTIVE_K3S_PROFILES_GIT_REMOTE_URL_DEFAULT", fmt.Sprintf("%s/%s.git", GitHubBaseURLDefault(), repoName))
}

func ProfilesRawURLDefault(path, ref string) string {
	repoName, _ := RepoNameDefault("profiles")
	if strings.TrimSpace(ref) == "" {
		ref = "main"
	}
	return fmt.Sprintf("%s/%s/%s/%s", GitHubRawBaseURLDefault(), repoName, ref, strings.TrimPrefix(path, "/"))
}

func MultipassProfileURLDefault() string {
	return envOrDefault("PRODUCTIVE_K3S_PROFILES_MULTIPASS_PROFILE_URL_DEFAULT", ProfilesRawURLDefault("profiles/local/multipass/1-server-2-agents.env", "main"))
}

func CatalogURLDefault() string {
	return envOrDefault("PRODUCTIVE_K3S_CATALOG_URL_DEFAULT", "https://catalogs.productive-k3s.io/catalogs/index.yaml")
}
