package app

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const defaultTelemetryEndpoint = "https://telemetry.productive-k3s.io/telemetry"
const defaultTelemetryMarker = "pk3s-public-v1"

type telemetryPreference struct {
	Telemetry struct {
		Enabled *bool `json:"enabled"`
	} `json:"telemetry"`
	Observability struct {
		BearerToken string `json:"bearer_token,omitempty"`
	} `json:"observability"`
}

type cliTelemetryContext struct {
	Enabled     bool
	SessionID   string
	RunID       string
	Endpoint    string
	Marker      string
	BearerToken string
	Command     string
	Target      string
	SourceMode  string
	HasProfile  bool
	MaxRetries  int
	ConnectTO   time.Duration
	RequestTO   time.Duration
	UserAgent   string
	WorkingDir  string
}

func parseTelemetryOverride(args []string, stderr ioWriter) ([]string, *bool, int) {
	filtered := make([]string, 0, len(args))
	var override *bool
	for i := 0; i < len(args); i++ {
		if args[i] != "--telemetry" {
			filtered = append(filtered, args[i])
			continue
		}
		if i+1 >= len(args) {
			fmt.Fprintln(stderr, "missing value for --telemetry; use enable or disable")
			return nil, nil, 2
		}
		switch strings.ToLower(strings.TrimSpace(args[i+1])) {
		case "enable":
			value := true
			override = &value
		case "disable":
			value := false
			override = &value
		default:
			fmt.Fprintln(stderr, "invalid value for --telemetry; use enable or disable")
			return nil, nil, 2
		}
		i++
	}
	return filtered, override, 0
}

type ioWriter interface {
	Write([]byte) (int, error)
}

func telemetryConfigPath() (string, error) {
	if xdg := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); xdg != "" {
		return filepath.Join(xdg, "productive-k3s-cli", "config.json"), nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "productive-k3s-cli", "config.json"), nil
}

func readTelemetryPreference() (*bool, error) {
	cfg, err := readTelemetryConfig()
	if err != nil {
		return nil, err
	}
	return cfg.Telemetry.Enabled, nil
}

func readTelemetryConfig() (*telemetryPreference, error) {
	path, err := telemetryConfigPath()
	if err != nil {
		return &telemetryPreference{}, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &telemetryPreference{}, nil
		}
		return &telemetryPreference{}, err
	}
	var cfg telemetryPreference
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return &telemetryPreference{}, err
	}
	return &cfg, nil
}

