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

func TestDetectAndCurrentEnvironment(t *testing.T) {
	found := Detect("sh")
	if !found.Found || found.Path == "" {
		t.Fatalf("expected sh to be detected: %#v", found)
	}
	missing := Detect("pk3s-command-that-does-not-exist")
	if missing.Found || missing.ErrorText == "" {
		t.Fatalf("expected missing tool status: %#v", missing)
	}
	t.Setenv("PK3S_COVERAGE_SENTINEL", "present")
	env := CurrentEnvWithKubeconfig("/tmp/current.yaml")
	if !slices.Contains(env, "KUBECONFIG=/tmp/current.yaml") || !slices.Contains(env, "PK3S_COVERAGE_SENTINEL=present") {
		t.Fatalf("unexpected current child environment: %#v", env)
	}
}
