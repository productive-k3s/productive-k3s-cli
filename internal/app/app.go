package app

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jemacchi/productive-k3s-cli/internal/bundles"
	"github.com/jemacchi/productive-k3s-cli/internal/platform"
)

var Version = "1.0.0"

type Dependencies struct {
	Stdout       io.Writer
	Stderr       io.Writer
	GOOS         string
	GOARCH       string
	WorkingDir   string
	CacheDir     string
	HTTPClient   *http.Client
	Exec         func(context.Context, Invocation) error
}

type Invocation struct {
	Path string
	Args []string
	Dir  string
	Env  []string
}

func DefaultDependencies() Dependencies {
	wd, _ := os.Getwd()
	cacheDir, _ := bundles.DefaultCacheDir()
	return Dependencies{
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
		GOOS:       runtime.GOOS,
		GOARCH:     runtime.GOARCH,
		WorkingDir: wd,
		CacheDir:   cacheDir,
		HTTPClient: http.DefaultClient,
		Exec:       osExec,
	}
}

func Run(ctx context.Context, args []string, deps Dependencies) int {
	if deps.Stdout == nil {
		deps.Stdout = io.Discard
	}
	if deps.Stderr == nil {
		deps.Stderr = io.Discard
	}
	if deps.WorkingDir == "" {
		if wd, err := os.Getwd(); err == nil {
			deps.WorkingDir = wd
		}
	}
	if deps.CacheDir == "" {
		cacheDir, err := bundles.DefaultCacheDir()
		if err != nil {
			fmt.Fprintf(deps.Stderr, "could not resolve cache directory: %v\n", err)
			return 1
		}
		deps.CacheDir = cacheDir
	}
	if deps.HTTPClient == nil {
		deps.HTTPClient = http.DefaultClient
	}
	if deps.Exec == nil {
		deps.Exec = osExec
	}

	if len(args) == 0 {
		printRootHelp(deps.Stdout)
		return 0
	}

	filteredArgs, telemetryOverride, code := parseTelemetryOverride(args, deps.Stderr)
	if code != 0 {
		return code
	}
	args = filteredArgs

	switch args[0] {
	case "help", "-h", "--help":
		return runHelp(args[1:], deps)
	case "version":
		fmt.Fprintln(deps.Stdout, Version)
		return 0
	case "config":
		return runConfig(args[1:], deps)
	case "bundle":
		return runBundle(ctx, args[1:], deps)
	case "profile":
		return runProfile(ctx, args[1:], deps, telemetryOverride)
	case "plan", "apply", "destroy", "status":
		return runInfraProfileCommand(ctx, args[0], args[1:], deps, telemetryOverride)
	case "doctor":
		return runDoctor(ctx, args[1:], deps, telemetryOverride)
	case "install":
		return runInstall(ctx, args[1:], deps, telemetryOverride)
	case "validate":
		return runValidate(ctx, args[1:], deps, telemetryOverride)
	case "backup":
		return runBackup(ctx, args[1:], deps, telemetryOverride)
	default:
		fmt.Fprintf(deps.Stderr, "Unsupported command: %s\n\n", args[0])
		printRootHelp(deps.Stderr)
		return 2
	}
}