func writeTelemetryPreference(enabled bool) error {
	path, err := telemetryConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	cfg, err := readTelemetryConfig()
	if err != nil {
		return err
	}
	cfg.Telemetry.Enabled = &enabled
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

func readObservabilityToken() (string, error) {
	cfg, err := readTelemetryConfig()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(cfg.Observability.BearerToken), nil
}

func writeObservabilityToken(token string) error {
	path, err := telemetryConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	cfg, err := readTelemetryConfig()
	if err != nil {
		return err
	}
	cfg.Observability.BearerToken = strings.TrimSpace(token)
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

func clearObservabilityToken() error {
	path, err := telemetryConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	cfg, err := readTelemetryConfig()
	if err != nil {
		return err
	}
	cfg.Observability.BearerToken = ""
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

func isInteractiveTerminal() bool {
	stdinInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	stdoutInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (stdinInfo.Mode()&os.ModeCharDevice) != 0 && (stdoutInfo.Mode()&os.ModeCharDevice) != 0
}

func promptTelemetryConsent(stderr ioWriter) (bool, error) {
	fmt.Fprint(stderr, "Productive K3S CLI can send anonymous telemetry about this command chain to help improve the installation flow. It does not include hostnames, IPs, or profile file contents. Enable anonymous telemetry for future pk3s runs? [Y/n]: ")
	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "" || answer == "y" || answer == "yes", nil
}

func parseExplicitTelemetryEnv() (*bool, bool) {
	raw, ok := os.LookupEnv("TELEMETRY_ENABLED")
	if !ok {
		return nil, false
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "y", "on":
		value := true
		return &value, true
	case "0", "false", "no", "n", "off":
		value := false
		return &value, true
	default:
		return nil, false
	}
}

func resolveTelemetryEnabled(override *bool) (bool, error) {
	if override != nil {
		return *override, nil
	}
	if value, ok := parseExplicitTelemetryEnv(); ok && value != nil {
		return *value, nil
	}
	if value, err := readTelemetryPreference(); err == nil && value != nil {
		return *value, nil
	} else if err != nil {
		return false, err
	}
	if isInteractiveTerminal() {
		enabled, err := promptTelemetryConsent(os.Stderr)
		if err != nil {
			return false, err
		}
		if err := writeTelemetryPreference(enabled); err != nil {
			return false, err
		}
		return enabled, nil
	}
	return false, nil
}

func resolveTelemetryEndpoint() string {
	if value, ok := os.LookupEnv("TELEMETRY_ENDPOINT"); ok {
		return value
	}
	return defaultTelemetryEndpoint
}

func resolveTelemetryMarker() string {
	if value, ok := os.LookupEnv("TELEMETRY_MARKER"); ok {
		return value
	}
	return defaultTelemetryMarker
}

func resolveTelemetryBearerToken() string {
	if value, ok := os.LookupEnv("TELEMETRY_BEARER_TOKEN"); ok {
		return strings.TrimSpace(value)
	}
	value, err := readObservabilityToken()
	if err != nil {
		return ""
	}
	return value
}

func resolveTelemetryInt(name string, fallback int) int {
	if value, ok := os.LookupEnv(name); ok {
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		if err == nil && parsed > 0 {
			return parsed
		}
	}
	return fallback
}

func generateTelemetryID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

func newCLITelemetryContext(command string, target string, hasProfile bool, deps Dependencies, enabled bool) *cliTelemetryContext {
	if !enabled {
		return &cliTelemetryContext{Enabled: false}
	}
	return &cliTelemetryContext{
		Enabled:     true,
		SessionID:   firstNonEmpty(os.Getenv("TELEMETRY_SESSION_ID"), generateTelemetryID()),
		RunID:       generateTelemetryID(),
		Endpoint:    resolveTelemetryEndpoint(),
		Marker:      resolveTelemetryMarker(),
		BearerToken: resolveTelemetryBearerToken(),
		Command:     command,
		Target:      target,
		SourceMode:  requestedSourceMode(),
		HasProfile:  hasProfile,
		MaxRetries:  resolveTelemetryInt("TELEMETRY_MAX_RETRIES", 3),
		ConnectTO:   time.Duration(resolveTelemetryInt("TELEMETRY_CONNECT_TIMEOUT_SECONDS", 5)) * time.Second,
		RequestTO:   time.Duration(resolveTelemetryInt("TELEMETRY_REQUEST_TIMEOUT_SECONDS", 10)) * time.Second,
		UserAgent:   fmt.Sprintf("productive-k3s-cli/%s", Version),
		WorkingDir:  deps.WorkingDir,
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func (c *cliTelemetryContext) childEnv() map[string]string {
	env := map[string]string{
		"TELEMETRY_ENABLED":       "false",
		"TELEMETRY_SESSION_ID":    "",
		"TELEMETRY_PARENT_RUN_ID": "",
		"TELEMETRY_BEARER_TOKEN":  "",
	}
	if !c.Enabled {
		return env
	}
	env["TELEMETRY_ENABLED"] = "true"
	env["TELEMETRY_SESSION_ID"] = c.SessionID
	env["TELEMETRY_PARENT_RUN_ID"] = c.RunID
	env["TELEMETRY_ENDPOINT"] = c.Endpoint
	env["TELEMETRY_MARKER"] = c.Marker
	env["TELEMETRY_BEARER_TOKEN"] = c.BearerToken
	if _, ok := os.LookupEnv("TELEMETRY_MAX_RETRIES"); ok {
		env["TELEMETRY_MAX_RETRIES"] = strconv.Itoa(c.MaxRetries)
	}
	if _, ok := os.LookupEnv("TELEMETRY_CONNECT_TIMEOUT_SECONDS"); ok {
		env["TELEMETRY_CONNECT_TIMEOUT_SECONDS"] = strconv.Itoa(int(c.ConnectTO / time.Second))
	}
	if _, ok := os.LookupEnv("TELEMETRY_REQUEST_TIMEOUT_SECONDS"); ok {
		env["TELEMETRY_REQUEST_TIMEOUT_SECONDS"] = strconv.Itoa(int(c.RequestTO / time.Second))
	}
	if value, ok := os.LookupEnv("TELEMETRY_OUTBOX_DIR"); ok {
		env["TELEMETRY_OUTBOX_DIR"] = value
	}
	return env
}

func (c *cliTelemetryContext) send(ctx context.Context, client *http.Client, eventName string, result string, deps Dependencies) {
	if !c.Enabled || strings.TrimSpace(c.Endpoint) == "" {
		return
	}
	payload := map[string]any{
		"schema_version": "1",
		"event_family":   "usage",
		"event_name":     eventName,
		"sent_at":        time.Now().Format(time.RFC3339),
		"session_id":     c.SessionID,
		"run_id":         c.RunID,
		"parent_run_id":  nil,
		"component":      "cli",
		"command": map[string]any{
			"name":        c.Command,
			"target":      c.Target,
			"source_mode": c.SourceMode,
			"has_profile": c.HasProfile,
			"result":      result,
		},
		"client": map[string]any{
			"repository":        "productive-k3s-cli",
			"version":           Version,
			"platform":          deps.GOOS,
			"arch":              deps.GOARCH,
			"user_agent":        c.UserAgent,
			"telemetry_enabled": "true",
		},
		"telemetry_meta": map[string]any{
			"delivery_mode":         "best-effort",
			"anonymous_by_contract": true,
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	attemptCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	for attempt := 0; attempt < c.MaxRetries; attempt++ {
		reqCtx, reqCancel := context.WithTimeout(attemptCtx, c.RequestTO)
		req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, c.Endpoint, bytes.NewReader(raw))
		if err != nil {
			reqCancel()
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", c.UserAgent)
		req.Header.Set("X-Productive-K3S-Telemetry", c.Marker)
		if strings.TrimSpace(c.BearerToken) != "" {
			req.Header.Set("Authorization", "Bearer "+c.BearerToken)
		}
		resp, err := client.Do(req)
		reqCancel()
		if err == nil && resp != nil {
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return
			}
		}
	}
}
