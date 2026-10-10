package app

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/productive-k3s/productive-k3s-cli/internal/clusters"
	"github.com/productive-k3s/productive-k3s-cli/internal/tui"
)

func TestClusterCommandUsageAndToolMatrix(t *testing.T) {
	t.Setenv("PK3S_CLUSTER_REGISTRY_PATH", filepath.Join(t.TempDir(), "registry.json"))
	missingTools := t.TempDir()
	t.Setenv("PATH", missingTools)

	cases := []struct {
		args []string
		code int
		text string
	}{
		{[]string{"cluster"}, 2, "Usage: pk3s cluster"},
		{[]string{"cluster", "show"}, 2, "cluster show"},
		{[]string{"cluster", "show", "missing"}, 2, "cluster not found"},
		{[]string{"cluster", "register"}, 2, "cluster register"},
		{[]string{"cluster", "remove"}, 2, "cluster remove"},
		{[]string{"cluster", "remove", "missing"}, 2, "cluster not found"},
		{[]string{"cluster", "test"}, 2, "cluster test"},
		{[]string{"cluster", "kubectl"}, 2, "cluster kubectl"},
		{[]string{"cluster", "k9s"}, 2, "cluster k9s"},
		{[]string{"cluster", "unknown"}, 2, "Unsupported cluster command"},
	}
	for _, tc := range cases {
		var stderr bytes.Buffer
		code := Run(context.Background(), tc.args, Dependencies{Stdout: &bytes.Buffer{}, Stderr: &stderr, GOOS: "linux", GOARCH: "amd64"})
		if code != tc.code || !strings.Contains(stderr.String(), tc.text) {
			t.Fatalf("args=%v code=%d stderr=%q", tc.args, code, stderr.String())
		}
	}

	var stdout bytes.Buffer
	if code := Run(context.Background(), []string{"cluster", "tools"}, Dependencies{Stdout: &stdout, Stderr: &bytes.Buffer{}, GOOS: "linux", GOARCH: "amd64"}); code != 0 {
		t.Fatalf("cluster tools failed: %d", code)
	}
	if !strings.Contains(stdout.String(), "kubectl\tnot-found") || !strings.Contains(stdout.String(), "k9s\tnot-found") {
		t.Fatalf("unexpected tools output: %q", stdout.String())
	}
}

func TestClusterRegisterFlagValidation(t *testing.T) {
	for _, tc := range []struct {
		args []string
		text string
	}{
		{[]string{"--name"}, "missing value"},
		{[]string{"--unknown", "x"}, "unsupported"},
		{[]string{"--name", "demo"}, "requires --kubeconfig"},
	} {
		var stderr bytes.Buffer
		if _, code := parseClusterRegisterFlags(tc.args, &stderr); code != 2 || !strings.Contains(stderr.String(), tc.text) {
			t.Fatalf("args=%v code=%d stderr=%q", tc.args, code, stderr.String())
		}
	}
	flags, code := parseClusterRegisterFlags([]string{"--kubeconfig", "/tmp/k", "--name", "Demo", "--type", "local", "--status", "Ready"}, &bytes.Buffer{})
	if code != 0 || flags["type"] != "local" || flags["status"] != "Ready" {
		t.Fatalf("unexpected parsed flags: %#v code=%d", flags, code)
	}
}

func TestRenderClusterIncludesOptionalState(t *testing.T) {
	var output bytes.Buffer
	renderCluster(&output, clusters.Cluster{
		ID: "demo", Name: "Demo", Type: "local", Status: "Ready",
		APIServer: "https://127.0.0.1:6443", Kubeconfig: "/tmp/k", Context: "ctx",
		Profile: "dev", Stacks: []string{"base"}, Addons: []string{"nginx"},
		KubeVersion: "v1.35.0", NodeCount: 2, LastCheckedAt: "now",
	})
	for _, expected := range []string{"Type: local", "API server:", "Profile: dev", "Stacks: base", "Add-ons: nginx", "Kubernetes: v1.35.0", "Nodes: 2", "Last checked: now"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("missing %q in %q", expected, output.String())
		}
	}
}