func runConfig(args []string, deps Dependencies) int {
	if len(args) < 2 {
		fmt.Fprintln(deps.Stderr, "Usage: pk3s config <telemetry|observability> ...")
		return 2
	}
	switch args[0] {
	case "telemetry":
		switch args[1] {
		case "enable":
			if err := writeTelemetryPreference(true); err != nil {
				fmt.Fprintf(deps.Stderr, "could not persist telemetry config: %v\n", err)
				return 1
			}
			fmt.Fprintln(deps.Stdout, "Telemetry has been enabled for future pk3s runs.")
			return 0
		case "disable":
			if err := writeTelemetryPreference(false); err != nil {
				fmt.Fprintf(deps.Stderr, "could not persist telemetry config: %v\n", err)
				return 1
			}
			fmt.Fprintln(deps.Stdout, "Telemetry has been disabled for future pk3s runs.")
			return 0
		case "status":
			value, err := readTelemetryPreference()
			if err != nil {
				fmt.Fprintf(deps.Stderr, "could not read telemetry config: %v\n", err)
				return 1
			}
			if value == nil {
				fmt.Fprintln(deps.Stdout, "Telemetry preference is not configured.")
				return 0
			}
			if *value {
				fmt.Fprintln(deps.Stdout, "Telemetry is currently enabled.")
			} else {
				fmt.Fprintln(deps.Stdout, "Telemetry is currently disabled.")
			}
			return 0
		default:
			fmt.Fprintln(deps.Stderr, "Usage: pk3s config telemetry <enable|disable|status>")
			return 2
		}
	case "observability":
		switch args[1] {
		case "set":
			if len(args) < 3 || strings.TrimSpace(args[2]) == "" {
				fmt.Fprintln(deps.Stderr, "Usage: pk3s config observability set <telemetry-bearer-token>")
				return 2
			}
			if err := writeObservabilityToken(args[2]); err != nil {
				fmt.Fprintf(deps.Stderr, "could not persist observability config: %v\n", err)
				return 1
			}
			fmt.Fprintln(deps.Stdout, "Observability token has been configured for future pk3s runs.")
			return 0
		case "clear":
			if err := clearObservabilityToken(); err != nil {
				fmt.Fprintf(deps.Stderr, "could not persist observability config: %v\n", err)
				return 1
			}
			fmt.Fprintln(deps.Stdout, "Observability token has been cleared.")
			return 0
		case "status":
			token, err := readObservabilityToken()
			if err != nil {
				fmt.Fprintf(deps.Stderr, "could not read observability config: %v\n", err)
				return 1
			}
			if token == "" {
				fmt.Fprintln(deps.Stdout, "Observability token is not configured.")
				return 0
			}
			fmt.Fprintf(deps.Stdout, "Observability token is configured: %s\n", redactObservabilityToken(token))
			return 0
		default:
			fmt.Fprintln(deps.Stderr, "Usage: pk3s config observability <set|clear|status>")
			return 2
		}
	default:
		fmt.Fprintln(deps.Stderr, "Usage: pk3s config <telemetry|observability> ...")
		return 2
	}
}

func redactObservabilityToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	if len(token) <= 12 {
		return token
	}
	return token[:12] + "..."
}

func runBundle(ctx context.Context, args []string, deps Dependencies) int {
	if len(args) < 2 {
		fmt.Fprintln(deps.Stderr, "Usage: pk3s bundle <core|infra> info [--json]")
		return 2
	}
	kind := args[0]
	if args[1] != "info" {
		fmt.Fprintln(deps.Stderr, "Usage: pk3s bundle <core|infra> info [--json]")
		return 2
	}
	extra := []string{"bundle", "info", "--json"}
	if len(args) > 2 {
		extra = []string{"bundle", "info"}
		extra = append(extra, args[2:]...)
	}
	return delegate(ctx, deps, kind, false, extra, nil, nil)
}

func runProfile(ctx context.Context, args []string, deps Dependencies, telemetryOverride *bool) int {
	if len(args) == 0 {
		fmt.Fprintln(deps.Stderr, "Usage: pk3s profile <list|validate> [profile]")
		return 2
	}
	switch args[0] {
	case "list":
		return delegate(ctx, deps, "infra", false, []string{"list-profiles"}, nil, newCLITelemetryContext("profile-list", "infra", false, deps, mustResolveTelemetry(telemetryOverride, deps)))
	case "validate":
		profile, code := resolveProfileArg(args[1:], deps)
		if code != 0 {
			return code
		}
		return delegate(ctx, deps, "infra", false, []string{"validate-profile", "--profile", profile}, nil, newCLITelemetryContext("profile-validate", "infra", true, deps, mustResolveTelemetry(telemetryOverride, deps)))
	default:
		fmt.Fprintf(deps.Stderr, "Unsupported profile command: %s\n", args[0])
		return 2
	}
}

