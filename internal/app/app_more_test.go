package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestReadOnlyCommandsDoNotPropagateTelemetryContext(t *testing.T) {
	workingDir := t.TempDir()
	t.Setenv("PRODUCTIVE_K3S_SOURCE", "local")
	profilePath := filepath.Join(workingDir, "basic.env")
	if err := os.WriteFile(profilePath, []byte("PK3S_INFRA_PROFILE_NAME=basic\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	infraDir := filepath.Join(workingDir, "productive-k3s-infra")
	if err := os.MkdirAll(infraDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(infraDir, "productive-k3s-infra.sh"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	var got Invocation
	code := Run(context.Background(), []string{"plan", "--profile", profilePath, "--telemetry", "enable"}, Dependencies{
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		GOOS:       "linux",
		GOARCH:     "amd64",
		WorkingDir: workingDir,
		CacheDir:   filepath.Join(workingDir, "cache"),
		Exec: func(_ context.Context, invocation Invocation) error {
			got = invocation
			return nil
		},
	})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}

	env := envMap(got.Env)
	if _, ok := env["TELEMETRY_SESSION_ID"]; ok {
		t.Fatalf("did not expect TELEMETRY_SESSION_ID for read-only command")
	}
	if _, ok := env["TELEMETRY_PARENT_RUN_ID"]; ok {
		t.Fatalf("did not expect TELEMETRY_PARENT_RUN_ID for read-only command")
	}
	if _, ok := env["TELEMETRY_MARKER"]; ok {
		t.Fatalf("did not expect TELEMETRY_MARKER for read-only command")
	}
}

func TestMutatingCommandsStillPropagateTelemetryContext(t *testing.T) {
	workingDir := t.TempDir()
	t.Setenv("PRODUCTIVE_K3S_SOURCE", "local")
	profilePath := filepath.Join(workingDir, "basic.env")
	if err := os.WriteFile(profilePath, []byte("PK3S_INFRA_PROFILE_NAME=basic\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	infraDir := filepath.Join(workingDir, "productive-k3s-infra")
	if err := os.MkdirAll(infraDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(infraDir, "productive-k3s-infra.sh"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	var got Invocation
	code := Run(context.Background(), []string{"apply", "--profile", profilePath, "--telemetry", "enable"}, Dependencies{
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		GOOS:       "linux",
		GOARCH:     "amd64",
		WorkingDir: workingDir,
		CacheDir:   filepath.Join(workingDir, "cache"),
		Exec: func(_ context.Context, invocation Invocation) error {
			got = invocation
			return nil
		},
	})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}

	env := envMap(got.Env)
	if env["TELEMETRY_ENABLED"] != "true" {
		t.Fatalf("expected TELEMETRY_ENABLED=true, got %q", env["TELEMETRY_ENABLED"])
	}
	if env["TELEMETRY_SESSION_ID"] == "" {
		t.Fatalf("expected TELEMETRY_SESSION_ID for mutating command")
	}
	if env["TELEMETRY_PARENT_RUN_ID"] == "" {
		t.Fatalf("expected TELEMETRY_PARENT_RUN_ID for mutating command")
	}
}

func expectedDownloadedTGZPath(cacheDir, rawURL string) string {
	sum := sha256.Sum256([]byte(rawURL))
	return filepath.Join(cacheDir, "tgz", "downloads", hex.EncodeToString(sum[:])+".tgz")
}

func TestRunBundleInfoDelegatesToCoreBundle(t *testing.T) {
	workingDir := t.TempDir()
	t.Setenv("PRODUCTIVE_K3S_SOURCE", "local")
	coreDir := filepath.Join(workingDir, "productive-k3s-core")
	if err := os.MkdirAll(coreDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(coreDir, "productive-k3s-core.sh"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	var got Invocation
	code := Run(context.Background(), []string{"bundle", "core", "info", "--json"}, noTelemetryDeps(t, Dependencies{
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		GOOS:       "linux",
		GOARCH:     "amd64",
		WorkingDir: workingDir,
		CacheDir:   filepath.Join(workingDir, "cache"),
		Exec: func(_ context.Context, invocation Invocation) error {
			got = invocation
			return nil
		},
	}))

	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if got.Path != filepath.Join(coreDir, "productive-k3s-core.sh") {
		t.Fatalf("unexpected path: %q", got.Path)
	}
	if strings.Join(got.Args, " ") != "bundle info --json" {
		t.Fatalf("unexpected args: %#v", got.Args)
	}
}

func TestRunDoctorAndBackupDelegationPaths(t *testing.T) {
	workingDir := t.TempDir()
	t.Setenv("PRODUCTIVE_K3S_SOURCE", "local")
	coreDir := filepath.Join(workingDir, "productive-k3s-core")
	infraDir := filepath.Join(workingDir, "productive-k3s-infra")
	for _, dir := range []string{coreDir, infraDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(coreDir, "productive-k3s-core.sh"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(infraDir, "productive-k3s-infra.sh"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	var invocations []Invocation
	deps := noTelemetryDeps(t, Dependencies{
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		GOOS:       "linux",
		GOARCH:     "amd64",
		WorkingDir: workingDir,
		CacheDir:   filepath.Join(workingDir, "cache"),
		Exec: func(_ context.Context, invocation Invocation) error {
			invocations = append(invocations, invocation)
			return nil
		},
	})

	if code := Run(context.Background(), []string{"doctor"}, deps); code != 0 {
		t.Fatalf("doctor expected exit 0, got %d", code)
	}
	if code := Run(context.Background(), []string{"doctor", "--infra", "--profile", "basic.env"}, deps); code != 0 {
		t.Fatalf("doctor infra expected exit 0, got %d", code)
	}
	if code := Run(context.Background(), []string{"backup", "--core"}, deps); code != 0 {
		t.Fatalf("backup expected exit 0, got %d", code)
	}

	if len(invocations) != 3 {
		t.Fatalf("expected 3 invocations, got %d", len(invocations))
	}
	if strings.Join(invocations[0].Args, " ") != "preflight" {
		t.Fatalf("unexpected doctor args: %#v", invocations[0].Args)
	}
	if strings.Join(invocations[1].Args, " ") != "doctor --infra --profile basic.env" {
		t.Fatalf("unexpected infra doctor args: %#v", invocations[1].Args)
	}
	if strings.Join(invocations[2].Args, " ") != "backup" {
		t.Fatalf("unexpected backup args: %#v", invocations[2].Args)
	}
}

func TestRunInstallRequiresCoreOnlyAndConfigStatusPaths(t *testing.T) {
	var stderr bytes.Buffer
	code := Run(context.Background(), []string{"install"}, Dependencies{
		Stdout: &bytes.Buffer{},
		Stderr: &stderr,
		GOOS:   "linux",
		GOARCH: "amd64",
	})
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("Only `pk3s install --core-only`")) {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}

	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	var stdout bytes.Buffer
	code = Run(context.Background(), []string{"config", "telemetry", "status"}, Dependencies{
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
		GOOS:   "linux",
		GOARCH: "amd64",
	})
	if code != 0 || !bytes.Contains(stdout.Bytes(), []byte("not configured")) {
		t.Fatalf("unexpected telemetry status output: code=%d out=%q", code, stdout.String())
	}

	stdout.Reset()
	if err := writeObservabilityToken("pk3s_live_clear_me"); err != nil {
		t.Fatal(err)
	}
	code = Run(context.Background(), []string{"config", "observability", "clear"}, Dependencies{
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
		GOOS:   "linux",
		GOARCH: "amd64",
	})
	if code != 0 || !bytes.Contains(stdout.Bytes(), []byte("cleared")) {
		t.Fatalf("unexpected observability clear output: code=%d out=%q", code, stdout.String())
	}
	token, err := readObservabilityToken()
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		t.Fatalf("expected token to be cleared, got %q", token)
	}
}

func TestRunInstallCoreOnlyDispatchAndBackupUnsupportedHost(t *testing.T) {
	workingDir := t.TempDir()
	t.Setenv("PRODUCTIVE_K3S_SOURCE", "local")
	coreDir := filepath.Join(workingDir, "productive-k3s-core")
	if err := os.MkdirAll(coreDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(coreDir, "productive-k3s-core.sh"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	var got Invocation
	code := Run(context.Background(), []string{"install", "--core-only", "--dry-run"}, noTelemetryDeps(t, Dependencies{
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		GOOS:       "linux",
		GOARCH:     "amd64",
		WorkingDir: workingDir,
		CacheDir:   filepath.Join(workingDir, "cache"),
		Exec: func(_ context.Context, invocation Invocation) error {
			got = invocation
			return nil
		},
	}))
	if code != 0 {
		t.Fatalf("expected install core-only dispatch to succeed, got %d", code)
	}
	if strings.Join(got.Args, " ") != "apply --dry-run" {
		t.Fatalf("unexpected install args: %#v", got.Args)
	}

	var stderr bytes.Buffer
	code = Run(context.Background(), []string{"backup", "--core"}, Dependencies{
		Stdout: &bytes.Buffer{},
		Stderr: &stderr,
		GOOS:   "windows",
		GOARCH: "amd64",
	})
	if code != 2 || !bytes.Contains(stderr.Bytes(), []byte("supported platforms")) {
		t.Fatalf("unexpected unsupported backup output: code=%d err=%q", code, stderr.String())
	}
}

func TestRunConfigCoversDisableAndUsageErrors(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	if err := writeTelemetryPreference(true); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run(context.Background(), []string{"config", "telemetry", "disable"}, Dependencies{
		Stdout: &stdout,
		Stderr: &stderr,
		GOOS:   "linux",
		GOARCH: "amd64",
	})
	if code != 0 || !bytes.Contains(stdout.Bytes(), []byte("disabled")) {
		t.Fatalf("unexpected disable output: code=%d out=%q err=%q", code, stdout.String(), stderr.String())
	}

	stdout.Reset()
	code = Run(context.Background(), []string{"config", "telemetry", "status"}, Dependencies{
		Stdout: &stdout,
		Stderr: &stderr,
		GOOS:   "linux",
		GOARCH: "amd64",
	})
	if code != 0 || !bytes.Contains(stdout.Bytes(), []byte("currently disabled")) {
		t.Fatalf("unexpected disabled status output: code=%d out=%q", code, stdout.String())
	}

	stderr.Reset()
	code = Run(context.Background(), []string{"config"}, Dependencies{
		Stdout: &bytes.Buffer{},
		Stderr: &stderr,
		GOOS:   "linux",
		GOARCH: "amd64",
	})
	if code != 2 || !bytes.Contains(stderr.Bytes(), []byte("Usage: pk3s config")) {
		t.Fatalf("unexpected config usage output: code=%d err=%q", code, stderr.String())
	}

	stderr.Reset()
	code = Run(context.Background(), []string{"config", "observability", "set"}, Dependencies{
		Stdout: &bytes.Buffer{},
		Stderr: &stderr,
		GOOS:   "linux",
		GOARCH: "amd64",
	})
	if code != 2 || !bytes.Contains(stderr.Bytes(), []byte("Usage: pk3s config observability set")) {
		t.Fatalf("unexpected observability set usage output: code=%d err=%q", code, stderr.String())
	}

	stderr.Reset()
	code = Run(context.Background(), []string{"config", "telemetry", "wat"}, Dependencies{
		Stdout: &bytes.Buffer{},
		Stderr: &stderr,
		GOOS:   "linux",
		GOARCH: "amd64",
	})
	if code != 2 || !bytes.Contains(stderr.Bytes(), []byte("Usage: pk3s config telemetry")) {
		t.Fatalf("unexpected telemetry usage output: code=%d err=%q", code, stderr.String())
	}
}

func TestRunBundleAndValidateUsagePaths(t *testing.T) {
	var stderr bytes.Buffer
	code := Run(context.Background(), []string{"bundle", "core"}, Dependencies{
		Stdout: &bytes.Buffer{},
		Stderr: &stderr,
		GOOS:   "linux",
		GOARCH: "amd64",
	})
	if code != 2 || !bytes.Contains(stderr.Bytes(), []byte("Usage: pk3s bundle <core|infra> info [--json]")) {
		t.Fatalf("unexpected bundle usage output: code=%d err=%q", code, stderr.String())
	}

	stderr.Reset()
	code = Run(context.Background(), []string{"validate", "--profile"}, Dependencies{
		Stdout: &bytes.Buffer{},
		Stderr: &stderr,
		GOOS:   "linux",
		GOARCH: "amd64",
	})
	if code != 2 || !bytes.Contains(stderr.Bytes(), []byte("missing value for --profile")) {
		t.Fatalf("unexpected validate profile error: code=%d err=%q", code, stderr.String())
	}
}

func TestRunValidateCoreAndInfraDispatch(t *testing.T) {
	workingDir := t.TempDir()
	t.Setenv("PRODUCTIVE_K3S_SOURCE", "local")
	coreDir := filepath.Join(workingDir, "productive-k3s-core")
	infraDir := filepath.Join(workingDir, "productive-k3s-infra")
	for _, dir := range []string{coreDir, infraDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(coreDir, "productive-k3s-core.sh"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(infraDir, "productive-k3s-infra.sh"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	profileDir := filepath.Join(workingDir, "tests", "fixtures", "profiles", "aws-single-node")
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		t.Fatal(err)
	}
	profilePath := filepath.Join(profileDir, "basic.env")
	if err := os.WriteFile(profilePath, []byte("PK3S_INFRA_PROFILE_NAME=basic\nPK3S_INFRA_ENGINE=opentofu\nPK3S_INFRA_SCENARIO=aws-single-node\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var invocations []Invocation
	deps := noTelemetryDeps(t, Dependencies{
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		GOOS:       "linux",
		GOARCH:     "amd64",
		WorkingDir: workingDir,
		CacheDir:   filepath.Join(workingDir, "cache"),
		Exec: func(_ context.Context, invocation Invocation) error {
			invocations = append(invocations, invocation)
			return nil
		},
	})

	if code := Run(context.Background(), []string{"validate", "--core"}, deps); code != 0 {
		t.Fatalf("validate --core expected exit 0, got %d", code)
	}
	if code := Run(context.Background(), []string{"validate", "basic"}, deps); code != 0 {
		t.Fatalf("validate basic expected exit 0, got %d", code)
	}

	if len(invocations) != 2 {
		t.Fatalf("expected 2 invocations, got %d", len(invocations))
	}
	if strings.Join(invocations[0].Args, " ") != "validate" {
		t.Fatalf("unexpected core validate args: %#v", invocations[0].Args)
	}
	if strings.Join(invocations[1].Args, " ") != "validate --profile "+profilePath {
		t.Fatalf("unexpected infra validate args: %#v", invocations[1].Args)
	}
}

func TestRunProfileListAndUnsupportedProfileCommand(t *testing.T) {
	workingDir := t.TempDir()
	var stdout bytes.Buffer
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.ReplaceAll(`apiVersion: catalogs.productive-k3s.io/v1alpha1
kind: ProductiveK3SCatalog
metadata:
  name: productive-k3s-catalog
entries:
  - id: multipass-1-server-2-agents
    name: multipass-1-server-2-agents
    kind: profile
    visibility: public
    category: local
    version: 0.9.62-0.9.4
    artifact:
      type: tgz
      url: SERVER_URL/infra/multipass-1-server-2-agents-0.9.62-0.9.4.tgz
  - id: aws-single-node-basic
    name: aws-single-node-basic
    kind: profile
    visibility: public
    category: cloud
    version: 0.9.62-0.9.4
    install:
      requiresLocalOverrides: true
    artifact:
      type: tgz
      url: SERVER_URL/infra/aws-single-node-basic-0.9.62-0.9.4.tgz
`, "SERVER_URL", server.URL)))
	}))
	defer server.Close()
	t.Setenv("PK3S_CATALOG_URLS", server.URL)

	code := Run(context.Background(), []string{"profile", "list"}, noTelemetryDeps(t, Dependencies{
		Stdout:     &stdout,
		Stderr:     &bytes.Buffer{},
		GOOS:       "linux",
		GOARCH:     "amd64",
		WorkingDir: workingDir,
		CacheDir:   filepath.Join(workingDir, "cache"),
		HTTPClient: server.Client(),
	}))
	if code != 0 {
		t.Fatalf("expected profile list to succeed, got %d", code)
	}
	if got := stdout.String(); !strings.Contains(got, "multipass-1-server-2-agents\t0.9.62-0.9.4\tlocal") {
		t.Fatalf("expected catalog-backed local profile listing, got %q", got)
	}
	if got := stdout.String(); !strings.Contains(got, "aws-single-node-basic\t0.9.62-0.9.4\tcloud\tneeds-env") {
		t.Fatalf("expected catalog-backed needs-env marker, got %q", got)
	}

	var stderr bytes.Buffer
	code = Run(context.Background(), []string{"profile", "wat"}, Dependencies{
		Stdout: &bytes.Buffer{},
		Stderr: &stderr,
		GOOS:   "linux",
		GOARCH: "amd64",
	})
	if code != 2 || !bytes.Contains(stderr.Bytes(), []byte("Unsupported profile command")) {
		t.Fatalf("unexpected unsupported profile output: code=%d err=%q", code, stderr.String())
	}
}

func TestRunProfileShowReadsCatalogInstallMetadata(t *testing.T) {
	workingDir := t.TempDir()
	var stdout bytes.Buffer

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.yaml":
			_, _ = w.Write([]byte(strings.ReplaceAll(`apiVersion: catalogs.productive-k3s.io/v1alpha1
kind: ProductiveK3SCatalog
metadata:
  name: productive-k3s-catalog
entries:
  - id: aws-single-node-basic
    name: aws-single-node-basic
    kind: profile
    visibility: public
    category: cloud
    description: "AWS single node profile."
    version: 0.1.0
    sourceRepository: productive-k3s-profiles
    artifact:
      type: tgz
      url: SERVER_URL/infra/aws-single-node-basic-0.1.0.tgz
    install:
      requiresLocalOverrides: true
      inputs:
        - name: AWS_REGION
          required: true
          sensitive: false
          source: package-default
          description: "Default AWS region"
        - name: AWS_KEY_PAIR_NAME
          required: true
          sensitive: false
          source: local-override
          description: "Existing key pair"
`, "SERVER_URL", server.URL)))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	t.Setenv("PK3S_CATALOG_URLS", server.URL+"/index.yaml")

	code := Run(context.Background(), []string{"profile", "show", "aws-single-node-basic"}, noTelemetryDeps(t, Dependencies{
		Stdout:     &stdout,
		Stderr:     &bytes.Buffer{},
		GOOS:       "linux",
		GOARCH:     "amd64",
		WorkingDir: workingDir,
		CacheDir:   filepath.Join(workingDir, "cache"),
		HTTPClient: server.Client(),
	}))
	if code != 0 {
		t.Fatalf("expected profile show to succeed, got %d", code)
	}
	got := stdout.String()
	if !strings.Contains(got, "Name: aws-single-node-basic") {
		t.Fatalf("missing profile name in output: %q", got)
	}
	if !strings.Contains(got, "Requires local overrides: true") {
		t.Fatalf("missing local overrides summary in output: %q", got)
	}
	if !strings.Contains(got, "AWS_KEY_PAIR_NAME [local-override] required") {
		t.Fatalf("missing input metadata in output: %q", got)
	}
}

func TestRunProfileValidateDelegatesTGZToInfra(t *testing.T) {
	workingDir := t.TempDir()
	t.Setenv("PRODUCTIVE_K3S_SOURCE", "local")
	infraDir := filepath.Join(workingDir, "productive-k3s-infra")
	if err := os.MkdirAll(infraDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(infraDir, "productive-k3s-infra.sh"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	tgzPath := filepath.Join(workingDir, "demo-profile.tgz")
	if err := os.WriteFile(tgzPath, []byte("tgz"), 0o644); err != nil {
		t.Fatal(err)
	}

	var got Invocation
	code := Run(context.Background(), []string{"profile", "validate", "--tgz", tgzPath}, noTelemetryDeps(t, Dependencies{
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		GOOS:       "linux",
		GOARCH:     "amd64",
		WorkingDir: workingDir,
		CacheDir:   filepath.Join(workingDir, "cache"),
		Exec: func(_ context.Context, invocation Invocation) error {
			got = invocation
			return nil
		},
	}))
	if code != 0 {
		t.Fatalf("expected profile validate tgz to succeed, got %d", code)
	}
	if strings.Join(got.Args, " ") != "profile validate --tgz "+tgzPath {
		t.Fatalf("unexpected profile tgz args: %#v", got.Args)
	}
}

func TestRunProfileInstallDelegatesTGZToInfra(t *testing.T) {
	workingDir := t.TempDir()
	t.Setenv("PRODUCTIVE_K3S_SOURCE", "local")
	infraDir := filepath.Join(workingDir, "productive-k3s-infra")
	if err := os.MkdirAll(infraDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(infraDir, "productive-k3s-infra.sh"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	tgzPath := filepath.Join(workingDir, "demo-profile.tgz")
	if err := os.WriteFile(tgzPath, []byte("tgz"), 0o644); err != nil {
		t.Fatal(err)
	}

	var got Invocation
	code := Run(context.Background(), []string{"profile", "install", "--tgz", tgzPath}, noTelemetryDeps(t, Dependencies{
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		GOOS:       "linux",
		GOARCH:     "amd64",
		WorkingDir: workingDir,
		CacheDir:   filepath.Join(workingDir, "cache"),
		Exec: func(_ context.Context, invocation Invocation) error {
			got = invocation
			return nil
		},
	}))
	if code != 0 {
		t.Fatalf("expected profile install tgz to succeed, got %d", code)
	}
	if strings.Join(got.Args, " ") != "profile install --tgz "+tgzPath {
		t.Fatalf("unexpected profile install tgz args: %#v", got.Args)
	}
}

func TestRunProfileInstallFromCatalogRequiresEnvFileWhenEntryNeedsLocalOverrides(t *testing.T) {
	workingDir := t.TempDir()
	var stderr bytes.Buffer

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.yaml":
			_, _ = w.Write([]byte(strings.ReplaceAll(`apiVersion: catalogs.productive-k3s.io/v1alpha1
kind: ProductiveK3SCatalog
metadata:
  name: productive-k3s-catalog
entries:
  - id: aws-single-node-basic
    name: aws-single-node-basic
    kind: profile
    visibility: public
    category: cloud
    description: "AWS single node profile."
    version: 0.1.0
    sourceRepository: productive-k3s-profiles
    artifact:
      type: tgz
      url: SERVER_URL/infra/aws-single-node-basic-0.1.0.tgz
    install:
      requiresLocalOverrides: true
      inputs:
        - name: AWS_KEY_PAIR_NAME
          required: true
          sensitive: false
          source: local-override
          description: "Existing key pair"
`, "SERVER_URL", server.URL)))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	t.Setenv("PK3S_CATALOG_URLS", server.URL+"/index.yaml")

	code := Run(context.Background(), []string{"profile", "install", "aws-single-node-basic"}, noTelemetryDeps(t, Dependencies{
		Stdout:     &bytes.Buffer{},
		Stderr:     &stderr,
		GOOS:       "linux",
		GOARCH:     "amd64",
		WorkingDir: workingDir,
		CacheDir:   filepath.Join(workingDir, "cache"),
		HTTPClient: server.Client(),
	}))
	if code != 2 {
		t.Fatalf("expected install preflight to fail, got %d", code)
	}
	if !strings.Contains(stderr.String(), "requires installation-specific local overrides; use --env-file <file>") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "AWS_KEY_PAIR_NAME") {
		t.Fatalf("missing input name in stderr: %q", stderr.String())
	}
}

func TestRunProfileInstallRequiresTGZ(t *testing.T) {
	var stderr bytes.Buffer
	code := Run(context.Background(), []string{"profile", "install"}, Dependencies{
		Stdout: &bytes.Buffer{},
		Stderr: &stderr,
		GOOS:   "linux",
		GOARCH: "amd64",
	})
	if code != 2 || !bytes.Contains(stderr.Bytes(), []byte("missing tgz; use --tgz <file|url>")) {
		t.Fatalf("unexpected missing tgz output: code=%d err=%q", code, stderr.String())
	}
}

func TestRunInfraInstallDelegatesTGZToInfra(t *testing.T) {
	workingDir := t.TempDir()
	t.Setenv("PRODUCTIVE_K3S_SOURCE", "local")
	infraDir := filepath.Join(workingDir, "productive-k3s-infra")
	if err := os.MkdirAll(infraDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(infraDir, "productive-k3s-infra.sh"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	tgzPath := filepath.Join(workingDir, "demo-profile.tgz")
	if err := os.WriteFile(tgzPath, []byte("tgz"), 0o644); err != nil {
		t.Fatal(err)
	}

	var got Invocation
	code := Run(context.Background(), []string{"infra", "install", "--tgz", tgzPath}, noTelemetryDeps(t, Dependencies{
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		GOOS:       "linux",
		GOARCH:     "amd64",
		WorkingDir: workingDir,
		CacheDir:   filepath.Join(workingDir, "cache"),
		Exec: func(_ context.Context, invocation Invocation) error {
			got = invocation
			return nil
		},
	}))
	if code != 0 {
		t.Fatalf("expected infra install tgz to succeed, got %d", code)
	}
	if strings.Join(got.Args, " ") != "profile install --tgz "+tgzPath {
		t.Fatalf("unexpected infra install tgz args: %#v", got.Args)
	}
}

func TestRunInfraStatusDelegatesTGZToInfra(t *testing.T) {
	workingDir := t.TempDir()
	t.Setenv("PRODUCTIVE_K3S_SOURCE", "local")
	infraDir := filepath.Join(workingDir, "productive-k3s-infra")
	if err := os.MkdirAll(infraDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(infraDir, "productive-k3s-infra.sh"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	tgzPath := filepath.Join(workingDir, "demo-profile.tgz")
	if err := os.WriteFile(tgzPath, []byte("tgz"), 0o644); err != nil {
		t.Fatal(err)
	}

	var got Invocation
	code := Run(context.Background(), []string{"infra", "status", "--tgz", tgzPath}, noTelemetryDeps(t, Dependencies{
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		GOOS:       "linux",
		GOARCH:     "amd64",
		WorkingDir: workingDir,
		CacheDir:   filepath.Join(workingDir, "cache"),
		Exec: func(_ context.Context, invocation Invocation) error {
			got = invocation
			return nil
		},
	}))
	if code != 0 {
		t.Fatalf("expected infra status tgz to succeed, got %d", code)
	}
	if strings.Join(got.Args, " ") != "profile status --tgz "+tgzPath {
		t.Fatalf("unexpected infra status tgz args: %#v", got.Args)
	}
}

func TestRunInfraPlanDelegatesTGZToInfra(t *testing.T) {
	workingDir := t.TempDir()
	t.Setenv("PRODUCTIVE_K3S_SOURCE", "local")
	infraDir := filepath.Join(workingDir, "productive-k3s-infra")
	if err := os.MkdirAll(infraDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(infraDir, "productive-k3s-infra.sh"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	tgzPath := filepath.Join(workingDir, "demo-profile.tgz")
	if err := os.WriteFile(tgzPath, []byte("tgz"), 0o644); err != nil {
		t.Fatal(err)
	}

	var got Invocation
	code := Run(context.Background(), []string{"infra", "plan", "--tgz", tgzPath}, noTelemetryDeps(t, Dependencies{
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		GOOS:       "linux",
		GOARCH:     "amd64",
		WorkingDir: workingDir,
		CacheDir:   filepath.Join(workingDir, "cache"),
		Exec: func(_ context.Context, invocation Invocation) error {
			got = invocation
			return nil
		},
	}))
	if code != 0 {
		t.Fatalf("expected infra plan tgz to succeed, got %d", code)
	}
	if strings.Join(got.Args, " ") != "profile plan --tgz "+tgzPath {
		t.Fatalf("unexpected infra plan tgz args: %#v", got.Args)
	}
}

func TestRunInfraApplyAndDestroyDelegateTGZToInfra(t *testing.T) {
	workingDir := t.TempDir()
	t.Setenv("PRODUCTIVE_K3S_SOURCE", "local")
	infraDir := filepath.Join(workingDir, "productive-k3s-infra")
	if err := os.MkdirAll(infraDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(infraDir, "productive-k3s-infra.sh"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	tgzPath := filepath.Join(workingDir, "demo-profile.tgz")
	if err := os.WriteFile(tgzPath, []byte("tgz"), 0o644); err != nil {
		t.Fatal(err)
	}

	var invocations []Invocation
	for _, args := range [][]string{
		{"infra", "apply", "--tgz", tgzPath},
		{"infra", "destroy", "--tgz", tgzPath},
	} {
		code := Run(context.Background(), args, noTelemetryDeps(t, Dependencies{
			Stdout:     &bytes.Buffer{},
			Stderr:     &bytes.Buffer{},
			GOOS:       "linux",
			GOARCH:     "amd64",
			WorkingDir: workingDir,
			CacheDir:   filepath.Join(workingDir, "cache"),
			Exec: func(_ context.Context, invocation Invocation) error {
				invocations = append(invocations, invocation)
				return nil
			},
		}))
		if code != 0 {
			t.Fatalf("expected %v to succeed, got %d", args, code)
		}
	}

	if len(invocations) != 2 {
		t.Fatalf("expected 2 infra invocations, got %d", len(invocations))
	}
	if strings.Join(invocations[0].Args, " ") != "profile apply --tgz "+tgzPath {
		t.Fatalf("unexpected infra apply tgz args: %#v", invocations[0].Args)
	}
	if strings.Join(invocations[1].Args, " ") != "profile destroy --tgz "+tgzPath {
		t.Fatalf("unexpected infra destroy tgz args: %#v", invocations[1].Args)
	}
}

func TestRunInfraRequiresTGZAndRejectsUnsupportedCommand(t *testing.T) {
	var stderr bytes.Buffer
	code := Run(context.Background(), []string{"infra", "status"}, Dependencies{
		Stdout: &bytes.Buffer{},
		Stderr: &stderr,
		GOOS:   "linux",
		GOARCH: "amd64",
	})
	if code != 2 || !bytes.Contains(stderr.Bytes(), []byte("missing tgz; use --tgz <file|url>")) {
		t.Fatalf("unexpected missing tgz output: code=%d err=%q", code, stderr.String())
	}

	stderr.Reset()
	code = Run(context.Background(), []string{"infra", "wat", "--tgz", "/tmp/demo.tgz"}, Dependencies{
		Stdout: &bytes.Buffer{},
		Stderr: &stderr,
		GOOS:   "linux",
		GOARCH: "amd64",
	})
	if code != 2 || !bytes.Contains(stderr.Bytes(), []byte("Unsupported infra command")) {
		t.Fatalf("unexpected unsupported infra output: code=%d err=%q", code, stderr.String())
	}
}

func TestRunAddonValidateDelegatesTGZToCore(t *testing.T) {
	workingDir := t.TempDir()
	t.Setenv("PRODUCTIVE_K3S_SOURCE", "local")
	coreDir := filepath.Join(workingDir, "productive-k3s-core")
	if err := os.MkdirAll(coreDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(coreDir, "productive-k3s-core.sh"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	tgzPath := filepath.Join(workingDir, "demo-addon.tgz")
	if err := os.WriteFile(tgzPath, []byte("tgz"), 0o644); err != nil {
		t.Fatal(err)
	}

	var got Invocation
	code := Run(context.Background(), []string{"addon", "validate", "--tgz", tgzPath}, noTelemetryDeps(t, Dependencies{
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		GOOS:       "linux",
		GOARCH:     "amd64",
		WorkingDir: workingDir,
		CacheDir:   filepath.Join(workingDir, "cache"),
		Exec: func(_ context.Context, invocation Invocation) error {
			got = invocation
			return nil
		},
	}))
	if code != 0 {
		t.Fatalf("expected addon validate tgz to succeed, got %d", code)
	}
	if strings.Join(got.Args, " ") != "addon validate --tgz "+tgzPath {
		t.Fatalf("unexpected addon tgz args: %#v", got.Args)
	}
}

func TestRunAddonInstallResolvesNameFromCatalog(t *testing.T) {
	workingDir := t.TempDir()
	t.Setenv("PRODUCTIVE_K3S_SOURCE", "local")
	coreDir := filepath.Join(workingDir, "productive-k3s-core")
	if err := os.MkdirAll(coreDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(coreDir, "productive-k3s-core.sh"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()
	catalogBody := strings.ReplaceAll(`apiVersion: catalogs.productive-k3s.io/v1alpha1
kind: ProductiveK3SCatalog
metadata:
  name: productive-k3s-catalog
entries:
  - kind: addon
    metadata:
      name: nginx
    artifact:
      type: tgz
      url: SERVER_URL/addons/nginx-0.1.0.tgz
`, "SERVER_URL", server.URL)
	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.yaml":
			_, _ = w.Write([]byte(catalogBody))
		case "/addons/nginx-0.1.0.tgz":
			_, _ = w.Write([]byte("tgz"))
		default:
			http.NotFound(w, r)
		}
	})
	t.Setenv("PK3S_CATALOG_URLS", server.URL+"/index.yaml")
	expectedTGZ := expectedDownloadedTGZPath(filepath.Join(workingDir, "cache"), server.URL+"/addons/nginx-0.1.0.tgz")

	var got Invocation
	code := Run(context.Background(), []string{"addon", "install", "nginx", "--cluster-context", "default"}, noTelemetryDeps(t, Dependencies{
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		GOOS:       "linux",
		GOARCH:     "amd64",
		WorkingDir: workingDir,
		CacheDir:   filepath.Join(workingDir, "cache"),
		HTTPClient: server.Client(),
		Exec: func(_ context.Context, invocation Invocation) error {
			got = invocation
			return nil
		},
	}))
	if code != 0 {
		t.Fatalf("expected addon install by catalog name to succeed, got %d", code)
	}
	if strings.Join(got.Args, " ") != "addon install --tgz "+expectedTGZ {
		t.Fatalf("unexpected addon catalog args: %#v", got.Args)
	}
	if !slices.Contains(got.Env, "PK3S_KUBE_CONTEXT=default") {
		t.Fatalf("expected cluster context to be propagated, env=%#v", got.Env)
	}
}

func TestRunAddonInstallForwardsPublicHost(t *testing.T) {
	workingDir := t.TempDir()
	t.Setenv("PRODUCTIVE_K3S_SOURCE", "local")
	coreDir := filepath.Join(workingDir, "productive-k3s-core")
	if err := os.MkdirAll(coreDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(coreDir, "productive-k3s-core.sh"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	tgzPath := filepath.Join(workingDir, "demo-addon.tgz")
	if err := os.WriteFile(tgzPath, []byte("tgz"), 0o644); err != nil {
		t.Fatal(err)
	}

	var got Invocation
	code := Run(context.Background(), []string{"addon", "install", "--tgz", tgzPath, "--cluster-context", "default", "--public-host", "demo.k3s.lab.internal"}, noTelemetryDeps(t, Dependencies{
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		GOOS:       "linux",
		GOARCH:     "amd64",
		WorkingDir: workingDir,
		CacheDir:   filepath.Join(workingDir, "cache"),
		Exec: func(_ context.Context, invocation Invocation) error {
			got = invocation
			return nil
		},
	}))
	if code != 0 {
		t.Fatalf("expected addon install with public host to succeed, got %d", code)
	}
	if strings.Join(got.Args, " ") != "addon install --tgz "+tgzPath+" --public-host demo.k3s.lab.internal" {
		t.Fatalf("unexpected addon public-host args: %#v", got.Args)
	}
	if !slices.Contains(got.Env, "PK3S_KUBE_CONTEXT=default") {
		t.Fatalf("expected cluster context to be propagated, env=%#v", got.Env)
	}
}

func TestRunAddonInstallResolvesProfileStateToKubeconfig(t *testing.T) {
	workingDir := t.TempDir()
	t.Setenv("PRODUCTIVE_K3S_SOURCE", "local")
	coreDir := filepath.Join(workingDir, "productive-k3s-core")
	if err := os.MkdirAll(coreDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(coreDir, "productive-k3s-core.sh"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	stateDir := filepath.Join(workingDir, "state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PK3S_PROFILE_STATE_DIR", stateDir)
	stateBody := `{
  "server_url": "https://10.0.0.10:6443",
  "ssh": {
    "user": "ubuntu",
    "port": 22,
    "key_path": "/tmp/test-key"
  },
  "server": {
    "ipv4": "10.0.0.10"
  }
}`
	if err := os.WriteFile(filepath.Join(stateDir, "multipass-1-server-2-agents.json"), []byte(stateBody), 0o644); err != nil {
		t.Fatal(err)
	}

	tgzPath := filepath.Join(workingDir, "demo-addon.tgz")
	if err := os.WriteFile(tgzPath, []byte("tgz"), 0o644); err != nil {
		t.Fatal(err)
	}

	var got Invocation
	code := Run(context.Background(), []string{"addon", "install", "--tgz", tgzPath, "--profile", "multipass-1-server-2-agents"}, noTelemetryDeps(t, Dependencies{
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		GOOS:       "linux",
		GOARCH:     "amd64",
		WorkingDir: workingDir,
		CacheDir:   filepath.Join(workingDir, "cache"),
		RunOutput: func(_ context.Context, invocation Invocation) ([]byte, error) {
			if invocation.Path != "ssh" {
				t.Fatalf("unexpected command: %s %#v", invocation.Path, invocation.Args)
			}
			return []byte("apiVersion: v1\nclusters:\n- cluster:\n    server: https://127.0.0.1:6443\n"), nil
		},
		Exec: func(_ context.Context, invocation Invocation) error {
			got = invocation
			kubeconfig := ""
			for _, kv := range invocation.Env {
				if strings.HasPrefix(kv, "KUBECONFIG=") {
					kubeconfig = strings.TrimPrefix(kv, "KUBECONFIG=")
				}
			}
			if kubeconfig == "" {
				t.Fatalf("expected KUBECONFIG env, got %#v", invocation.Env)
			}
			body, err := os.ReadFile(kubeconfig)
			if err != nil {
				t.Fatalf("expected kubeconfig file to exist: %v", err)
			}
			if !strings.Contains(string(body), "server: https://10.0.0.10:6443") {
				t.Fatalf("expected rewritten kubeconfig server, got %q", string(body))
			}
			return nil
		},
	}))
	if code != 0 {
		t.Fatalf("expected addon install by profile to succeed, got %d", code)
	}
	if strings.Join(got.Args, " ") != "addon install --tgz "+tgzPath {
		t.Fatalf("unexpected addon profile args: %#v", got.Args)
	}
}

func TestRunProfileInstallResolvesNameFromCatalogWithMirrorFallback(t *testing.T) {
	workingDir := t.TempDir()
	t.Setenv("PRODUCTIVE_K3S_SOURCE", "local")
	infraDir := filepath.Join(workingDir, "productive-k3s-infra")
	if err := os.MkdirAll(infraDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(infraDir, "productive-k3s-infra.sh"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer fallback.Close()
	fallbackCatalog := strings.ReplaceAll(`apiVersion: catalogs.productive-k3s.io/v1alpha1
kind: ProductiveK3SCatalog
metadata:
  name: productive-k3s-catalog
entries:
  - kind: profile
    metadata:
      name: multipass-1-server-2-agents
    artifact:
      type: tgz
      url: SERVER_URL/infra/multipass-1-server-2-agents-0.1.0.tgz
`, "SERVER_URL", fallback.URL)
	fallback.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.yaml":
			_, _ = w.Write([]byte(fallbackCatalog))
		case "/infra/multipass-1-server-2-agents-0.1.0.tgz":
			_, _ = w.Write([]byte("tgz"))
		default:
			http.NotFound(w, r)
		}
	})
	t.Setenv("PK3S_CATALOG_URLS", "http://127.0.0.1:1/index.yaml,"+fallback.URL+"/index.yaml")
	expectedTGZ := expectedDownloadedTGZPath(filepath.Join(workingDir, "cache"), fallback.URL+"/infra/multipass-1-server-2-agents-0.1.0.tgz")

	var got Invocation
	code := Run(context.Background(), []string{"profile", "install", "multipass-1-server-2-agents"}, noTelemetryDeps(t, Dependencies{
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		GOOS:       "linux",
		GOARCH:     "amd64",
		WorkingDir: workingDir,
		CacheDir:   filepath.Join(workingDir, "cache"),
		HTTPClient: fallback.Client(),
		Exec: func(_ context.Context, invocation Invocation) error {
			got = invocation
			return nil
		},
	}))
	if code != 0 {
		t.Fatalf("expected profile install by catalog name to succeed, got %d", code)
	}
	if strings.Join(got.Args, " ") != "profile install --tgz "+expectedTGZ {
		t.Fatalf("unexpected profile catalog args: %#v", got.Args)
	}
}

func TestRunProfileInstallPassesEnvFileOverrideToInfra(t *testing.T) {
	workingDir := t.TempDir()
	t.Setenv("PRODUCTIVE_K3S_SOURCE", "local")
	infraDir := filepath.Join(workingDir, "productive-k3s-infra")
	if err := os.MkdirAll(infraDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(infraDir, "productive-k3s-infra.sh"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	tgzPath := filepath.Join(workingDir, "demo-profile.tgz")
	if err := os.WriteFile(tgzPath, []byte("tgz"), 0o644); err != nil {
		t.Fatal(err)
	}
	envPath := filepath.Join(workingDir, "aws.env")
	if err := os.WriteFile(envPath, []byte("AWS_REGION=us-east-1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var got Invocation
	code := Run(context.Background(), []string{"profile", "install", "--tgz", tgzPath, "--env-file", envPath}, noTelemetryDeps(t, Dependencies{
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		GOOS:       "linux",
		GOARCH:     "amd64",
		WorkingDir: workingDir,
		CacheDir:   filepath.Join(workingDir, "cache"),
		Exec: func(_ context.Context, invocation Invocation) error {
			got = invocation
			return nil
		},
	}))
	if code != 0 {
		t.Fatalf("expected profile install with env-file to succeed, got %d", code)
	}
	if !slices.Contains(got.Env, "PK3S_PROFILE_OVERRIDE_ENV_FILE="+envPath) {
		t.Fatalf("expected env override path in env, got %#v", got.Env)
	}
}

func TestRunInfraStatusResolvesNameFromCatalog(t *testing.T) {
	workingDir := t.TempDir()
	t.Setenv("PRODUCTIVE_K3S_SOURCE", "local")
	infraDir := filepath.Join(workingDir, "productive-k3s-infra")
	if err := os.MkdirAll(infraDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(infraDir, "productive-k3s-infra.sh"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()
	catalogBody := strings.ReplaceAll(`apiVersion: catalogs.productive-k3s.io/v1alpha1
kind: ProductiveK3SCatalog
metadata:
  name: productive-k3s-catalog
entries:
  - kind: profile
    metadata:
      name: aws-single-node-basic
    artifact:
      type: tgz
      url: SERVER_URL/infra/aws-single-node-basic-0.1.0.tgz
`, "SERVER_URL", server.URL)
	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.yaml":
			_, _ = w.Write([]byte(catalogBody))
		case "/infra/aws-single-node-basic-0.1.0.tgz":
			_, _ = w.Write([]byte("tgz"))
		default:
			http.NotFound(w, r)
		}
	})
	t.Setenv("PK3S_CATALOG_URLS", server.URL+"/index.yaml")
	expectedTGZ := expectedDownloadedTGZPath(filepath.Join(workingDir, "cache"), server.URL+"/infra/aws-single-node-basic-0.1.0.tgz")

	var got Invocation
	code := Run(context.Background(), []string{"infra", "status", "aws-single-node-basic"}, noTelemetryDeps(t, Dependencies{
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		GOOS:       "linux",
		GOARCH:     "amd64",
		WorkingDir: workingDir,
		CacheDir:   filepath.Join(workingDir, "cache"),
		HTTPClient: server.Client(),
		Exec: func(_ context.Context, invocation Invocation) error {
			got = invocation
			return nil
		},
	}))
	if code != 0 {
		t.Fatalf("expected infra status by catalog name to succeed, got %d", code)
	}
	if strings.Join(got.Args, " ") != "profile status --tgz "+expectedTGZ {
		t.Fatalf("unexpected infra catalog args: %#v", got.Args)
	}
}

func TestRunInfraApplyPassesEnvFileOverrideToInfra(t *testing.T) {
	workingDir := t.TempDir()
	t.Setenv("PRODUCTIVE_K3S_SOURCE", "local")
	infraDir := filepath.Join(workingDir, "productive-k3s-infra")
	if err := os.MkdirAll(infraDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(infraDir, "productive-k3s-infra.sh"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	tgzPath := filepath.Join(workingDir, "demo-profile.tgz")
	if err := os.WriteFile(tgzPath, []byte("tgz"), 0o644); err != nil {
		t.Fatal(err)
	}
	envPath := filepath.Join(workingDir, "aws.env")
	if err := os.WriteFile(envPath, []byte("AWS_REGION=us-east-1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var got Invocation
	code := Run(context.Background(), []string{"infra", "apply", "--tgz", tgzPath, "--env-file", envPath}, noTelemetryDeps(t, Dependencies{
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		GOOS:       "linux",
		GOARCH:     "amd64",
		WorkingDir: workingDir,
		CacheDir:   filepath.Join(workingDir, "cache"),
		Exec: func(_ context.Context, invocation Invocation) error {
			got = invocation
			return nil
		},
	}))
	if code != 0 {
		t.Fatalf("expected infra apply with env-file to succeed, got %d", code)
	}
	if !slices.Contains(got.Env, "PK3S_PROFILE_OVERRIDE_ENV_FILE="+envPath) {
		t.Fatalf("expected env override path in env, got %#v", got.Env)
	}
}

func TestRunInfraApplyFromCatalogRequiresEnvFileWhenEntryNeedsLocalOverrides(t *testing.T) {
	workingDir := t.TempDir()
	var stderr bytes.Buffer

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.yaml":
			_, _ = w.Write([]byte(strings.ReplaceAll(`apiVersion: catalogs.productive-k3s.io/v1alpha1
kind: ProductiveK3SCatalog
metadata:
  name: productive-k3s-catalog
entries:
  - id: on-prem-basic
    name: on-prem-basic
    kind: profile
    visibility: public
    category: edge
    description: "On-prem profile."
    version: 0.1.0
    sourceRepository: productive-k3s-profiles
    artifact:
      type: tgz
      url: SERVER_URL/infra/on-prem-basic-0.1.0.tgz
    install:
      requiresLocalOverrides: true
      inputs:
        - name: ONPREM_SERVER_IP
          required: true
          sensitive: false
          source: local-override
          description: "Server IP"
`, "SERVER_URL", server.URL)))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	t.Setenv("PK3S_CATALOG_URLS", server.URL+"/index.yaml")

	code := Run(context.Background(), []string{"infra", "apply", "on-prem-basic"}, noTelemetryDeps(t, Dependencies{
		Stdout:     &bytes.Buffer{},
		Stderr:     &stderr,
		GOOS:       "linux",
		GOARCH:     "amd64",
		WorkingDir: workingDir,
		CacheDir:   filepath.Join(workingDir, "cache"),
		HTTPClient: server.Client(),
	}))
	if code != 2 {
		t.Fatalf("expected infra apply preflight to fail, got %d", code)
	}
	if !strings.Contains(stderr.String(), "use --env-file <file>") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "ONPREM_SERVER_IP") {
		t.Fatalf("missing local input in stderr: %q", stderr.String())
	}
}

func TestRunAddonInstallWithProfileSuggestsInfraCommandsWhenStateIsMissing(t *testing.T) {
	workingDir := t.TempDir()
	t.Setenv("PRODUCTIVE_K3S_SOURCE", "local")
	coreDir := filepath.Join(workingDir, "productive-k3s-core")
	if err := os.MkdirAll(coreDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(coreDir, "productive-k3s-core.sh"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	tgzPath := filepath.Join(workingDir, "demo-addon.tgz")
	if err := os.WriteFile(tgzPath, []byte("tgz"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	code := Run(context.Background(), []string{"addon", "install", "--tgz", tgzPath, "--profile", "aws-single-node-basic"}, noTelemetryDeps(t, Dependencies{
		Stdout:     &bytes.Buffer{},
		Stderr:     &stderr,
		GOOS:       "linux",
		GOARCH:     "amd64",
		WorkingDir: workingDir,
		CacheDir:   filepath.Join(workingDir, "cache"),
	}))
	if code != 2 {
		t.Fatalf("expected addon install with missing profile state to fail, got %d", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("pk3s infra install aws-single-node-basic [--env-file <file>]")) {
		t.Fatalf("expected install suggestion in stderr, got %q", stderr.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte("pk3s infra status aws-single-node-basic [--env-file <file>]")) {
		t.Fatalf("expected status suggestion in stderr, got %q", stderr.String())
	}
}

func TestRunAddonListUsesDefaultCatalogURLWhenEnvUnset(t *testing.T) {
	workingDir := t.TempDir()
	if err := os.Unsetenv("PK3S_CATALOG_URLS"); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	code := Run(context.Background(), []string{"addon", "list"}, noTelemetryDeps(t, Dependencies{
		Stdout:     &stdout,
		Stderr:     &bytes.Buffer{},
		GOOS:       "linux",
		GOARCH:     "amd64",
		WorkingDir: workingDir,
		CacheDir:   filepath.Join(workingDir, "cache"),
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.String() != defaultCatalogURL {
					t.Fatalf("expected default catalog URL, got %s", req.URL.String())
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`apiVersion: catalogs.productive-k3s.io/v1alpha1
kind: ProductiveK3SCatalog
metadata:
  name: productive-k3s-catalog
entries:
  - kind: addon
    name: nginx
    category: example
    version: 0.1.0
    artifact:
      type: tgz
      url: https://downloads.productive-k3s.io/addons/nginx-0.1.0.tgz
`)),
					Header: make(http.Header),
				}, nil
			}),
		},
	}))
	if code != 0 {
		t.Fatalf("expected addon list to succeed, got %d", code)
	}
	if !strings.Contains(stdout.String(), "nginx\t0.1.0\texample") {
		t.Fatalf("unexpected addon list output: %q", stdout.String())
	}
}

func TestRunCatalogNameDoesNotEmitLocalTGZLookupNoise(t *testing.T) {
	workingDir := t.TempDir()
	t.Setenv("PRODUCTIVE_K3S_SOURCE", "local")
	coreDir := filepath.Join(workingDir, "productive-k3s-core")
	if err := os.MkdirAll(coreDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(coreDir, "productive-k3s-core.sh"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.yaml":
			_, _ = w.Write([]byte(strings.ReplaceAll(`apiVersion: catalogs.productive-k3s.io/v1alpha1
kind: ProductiveK3SCatalog
metadata:
  name: productive-k3s-catalog
entries:
  - kind: addon
    metadata:
      name: nginx
    artifact:
      type: tgz
      url: SERVER_URL/addons/nginx-0.1.0.tgz
`, "SERVER_URL", server.URL)))
		case "/addons/nginx-0.1.0.tgz":
			http.Error(w, "nope", http.StatusBadGateway)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	t.Setenv("PK3S_CATALOG_URLS", server.URL+"/index.yaml")

	var stderr bytes.Buffer
	code := Run(context.Background(), []string{"addon", "install", "nginx", "--cluster-context", "default"}, noTelemetryDeps(t, Dependencies{
		Stdout:     &bytes.Buffer{},
		Stderr:     &stderr,
		GOOS:       "linux",
		GOARCH:     "amd64",
		WorkingDir: workingDir,
		CacheDir:   filepath.Join(workingDir, "cache"),
		HTTPClient: server.Client(),
	}))
	if code != 2 {
		t.Fatalf("expected addon install by catalog name to fail on download error, got %d", code)
	}
	if bytes.Contains(stderr.Bytes(), []byte("tgz not found: nginx")) {
		t.Fatalf("unexpected local tgz lookup noise in stderr: %q", stderr.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte("download failed")) {
		t.Fatalf("expected download failure in stderr, got %q", stderr.String())
	}
}

func TestRunAddonListReadsCatalog(t *testing.T) {
	workingDir := t.TempDir()
	var stdout bytes.Buffer

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.yaml":
			_, _ = w.Write([]byte(strings.ReplaceAll(`apiVersion: catalogs.productive-k3s.io/v1alpha1
kind: ProductiveK3SCatalog
metadata:
  name: productive-k3s-catalog
entries:
  - kind: addon
    name: nginx
    category: example
    version: 0.1.0
    artifact:
      type: tgz
      url: SERVER_URL/addons/nginx-0.1.0.tgz
  - kind: profile
    name: multipass-1-server-2-agents
    category: local
    version: 0.1.0
    artifact:
      type: tgz
      url: SERVER_URL/infra/multipass-1-server-2-agents-0.1.0.tgz
`, "SERVER_URL", server.URL)))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	t.Setenv("PK3S_CATALOG_URLS", server.URL+"/index.yaml")

	code := Run(context.Background(), []string{"addon", "list"}, noTelemetryDeps(t, Dependencies{
		Stdout:     &stdout,
		Stderr:     &bytes.Buffer{},
		GOOS:       "linux",
		GOARCH:     "amd64",
		WorkingDir: workingDir,
		CacheDir:   filepath.Join(workingDir, "cache"),
		HTTPClient: server.Client(),
	}))
	if code != 0 {
		t.Fatalf("expected addon list to succeed, got %d", code)
	}
	if got := stdout.String(); !strings.Contains(got, "nginx\t0.1.0\texample") {
		t.Fatalf("unexpected addon list output: %q", got)
	}
	if strings.Contains(stdout.String(), "multipass-1-server-2-agents") {
		t.Fatalf("addon list should not include profiles: %q", stdout.String())
	}
}

func TestRunAddonInstallRequiresExplicitTarget(t *testing.T) {
	var stderr bytes.Buffer
	code := Run(context.Background(), []string{"addon", "install", "nginx"}, noTelemetryDeps(t, Dependencies{
		Stdout: &bytes.Buffer{},
		Stderr: &stderr,
		GOOS:   "linux",
		GOARCH: "amd64",
	}))
	if code != 2 || !strings.Contains(stderr.String(), "explicit target") {
		t.Fatalf("unexpected addon install target error: code=%d err=%q", code, stderr.String())
	}
}

func TestResolveProfileArgAndPathHelpers(t *testing.T) {
	workingDir := t.TempDir()
	deps := noTelemetryDeps(t, Dependencies{
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		GOOS:       "linux",
		GOARCH:     "amd64",
		WorkingDir: workingDir,
		CacheDir:   filepath.Join(workingDir, "cache"),
	})

	fixturesDir := filepath.Join(workingDir, "tests", "fixtures", "profiles", "aws-single-node")
	if err := os.MkdirAll(fixturesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	shortProfile := filepath.Join(fixturesDir, "basic.env")
	if err := os.WriteFile(shortProfile, []byte("PK3S_INFRA_PROFILE_NAME=basic\nPK3S_INFRA_ENGINE=opentofu\nPK3S_INFRA_SCENARIO=aws-single-node\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got, code := resolveProfileArg([]string{"--profile", shortProfile}, deps); code != 0 || got != shortProfile {
		t.Fatalf("expected explicit profile path, got %q code=%d", got, code)
	}
	if got, code := resolveProfileArg([]string{"basic"}, deps); code != 0 || got != shortProfile {
		t.Fatalf("expected short name resolution, got %q code=%d", got, code)
	}
	if _, code := resolveProfileArg([]string{"--profile"}, deps); code != 2 {
		t.Fatalf("expected missing profile value to fail with 2, got %d", code)
	}
	if _, code := resolveProfileArg(nil, deps); code != 2 {
		t.Fatalf("expected missing profile to fail with 2, got %d", code)
	}
}

func TestDefaultDependenciesAndHelpers(t *testing.T) {
	deps := DefaultDependencies()
	if deps.Stdout == nil || deps.Stderr == nil || deps.HTTPClient == nil || deps.Exec == nil {
		t.Fatalf("default dependencies must populate io/http/exec")
	}
	if deps.GOOS == "" || deps.GOARCH == "" {
		t.Fatalf("default dependencies must capture platform")
	}

	if got := redactObservabilityToken("short-token"); got != "short-token" {
		t.Fatalf("unexpected short token redaction: %q", got)
	}
	if got := removeFlag([]string{"backup", "--core", "--foo"}, "--core"); strings.Join(got, " ") != "backup --foo" {
		t.Fatalf("unexpected removeFlag result: %#v", got)
	}
}

func TestNormalizeProfileURLAndValidationHelpers(t *testing.T) {
	rawGitHub := "https://github.com/jemacchi/productive-k3s-profiles/blob/main/profiles/aws-single-node/basic.env"
	wantRaw := "https://raw.githubusercontent.com/jemacchi/productive-k3s-profiles/main/profiles/aws-single-node/basic.env"
	if got := normalizeProfileURL(rawGitHub); got != wantRaw {
		t.Fatalf("unexpected normalized github url: %q", got)
	}
	if got := normalizeProfileURL("https://example.test/basic.env"); got != "https://example.test/basic.env" {
		t.Fatalf("unexpected passthrough url: %q", got)
	}

	valid := []byte("export PK3S_INFRA_PROFILE_NAME=basic\nPK3S_INFRA_ENGINE=opentofu\nPK3S_INFRA_SCENARIO=aws-single-node\n")
	if err := validateProfileEnv(valid); err != nil {
		t.Fatalf("expected valid exported env format: %v", err)
	}
	if err := validateProfileEnv([]byte("")); err == nil {
		t.Fatalf("expected empty profile to fail")
	}
	if err := validateProfileEnv([]byte("NOPE\n")); err == nil {
		t.Fatalf("expected invalid line to fail")
	}
	if err := validateProfileEnv([]byte("PK3S_INFRA_PROFILE_NAME=basic\nPK3S_INFRA_ENGINE=nomad\nPK3S_INFRA_SCENARIO=aws-single-node\n")); err == nil {
		t.Fatalf("expected unsupported engine to fail")
	}
}

func TestOsExecRunsCommand(t *testing.T) {
	tmp := t.TempDir()
	script := filepath.Join(tmp, "echo.sh")
	if err := os.WriteFile(script, []byte("#!/usr/bin/env bash\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := osExec(context.Background(), Invocation{
		Path: script,
		Args: nil,
		Dir:  tmp,
		Env:  os.Environ(),
	}); err != nil {
		t.Fatalf("osExec should run executable script: %v", err)
	}
}

func TestDelegateReportsExecAndBundleResolutionErrors(t *testing.T) {
	var stderr bytes.Buffer
	workingDir := t.TempDir()
	t.Setenv("PRODUCTIVE_K3S_SOURCE", "local")
	coreDir := filepath.Join(workingDir, "productive-k3s-core")
	if err := os.MkdirAll(coreDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(coreDir, "productive-k3s-core.sh"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	code := delegate(context.Background(), Dependencies{
		Stdout:     &bytes.Buffer{},
		Stderr:     &stderr,
		GOOS:       "linux",
		GOARCH:     "amd64",
		WorkingDir: workingDir,
		CacheDir:   filepath.Join(workingDir, "cache"),
		Exec: func(_ context.Context, invocation Invocation) error {
			return errors.New("boom")
		},
	}, "core", false, []string{"bundle", "info", "--json"}, nil, nil)
	if code != 1 {
		t.Fatalf("expected delegate exec failure, got %d", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("command failed: boom")) {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}

	stderr.Reset()
	t.Setenv("PRODUCTIVE_K3S_SOURCE", "wat")
	code = delegate(context.Background(), Dependencies{
		Stdout:     &bytes.Buffer{},
		Stderr:     &stderr,
		GOOS:       "linux",
		GOARCH:     "amd64",
		WorkingDir: workingDir,
		CacheDir:   filepath.Join(workingDir, "cache"),
		Exec:       func(_ context.Context, invocation Invocation) error { return nil },
	}, "core", false, []string{"bundle", "info", "--json"}, nil, nil)
	if code != 1 {
		t.Fatalf("expected delegate bundle resolution failure, got %d", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("bundle resolution failed")) {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}