func TestRunBOMUsageAndBundleResolutionFailures(t *testing.T) {
	var stderr bytes.Buffer
	if code := Run(context.Background(), []string{"bom"}, Dependencies{Stdout: &bytes.Buffer{}, Stderr: &stderr, GOOS: "linux", GOARCH: "amd64"}); code != 2 {
		t.Fatalf("expected BOM usage failure, got %d", code)
	}

	workingDir := t.TempDir()
	t.Setenv("PRODUCTIVE_K3S_SOURCE", "local")
	stderr.Reset()
	code := Run(context.Background(), []string{"bom", "--json"}, noTelemetryDeps(t, Dependencies{
		Stdout: &bytes.Buffer{}, Stderr: &stderr, GOOS: "linux", GOARCH: "amd64",
		WorkingDir: workingDir, CacheDir: filepath.Join(workingDir, "cache"),
	}))
	if code != 1 || !strings.Contains(stderr.String(), "could not resolve core BOM") {
		t.Fatalf("unexpected missing core result: code=%d stderr=%q", code, stderr.String())
	}
}

func TestResolveDelegatedBOMRejectsInvalidOutputAndFallbackFailure(t *testing.T) {
	workingDir := t.TempDir()
	t.Setenv("PRODUCTIVE_K3S_SOURCE", "local")
	coreDir := filepath.Join(workingDir, "productive-k3s-core")
	if err := os.MkdirAll(coreDir, 0o755); err != nil {
		t.Fatal(err)
	}
	entrypoint := filepath.Join(coreDir, "productive-k3s-core.sh")
	if err := os.WriteFile(entrypoint, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	deps := Dependencies{WorkingDir: workingDir, CacheDir: filepath.Join(workingDir, "cache"), RunOutput: func(context.Context, Invocation) ([]byte, error) {
		return []byte("not-json"), nil
	}}
	if _, err := resolveDelegatedBOM(context.Background(), deps, "core"); err == nil {
		t.Fatal("invalid delegated BOM accepted")
	}

	deps.RunOutput = func(_ context.Context, invocation Invocation) ([]byte, error) {
		if len(invocation.Args) > 0 && invocation.Args[0] == "bom" {
			return nil, errors.New("bom unavailable")
		}
		return nil, errors.New("bundle info unavailable")
	}
	if _, err := resolveDelegatedBOM(context.Background(), deps, "core"); err == nil || !strings.Contains(err.Error(), "bom unavailable") {
		t.Fatalf("unexpected fallback error: %v", err)
	}
}

func TestStreamWriterAndCommandEventSupport(t *testing.T) {
	var output bytes.Buffer
	var lines []string
	writer := streamWriter{buffer: &output, sink: func(line string) { lines = append(lines, line) }}
	if n, err := writer.Write([]byte("one\ntwo\n")); err != nil || n != 8 {
		t.Fatalf("write failed: n=%d err=%v", n, err)
	}
	if output.String() != "one\ntwo\n" || strings.Join(lines, ",") != "one,two" {
		t.Fatalf("unexpected stream output=%q lines=%#v", output.String(), lines)
	}
	if !commandSupportsOperationEvents(Invocation{Path: "/tmp/productive-k3s-core.sh"}) ||
		!commandSupportsOperationEvents(Invocation{Path: "productive-k3s-infra.sh"}) ||
		commandSupportsOperationEvents(Invocation{Path: "other.sh"}) {
		t.Fatal("operation event command detection failed")
	}
	if got := withOperationEvents([]string{"--events", "ndjson", "status"}); strings.Join(got, " ") != "--events ndjson status" {
		t.Fatalf("existing event args changed: %#v", got)
	}
}

func TestOSExecOutputCapturesCommandOutput(t *testing.T) {
	out, err := osExecOutput(context.Background(), Invocation{Path: "/bin/sh", Args: []string{"-c", "printf captured"}})
	if err != nil || string(out) != "captured" {
		t.Fatalf("unexpected output=%q err=%v", out, err)
	}
}

func TestRunEventedInvocationSeparatesEventsAndLogs(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "productive-k3s-core.sh")
	body := `#!/bin/sh
printf '%s\n' '{"schema_version":"productive-k3s-operation-event/v1","component":"core","operation":"addon.install","step":"operation.started","status":"running","message":"Started"}'
printf '%s\n' 'human output'
printf '%s\n' 'human error' >&2
`
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	var eventSteps, logs []string
	err := runEventedInvocation(
		context.Background(),
		Invocation{Path: script, Args: []string{"status"}},
		func(event tui.OperationEvent) { eventSteps = append(eventSteps, event.Step) },
		func(line string) { logs = append(logs, line) },
		&stdout,
		&stderr,
	)
	if err != nil {
		t.Fatalf("evented invocation failed: %v", err)
	}
	if strings.Join(eventSteps, ",") != "operation.started" {
		t.Fatalf("unexpected events: %#v", eventSteps)
	}
	if stdout.String() != "human output\n" || !strings.Contains(stderr.String(), "human error") {
		t.Fatalf("unexpected stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if !strings.Contains(strings.Join(logs, ","), "human output") || !strings.Contains(strings.Join(logs, ","), "human error") {
		t.Fatalf("unexpected logs: %#v", logs)
	}
}

func TestResolveTGZPathLocalRemoteAndFailure(t *testing.T) {
	local := filepath.Join(t.TempDir(), "package.tgz")
	if err := os.WriteFile(local, []byte("tgz"), 0o600); err != nil {
		t.Fatal(err)
	}
	deps := Dependencies{Stderr: &bytes.Buffer{}, CacheDir: t.TempDir(), HTTPClient: http.DefaultClient}
	if got, code, ok := resolveTGZPath(local, deps); !ok || code != 0 || got != local {
		t.Fatalf("unexpected local result: %q %d %v", got, code, ok)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("remote-tgz"))
	}))
	defer server.Close()
	deps.HTTPClient = server.Client()
	got, code, ok := resolveTGZPath(server.URL+"/package.tgz", deps)
	if !ok || code != 0 || got == "" {
		t.Fatalf("unexpected remote result: %q %d %v", got, code, ok)
	}
	if body, err := os.ReadFile(got); err != nil || string(body) != "remote-tgz" {
		t.Fatalf("unexpected downloaded TGZ: %q %v", body, err)
	}

	var stderr bytes.Buffer
	deps.Stderr = &stderr
	if _, code, ok := resolveTGZPath(filepath.Join(t.TempDir(), "missing.tgz"), deps); !ok || code != 2 || !strings.Contains(stderr.String(), "tgz not found") {
		t.Fatalf("unexpected missing result: code=%d ok=%v stderr=%q", code, ok, stderr.String())
	}
}