func runInfraProfileCommand(ctx context.Context, command string, args []string, deps Dependencies, telemetryOverride *bool) int {
	profile, code := resolveProfileArg(args, deps)
	if code != 0 {
		return code
	}
	return delegate(ctx, deps, "infra", false, []string{command, "--profile", profile}, nil, newCLITelemetryContext(command, "infra", true, deps, mustResolveTelemetry(telemetryOverride, deps)))
}

func runDoctor(ctx context.Context, args []string, deps Dependencies, telemetryOverride *bool) int {
	if contains(args, "--core") || len(args) == 0 {
		if !platform.SupportsCoreHost(deps.GOOS, deps.GOARCH) {
			return unsupportedCoreHost(deps)
		}
			return delegate(ctx, deps, "core", true, []string{"preflight"}, nil, newCLITelemetryContext("doctor", "core", false, deps, mustResolveTelemetry(telemetryOverride, deps)))
		}
	return delegate(ctx, deps, "infra", false, append([]string{"doctor"}, args...), nil, newCLITelemetryContext("doctor", "infra", false, deps, mustResolveTelemetry(telemetryOverride, deps)))
}

func runInstall(ctx context.Context, args []string, deps Dependencies, telemetryOverride *bool) int {
	if !contains(args, "--core-only") {
		fmt.Fprintln(deps.Stderr, "Only `pk3s install --core-only` is implemented in this release.")
		return 2
	}
	if !platform.SupportsCoreHost(deps.GOOS, deps.GOARCH) {
		return unsupportedCoreHost(deps)
	}
	filtered := removeFlag(args, "--core-only")
	return delegate(ctx, deps, "core", true, append([]string{"bootstrap"}, filtered...), nil, newCLITelemetryContext("install", "core", false, deps, mustResolveTelemetry(telemetryOverride, deps)))
}

func runValidate(ctx context.Context, args []string, deps Dependencies, telemetryOverride *bool) int {
	if contains(args, "--core") || len(args) == 0 {
		if !platform.SupportsCoreHost(deps.GOOS, deps.GOARCH) {
			return unsupportedCoreHost(deps)
		}
		filtered := removeFlag(args, "--core")
			return delegate(ctx, deps, "core", true, append([]string{"validate"}, filtered...), nil, newCLITelemetryContext("validate", "core", false, deps, mustResolveTelemetry(telemetryOverride, deps)))
		}
	profile, code := resolveProfileArg(args, deps)
	if code != 0 {
		return code
	}
	return delegate(ctx, deps, "infra", false, []string{"validate", "--profile", profile}, nil, newCLITelemetryContext("validate", "infra", true, deps, mustResolveTelemetry(telemetryOverride, deps)))
}

func runBackup(ctx context.Context, args []string, deps Dependencies, telemetryOverride *bool) int {
	if !platform.SupportsCoreHost(deps.GOOS, deps.GOARCH) {
		return unsupportedCoreHost(deps)
	}
	filtered := removeFlag(args, "--core")
	return delegate(ctx, deps, "core", true, append([]string{"backup"}, filtered...), nil, newCLITelemetryContext("backup", "core", false, deps, mustResolveTelemetry(telemetryOverride, deps)))
}

func mustResolveTelemetry(override *bool, deps Dependencies) bool {
	enabled, err := resolveTelemetryEnabled(override)
	if err != nil {
		fmt.Fprintf(deps.Stderr, "could not resolve telemetry preference: %v\n", err)
		return false
	}
	return enabled
}

func unsupportedCoreHost(deps Dependencies) int {
	fmt.Fprintf(
		deps.Stderr,
		"Productive K3S Core host-installation commands are not supported on %s/%s. Check the supported platforms: %s\n",
		deps.GOOS,
		deps.GOARCH,
		platform.CoreSupportedPlatformsURL,
	)
	return 2
}

