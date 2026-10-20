package bundles

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepositoryAndURLDefaults(t *testing.T) {
	t.Setenv("PRODUCTIVE_K3S_GITHUB_OWNER_DEFAULT", "example")
	t.Setenv("PRODUCTIVE_K3S_GITHUB_BASE_URL_DEFAULT", "https://git.example")
	t.Setenv("PRODUCTIVE_K3S_GITHUB_RAW_BASE_URL_DEFAULT", "https://raw.example")
	if GitHubOwnerDefault() != "example" || GitHubBaseURLDefault() != "https://git.example" || GitHubRawBaseURLDefault() != "https://raw.example" {
		t.Fatal("github defaults did not honor environment")
	}
	for _, kind := range []string{"core", "infra", "profiles", "cli"} {
		if value, err := RepoNameDefault(kind); err != nil || !strings.Contains(value, kind) {
			t.Fatalf("unexpected repo default for %s: %q %v", kind, value, err)
		}
	}
	if _, err := RepoNameDefault("bad"); err == nil {
		t.Fatal("unsupported repository kind accepted")
	}
	for _, kind := range []string{"core", "infra"} {
		if value, err := ReleaseRepoDefault(kind); err != nil || !strings.HasPrefix(value, "example/") {
			t.Fatalf("unexpected release repo for %s: %q %v", kind, value, err)
		}
	}
	if _, err := ReleaseRepoDefault("profiles"); err == nil {
		t.Fatal("unsupported release kind accepted")
	}
	if got := ProfilesGitRemoteURLDefault(); got != "https://git.example/productive-k3s-profiles.git" {
		t.Fatalf("unexpected profiles remote: %q", got)
	}
	if got := ProfilesRawURLDefault("/path/file.env", ""); got != "https://raw.example/productive-k3s-profiles/main/path/file.env" {
		t.Fatalf("unexpected profiles raw URL: %q", got)
	}
	if MultipassProfileURLDefault() == "" || CatalogURLDefault() == "" {
		t.Fatal("public defaults must not be empty")
	}
}

func TestCatalogSHA256Default(t *testing.T) {
	t.Run("release fallback", func(t *testing.T) {
		if err := os.Unsetenv("PRODUCTIVE_K3S_CATALOG_SHA256_DEFAULT"); err != nil {
			t.Fatal(err)
		}
		if got := CatalogSHA256Default(); got != "93be2e919e58b6ce9eadc38505490625ec4f67d6855642b63adc4c4c9d0f46a7" {
			t.Fatalf("unexpected release catalog digest: %q", got)
		}
	})

	t.Run("explicit override", func(t *testing.T) {
		t.Setenv("PRODUCTIVE_K3S_CATALOG_SHA256_DEFAULT", "  custom-digest  ")
		if got := CatalogSHA256Default(); got != "custom-digest" {
			t.Fatalf("unexpected catalog digest override: %q", got)
		}
	})

	t.Run("explicit development opt out", func(t *testing.T) {
		t.Setenv("PRODUCTIVE_K3S_CATALOG_SHA256_DEFAULT", "")
		if got := CatalogSHA256Default(); got != "" {
			t.Fatalf("expected empty catalog digest override, got %q", got)
		}
	})
}

func TestLocalBundleEnvironmentAndRemoteDefaults(t *testing.T) {
	for _, kind := range []string{"core", "infra"} {
		dirVar, urlVar, refVar, err := localBundleEnvNames(kind)
		if err != nil || dirVar == "" || urlVar == "" || refVar == "" {
			t.Fatalf("unexpected env names for %s: %q %q %q %v", kind, dirVar, urlVar, refVar, err)
		}
		if remote := DefaultGitRemoteURL(kind); !strings.HasSuffix(remote, "productive-k3s-"+kind+".git") {
			t.Fatalf("unexpected remote for %s: %q", kind, remote)
		}
	}
	if _, _, _, err := localBundleEnvNames("bad"); err == nil {
		t.Fatal("unsupported local bundle kind accepted")
	}
	if DefaultGitRemoteURL("bad") != "" {
		t.Fatal("unsupported remote kind returned a URL")
	}

	repoDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoDir, "productive-k3s-core.sh"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PRODUCTIVE_K3S_CORE_REPO_DIR", "")
	t.Setenv("PRODUCTIVE_K3S_REPO", repoDir)
	ref, ok, err := resolveExplicitLocalBundle("core")
	if err != nil || !ok || ref.Root != repoDir {
		t.Fatalf("legacy core override failed: %#v %v %v", ref, ok, err)
	}
}
