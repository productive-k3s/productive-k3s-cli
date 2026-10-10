package clusters

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRegistryUpsertListGetDelete(t *testing.T) {
	reg := NewRegistry(filepath.Join(t.TempDir(), "registry.json"))

	if err := reg.Upsert(Cluster{ID: "local-dev", Kubeconfig: "/tmp/kubeconfig", Context: "pk3s-local-dev"}); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}
	if err := reg.Upsert(Cluster{ID: "edge", Name: "Edge", Kubeconfig: "/tmp/edge", Context: "pk3s-edge"}); err != nil {
		t.Fatalf("second upsert failed: %v", err)
	}

	items, err := reg.List()
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(items) != 2 || items[0].ID != "edge" || items[1].ID != "local-dev" {
		t.Fatalf("unexpected sorted items: %#v", items)
	}

	got, err := reg.Get("local-dev")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got.Name != "local-dev" || got.Status != "Unknown" {
		t.Fatalf("unexpected defaults: %#v", got)
	}

	if err := reg.Delete("local-dev"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	items, err = reg.List()
	if err != nil {
		t.Fatalf("list after delete failed: %v", err)
	}
	if len(items) != 1 || items[0].ID != "edge" {
		t.Fatalf("unexpected items after delete: %#v", items)
	}
}

func TestRegistryDefaultsAndFailurePaths(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registry.json")
	t.Setenv("PK3S_CLUSTER_REGISTRY_PATH", path)
	if got, err := DefaultRegistryPath(); err != nil || got != path {
		t.Fatalf("unexpected default registry path: %q %v", got, err)
	}
	reg := NewRegistry(path)
	if _, err := reg.Get("missing"); err == nil {
		t.Fatal("missing cluster was returned")
	}
	if err := reg.Delete("missing"); err == nil {
		t.Fatal("missing cluster was deleted")
	}
	if err := reg.Upsert(Cluster{}); err == nil {
		t.Fatal("empty cluster id accepted")
	}
	if err := os.WriteFile(path, []byte("not-json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := reg.List(); err == nil {
		t.Fatal("invalid registry JSON accepted")
	}
	if _, err := NewRegistry("").List(); err == nil {
		t.Fatal("empty registry path accepted")
	}
}
