package bundles

import (
	"fmt"
	"os"
	"path/filepath"
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