func resolveProfileArg(args []string, deps Dependencies) (string, int) {
	for i := 0; i < len(args); i++ {
		if args[i] == "--profile" {
			if i+1 >= len(args) {
				fmt.Fprintln(deps.Stderr, "missing value for --profile")
				return "", 2
			}
			return resolveProfilePath(args[i+1], deps)
		}
	}
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		return resolveProfilePath(args[0], deps)
	}
	fmt.Fprintln(deps.Stderr, "missing profile; use --profile <file>")
	return "", 2
}

func resolveProfilePath(value string, deps Dependencies) (string, int) {
	if isHTTPProfileURL(value) {
		path, err := downloadAndValidateProfile(ctxless(), value, deps)
		if err != nil {
			fmt.Fprintf(deps.Stderr, "invalid profile URL: %v\n", err)
			return "", 2
		}
		return path, 0
	}

	if statFile(value) {
		return value, 0
	}

	searchRoots := []string{
		filepath.Join(deps.WorkingDir, "tests", "fixtures", "profiles"),
	}

	switch requestedSourceMode() {
	case "local":
		localInfra, err := bundles.ResolveLocalSibling("infra", deps.WorkingDir)
		if err == nil {
			searchRoots = append(searchRoots, filepath.Join(localInfra.Root, "profiles"))
		}
	case "remote":
		remoteInfra, err := resolveBundle(ctxless(), deps, "infra")
		if err == nil {
			searchRoots = append(searchRoots, filepath.Join(remoteInfra.Root, "profiles"))
		}
	}

	names := []string{value}
	if !strings.HasSuffix(value, ".env") {
		names = append(names, value+".env")
	}
	for _, root := range searchRoots {
		for _, name := range names {
			candidate := filepath.Join(root, name)
			if statFile(candidate) {
				return candidate, 0
			}
			matches, _ := filepath.Glob(filepath.Join(root, "*", name))
			for _, match := range matches {
				if statFile(match) {
					return match, 0
				}
			}
		}
	}

	fmt.Fprintf(deps.Stderr, "profile not found: %s\n", value)
	return "", 2
}

func delegate(ctx context.Context, deps Dependencies, kind string, requiresCoreHost bool, args []string, env map[string]string, telemetry *cliTelemetryContext) int {
	if requiresCoreHost && !platform.SupportsCoreHost(deps.GOOS, deps.GOARCH) {
		return unsupportedCoreHost(deps)
	}
	ref, err := resolveBundle(ctx, deps, kind)
	if err != nil {
		fmt.Fprintf(deps.Stderr, "bundle resolution failed: %v\n", err)
		return 1
	}
	extraEnv := map[string]string{}
	for k, v := range env {
		extraEnv[k] = v
	}
	if telemetry != nil {
		for k, v := range telemetry.childEnv() {
			extraEnv[k] = v
		}
	}
	if ref.Source == "remote" {
		extraEnv["PRODUCTIVE_K3S_SOURCE"] = "remote"
		extraEnv["PRODUCTIVE_K3S_VERSION"] = bundles.DefaultReleaseManifest().CoreVersion
		if kind == "infra" {
			extraEnv["PRODUCTIVE_K3S_INFRA_VERSION"] = bundles.DefaultReleaseManifest().InfraVersion
		}
	}
	if telemetry != nil {
		telemetry.send(ctx, deps.HTTPClient, "cli.command.started", "started", deps)
	}
	if err := deps.Exec(ctx, Invocation{
		Path: ref.Entrypoint,
		Args: args,
		Dir:  ref.Root,
		Env:  mergedEnv(extraEnv),
	}); err != nil {
		if telemetry != nil {
			telemetry.send(ctx, deps.HTTPClient, "cli.command.completed", "failed", deps)
		}
		fmt.Fprintf(deps.Stderr, "command failed: %v\n", err)
		return 1
	}
	if telemetry != nil {
		telemetry.send(ctx, deps.HTTPClient, "cli.command.completed", "success", deps)
	}
	return 0
}

