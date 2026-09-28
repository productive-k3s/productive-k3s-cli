package kubeconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyManagedUsesUserScopedClusterDirAndPrivateFile(t *testing.T) {
	base := t.TempDir()
	t.Setenv("PK3S_CLUSTER_CONFIG_DIR", base)
	source := filepath.Join(t.TempDir(), "k3s.yaml")
	if err := os.WriteFile(source, []byte("apiVersion: v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	target, err := CopyManaged("Local Dev", source)
	if err != nil {
		t.Fatalf("copy failed: %v", err)
	}
	if target != filepath.Join(base, "local-dev", "kubeconfig.yaml") {
		t.Fatalf("unexpected target: %s", target)
	}
	body, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target failed: %v", err)
	}
	if string(body) != "apiVersion: v1\n" {
		t.Fatalf("unexpected target body: %q", string(body))
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected private kubeconfig permissions, got %o", info.Mode().Perm())
	}
}

func TestContextNameIsDeterministic(t *testing.T) {
	if got := ContextName("Local Dev"); got != "pk3s-local-dev" {
		t.Fatalf("unexpected context name: %s", got)
	}
}
