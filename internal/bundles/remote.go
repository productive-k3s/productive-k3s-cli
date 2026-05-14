package bundles

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func ResolveRemoteBundle(ctx context.Context, httpClient *http.Client, cacheDir string, spec RemoteSpec) (BundleRef, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	bundleDir := CacheBundleDir(cacheDir, spec.Kind, spec.Version)
	if entrypoint, err := findEntrypoint(bundleDir, spec.Entrypoint); err == nil {
		root := filepath.Dir(entrypoint)
		if err := validateRemoteBundleContents(spec.Kind, root); err != nil {
			return BundleRef{}, err
		}
		return BundleRef{
			Kind:       spec.Kind,
			Version:    spec.Version,
			Source:     "remote",
			Root:       root,
			Entrypoint: entrypoint,
		}, nil
	}

	downloadsDir := filepath.Join(cacheDir, "downloads")
	if err := os.MkdirAll(downloadsDir, 0o755); err != nil {
		return BundleRef{}, err
	}
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		return BundleRef{}, err
	}

	archivePath := filepath.Join(downloadsDir, spec.ArchiveName)
	checksumPath := filepath.Join(downloadsDir, filepath.Base(spec.ChecksumURL))

	if err := downloadFile(ctx, httpClient, spec.ArchiveURL, archivePath); err != nil {
		return BundleRef{}, err
	}
	if err := downloadFile(ctx, httpClient, spec.ChecksumURL, checksumPath); err != nil {
		return BundleRef{}, err
	}
	if err := verifyChecksumFile(archivePath, checksumPath, spec.ArchiveName); err != nil {
		return BundleRef{}, err
	}
	if err := extractTarGz(archivePath, bundleDir); err != nil {
		return BundleRef{}, err
	}

	entrypoint, err := findEntrypoint(bundleDir, spec.Entrypoint)
	if err != nil {
		return BundleRef{}, err
	}
	root := filepath.Dir(entrypoint)
	if err := validateRemoteBundleContents(spec.Kind, root); err != nil {
		return BundleRef{}, err
	}

	return BundleRef{
		Kind:       spec.Kind,
		Version:    spec.Version,
		Source:     "remote",
		Root:       root,
		Entrypoint: entrypoint,
	}, nil
}

func validateRemoteBundleContents(kind, root string) error {
	required := requiredRemoteBundlePaths(kind)
	if len(required) == 0 {
		return nil
	}

	var missing []string
	for _, rel := range required {
		path := filepath.Join(root, rel)
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			missing = append(missing, rel)
		}
	}
	if len(missing) == 0 {
		return nil
	}

	sort.Strings(missing)
	return fmt.Errorf("incomplete %s bundle release: missing required files: %s", kind, strings.Join(missing, ", "))
}

func requiredRemoteBundlePaths(kind string) []string {
	switch kind {
	case "core":
		return []string{
			"bundle-info.json",
			"scripts/productive-k3s-core.sh",
			"scripts/preflight-host.sh",
			"scripts/bootstrap-k3s-stack.sh",
			"scripts/backup-k3s-stack.sh",
			"scripts/validate-k3s-stack.sh",
			"scripts/send-telemetry.sh",
		}
	case "infra":
		return []string{
			"scripts/release-config.sh",
			"scenarios/multipass/opentofu/cloud-init/server.yaml",
			"scenarios/multipass/opentofu/cloud-init/agent-1.yaml",
			"scenarios/multipass/opentofu/cloud-init/agent-2.yaml",
			"ansible/roles/remote_cluster/files/bootstrap-agents.sh",
			"ansible/roles/remote_cluster/files/bootstrap-server.sh",
			"ansible/roles/remote_cluster/files/bootstrap-stack.sh",
			"ansible/roles/remote_cluster/files/cluster-up.sh",
			"ansible/roles/remote_cluster/files/common.sh",
			"ansible/roles/remote_cluster/files/preflight-productive-k3s-core.sh",
			"ansible/roles/remote_cluster/files/preflight.sh",
			"ansible/roles/remote_cluster/files/push-productive-k3s-core.sh",
			"ansible/roles/remote_cluster/files/reconcile-cluster-defaults.sh",
			"ansible/roles/remote_cluster/files/refresh-generated-artifacts.sh",
			"ansible/roles/remote_cluster/files/run_remote_bootstrap_session.py",
			"ansible/roles/remote_cluster/files/sync-hosts.sh",
			"ansible/roles/remote_cluster/files/validate-cluster.sh",
		}
	default:
		return nil
	}
}

func downloadFile(ctx context.Context, httpClient *http.Client, url, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed for %s: %s", url, resp.Status)
	}

	tmp := dest + ".tmp"
	file, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(file, resp.Body); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, dest)
}

func verifyChecksumFile(archivePath, checksumPath, archiveName string) error {
	expected, err := expectedChecksum(checksumPath, archiveName)
	if err != nil {
		return err
	}

	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("checksum mismatch for %s", archiveName)
	}
	return nil
}

func expectedChecksum(checksumPath, archiveName string) (string, error) {
	content, err := os.ReadFile(checksumPath)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(content), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := filepath.Base(strings.TrimPrefix(fields[1], "*"))
		if name == archiveName {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("checksum for %s not found in %s", archiveName, checksumPath)
}

func extractTarGz(archivePath, dest string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gz, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		target := filepath.Join(dest, header.Name)
		if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid archive path: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		}
	}
}

func findEntrypoint(root, entrypointName string) (string, error) {
	var found string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Base(path) == entrypointName {
			found = path
			return io.EOF
		}
		return nil
	})
	if err != nil && err != io.EOF {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("entrypoint %s not found under %s", entrypointName, root)
	}
	return found, nil
}