func resolveBundle(ctx context.Context, deps Dependencies, kind string) (bundles.BundleRef, error) {
	source := requestedSourceMode()
	switch source {
	case "local":
		return bundles.ResolveLocalSibling(kind, deps.WorkingDir)
	case "remote":
		spec, err := bundles.DefaultRemoteSpec(kind)
		if err != nil {
			return bundles.BundleRef{}, err
		}
		return bundles.ResolveRemoteBundle(ctx, deps.HTTPClient, deps.CacheDir, spec)
	default:
		return bundles.BundleRef{}, fmt.Errorf("unsupported PRODUCTIVE_K3S_SOURCE %q; use local or remote", source)
	}
}

func requestedSourceMode() string {
	source := strings.ToLower(strings.TrimSpace(os.Getenv("PRODUCTIVE_K3S_SOURCE")))
	if source == "" {
		return "remote"
	}
	return source
}

func osExec(ctx context.Context, invocation Invocation) error {
	cmd := exec.CommandContext(ctx, invocation.Path, invocation.Args...)
	cmd.Dir = invocation.Dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = invocation.Env
	return cmd.Run()
}

func runHelp(args []string, deps Dependencies) int {
	if len(args) == 0 {
		printRootHelp(deps.Stdout)
		return 0
	}

	topic := args[0]
	if helpText, ok := helpTopics()[topic]; ok {
		fmt.Fprint(deps.Stdout, helpText)
		return 0
	}

	fmt.Fprintf(deps.Stderr, "unknown help topic: %s\n", topic)
	fmt.Fprintln(deps.Stderr, "Run `pk3s help` to see the available commands.")
	return 2
}

func printRootHelp(w io.Writer) {
	fmt.Fprint(w, `Productive K3S CLI

Usage:
  pk3s <command> [flags]

  Commands:
  help
  version
  config
  doctor
  install
  validate
  backup
  bundle
  profile
  plan
  apply
  destroy
  status

Use:
  pk3s help profile
  pk3s help plan
  pk3s help bundle
`)
}

func helpTopics() map[string]string {
	return map[string]string{
		"profile": `Profile commands

Usage:
  pk3s profile <list|validate> [flags]

Subcommands:
  list
  validate --profile <file|name>

Examples:
  pk3s profile list
  pk3s profile validate --profile profiles/multipass/1-server-2-agents.env
  pk3s profile validate --profile basic.env

Notes:
  --profile accepts either an explicit file path or a short profile name.
  This validates the profile contract only, not the deployed scenario state.
`,
		"plan": `Plan command

Usage:
  pk3s plan --profile <file|name>

Examples:
  pk3s plan --profile profiles/aws-single-node/basic.env
  pk3s plan --profile basic.env

Notes:
  This delegates to Productive K3S Infra using the resolved profile.
`,
		"apply": `Apply command

Usage:
  pk3s apply --profile <file|name>

Examples:
  pk3s apply --profile profiles/multipass/1-server-2-agents.env
  pk3s apply --profile basic.env

Notes:
  This delegates to Productive K3S Infra using the resolved profile.
`,
		"destroy": `Destroy command

Usage:
  pk3s destroy --profile <file|name>

Examples:
  pk3s destroy --profile profiles/multipass/1-server-2-agents.env
  pk3s destroy --profile basic.env

Notes:
  This delegates to Productive K3S Infra using the resolved profile.
`,
		"status": `Status command

Usage:
  pk3s status --profile <file|name>

Examples:
  pk3s status --profile profiles/on-prem/basic.env
  pk3s status --profile basic.env

Notes:
  This delegates to Productive K3S Infra using the resolved profile.
`,
		"bundle": `Bundle commands

Usage:
  pk3s bundle <core|infra> info [--json]

Examples:
  pk3s bundle core info --json
  pk3s bundle infra info --json

Notes:
  Bundle metadata comes from the resolved local or remote bundle.
`,
		"install": `Install command

Usage:
  pk3s install --core-only [flags]

Examples:
  pk3s install --core-only

Notes:
  This delegates to Productive K3S Core bootstrap on a supported Linux host.
`,
		"doctor": `Doctor command

Usage:
  pk3s doctor [--core]

Examples:
  pk3s doctor --core

Notes:
  Core host checks require a supported Linux host.
`,
		"validate": `Validate command

Usage:
  pk3s validate --core
  pk3s validate --profile <file|name>

Examples:
  pk3s validate --core
  pk3s validate --profile profiles/aws-single-node/basic.env

Notes:
  Core validation is local-host oriented. Profile validation here delegates to Infra scenario validation and may require generated deployment state.
`,
		"backup": `Backup command

Usage:
  pk3s backup --core

Examples:
  pk3s backup --core

Notes:
  This delegates to Productive K3S Core backup on a supported Linux host.
`,
		"help": `Help command

Usage:
  pk3s help
  pk3s help <command>

Examples:
  pk3s help profile
  pk3s help plan
`,
		"version": `Version command

Usage:
  pk3s version
`,
	}
}