func TestPropagateLegacyCoreRepositoryVariables(t *testing.T) {
	propagateLegacyCoreRepoDir(nil)
	t.Setenv("PRODUCTIVE_K3S_CORE_REPO_DIR", "/core")
	values := map[string]string{}
	propagateLegacyCoreRepoDir(values)
	if values["PRODUCTIVE_K3S_REPO"] != "/core" || values["PK3S_PROFILE_PACKAGE_PRODUCTIVE_K3S_REPO"] != "/core" {
		t.Fatalf("core directory was not propagated: %#v", values)
	}

	t.Setenv("PRODUCTIVE_K3S_CORE_REPO_DIR", "")
	t.Setenv("PRODUCTIVE_K3S_REPO", "/legacy")
	values = map[string]string{}
	propagateLegacyCoreRepoDir(values)
	if values["PRODUCTIVE_K3S_CORE_REPO_DIR"] != "/legacy" || values["PK3S_PROFILE_PACKAGE_PRODUCTIVE_K3S_REPO"] != "/legacy" {
		t.Fatalf("legacy directory was not propagated: %#v", values)
	}
}

func TestPublicPackageCommandUsageMatrix(t *testing.T) {
	cases := []struct {
		args []string
		text string
	}{
		{[]string{"profile"}, "Usage: pk3s profile"},
		{[]string{"profile", "show"}, "profile show"},
		{[]string{"profile", "unsupported"}, "Unsupported profile command"},
		{[]string{"infra"}, "Usage: pk3s infra"},
		{[]string{"infra", "unsupported"}, "Unsupported infra command"},
		{[]string{"addon"}, "Usage: pk3s addon"},
		{[]string{"addon", "show"}, "addon show"},
		{[]string{"addon", "unsupported"}, "Unsupported addon command"},
		{[]string{"stack"}, "Usage: pk3s stack"},
		{[]string{"stack", "show"}, "stack show"},
		{[]string{"stack", "unsupported"}, "Unsupported stack command"},
	}
	for _, tc := range cases {
		var stderr bytes.Buffer
		code := Run(context.Background(), tc.args, noTelemetryDeps(t, Dependencies{
			Stdout: &bytes.Buffer{}, Stderr: &stderr, GOOS: "linux", GOARCH: "amd64",
			WorkingDir: t.TempDir(), CacheDir: t.TempDir(),
		}))
		if code != 2 || !strings.Contains(stderr.String(), tc.text) {
			t.Fatalf("args=%v code=%d stderr=%q", tc.args, code, stderr.String())
		}
	}
}

