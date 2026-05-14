package app

import (
	"bytes"
	"context"
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
