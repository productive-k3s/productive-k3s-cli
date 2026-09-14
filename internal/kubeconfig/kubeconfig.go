package kubeconfig

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func DefaultClusterDir(id string) (string, error) {
	if base := strings.TrimSpace(os.Getenv("PK3S_CLUSTER_CONFIG_DIR")); base != "" {
		return filepath.Join(base, SafeName(id)), nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "pk3s", "clusters", SafeName(id)), nil
}

func ManagedPath(id string) (string, error) {
	dir, err := DefaultClusterDir(id)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "kubeconfig.yaml"), nil
}

func CopyManaged(id string, source string) (string, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return "", fmt.Errorf("kubeconfig path is required")
	}
	target, err := ManagedPath(id)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", err
	}
	in, err := os.Open(source)
	if err != nil {
		return "", err
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return "", err
	}
	if err := out.Close(); err != nil {
		return "", err
	}
	_ = os.Chmod(target, 0o600)
	return target, nil
}

func WriteManaged(id string, content string) (string, error) {
	target, err := ManagedPath(id)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(target, []byte(content), 0o600); err != nil {
		return "", err
	}
	_ = os.Chmod(target, 0o600)
	return target, nil
}

func ContextName(id string) string {
	return "pk3s-" + SafeName(id)
}

func SafeName(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if ok {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	result := strings.Trim(b.String(), "-")
	if result == "" {
		return "cluster"
	}
	return result
}
