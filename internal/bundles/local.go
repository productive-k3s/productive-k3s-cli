package bundles

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func ResolveLocalSibling(kind, startDir string) (BundleRef, error) {
	repoName, err := LocalRepoName(kind)
	if err != nil {
		return BundleRef{}, err
	}
	entryName, err := EntryPointName(kind)
	if err != nil {
		return BundleRef{}, err
	}

	candidates := []string{
		filepath.Join(startDir, repoName),
		filepath.Join(filepath.Dir(startDir), repoName),
	}

	for _, candidate := range candidates {
		entrypoint := filepath.Join(candidate, entryName)
		info, err := os.Stat(entrypoint)
		if err == nil && !info.IsDir() {
			return BundleRef{
				Kind:       kind,
				Source:     "local",
				Root:       candidate,
				Entrypoint: entrypoint,
			}, nil
		}
	}

	return BundleRef{}, fmt.Errorf("local %s bundle not found near %s", kind, startDir)
}

func ResolveLocalBundle(kind, startDir, cacheDir string) (BundleRef, error) {
	if ref, ok, err := resolveExplicitLocalBundle(kind); ok {
		return ref, err
	}
	if ref, ok, err := resolveLocalCloneOverride(kind, cacheDir); ok {
		return ref, err
	}
	return ResolveLocalSibling(kind, startDir)
}

func resolveExplicitLocalBundle(kind string) (BundleRef, bool, error) {
	dirVar, _, _, err := localBundleEnvNames(kind)
	if err != nil {
		return BundleRef{}, true, err
	}
	dir := strings.TrimSpace(os.Getenv(dirVar))
	if dir == "" && kind == "core" {
		dir = strings.TrimSpace(os.Getenv("PRODUCTIVE_K3S_REPO"))
	}
	if dir == "" {
		return BundleRef{}, false, nil
	}
	ref, err := validateLocalBundleDir(kind, dir)
	if err != nil {
		return BundleRef{}, true, fmt.Errorf("invalid %s: %w", dirVar, err)
	}
	return ref, true, nil
}

func resolveLocalCloneOverride(kind, cacheDir string) (BundleRef, bool, error) {
	_, urlVar, refVar, err := localBundleEnvNames(kind)
	if err != nil {
		return BundleRef{}, true, err
	}
	repoURL := strings.TrimSpace(os.Getenv(urlVar))
	repoRef := strings.TrimSpace(os.Getenv(refVar))
	if repoURL == "" && repoRef == "" {
		return BundleRef{}, false, nil
	}
	if repoURL == "" {
		repoURL = DefaultGitRemoteURL(kind)
	}
	if cacheDir == "" {
		return BundleRef{}, true, fmt.Errorf("cache directory is required to clone %s", kind)
	}
	targetDir := localCloneCacheDir(cacheDir, kind, repoURL, repoRef)
	if ref, err := validateLocalBundleDir(kind, targetDir); err == nil {
		return ref, true, nil
	}
	if err := os.RemoveAll(targetDir); err != nil {
		return BundleRef{}, true, err
	}
	if err := os.MkdirAll(filepath.Dir(targetDir), 0o755); err != nil {
		return BundleRef{}, true, err
	}
	if err := cloneRepoToDir(repoURL, repoRef, targetDir); err != nil {
		return BundleRef{}, true, fmt.Errorf("clone override failed: %w", err)
	}
	ref, err := validateLocalBundleDir(kind, targetDir)
	if err != nil {
		return BundleRef{}, true, err
	}
	return ref, true, nil
}

func validateLocalBundleDir(kind, candidate string) (BundleRef, error) {
	entryName, err := EntryPointName(kind)
	if err != nil {
		return BundleRef{}, err
	}
	entrypoint := filepath.Join(candidate, entryName)
	info, err := os.Stat(entrypoint)
	if err != nil {
		return BundleRef{}, err
	}
	if info.IsDir() {
		return BundleRef{}, fmt.Errorf("entrypoint is a directory: %s", entrypoint)
	}
	return BundleRef{
		Kind:       kind,
		Source:     "local",
		Root:       candidate,
		Entrypoint: entrypoint,
	}, nil
}

func localBundleEnvNames(kind string) (dirVar, urlVar, refVar string, err error) {
	switch kind {
	case "core":
		return "PRODUCTIVE_K3S_CORE_REPO_DIR", "PRODUCTIVE_K3S_CORE_REPO_URL", "PRODUCTIVE_K3S_CORE_REPO_REF", nil
	case "infra":
		return "PRODUCTIVE_K3S_INFRA_REPO_DIR", "PRODUCTIVE_K3S_INFRA_REPO_URL", "PRODUCTIVE_K3S_INFRA_REPO_REF", nil
	default:
		return "", "", "", fmt.Errorf("unsupported bundle kind: %s", kind)
	}
}

func DefaultGitRemoteURL(kind string) string {
	repo, err := RepoNameDefault(kind)
	if err != nil {
		return ""
	}
	switch kind {
	case "profiles":
		return ProfilesGitRemoteURLDefault()
	default:
		return fmt.Sprintf("%s/%s.git", GitHubBaseURLDefault(), repo)
	}
}

func localCloneCacheDir(cacheDir, kind, repoURL, repoRef string) string {
	hash := sha256.Sum256([]byte(kind + "\n" + repoURL + "\n" + repoRef))
	return filepath.Join(cacheDir, "local-clones", kind, hex.EncodeToString(hash[:12]))
}

func cloneRepoToDir(repoURL, repoRef, targetDir string) error {
	args := []string{"clone", "--depth", "1"}
	if repoRef != "" {
		args = append(args, "--branch", repoRef)
	}
	args = append(args, repoURL, targetDir)
	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