func TestClusterRuntimeFailurePaths(t *testing.T) {
	dir := t.TempDir()
	registry := clusters.NewRegistry(filepath.Join(dir, "registry.json"))
	if err := registry.Upsert(clusters.Cluster{ID: "demo", Kubeconfig: filepath.Join(dir, "missing.yaml"), Context: "ctx"}); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	deps := Dependencies{Stdout: &bytes.Buffer{}, Stderr: &stderr, GOOS: "linux", GOARCH: "amd64", Exec: func(context.Context, Invocation) error { return nil }}
	if code := runClusterKubectl(context.Background(), registry, "demo", nil, deps); code != 2 || !strings.Contains(stderr.String(), "kubeconfig not found") {
		t.Fatalf("unexpected missing kubeconfig result: code=%d stderr=%q", code, stderr.String())
	}
	stderr.Reset()
	if code := runClusterK9s(context.Background(), registry, "demo", deps); code != 2 || !strings.Contains(stderr.String(), "kubeconfig not found") {
		t.Fatalf("unexpected k9s missing kubeconfig result: code=%d stderr=%q", code, stderr.String())
	}

	kubeconfig := filepath.Join(dir, "kubeconfig.yaml")
	if err := os.WriteFile(kubeconfig, []byte("apiVersion: v1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := registry.Upsert(clusters.Cluster{ID: "demo", Kubeconfig: kubeconfig, Context: "ctx"}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", t.TempDir())
	stderr.Reset()
	if code := runClusterKubectl(context.Background(), registry, "demo", nil, deps); code != 2 || !strings.Contains(stderr.String(), "kubectl is not installed") {
		t.Fatalf("unexpected missing kubectl result: code=%d stderr=%q", code, stderr.String())
	}
	stderr.Reset()
	if code := runClusterK9s(context.Background(), registry, "demo", deps); code != 2 || !strings.Contains(stderr.String(), "k9s is not installed") {
		t.Fatalf("unexpected missing k9s result: code=%d stderr=%q", code, stderr.String())
	}

	t.Setenv("PATH", fakeExecutableDir(t, "kubectl"))
	stderr.Reset()
	deps.RunOutput = func(context.Context, Invocation) ([]byte, error) { return nil, errors.New("kubectl boom") }
	if code := runClusterKubectl(context.Background(), registry, "demo", []string{"get", "nodes"}, deps); code != 1 || !strings.Contains(stderr.String(), "kubectl failed") {
		t.Fatalf("unexpected kubectl execution result: code=%d stderr=%q", code, stderr.String())
	}

	t.Setenv("PATH", fakeExecutableDir(t, "k9s"))
	stderr.Reset()
	deps.Exec = func(context.Context, Invocation) error { return errors.New("k9s boom") }
	if code := runClusterK9s(context.Background(), registry, "demo", deps); code != 1 || !strings.Contains(stderr.String(), "k9s failed") {
		t.Fatalf("unexpected k9s execution result: code=%d stderr=%q", code, stderr.String())
	}
}
