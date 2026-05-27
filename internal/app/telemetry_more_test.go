package app

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestResolveTelemetryEnabledRespectsStoredPreferenceAndFallbacks(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	if err := writeTelemetryPreference(false); err != nil {
		t.Fatal(err)
	}

	enabled, err := resolveTelemetryEnabled(nil)
	if err != nil {
		t.Fatalf("resolveTelemetryEnabled: %v", err)
	}
	if enabled {
		t.Fatalf("expected stored preference to disable telemetry")
	}

	t.Setenv("TELEMETRY_ENABLED", "garbage")
	enabled, err = resolveTelemetryEnabled(nil)
	if err != nil {
		t.Fatalf("resolveTelemetryEnabled with invalid env fallback: %v", err)
	}
	if enabled {
		t.Fatalf("expected stored false preference to still win")
	}
}

func TestResolveTelemetryHelpersAndChildEnv(t *testing.T) {
	t.Setenv("TELEMETRY_MAX_RETRIES", "7")
	t.Setenv("TELEMETRY_CONNECT_TIMEOUT_SECONDS", "9")
	t.Setenv("TELEMETRY_REQUEST_TIMEOUT_SECONDS", "11")
	t.Setenv("TELEMETRY_OUTBOX_DIR", "/tmp/pk3s-outbox")
	t.Setenv("TELEMETRY_BEARER_TOKEN", "  bearer-1  ")

	ctx := newCLITelemetryContext("plan", "infra", true, Dependencies{
		GOOS:       "linux",
		GOARCH:     "amd64",
		WorkingDir: t.TempDir(),
	}, true)
	if ctx.MaxRetries != 7 || ctx.ConnectTO != 9*time.Second || ctx.RequestTO != 11*time.Second {
		t.Fatalf("unexpected telemetry timings: %+v", ctx)
	}
	if ctx.BearerToken != "bearer-1" {
		t.Fatalf("unexpected bearer token: %q", ctx.BearerToken)
	}

	child := ctx.childEnv()
	if child["TELEMETRY_ENABLED"] != "true" {
		t.Fatalf("expected enabled child env, got %#v", child)
	}
	if child["TELEMETRY_MAX_RETRIES"] != "7" || child["TELEMETRY_CONNECT_TIMEOUT_SECONDS"] != "9" || child["TELEMETRY_REQUEST_TIMEOUT_SECONDS"] != "11" {
		t.Fatalf("unexpected child env timing overrides: %#v", child)
	}
	if child["TELEMETRY_OUTBOX_DIR"] != "/tmp/pk3s-outbox" {
		t.Fatalf("unexpected child outbox dir: %#v", child)
	}

	disabled := (&cliTelemetryContext{Enabled: false}).childEnv()
	if disabled["TELEMETRY_ENABLED"] != "false" {
		t.Fatalf("expected disabled child env, got %#v", disabled)
	}
}

func TestResolveTelemetryBearerTokenFallsBackToConfig(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	if err := os.Unsetenv("TELEMETRY_BEARER_TOKEN"); err != nil {
		t.Fatal(err)
	}
	if err := writeObservabilityToken("stored-token"); err != nil {
		t.Fatal(err)
	}
	if got := resolveTelemetryBearerToken(); got != "stored-token" {
		t.Fatalf("expected stored token, got %q", got)
	}
}

func TestReadTelemetryConfigInvalidJSONAndResolveTelemetryIntFallback(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	configPath := filepath.Join(configHome, "productive-k3s-cli", "config.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readTelemetryConfig(); err == nil {
		t.Fatalf("expected invalid config json to fail")
	}

	t.Setenv("TELEMETRY_MAX_RETRIES", "0")
	if got := resolveTelemetryInt("TELEMETRY_MAX_RETRIES", 3); got != 3 {
		t.Fatalf("expected fallback retry count, got %d", got)
	}
}

func TestParseTelemetryOverrideMissingValue(t *testing.T) {
	var stderr bytes.Buffer
	_, _, code := parseTelemetryOverride([]string{"plan", "--telemetry"}, &stderr)
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("missing value for --telemetry")) {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestParseExplicitTelemetryEnvFalseAndMarkerOverride(t *testing.T) {
	t.Setenv("TELEMETRY_ENABLED", "off")
	value, ok := parseExplicitTelemetryEnv()
	if !ok || value == nil || *value {
		t.Fatalf("expected explicit false telemetry env")
	}

	t.Setenv("TELEMETRY_MARKER", "custom-marker")
	if got := resolveTelemetryMarker(); got != "custom-marker" {
		t.Fatalf("unexpected telemetry marker %q", got)
	}
}

func TestPromptTelemetryConsentAcceptsAndRejects(t *testing.T) {
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	inputYes, err := os.CreateTemp(t.TempDir(), "stdin-yes")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := inputYes.WriteString("yes\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := inputYes.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	os.Stdin = inputYes

	var stderr bytes.Buffer
	enabled, err := promptTelemetryConsent(&stderr)
	if err != nil {
		t.Fatalf("promptTelemetryConsent yes: %v", err)
	}
	if !enabled {
		t.Fatalf("expected yes prompt to enable telemetry")
	}

	inputNo, err := os.CreateTemp(t.TempDir(), "stdin-no")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := inputNo.WriteString("n\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := inputNo.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	os.Stdin = inputNo

	enabled, err = promptTelemetryConsent(&stderr)
	if err != nil {
		t.Fatalf("promptTelemetryConsent no: %v", err)
	}
	if enabled {
		t.Fatalf("expected no prompt to disable telemetry")
	}
	if !bytes.Contains(stderr.Bytes(), []byte("Enable anonymous telemetry")) {
		t.Fatalf("expected consent prompt text, got %q", stderr.String())
	}
}
