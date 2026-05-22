package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestRunVersion(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run(context.Background(), []string{"version"}, Dependencies{
		Stdout: &stdout,
		Stderr: &stderr,
		GOOS:   "linux",
		GOARCH: "amd64",
	})

	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if got := stdout.String(); got != "1.0.0\n" {
		t.Fatalf("unexpected version output: %q", got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunHelpListsCommands(t *testing.T) {
	var stdout bytes.Buffer

	code := Run(context.Background(), []string{"help"}, Dependencies{
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
		GOOS:   "linux",
		GOARCH: "amd64",
	})

	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}

	for _, expected := range []string{"install", "doctor", "validate", "bundle", "profile", "version"} {
		if !bytes.Contains(stdout.Bytes(), []byte(expected)) {
			t.Fatalf("help output missing %q: %s", expected, stdout.String())
		}
	}
}

func TestRunHelpProfileShowsUsageAndExamples(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run(context.Background(), []string{"help", "profile"}, Dependencies{
		Stdout: &stdout,
		Stderr: &stderr,
		GOOS:   "linux",
		GOARCH: "amd64",
	})

	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	for _, expected := range []string{
		"pk3s profile <list|validate>",
		"pk3s profile list",
		"pk3s profile validate --profile profiles/multipass/1-server-2-agents.env",
	} {
		if !bytes.Contains(stdout.Bytes(), []byte(expected)) {
			t.Fatalf("profile help missing %q: %s", expected, stdout.String())
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunHelpPlanShowsProfileExamples(t *testing.T) {
	var stdout bytes.Buffer

	code := Run(context.Background(), []string{"help", "plan"}, Dependencies{
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
		GOOS:   "linux",
		GOARCH: "amd64",
	})

	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	for _, expected := range []string{
		"pk3s plan --profile <file|name>",
		"pk3s plan --profile profiles/aws-single-node/basic.env",
		"pk3s plan --profile basic.env",
	} {
		if !bytes.Contains(stdout.Bytes(), []byte(expected)) {
			t.Fatalf("plan help missing %q: %s", expected, stdout.String())
		}
	}
}

func TestRunHelpRejectsUnknownTopic(t *testing.T) {
	var stderr bytes.Buffer

	code := Run(context.Background(), []string{"help", "wat"}, Dependencies{
		Stdout: &bytes.Buffer{},
		Stderr: &stderr,
		GOOS:   "linux",
		GOARCH: "amd64",
	})

	if code == 0 {
		t.Fatalf("expected non-zero exit code")
	}
	for _, expected := range []string{
		"unknown help topic",
		"pk3s help",
	} {
		if !bytes.Contains(bytes.ToLower(stderr.Bytes()), []byte(expected)) {
			t.Fatalf("expected %q in stderr, got %q", expected, stderr.String())
		}
	}
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	var stderr bytes.Buffer

	code := Run(context.Background(), []string{"unsupported-command"}, Dependencies{
		Stdout: &bytes.Buffer{},
		Stderr: &stderr,
		GOOS:   "linux",
		GOARCH: "amd64",
	})

	if code == 0 {
		t.Fatalf("expected non-zero exit code")
	}
	if !bytes.Contains(bytes.ToLower(stderr.Bytes()), []byte("unsupported")) &&
		!bytes.Contains(bytes.ToLower(stderr.Bytes()), []byte("unknown")) {
		t.Fatalf("unexpected error output: %s", stderr.String())
	}
}

func TestRunBlocksCoreInstallOnUnsupportedHost(t *testing.T) {
	var stderr bytes.Buffer

	code := Run(context.Background(), []string{"install", "--core-only"}, Dependencies{
		Stdout: &bytes.Buffer{},
		Stderr: &stderr,
		GOOS:   "windows",
		GOARCH: "amd64",
	})

	if code == 0 {
		t.Fatalf("expected non-zero exit code")
	}
	if !bytes.Contains(stderr.Bytes(), []byte("supported platforms")) {
		t.Fatalf("expected supported platforms guidance, got %q", stderr.String())
	}
}

func TestRequestedSourceModeDefaultsToRemote(t *testing.T) {
	t.Setenv("PRODUCTIVE_K3S_SOURCE", "")
	if got := requestedSourceMode(); got != "remote" {
		t.Fatalf("expected default source mode to be remote, got %q", got)
	}
}

func TestRequestedSourceModeNormalizesExplicitLocal(t *testing.T) {
	t.Setenv("PRODUCTIVE_K3S_SOURCE", "LOCAL")
	if got := requestedSourceMode(); got != "local" {
		t.Fatalf("expected explicit local source mode, got %q", got)
	}
}

func TestRunProfileValidateDownloadsHTTPProfileAndDelegates(t *testing.T) {
	workingDir := t.TempDir()
	t.Setenv("PRODUCTIVE_K3S_SOURCE", "local")
	infraDir := filepath.Join(workingDir, "productive-k3s-infra")
	if err := os.MkdirAll(infraDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(infraDir, "productive-k3s-infra.sh"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("PK3S_INFRA_PROFILE_NAME=multipass-1-server-2-agents\nPK3S_INFRA_ENGINE=opentofu\nPK3S_INFRA_SCENARIO=multipass\n"))
	}))
	defer server.Close()

	var got Invocation
	code := Run(context.Background(), []string{"profile", "validate", "--profile", server.URL + "/profile.env"}, Dependencies{
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
	})

	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if len(got.Args) != 3 || got.Args[0] != "validate-profile" || got.Args[1] != "--profile" {
		t.Fatalf("unexpected delegated args: %#v", got.Args)
	}
	if got.Args[2] == server.URL+"/profile.env" {
		t.Fatalf("expected downloaded local path, got URL")
	}
	if _, err := os.Stat(got.Args[2]); err != nil {
		t.Fatalf("expected downloaded profile to exist: %v", err)
	}
}

func TestRunValidateProfileDelegatesToInfraScenarioValidate(t *testing.T) {
	workingDir := t.TempDir()
	t.Setenv("PRODUCTIVE_K3S_SOURCE", "local")
	infraDir := filepath.Join(workingDir, "productive-k3s-infra")
	if err := os.MkdirAll(infraDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(infraDir, "productive-k3s-infra.sh"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	profilePath := filepath.Join(workingDir, "scenario.env")
	if err := os.WriteFile(profilePath, []byte("PK3S_INFRA_PROFILE_NAME=test\nPK3S_INFRA_ENGINE=opentofu\nPK3S_INFRA_SCENARIO=multipass\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var got Invocation
	code := Run(context.Background(), []string{"validate", "--profile", profilePath}, Dependencies{
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
	if len(got.Args) != 3 || got.Args[0] != "validate" || got.Args[1] != "--profile" || got.Args[2] != profilePath {
		t.Fatalf("unexpected delegated args: %#v", got.Args)
	}
}

func TestRunProfileValidateRejectsInvalidHTTPProfile(t *testing.T) {
	workingDir := t.TempDir()
	t.Setenv("PRODUCTIVE_K3S_SOURCE", "local")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("this is not an env file\n"))
	}))
	defer server.Close()

	var stderr bytes.Buffer
	execCalled := false
	code := Run(context.Background(), []string{"profile", "validate", "--profile", server.URL + "/profile.env"}, Dependencies{
		Stdout:     &bytes.Buffer{},
		Stderr:     &stderr,
		GOOS:       "linux",
		GOARCH:     "amd64",
		WorkingDir: workingDir,
		CacheDir:   filepath.Join(workingDir, "cache"),
		HTTPClient: server.Client(),
		Exec: func(_ context.Context, invocation Invocation) error {
			execCalled = true
			return nil
		},
	})

	if code == 0 {
		t.Fatalf("expected non-zero exit code")
	}
	if execCalled {
		t.Fatalf("expected delegation to be skipped for invalid profile")
	}
	if !bytes.Contains(bytes.ToLower(stderr.Bytes()), []byte("invalid profile")) {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRunConfigTelemetryEnablePersistsPreference(t *testing.T) {
	workingDir := t.TempDir()
	configHome := filepath.Join(workingDir, "config")
	t.Setenv("XDG_CONFIG_HOME", configHome)

	var stdout bytes.Buffer
	code := Run(context.Background(), []string{"config", "telemetry", "enable"}, Dependencies{
		Stdout:     &stdout,
		Stderr:     &bytes.Buffer{},
		GOOS:       "linux",
		GOARCH:     "amd64",
		WorkingDir: workingDir,
		CacheDir:   filepath.Join(workingDir, "cache"),
	})

	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	raw, err := os.ReadFile(filepath.Join(configHome, "productive-k3s-cli", "config.json"))
	if err != nil {
		t.Fatalf("expected config file to exist: %v", err)
	}
	var payload map[string]map[string]bool
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("invalid config json: %v", err)
	}
	if !payload["telemetry"]["enabled"] {
		t.Fatalf("expected telemetry.enabled=true, got %s", raw)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("enabled")) {
		t.Fatalf("expected status message, got %q", stdout.String())
	}
}

func TestRunConfigObservabilitySetPersistsToken(t *testing.T) {
	workingDir := t.TempDir()
	configHome := filepath.Join(workingDir, "config")
	t.Setenv("XDG_CONFIG_HOME", configHome)

	var stdout bytes.Buffer
	code := Run(context.Background(), []string{"config", "observability", "set", "pk3s_live_abcdef123456"}, Dependencies{
		Stdout:     &stdout,
		Stderr:     &bytes.Buffer{},
		GOOS:       "linux",
		GOARCH:     "amd64",
		WorkingDir: workingDir,
		CacheDir:   filepath.Join(workingDir, "cache"),
	})

	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	raw, err := os.ReadFile(filepath.Join(configHome, "productive-k3s-cli", "config.json"))
	if err != nil {
		t.Fatalf("expected config file to exist: %v", err)
	}
	var payload map[string]map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("invalid config json: %v", err)
	}
	if payload["observability"]["bearer_token"] != "pk3s_live_abcdef123456" {
		t.Fatalf("expected observability token to be persisted, got %s", raw)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("configured")) {
		t.Fatalf("expected status message, got %q", stdout.String())
	}
}

func TestRunConfigObservabilityStatusRedactsToken(t *testing.T) {
	workingDir := t.TempDir()
	configHome := filepath.Join(workingDir, "config")
	t.Setenv("XDG_CONFIG_HOME", configHome)
	if err := writeObservabilityToken("pk3s_live_abcdef1234567890"); err != nil {
		t.Fatalf("writeObservabilityToken: %v", err)
	}

	var stdout bytes.Buffer
	code := Run(context.Background(), []string{"config", "observability", "status"}, Dependencies{
		Stdout:     &stdout,
		Stderr:     &bytes.Buffer{},
		GOOS:       "linux",
		GOARCH:     "amd64",
		WorkingDir: workingDir,
		CacheDir:   filepath.Join(workingDir, "cache"),
	})

	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("pk3s_live_ab...")) {
		t.Fatalf("expected redacted token in status output, got %q", stdout.String())
	}
	if bytes.Contains(stdout.Bytes(), []byte("abcdef1234567890")) {
		t.Fatalf("expected full token to stay hidden, got %q", stdout.String())
	}
}

func TestRunPlanTelemetryOverridePropagatesCorrelationAndSendsCliEvents(t *testing.T) {
	workingDir := t.TempDir()
	t.Setenv("PRODUCTIVE_K3S_SOURCE", "local")
	t.Setenv("TELEMETRY_ENDPOINT", "")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(workingDir, "config"))
	infraDir := filepath.Join(workingDir, "productive-k3s-infra")
	if err := os.MkdirAll(infraDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(infraDir, "productive-k3s-infra.sh"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	profilePath := filepath.Join(workingDir, "scenario.env")
	if err := os.WriteFile(profilePath, []byte("PK3S_INFRA_PROFILE_NAME=test\nPK3S_INFRA_ENGINE=opentofu\nPK3S_INFRA_SCENARIO=multipass\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var requests []map[string]any
	var markers []string
	var authz []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("invalid telemetry payload: %v", err)
		}
		requests = append(requests, payload)
		markers = append(markers, r.Header.Get("X-Productive-K3S-Telemetry"))
		authz = append(authz, r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	t.Setenv("TELEMETRY_ENDPOINT", server.URL)
	t.Setenv("TELEMETRY_BEARER_TOKEN", "pk3s_live_testtoken")

	var got Invocation
	code := Run(context.Background(), []string{"plan", "--profile", profilePath, "--telemetry", "enable"}, Dependencies{
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
	})

	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	env := map[string]string{}
	for _, item := range got.Env {
		parts := bytes.SplitN([]byte(item), []byte("="), 2)
		if len(parts) == 2 {
			env[string(parts[0])] = string(parts[1])
		}
	}
	if env["TELEMETRY_ENABLED"] != "true" {
		t.Fatalf("expected TELEMETRY_ENABLED=true, got %q", env["TELEMETRY_ENABLED"])
	}
	if env["TELEMETRY_SESSION_ID"] == "" {
		t.Fatalf("expected TELEMETRY_SESSION_ID to be propagated")
	}
	if env["TELEMETRY_PARENT_RUN_ID"] == "" {
		t.Fatalf("expected TELEMETRY_PARENT_RUN_ID to be propagated")
	}
	if env["TELEMETRY_PARENT_RUN_ID"] == env["TELEMETRY_SESSION_ID"] {
		t.Fatalf("expected parent run id to differ from session id")
	}
	if env["TELEMETRY_ENDPOINT"] != server.URL {
		t.Fatalf("expected TELEMETRY_ENDPOINT=%q, got %q", server.URL, env["TELEMETRY_ENDPOINT"])
	}
	if env["TELEMETRY_MARKER"] != "pk3s-public-v1" {
		t.Fatalf("expected TELEMETRY_MARKER to be propagated, got %q", env["TELEMETRY_MARKER"])
	}
	if env["TELEMETRY_BEARER_TOKEN"] != "pk3s_live_testtoken" {
		t.Fatalf("expected TELEMETRY_BEARER_TOKEN to be propagated, got %q", env["TELEMETRY_BEARER_TOKEN"])
	}
	if len(requests) != 2 {
		t.Fatalf("expected started/completed telemetry events, got %d", len(requests))
	}
	if requests[0]["event_name"] != "cli.command.started" {
		t.Fatalf("unexpected first event: %#v", requests[0]["event_name"])
	}
	if requests[1]["event_name"] != "cli.command.completed" {
		t.Fatalf("unexpected second event: %#v", requests[1]["event_name"])
	}
	if len(markers) != 2 || markers[0] != "pk3s-public-v1" || markers[1] != "pk3s-public-v1" {
		t.Fatalf("expected telemetry marker header on both requests, got %#v", markers)
	}
	if len(authz) != 2 || authz[0] != "Bearer pk3s_live_testtoken" || authz[1] != "Bearer pk3s_live_testtoken" {
		t.Fatalf("expected observability bearer token on both requests, got %#v", authz)
	}
}

func TestCLITelemetryDefaultsUseCanonicalEndpointAndMarker(t *testing.T) {
	ctx := newCLITelemetryContext("plan", "infra", true, Dependencies{
		GOOS:       "linux",
		GOARCH:     "amd64",
		WorkingDir: t.TempDir(),
	}, true)
	if ctx.Endpoint != defaultTelemetryEndpoint {
		t.Fatalf("expected default endpoint %q, got %q", defaultTelemetryEndpoint, ctx.Endpoint)
	}
	if ctx.Marker != defaultTelemetryMarker {
		t.Fatalf("expected default marker %q, got %q", defaultTelemetryMarker, ctx.Marker)
	}
}
