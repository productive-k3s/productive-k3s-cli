package bundles

import (
	"os"
	"path/filepath"
	"runtime"
)

func DefaultCacheDir() (string, error) {
	if runtime.GOOS == "windows" {
		if localAppData := os.Getenv("LocalAppData"); localAppData != "" {
			return filepath.Join(localAppData, "pk3s"), nil
		}
	}

	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "pk3s"), nil
}
