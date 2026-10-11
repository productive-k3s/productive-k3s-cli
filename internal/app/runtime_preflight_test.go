package app

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeProfilePackage(t *testing.T, profileYAML string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "profile.tgz")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(file)
	tw := tar.NewWriter(gz)
	body := []byte(profileYAML)
	if err := tw.WriteHeader(&tar.Header{Name: "./profile.yaml", Mode: 0o600, Size: int64(len(body))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPackagedProfileRuntimePreflightRejectsDisabledScenario(t *testing.T) {
	t.Setenv("PK3S_RUNTIME_DISABLED_SCENARIOS", " multipass, nested-vm ")
	path := writeProfilePackage(t, `
metadata:
  name: multipass-1-server-2-agents
spec:
  scenario:
    type: multipass
  engine:
    type: opentofu
`)
	stderr := &bytes.Buffer{}
	code := packagedProfileRuntimePreflight(path, Dependencies{
		Stderr: stderr,
		LookPath: func(string) (string, error) {
			return "/bin/tool", nil
		},
	})
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
	if got := stderr.String(); !strings.Contains(got, `scenario "multipass" is disabled by runtime policy`) {
		t.Fatalf("unexpected diagnostic: %s", got)
	}
}

func TestPackagedProfileRuntimePreflightReportsMissingCommands(t *testing.T) {
	t.Setenv("PK3S_RUNTIME_DISABLED_SCENARIOS", "")
	path := writeProfilePackage(t, `
metadata:
  name: multipass-demo
spec:
  scenario:
    type: multipass
  engine:
    type: opentofu
`)
	stderr := &bytes.Buffer{}
	code := packagedProfileRuntimePreflight(path, Dependencies{
		Stderr: stderr,
		LookPath: func(string) (string, error) {
			return "", errors.New("missing")
		},
	})
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
	if got := stderr.String(); !strings.Contains(got, "missing required command(s): curl, jq, multipass, sha256sum, ssh, ssh-keygen, tar, timeout, tofu") {
		t.Fatalf("unexpected diagnostic: %s", got)
	}
}

func TestPackagedProfileRuntimePreflightAcceptsAvailableCommands(t *testing.T) {
	t.Setenv("PK3S_RUNTIME_DISABLED_SCENARIOS", "")
	path := writeProfilePackage(t, `
metadata:
  name: aws-demo
spec:
  scenario:
    type: aws-single-node
  engine:
    type: opentofu
`)
	code := packagedProfileRuntimePreflight(path, Dependencies{
		Stderr: &bytes.Buffer{},
		LookPath: func(command string) (string, error) {
			if command != "jq" && command != "tofu" {
				t.Fatalf("unexpected command lookup: %s", command)
			}
			return "/usr/local/bin/" + command, nil
		},
	})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
}

func TestProfileRuntimeFromEnv(t *testing.T) {
	requirements := profileRuntimeFromEnv([]byte(`
export PK3S_INFRA_PROFILE_NAME=demo
PK3S_INFRA_SCENARIO='multipass'
PK3S_INFRA_ENGINE="opentofu"
`))
	if requirements.Name != "demo" || requirements.Scenario != "multipass" || requirements.Engine != "opentofu" {
		t.Fatalf("unexpected requirements: %#v", requirements)
	}
}

func TestProfileEnvRuntimePreflightHandlesAvailableAndMissingProfiles(t *testing.T) {
	t.Setenv("PK3S_RUNTIME_DISABLED_SCENARIOS", "")
	stderr := &bytes.Buffer{}
	deps := Dependencies{
		Stderr: stderr,
		LookPath: func(string) (string, error) {
			return "/bin/tool", nil
		},
	}
	path := filepath.Join(t.TempDir(), "shell.env")
	if err := os.WriteFile(path, []byte("PK3S_INFRA_PROFILE_NAME=shell-demo\nPK3S_INFRA_ENGINE=shell\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if code := profileEnvRuntimePreflight(path, deps); code != 0 {
		t.Fatalf("expected available shell profile, got %d: %s", code, stderr.String())
	}
	if code := profileEnvRuntimePreflight(path+".missing", deps); code != 2 {
		t.Fatalf("expected missing profile to fail, got %d", code)
	}
}

func TestPackagedProfileRuntimePreflightDefersPackagesWithoutMetadata(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.tgz")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(file)
	tw := tar.NewWriter(gz)
	body := []byte("legacy")
	if err := tw.WriteHeader(&tar.Header{Name: "README.md", Mode: 0o600, Size: int64(len(body))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if code := packagedProfileRuntimePreflight(path, Dependencies{Stderr: &bytes.Buffer{}}); code != 0 {
		t.Fatalf("expected package validation to remain delegated, got %d", code)
	}
}

func TestPackagedProfileRuntimePreflightReportsUnreadablePackage(t *testing.T) {
	stderr := &bytes.Buffer{}
	code := packagedProfileRuntimePreflight(filepath.Join(t.TempDir(), "missing.tgz"), Dependencies{Stderr: stderr})
	if code != 2 || !strings.Contains(stderr.String(), "could not inspect profile runtime requirements") {
		t.Fatalf("unexpected result: code=%d stderr=%s", code, stderr.String())
	}
}

func TestProfileRuntimePreflightReportsAnsibleRequirements(t *testing.T) {
	t.Setenv("PK3S_RUNTIME_DISABLED_SCENARIOS", "")
	stderr := &bytes.Buffer{}
	code := profileRuntimePreflight(profileRuntimeRequirements{Engine: "ansible"}, Dependencies{
		Stderr: stderr,
		LookPath: func(string) (string, error) {
			return "", errors.New("missing")
		},
	})
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
	if got := stderr.String(); !strings.Contains(got, `profile "unknown"`) || !strings.Contains(got, "ansible-playbook") {
		t.Fatalf("unexpected diagnostic: %s", got)
	}
}

func TestRunProfileInstallStopsBeforeDelegationWhenScenarioIsDisabled(t *testing.T) {
	t.Setenv("PK3S_RUNTIME_DISABLED_SCENARIOS", "multipass")
	path := writeProfilePackage(t, `
metadata:
  name: multipass-demo
spec:
  scenario:
    type: multipass
  engine:
    type: opentofu
`)
	called := false
	stderr := &bytes.Buffer{}
	code := Run(context.Background(), []string{"profile", "install", "--tgz", path}, noTelemetryDeps(t, Dependencies{
		Stdout:     &bytes.Buffer{},
		Stderr:     stderr,
		WorkingDir: t.TempDir(),
		CacheDir:   t.TempDir(),
		Exec: func(context.Context, Invocation) error {
			called = true
			return nil
		},
		LookPath: func(string) (string, error) {
			return "/bin/tool", nil
		},
	}))
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
	if called {
		t.Fatal("runtime delegation must not start after a failed preflight")
	}
	if got := stderr.String(); !strings.Contains(got, "disabled by runtime policy") {
		t.Fatalf("unexpected diagnostic: %s", got)
	}
}
