package bundles

import (
	"fmt"
	"path/filepath"
)

type BundleRef struct {
	Kind       string
	Version    string
	Source     string
	Root       string
	Entrypoint string
}

type RemoteSpec struct {
	Kind        string
	Version     string
	ArchiveName string
	ArchiveURL  string
	ChecksumURL string
	Entrypoint  string
}

type ReleaseManifest struct {
	CoreVersion  string `json:"core_version"`
	InfraVersion string `json:"infra_version"`
}

func DefaultReleaseManifest() ReleaseManifest {
	return ReleaseManifest{
		CoreVersion:  "0.9.1",
		InfraVersion: "0.9.41-0.9.1",
	}
}

func DefaultRemoteSpec(kind string) (RemoteSpec, error) {
	manifest := DefaultReleaseManifest()
	switch kind {
	case "core":
		version := manifest.CoreVersion
		archiveName := fmt.Sprintf("productive-k3s-core-%s.tar.gz", version)
		baseURL := fmt.Sprintf("https://github.com/jemacchi/productive-k3s-core/releases/download/%s", version)
		return RemoteSpec{
			Kind:        "core",
			Version:     version,
			ArchiveName: archiveName,
			ArchiveURL:  fmt.Sprintf("%s/%s", baseURL, archiveName),
			ChecksumURL: fmt.Sprintf("%s/%s.sha256", baseURL, archiveName),
			Entrypoint:  "productive-k3s-core.sh",
		}, nil
	case "infra":
		version := manifest.InfraVersion
		archiveName := fmt.Sprintf("productive-k3s-infra-%s.tar.gz", version)
		baseURL := fmt.Sprintf("https://github.com/jemacchi/productive-k3s-infra/releases/download/%s", version)
		return RemoteSpec{
			Kind:        "infra",
			Version:     version,
			ArchiveName: archiveName,
			ArchiveURL:  fmt.Sprintf("%s/%s", baseURL, archiveName),
			ChecksumURL: fmt.Sprintf("%s/checksums.txt", baseURL),
			Entrypoint:  "productive-k3s-infra.sh",
		}, nil
	default:
		return RemoteSpec{}, fmt.Errorf("unsupported bundle kind: %s", kind)
	}
}

func EntryPointName(kind string) (string, error) {
	switch kind {
	case "core":
		return "productive-k3s-core.sh", nil
	case "infra":
		return "productive-k3s-infra.sh", nil
	default:
		return "", fmt.Errorf("unsupported bundle kind: %s", kind)
	}
}

func LocalRepoName(kind string) (string, error) {
	switch kind {
	case "core":
		return "productive-k3s-core", nil
	case "infra":
		return "productive-k3s-infra", nil
	default:
		return "", fmt.Errorf("unsupported bundle kind: %s", kind)
	}
}

func CacheBundleDir(cacheDir, kind, version string) string {
	return filepath.Join(cacheDir, "bundles", kind, version)
}