func contains(args []string, needle string) bool {
	for _, arg := range args {
		if arg == needle {
			return true
		}
	}
	return false
}

func removeFlag(args []string, needle string) []string {
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		if arg != needle {
			filtered = append(filtered, arg)
		}
	}
	return filtered
}

func mergedEnv(extra map[string]string) []string {
	envMap := map[string]string{}
	for _, item := range os.Environ() {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	for k, v := range extra {
		envMap[k] = v
	}
	result := make([]string, 0, len(envMap))
	for k, v := range envMap {
		result = append(result, k+"="+v)
	}
	return result
}

func statFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func ctxless() context.Context {
	return context.Background()
}

var errUsage = errors.New("usage")

func isHTTPProfileURL(value string) bool {
	return strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")
}

func downloadAndValidateProfile(ctx context.Context, rawURL string, deps Dependencies) (string, error) {
	sourceURL := normalizeProfileURL(rawURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := deps.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if err := validateProfileEnv(body); err != nil {
		return "", err
	}

	hash := sha256.Sum256([]byte(rawURL))
	filename := hex.EncodeToString(hash[:]) + ".env"
	targetDir := filepath.Join(deps.CacheDir, "profiles", "downloads")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", err
	}
	targetPath := filepath.Join(targetDir, filename)
	if err := os.WriteFile(targetPath, body, 0o600); err != nil {
		return "", err
	}
	return targetPath, nil
}

func normalizeProfileURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	if parsed.Host != "github.com" {
		return rawURL
	}
	parts := strings.Split(strings.TrimPrefix(parsed.Path, "/"), "/")
	if len(parts) < 5 || parts[2] != "blob" {
		return rawURL
	}
	owner := parts[0]
	repo := parts[1]
	ref := parts[3]
	path := strings.Join(parts[4:], "/")
	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", owner, repo, ref, path)
}

func validateProfileEnv(body []byte) error {
	if len(strings.TrimSpace(string(body))) == 0 {
		return fmt.Errorf("empty profile content")
	}

	required := map[string]bool{
		"PK3S_INFRA_PROFILE_NAME": false,
		"PK3S_INFRA_ENGINE":       false,
		"PK3S_INFRA_SCENARIO":     false,
	}
	values := map[string]string{}

	scanner := bufio.NewScanner(strings.NewReader(string(body)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid profile line: %s", line)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" {
			return fmt.Errorf("invalid profile line: %s", line)
		}
		values[key] = value
		if _, ok := required[key]; ok {
			required[key] = true
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	for key, present := range required {
		if !present {
			return fmt.Errorf("missing required variable: %s", key)
		}
	}
	switch values["PK3S_INFRA_ENGINE"] {
	case "opentofu", "ansible", "shell":
	default:
		return fmt.Errorf("unsupported PK3S_INFRA_ENGINE: %s", values["PK3S_INFRA_ENGINE"])
	}
	return nil
}
