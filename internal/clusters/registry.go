package clusters

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Cluster struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Type          string   `json:"type,omitempty"`
	CreatedAt     string   `json:"created_at,omitempty"`
	Status        string   `json:"status,omitempty"`
	APIServer     string   `json:"api_server,omitempty"`
	Kubeconfig    string   `json:"kubeconfig"`
	Context       string   `json:"context"`
	Profile       string   `json:"profile,omitempty"`
	Stacks        []string `json:"stacks,omitempty"`
	Addons        []string `json:"addons,omitempty"`
	KubeVersion   string   `json:"kubernetes_version,omitempty"`
	NodeCount     int      `json:"node_count,omitempty"`
	LastCheckedAt string   `json:"last_checked_at,omitempty"`
}

type Registry struct {
	Path string
}

type registryFile struct {
	Clusters []Cluster `json:"clusters"`
}

func DefaultRegistryPath() (string, error) {
	if path := strings.TrimSpace(os.Getenv("PK3S_CLUSTER_REGISTRY_PATH")); path != "" {
		return path, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "pk3s", "clusters", "registry.json"), nil
}

func NewRegistry(path string) Registry {
	return Registry{Path: path}
}

func (r Registry) List() ([]Cluster, error) {
	data, err := r.read()
	if err != nil {
		return nil, err
	}
	clusters := append([]Cluster(nil), data.Clusters...)
	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].ID < clusters[j].ID
	})
	return clusters, nil
}

func (r Registry) Get(id string) (Cluster, error) {
	id = strings.TrimSpace(id)
	data, err := r.read()
	if err != nil {
		return Cluster{}, err
	}
	for _, cluster := range data.Clusters {
		if cluster.ID == id {
			return cluster, nil
		}
	}
	return Cluster{}, fmt.Errorf("cluster not found: %s", id)
}

func (r Registry) Upsert(cluster Cluster) error {
	cluster.ID = strings.TrimSpace(cluster.ID)
	if cluster.ID == "" {
		return fmt.Errorf("cluster id is required")
	}
	if cluster.Name == "" {
		cluster.Name = cluster.ID
	}
	if cluster.Context == "" {
		cluster.Context = "pk3s-" + cluster.ID
	}
	if cluster.CreatedAt == "" {
		cluster.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if cluster.Status == "" {
		cluster.Status = "Unknown"
	}
	data, err := r.read()
	if err != nil {
		return err
	}
	replaced := false
	for i := range data.Clusters {
		if data.Clusters[i].ID == cluster.ID {
			if cluster.CreatedAt == "" {
				cluster.CreatedAt = data.Clusters[i].CreatedAt
			}
			data.Clusters[i] = cluster
			replaced = true
			break
		}
	}
	if !replaced {
		data.Clusters = append(data.Clusters, cluster)
	}
	return r.write(data)
}

func (r Registry) Delete(id string) error {
	id = strings.TrimSpace(id)
	data, err := r.read()
	if err != nil {
		return err
	}
	next := data.Clusters[:0]
	found := false
	for _, cluster := range data.Clusters {
		if cluster.ID == id {
			found = true
			continue
		}
		next = append(next, cluster)
	}
	if !found {
		return fmt.Errorf("cluster not found: %s", id)
	}
	data.Clusters = next
	return r.write(data)
}

func (r Registry) read() (registryFile, error) {
	if strings.TrimSpace(r.Path) == "" {
		return registryFile{}, fmt.Errorf("registry path is required")
	}
	body, err := os.ReadFile(r.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return registryFile{}, nil
		}
		return registryFile{}, err
	}
	var data registryFile
	if err := json.Unmarshal(body, &data); err != nil {
		return registryFile{}, err
	}
	return data, nil
}

func (r Registry) write(data registryFile) error {
	if err := os.MkdirAll(filepath.Dir(r.Path), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.Path, append(body, '\n'), 0o600)
}
