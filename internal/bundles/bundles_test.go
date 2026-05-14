package bundles

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveLocalSiblingBundle(t *testing.T) {
	root := t.TempDir()
	coreDir := filepath.Join(root, "productive-k3s-core")
	if err := os.MkdirAll(coreDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(coreDir, "productive-k3s-core.sh"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	ref, err := ResolveLocalSibling("core", root)
	if err != nil {
		t.Fatalf("expected local sibling resolution to succeed: %v", err)
	}
	if ref.Source != "local" {
		t.Fatalf("expected local source, got %q", ref.Source)
	}
	if ref.Entrypoint != filepath.Join(coreDir, "productive-k3s-core.sh") {
		t.Fatalf("unexpected entrypoint: %q", ref.Entrypoint)
	}
}

func TestResolveRemoteBundleDownloadsAndExtracts(t *testing.T) {
	archiveName := "productive-k3s-core-0.9.1.tar.gz"
	archiveBytes := buildTarGzFiles(t, map[string]string{
		"productive-k3s-core-0.9.1/productive-k3s-core.sh":            "#!/usr/bin/env bash\n",
		"productive-k3s-core-0.9.1/bundle-info.json":                  "{}\n",
		"productive-k3s-core-0.9.1/scripts/productive-k3s-core.sh":    "#!/usr/bin/env bash\n",
		"productive-k3s-core-0.9.1/scripts/preflight-host.sh":         "#!/usr/bin/env bash\n",
		"productive-k3s-core-0.9.1/scripts/bootstrap-k3s-stack.sh":    "#!/usr/bin/env bash\n",
		"productive-k3s-core-0.9.1/scripts/backup-k3s-stack.sh":       "#!/usr/bin/env bash\n",
		"productive-k3s-core-0.9.1/scripts/validate-k3s-stack.sh":     "#!/usr/bin/env bash\n",
		"productive-k3s-core-0.9.1/scripts/send-telemetry.sh":         "#!/usr/bin/env bash\n",
	})
	sum := sha256.Sum256(archiveBytes)
	checksum := hex.EncodeToString(sum[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/productive-k3s-core-0.9.1.tar.gz":
			_, _ = w.Write(archiveBytes)
		case "/productive-k3s-core-0.9.1.tar.gz.sha256":
			_, _ = w.Write([]byte(checksum + "  " + archiveName + "\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	ref, err := ResolveRemoteBundle(context.Background(), http.DefaultClient, cacheDir, RemoteSpec{
		Kind:        "core",
		Version:     "0.9.1",
		ArchiveName: archiveName,
		ArchiveURL:  server.URL + "/" + archiveName,
		ChecksumURL: server.URL + "/" + archiveName + ".sha256",
		Entrypoint:  "productive-k3s-core.sh",
	})
	if err != nil {
		t.Fatalf("expected remote resolution to succeed: %v", err)
	}

	if ref.Source != "remote" {
		t.Fatalf("expected remote source, got %q", ref.Source)
	}
	if _, err := os.Stat(ref.Entrypoint); err != nil {
		t.Fatalf("expected extracted entrypoint to exist: %v", err)
	}
}

func TestResolveRemoteCoreBundleRejectsIncompleteRelease(t *testing.T) {
	archiveName := "productive-k3s-core-0.9.1.tar.gz"
	archiveBytes := buildTarGzFiles(t, map[string]string{
		"productive-k3s-core-0.9.1/productive-k3s-core.sh": "#!/usr/bin/env bash\n",
	})
	sum := sha256.Sum256(archiveBytes)
	checksum := hex.EncodeToString(sum[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/" + archiveName:
			_, _ = w.Write(archiveBytes)
		case "/" + archiveName + ".sha256":
			_, _ = w.Write([]byte(checksum + "  " + archiveName + "\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	_, err := ResolveRemoteBundle(context.Background(), http.DefaultClient, t.TempDir(), RemoteSpec{
		Kind:        "core",
		Version:     "0.9.1",
		ArchiveName: archiveName,
		ArchiveURL:  server.URL + "/" + archiveName,
		ChecksumURL: server.URL + "/" + archiveName + ".sha256",
		Entrypoint:  "productive-k3s-core.sh",
	})
	if err == nil {
		t.Fatalf("expected incomplete core release to fail")
	}
	for _, expected := range []string{
		"bundle-info.json",
		"scripts/productive-k3s-core.sh",
		"scripts/preflight-host.sh",
		"scripts/bootstrap-k3s-stack.sh",
		"scripts/backup-k3s-stack.sh",
		"scripts/validate-k3s-stack.sh",
		"scripts/send-telemetry.sh",
	} {
		if !strings.Contains(err.Error(), expected) {
			t.Fatalf("expected missing file %q in error, got %v", expected, err)
		}
	}
}

func TestResolveRemoteInfraBundleRejectsIncompleteRelease(t *testing.T) {
	archiveName := "productive-k3s-infra-0.9.41-0.9.1.tar.gz"
	archiveBytes := buildTarGzFiles(t, map[string]string{
		"productive-k3s-infra-0.9.41-0.9.1/productive-k3s-infra.sh": "#!/usr/bin/env bash\n",
		"productive-k3s-infra-0.9.41-0.9.1/scripts/release.env":    "PK3S_INFRA_RELEASE_TAG=0.9.41-0.9.1\n",
	})
	sum := sha256.Sum256(archiveBytes)
	checksum := hex.EncodeToString(sum[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/" + archiveName:
			_, _ = w.Write(archiveBytes)
		case "/checksums.txt":
			_, _ = w.Write([]byte(checksum + "  " + archiveName + "\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	_, err := ResolveRemoteBundle(context.Background(), http.DefaultClient, t.TempDir(), RemoteSpec{
		Kind:        "infra",
		Version:     "0.9.41-0.9.1",
		ArchiveName: archiveName,
		ArchiveURL:  server.URL + "/" + archiveName,
		ChecksumURL: server.URL + "/checksums.txt",
		Entrypoint:  "productive-k3s-infra.sh",
	})
	if err == nil {
		t.Fatalf("expected incomplete infra release to fail")
	}
	for _, expected := range []string{
		"scenarios/multipass/opentofu/cloud-init/server.yaml",
		"scenarios/multipass/opentofu/cloud-init/agent-1.yaml",
		"scenarios/multipass/opentofu/cloud-init/agent-2.yaml",
		"ansible/roles/remote_cluster/files/bootstrap-agents.sh",
		"ansible/roles/remote_cluster/files/bootstrap-server.sh",
		"ansible/roles/remote_cluster/files/bootstrap-stack.sh",
		"ansible/roles/remote_cluster/files/cluster-up.sh",
		"scripts/release-config.sh",
		"ansible/roles/remote_cluster/files/common.sh",
		"ansible/roles/remote_cluster/files/preflight-productive-k3s-core.sh",
		"ansible/roles/remote_cluster/files/preflight.sh",
		"ansible/roles/remote_cluster/files/push-productive-k3s-core.sh",
		"ansible/roles/remote_cluster/files/reconcile-cluster-defaults.sh",
		"ansible/roles/remote_cluster/files/refresh-generated-artifacts.sh",
		"ansible/roles/remote_cluster/files/run_remote_bootstrap_session.py",
		"ansible/roles/remote_cluster/files/sync-hosts.sh",
		"ansible/roles/remote_cluster/files/validate-cluster.sh",
	} {
		if !strings.Contains(err.Error(), expected) {
			t.Fatalf("expected missing file %q in error, got %v", expected, err)
		}
	}
}

func TestResolveRemoteInfraBundleAcceptsCompleteRelease(t *testing.T) {
	archiveName := "productive-k3s-infra-0.9.41-0.9.1.tar.gz"
	archiveBytes := buildTarGzFiles(t, map[string]string{
		"productive-k3s-infra-0.9.41-0.9.1/productive-k3s-infra.sh":                                         "#!/usr/bin/env bash\n",
		"productive-k3s-infra-0.9.41-0.9.1/scripts/release.env":                                              "PK3S_INFRA_RELEASE_TAG=0.9.41-0.9.1\n",
		"productive-k3s-infra-0.9.41-0.9.1/scripts/release-config.sh":                                        "#!/usr/bin/env bash\n",
		"productive-k3s-infra-0.9.41-0.9.1/scenarios/multipass/opentofu/cloud-init/server.yaml":             "#cloud-config\n",
		"productive-k3s-infra-0.9.41-0.9.1/scenarios/multipass/opentofu/cloud-init/agent-1.yaml":            "#cloud-config\n",
		"productive-k3s-infra-0.9.41-0.9.1/scenarios/multipass/opentofu/cloud-init/agent-2.yaml":            "#cloud-config\n",
		"productive-k3s-infra-0.9.41-0.9.1/ansible/roles/remote_cluster/files/bootstrap-agents.sh":           "#!/usr/bin/env bash\n",
		"productive-k3s-infra-0.9.41-0.9.1/ansible/roles/remote_cluster/files/bootstrap-server.sh":           "#!/usr/bin/env bash\n",
		"productive-k3s-infra-0.9.41-0.9.1/ansible/roles/remote_cluster/files/bootstrap-stack.sh":            "#!/usr/bin/env bash\n",
		"productive-k3s-infra-0.9.41-0.9.1/ansible/roles/remote_cluster/files/cluster-up.sh":                 "#!/usr/bin/env bash\n",
		"productive-k3s-infra-0.9.41-0.9.1/ansible/roles/remote_cluster/files/common.sh":                     "#!/usr/bin/env bash\n",
		"productive-k3s-infra-0.9.41-0.9.1/ansible/roles/remote_cluster/files/preflight-productive-k3s-core.sh": "#!/usr/bin/env bash\n",
		"productive-k3s-infra-0.9.41-0.9.1/ansible/roles/remote_cluster/files/preflight.sh":                  "#!/usr/bin/env bash\n",
		"productive-k3s-infra-0.9.41-0.9.1/ansible/roles/remote_cluster/files/push-productive-k3s-core.sh":  "#!/usr/bin/env bash\n",
		"productive-k3s-infra-0.9.41-0.9.1/ansible/roles/remote_cluster/files/reconcile-cluster-defaults.sh": "#!/usr/bin/env bash\n",
		"productive-k3s-infra-0.9.41-0.9.1/ansible/roles/remote_cluster/files/refresh-generated-artifacts.sh": "#!/usr/bin/env bash\n",
		"productive-k3s-infra-0.9.41-0.9.1/ansible/roles/remote_cluster/files/run_remote_bootstrap_session.py": "print('ok')\n",
		"productive-k3s-infra-0.9.41-0.9.1/ansible/roles/remote_cluster/files/sync-hosts.sh":                "#!/usr/bin/env bash\n",
		"productive-k3s-infra-0.9.41-0.9.1/ansible/roles/remote_cluster/files/validate-cluster.sh":          "#!/usr/bin/env bash\n",
	})
	sum := sha256.Sum256(archiveBytes)
	checksum := hex.EncodeToString(sum[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/" + archiveName:
			_, _ = w.Write(archiveBytes)
		case "/checksums.txt":
			_, _ = w.Write([]byte(checksum + "  " + archiveName + "\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	ref, err := ResolveRemoteBundle(context.Background(), http.DefaultClient, t.TempDir(), RemoteSpec{
		Kind:        "infra",
		Version:     "0.9.41-0.9.1",
		ArchiveName: archiveName,
		ArchiveURL:  server.URL + "/" + archiveName,
		ChecksumURL: server.URL + "/checksums.txt",
		Entrypoint:  "productive-k3s-infra.sh",
	})
	if err != nil {
		t.Fatalf("expected complete infra release to succeed: %v", err)
	}
	if ref.Source != "remote" {
		t.Fatalf("expected remote source, got %q", ref.Source)
	}
}

func TestReleaseManifestMatchesExpectedVersions(t *testing.T) {
	manifest := DefaultReleaseManifest()
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte(`"core_version":"0.9.1"`)) {
		t.Fatalf("manifest missing core version: %s", raw)
	}
	if !bytes.Contains(raw, []byte(`"infra_version":"0.9.41-0.9.1"`)) {
		t.Fatalf("manifest missing infra version: %s", raw)
	}
}

func buildTarGz(t *testing.T, name, content string) []byte {
	t.Helper()

	return buildTarGzFiles(t, map[string]string{name: content})
}

func buildTarGzFiles(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	for name, content := range files {
		data := []byte(content)
		if err := tw.WriteHeader(&tar.Header{
			Name: name,
			Mode: 0o755,
			Size: int64(len(data)),
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(data); err != nil {
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
