package tools

import (
	"os"
	"slices"
	"testing"
)

func TestEnvWithKubeconfigOnlyChangesChildEnv(t *testing.T) {
	t.Setenv("KUBECONFIG", "parent")
	parent := os.Getenv("KUBECONFIG")

	child := EnvWithKubeconfig([]string{"PATH=/bin", "KUBECONFIG=old"}, "/tmp/pk3s.yaml")
	if os.Getenv("KUBECONFIG") != parent {
		t.Fatalf("parent KUBECONFIG changed")
	}
	if !slices.Contains(child, "KUBECONFIG=/tmp/pk3s.yaml") {
		t.Fatalf("child env missing kubeconfig override: %#v", child)
	}
}
