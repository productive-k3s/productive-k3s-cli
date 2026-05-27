package app

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
	code := Run(context.Background(), []string{"bundle", "core", "info", "--json"}, Dependencies{
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
	deps := Dependencies{
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
	}

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
	code := Run(context.Background(), []string{"install", "--core-only", "--dry-run"}, Dependencies{
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
		t.Fatalf("expected install core-only dispatch to succeed, got %d", code)
	}
	if strings.Join(got.Args, " ") != "bootstrap --dry-run" {
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
	deps := Dependencies{
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
	}

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
	t.Setenv("PRODUCTIVE_K3S_SOURCE", "local")
	infraDir := filepath.Join(workingDir, "productive-k3s-infra")
	if err := os.MkdirAll(infraDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(infraDir, "productive-k3s-infra.sh"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	var got Invocation
	code := Run(context.Background(), []string{"profile", "list"}, Dependencies{
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
		t.Fatalf("expected profile list to succeed, got %d", code)
	}
	if strings.Join(got.Args, " ") != "list-profiles" {
		t.Fatalf("unexpected profile list args: %#v", got.Args)
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

func TestResolveProfileArgAndPathHelpers(t *testing.T) {
	workingDir := t.TempDir()
	deps := Dependencies{
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		GOOS:       "linux",
		GOARCH:     "amd64",
		WorkingDir: workingDir,
		CacheDir:   filepath.Join(workingDir, "cache"),
	}

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
	rawGitHub := "https://github.com/jemacchi/productive-k3s-infra/blob/main/profiles/aws-single-node/basic.env"
	wantRaw := "https://raw.githubusercontent.com/jemacchi/productive-k3s-infra/main/profiles/aws-single-node/basic.env"
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
