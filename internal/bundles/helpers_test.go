package bundles

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDefaultCacheDirUsesUserCacheDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("linux/mac cache path test")
	}
	t.Setenv("XDG_CACHE_HOME", filepath.Join(t.TempDir(), "cache"))

	got, err := DefaultCacheDir()
	if err != nil {
		t.Fatalf("DefaultCacheDir: %v", err)
	}
	if !strings.HasSuffix(got, filepath.Join("cache", "pk3s")) {
		t.Fatalf("unexpected cache dir %q", got)
	}
}

func TestDefaultRemoteSpecAndBundleHelpers(t *testing.T) {
	coreSpec, err := DefaultRemoteSpec("core")
	if err != nil {
		t.Fatalf("DefaultRemoteSpec(core): %v", err)
	}
	if coreSpec.ArchiveName != "productive-k3s-core-0.9.5.tar.gz" {
		t.Fatalf("unexpected core archive name: %q", coreSpec.ArchiveName)
	}

	infraSpec, err := DefaultRemoteSpec("infra")
	if err != nil {
		t.Fatalf("DefaultRemoteSpec(infra): %v", err)
	}
	if infraSpec.ChecksumURL == "" || !strings.Contains(infraSpec.ChecksumURL, "checksums.txt") {
		t.Fatalf("unexpected infra checksum url: %q", infraSpec.ChecksumURL)
	}
	if infraSpec.ArchiveName != "productive-k3s-infra-0.9.64-0.9.5.tar.gz" {
		t.Fatalf("unexpected infra archive name: %q", infraSpec.ArchiveName)
	}

	if _, err := DefaultRemoteSpec("wat"); err == nil {
		t.Fatalf("expected unsupported bundle kind to fail")
	}
	if _, err := EntryPointName("wat"); err == nil {
		t.Fatalf("expected unsupported entrypoint kind to fail")
	}
	if _, err := LocalRepoName("wat"); err == nil {
		t.Fatalf("expected unsupported local repo kind to fail")
	}
}

func TestResolveLocalSiblingUsesParentCandidateAndFailsCleanly(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	startDir := filepath.Join(workspace, "productive-k3s-cli")
	coreDir := filepath.Join(workspace, "productive-k3s-core")
	if err := os.MkdirAll(startDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(coreDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(coreDir, "productive-k3s-core.sh"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	ref, err := ResolveLocalSibling("core", startDir)
	if err != nil {
		t.Fatalf("ResolveLocalSibling parent candidate: %v", err)
	}
	if ref.Root != coreDir {
		t.Fatalf("unexpected root: %q", ref.Root)
	}

	if _, err := ResolveLocalSibling("infra", t.TempDir()); err == nil {
		t.Fatalf("expected missing local sibling to fail")
	}
}

func TestResolveLocalBundleUsesExplicitRepoDir(t *testing.T) {
	repoDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoDir, "productive-k3s-core.sh"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PRODUCTIVE_K3S_CORE_REPO_DIR", repoDir)
	t.Setenv("PRODUCTIVE_K3S_CORE_REPO_URL", "")
	t.Setenv("PRODUCTIVE_K3S_CORE_REPO_REF", "")

	ref, err := ResolveLocalBundle("core", t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatalf("ResolveLocalBundle explicit dir: %v", err)
	}
	if ref.Root != repoDir {
		t.Fatalf("unexpected root: %q", ref.Root)
	}
}

func TestResolveLocalBundleClonesRepoOverride(t *testing.T) {
	originDir := t.TempDir()
	runGit(t, originDir, "init", "-b", "development")
	runGit(t, originDir, "config", "user.email", "test@example.com")
	runGit(t, originDir, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(originDir, "productive-k3s-core.sh"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	runGit(t, originDir, "add", "productive-k3s-core.sh")
	runGit(t, originDir, "commit", "-m", "seed")

	t.Setenv("PRODUCTIVE_K3S_CORE_REPO_DIR", "")
	t.Setenv("PRODUCTIVE_K3S_CORE_REPO_URL", originDir)
	t.Setenv("PRODUCTIVE_K3S_CORE_REPO_REF", "development")

	ref, err := ResolveLocalBundle("core", t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatalf("ResolveLocalBundle repo override: %v", err)
	}
	if ref.Root == originDir {
		t.Fatalf("expected cloned repo, got original dir %q", ref.Root)
	}
	if _, err := os.Stat(filepath.Join(ref.Root, "productive-k3s-core.sh")); err != nil {
		t.Fatalf("expected cloned entrypoint to exist: %v", err)
	}
}

func TestDownloadFileFailsOnHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusBadGateway)
	}))
	defer server.Close()

	err := downloadFile(context.Background(), server.Client(), server.URL, filepath.Join(t.TempDir(), "artifact.tgz"))
	if err == nil || !strings.Contains(err.Error(), "download failed") {
		t.Fatalf("expected download failure, got %v", err)
	}
}

func TestExpectedChecksumSupportsWildcardAndVerifyMismatch(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "bundle.tgz")
	if err := os.WriteFile(archive, []byte("bundle"), 0o644); err != nil {
		t.Fatal(err)
	}
	checksum := filepath.Join(dir, "checksums.txt")
	if err := os.WriteFile(checksum, []byte("deadbeef *bundle.tgz\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := expectedChecksum(checksum, "bundle.tgz")
	if err != nil {
		t.Fatalf("expectedChecksum: %v", err)
	}
	if got != "deadbeef" {
		t.Fatalf("unexpected checksum %q", got)
	}
	if err := verifyChecksumFile(archive, checksum, "bundle.tgz"); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected checksum mismatch, got %v", err)
	}
	if _, err := expectedChecksum(checksum, "missing.tgz"); err == nil {
		t.Fatalf("expected missing checksum lookup to fail")
	}
}

func TestExtractTarGzRejectsPathTraversalAndFindEntrypointErrors(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "bad.tar.gz")
	raw := buildTarGzBytes(t, map[string]string{
		"../escape.sh": "#!/usr/bin/env bash\n",
	})
	if err := os.WriteFile(archivePath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := extractTarGz(archivePath, t.TempDir()); err == nil || !strings.Contains(err.Error(), "invalid archive path") {
		t.Fatalf("expected invalid archive path error, got %v", err)
	}

	if _, err := findEntrypoint(t.TempDir(), "missing.sh"); err == nil {
		t.Fatalf("expected missing entrypoint to fail")
	}
}

func buildTarGzBytes(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, content := range files {
		body := []byte(content)
		if err := tw.WriteHeader(&tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(body)),
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v: %s", args, err, strings.TrimSpace(string(out)))
	}
}
