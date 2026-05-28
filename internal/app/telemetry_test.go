package app

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestParseTelemetryOverrideEnable(t *testing.T) {
	var stderr bytes.Buffer

	args, override, code := parseTelemetryOverride([]string{"profile", "list", "--telemetry", "enable"}, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if len(args) != 2 || args[0] != "profile" || args[1] != "list" {
		t.Fatalf("unexpected filtered args: %#v", args)
	}
	if override == nil || !*override {
		t.Fatalf("expected telemetry override to be enabled")
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestParseTelemetryOverrideRejectsInvalidValue(t *testing.T) {
	var stderr bytes.Buffer

	_, _, code := parseTelemetryOverride([]string{"bundle", "info", "--telemetry", "maybe"}, &stderr)
	if code == 0 {
		t.Fatalf("expected non-zero exit code")
	}
	if !bytes.Contains(stderr.Bytes(), []byte("invalid value for --telemetry")) {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestResolveTelemetryEnabledUsesExplicitEnv(t *testing.T) {
	t.Setenv("TELEMETRY_ENABLED", "true")
	enabled, err := resolveTelemetryEnabled(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !enabled {
		t.Fatalf("expected telemetry to resolve to true")
	}
}

func TestWriteTelemetryPreferenceRoundTrip(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	if err := writeTelemetryPreference(true); err != nil {
		t.Fatalf("writeTelemetryPreference failed: %v", err)
	}
	value, err := readTelemetryPreference()
	if err != nil {
		t.Fatalf("readTelemetryPreference failed: %v", err)
	}
	if value == nil || !*value {
		t.Fatalf("expected stored telemetry preference to be true")
	}

	configPath := filepath.Join(configHome, "productive-k3s-cli", "config.json")
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("expected config file to exist: %v", err)
	}
}

func TestObservabilityTokenRoundTrip(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	if err := writeObservabilityToken("secret-token-value"); err != nil {
		t.Fatalf("writeObservabilityToken failed: %v", err)
	}
	token, err := readObservabilityToken()
	if err != nil {
		t.Fatalf("readObservabilityToken failed: %v", err)
	}
	if token != "secret-token-value" {
		t.Fatalf("unexpected token: %q", token)
	}

	if err := clearObservabilityToken(); err != nil {
		t.Fatalf("clearObservabilityToken failed: %v", err)
	}
	token, err = readObservabilityToken()
	if err != nil {
		t.Fatalf("readObservabilityToken after clear failed: %v", err)
	}
	if token != "" {
		t.Fatalf("expected cleared token, got %q", token)
	}
}
