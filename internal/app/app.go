package app

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"

	"github.com/jemacchi/productive-k3s-cli/internal/bundles"
	"github.com/jemacchi/productive-k3s-cli/internal/platform"
)

var Version = "1.0.0"

const defaultCatalogURL = "https://catalogs.productive-k3s.io/catalogs/index.yaml"

type Dependencies struct {
	Stdout     io.Writer
	Stderr     io.Writer
	GOOS       string
	GOARCH     string
	WorkingDir string
	CacheDir   string
	HTTPClient *http.Client
	Exec       func(context.Context, Invocation) error
	RunOutput  func(context.Context, Invocation) ([]byte, error)
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
		RunOutput:  osExecOutput,
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
	if deps.RunOutput == nil {
		deps.RunOutput = osExecOutput
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
	case "bom":
		return runBOM(ctx, args[1:], deps)
	case "config":
		return runConfig(args[1:], deps)
	case "bundle":
		return runBundle(ctx, args[1:], deps)
	case "profile":
		return runProfile(ctx, args[1:], deps, telemetryOverride)
	case "infra":
		return runInfra(ctx, args[1:], deps, telemetryOverride)
	case "addon":
		return runAddon(ctx, args[1:], deps, telemetryOverride)
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

func runBOM(ctx context.Context, args []string, deps Dependencies) int {
	if len(args) != 1 || args[0] != "--json" {
		fmt.Fprintln(deps.Stderr, "Usage: pk3s bom --json")
		return 2
	}

	coreBOM, err := resolveDelegatedBOM(ctx, deps, "core")
	if err != nil {
		fmt.Fprintf(deps.Stderr, "could not resolve core BOM: %v\n", err)
		return 1
	}
	infraBOM, err := resolveDelegatedBOM(ctx, deps, "infra")
	if err != nil {
		fmt.Fprintf(deps.Stderr, "could not resolve infra BOM: %v\n", err)
		return 1
	}

	configuredCatalogs := catalogURLsFromEnv()

	bom := map[string]any{
		"schema_version": "1",
		"bom_type":       "productive-k3s-cli-bom/v1",
		"cli": map[string]any{
			"name":       "pk3s",
			"version":    Version,
			"entrypoint": "pk3s",
		},
		"implementation": map[string]any{
			"language":   "go",
			"go_version": runtime.Version(),
			"goos":       deps.GOOS,
			"goarch":     deps.GOARCH,
		},
		"catalog": map[string]any{
			"default_url":    defaultCatalogURL,
			"configured_urls": configuredCatalogs,
		},
		"bundles": map[string]any{
			"source_mode": requestedSourceMode(),
			"core":        coreBOM,
			"infra":       infraBOM,
		},
	}

	encoded, err := json.MarshalIndent(bom, "", "  ")
	if err != nil {
		fmt.Fprintf(deps.Stderr, "could not encode BOM: %v\n", err)
		return 1
	}
	fmt.Fprintln(deps.Stdout, string(encoded))
	return 0
}

func runProfile(ctx context.Context, args []string, deps Dependencies, telemetryOverride *bool) int {
	if len(args) == 0 {
		fmt.Fprintln(deps.Stderr, "Usage: pk3s profile <list|show|validate|install> [profile]")
		return 2
	}
	switch args[0] {
	case "list":
		return runProfileList(ctx, deps)
	case "show":
		if len(args) < 2 || strings.TrimSpace(args[1]) == "" {
			fmt.Fprintln(deps.Stderr, "Usage: pk3s profile show <name>")
			return 2
		}
		entry, err := findCatalogEntry(ctx, deps, "profile", args[1])
		if err != nil {
			fmt.Fprintln(deps.Stderr, err.Error())
			return 2
		}
		renderProfileCatalogEntry(deps.Stdout, entry)
		return 0
	case "validate":
		if value, found := firstNonFlagArg(args[1:]); found && !contains(args[1:], "--tgz") && !contains(args[1:], "--profile") && !isHTTPAssetURL(value) && !statFile(value) && !looksLikeLocalTGZPath(value) {
			if profile, ok := lookupProfilePath(value, deps); ok {
				return delegate(ctx, deps, "infra", false, []string{"validate-profile", "--profile", profile}, nil, maybeCLITelemetryContext("profile-validate", "infra", true, deps, telemetryOverride))
			}
			tgz, err := resolveCatalogTGZ(ctx, deps, "profile", value)
			if err == nil {
				return delegate(ctx, deps, "infra", false, []string{"profile", "validate", "--tgz", tgz}, nil, maybeCLITelemetryContext("profile-validate", "infra", false, deps, telemetryOverride))
			}
			fmt.Fprintln(deps.Stderr, err.Error())
			return 2
		}
		if tgz, code, ok := resolveTGZArg(args[1:], deps); ok {
			if code != 0 {
				return code
			}
			return delegate(ctx, deps, "infra", false, []string{"profile", "validate", "--tgz", tgz}, nil, maybeCLITelemetryContext("profile-validate", "infra", false, deps, telemetryOverride))
		}
		profile, code := resolveProfileArg(args[1:], deps)
		if code != 0 {
			return code
		}
		return delegate(ctx, deps, "infra", false, []string{"validate-profile", "--profile", profile}, nil, maybeCLITelemetryContext("profile-validate", "infra", true, deps, telemetryOverride))
	case "install":
		filteredArgs, env, code := parsePackageEnvFile(args[1:], deps.Stderr)
		if code != 0 {
			return code
		}
		if value, found := firstNonFlagArg(filteredArgs); found && !contains(filteredArgs, "--tgz") && !isHTTPAssetURL(value) && !statFile(value) && !looksLikeLocalTGZPath(value) {
			entry, err := findCatalogEntry(ctx, deps, "profile", value)
			if err != nil {
				fmt.Fprintln(deps.Stderr, err.Error())
				return 2
			}
			if exitCode := profileCatalogInstallPreflight(entry, env, deps.Stderr); exitCode != 0 {
				return exitCode
			}
			resolved, err := downloadCatalogEntryTGZ(ctx, deps, entry)
			if err != nil {
				fmt.Fprintln(deps.Stderr, err.Error())
				return 2
			}
			return delegate(ctx, deps, "infra", false, []string{"profile", "install", "--tgz", resolved}, env, maybeCLITelemetryContext("profile-install", "infra", false, deps, telemetryOverride))
		}
		tgz, code, ok := resolveTGZArg(filteredArgs, deps)
		if !ok {
			fmt.Fprintln(deps.Stderr, "missing tgz; use --tgz <file|url>")
			return 2
		}
		if code != 0 {
			return code
		}
		return delegate(ctx, deps, "infra", false, []string{"profile", "install", "--tgz", tgz}, env, maybeCLITelemetryContext("profile-install", "infra", false, deps, telemetryOverride))
	default:
		fmt.Fprintf(deps.Stderr, "Unsupported profile command: %s\n", args[0])
		return 2
	}
}

func runInfra(ctx context.Context, args []string, deps Dependencies, telemetryOverride *bool) int {
	if len(args) == 0 {
		fmt.Fprintln(deps.Stderr, "Usage: pk3s infra <install|plan|apply|destroy|status> --tgz <file|url>")
		return 2
	}
	switch args[0] {
	case "install", "plan", "apply", "destroy", "status":
		filteredArgs, env, code := parsePackageEnvFile(args[1:], deps.Stderr)
		if code != 0 {
			return code
		}
		if value, found := firstNonFlagArg(filteredArgs); found && !contains(filteredArgs, "--tgz") && !isHTTPAssetURL(value) && !statFile(value) && !looksLikeLocalTGZPath(value) {
			entry, err := findCatalogEntry(ctx, deps, "profile", value)
			if err != nil {
				fmt.Fprintln(deps.Stderr, err.Error())
				return 2
			}
			if exitCode := profileCatalogInstallPreflight(entry, env, deps.Stderr); exitCode != 0 {
				return exitCode
			}
			resolved, err := downloadCatalogEntryTGZ(ctx, deps, entry)
			if err != nil {
				fmt.Fprintln(deps.Stderr, err.Error())
				return 2
			}
			return delegate(ctx, deps, "infra", false, []string{"profile", args[0], "--tgz", resolved}, env, maybeCLITelemetryContext("infra-"+args[0], "infra", false, deps, telemetryOverride))
		}
		tgz, code, ok := resolveTGZArg(filteredArgs, deps)
		if !ok {
			fmt.Fprintln(deps.Stderr, "missing tgz; use --tgz <file|url>")
			return 2
		}
		if code != 0 {
			return code
		}
		return delegate(ctx, deps, "infra", false, []string{"profile", args[0], "--tgz", tgz}, env, maybeCLITelemetryContext("infra-"+args[0], "infra", false, deps, telemetryOverride))
	default:
		fmt.Fprintf(deps.Stderr, "Unsupported infra command: %s\n", args[0])
		return 2
	}
}

func runAddon(ctx context.Context, args []string, deps Dependencies, telemetryOverride *bool) int {
	if len(args) == 0 {
		fmt.Fprintln(deps.Stderr, "Usage: pk3s addon <list|install|validate> [flags]")
		return 2
	}
	switch args[0] {
	case "list":
		return runAddonList(ctx, deps)
	case "validate":
		if resolved, handled, exitCode := maybeResolveCatalogName(ctx, deps, "addon", args[1:], "", 0, false); handled {
			if exitCode != 0 {
				return exitCode
			}
			return delegate(ctx, deps, "core", true, []string{"addon", "validate", "--tgz", resolved}, nil, maybeCLITelemetryContext("addon-validate", "core", false, deps, telemetryOverride))
		}
		tgz, code, ok := resolveTGZArg(args[1:], deps)
		if !ok {
			fmt.Fprintln(deps.Stderr, "missing tgz; use --tgz <file|url>")
			return 2
		}
		if code != 0 {
			return code
		}
		return delegate(ctx, deps, "core", true, []string{"addon", "validate", "--tgz", tgz}, nil, maybeCLITelemetryContext("addon-validate", "core", false, deps, telemetryOverride))
	case "install":
		target, code := parseAddonInstallTarget(args[1:], deps.Stderr)
		if code != 0 {
			return code
		}
		env, cleanup, err := resolveAddonInstallEnv(ctx, deps, target)
		if err != nil {
			fmt.Fprintln(deps.Stderr, err.Error())
			return 2
		}
		defer cleanup()
		if resolved, handled, exitCode := maybeResolveCatalogName(ctx, deps, "addon", target.filteredArgs, "", 0, false); handled {
			if exitCode != 0 {
				return exitCode
			}
			coreArgs := []string{"addon", "install", "--tgz", resolved}
			if target.publicHost != "" {
				coreArgs = append(coreArgs, "--public-host", target.publicHost)
			}
			return delegate(ctx, deps, "core", true, coreArgs, env, maybeCLITelemetryContext("addon-install", "core", false, deps, telemetryOverride))
		}
		tgz, code, ok := resolveTGZArg(target.filteredArgs, deps)
		if !ok {
			fmt.Fprintln(deps.Stderr, "missing tgz; use --tgz <file|url>")
			return 2
		}
		if code != 0 {
			return code
		}
		coreArgs := []string{"addon", "install", "--tgz", tgz}
		if target.publicHost != "" {
			coreArgs = append(coreArgs, "--public-host", target.publicHost)
		}
		return delegate(ctx, deps, "core", true, coreArgs, env, maybeCLITelemetryContext("addon-install", "core", false, deps, telemetryOverride))
	default:
		fmt.Fprintf(deps.Stderr, "Unsupported addon command: %s\n", args[0])
		return 2
	}
}

func runAddonList(ctx context.Context, deps Dependencies) int {
	entries, _, err := loadCatalogEntriesFromConfiguredSources(ctx, deps)
	if err != nil {
		fmt.Fprintf(deps.Stderr, "%v\n", err)
		return 1
	}
	filtered := make([]catalogEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.Kind == "addon" {
			filtered = append(filtered, entry)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Name < filtered[j].Name
	})
	for _, entry := range filtered {
		line := entry.Name
		if entry.Version != "" {
			line += "\t" + entry.Version
		}
		if entry.Category != "" {
			line += "\t" + entry.Category
		}
		fmt.Fprintln(deps.Stdout, line)
	}
	return 0
}

func runProfileList(ctx context.Context, deps Dependencies) int {
	entries, _, err := loadCatalogEntriesFromConfiguredSources(ctx, deps)
	if err != nil {
		fmt.Fprintf(deps.Stderr, "%v\n", err)
		return 1
	}
	filtered := make([]catalogEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.Kind == "profile" {
			filtered = append(filtered, entry)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Name < filtered[j].Name
	})
	for _, entry := range filtered {
		line := entry.Name
		if entry.Version != "" {
			line += "\t" + entry.Version
		}
		if entry.Category != "" {
			line += "\t" + entry.Category
		}
		if entry.Install.RequiresLocalOverrides {
			line += "\tneeds-env"
		}
		fmt.Fprintln(deps.Stdout, line)
	}
	return 0
}

type addonInstallTarget struct {
	filteredArgs   []string
	kubeconfig     string
	clusterContext string
	profileName    string
	publicHost     string
}

func parseAddonInstallTarget(args []string, stderr io.Writer) (addonInstallTarget, int) {
	filtered := make([]string, 0, len(args))
	target := addonInstallTarget{}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--kubeconfig":
			if i+1 >= len(args) || strings.TrimSpace(args[i+1]) == "" {
				fmt.Fprintln(stderr, "missing kubeconfig path; use --kubeconfig <file>")
				return addonInstallTarget{}, 2
			}
			target.kubeconfig = args[i+1]
			i++
		case "--cluster-context":
			if i+1 >= len(args) || strings.TrimSpace(args[i+1]) == "" {
				fmt.Fprintln(stderr, "missing cluster context; use --cluster-context <name>")
				return addonInstallTarget{}, 2
			}
			target.clusterContext = args[i+1]
			i++
		case "--profile":
			if i+1 >= len(args) || strings.TrimSpace(args[i+1]) == "" {
				fmt.Fprintln(stderr, "missing profile name; use --profile <name>")
				return addonInstallTarget{}, 2
			}
			target.profileName = args[i+1]
			i++
		case "--public-host":
			if i+1 >= len(args) || strings.TrimSpace(args[i+1]) == "" {
				fmt.Fprintln(stderr, "missing public host; use --public-host <fqdn>")
				return addonInstallTarget{}, 2
			}
			target.publicHost = args[i+1]
			i++
		default:
			filtered = append(filtered, args[i])
		}
	}

	if strings.TrimSpace(target.kubeconfig) == "" && strings.TrimSpace(target.clusterContext) == "" && strings.TrimSpace(target.profileName) == "" {
		fmt.Fprintln(stderr, "addon install requires an explicit target; use --profile <name>, --kubeconfig <file> or --cluster-context <name>")
		return addonInstallTarget{}, 2
	}
	target.filteredArgs = filtered
	return target, 0
}

func parsePackageEnvFile(args []string, stderr io.Writer) ([]string, map[string]string, int) {
	filtered := make([]string, 0, len(args))
	env := map[string]string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--env-file":
			if i+1 >= len(args) || strings.TrimSpace(args[i+1]) == "" {
				fmt.Fprintln(stderr, "missing env file path; use --env-file <file>")
				return nil, nil, 2
			}
			env["PK3S_PROFILE_OVERRIDE_ENV_FILE"] = args[i+1]
			i++
		default:
			filtered = append(filtered, args[i])
		}
	}
	return filtered, env, 0
}

func runInfraProfileCommand(ctx context.Context, command string, args []string, deps Dependencies, telemetryOverride *bool) int {
	profile, code := resolveProfileArg(args, deps)
	if code != 0 {
		return code
	}
	return delegate(ctx, deps, "infra", false, []string{command, "--profile", profile}, nil, maybeCLITelemetryContext(command, "infra", true, deps, telemetryOverride))
}

func runDoctor(ctx context.Context, args []string, deps Dependencies, telemetryOverride *bool) int {
	if contains(args, "--core") || len(args) == 0 {
		if !platform.SupportsCoreHost(deps.GOOS, deps.GOARCH) {
			return unsupportedCoreHost(deps)
		}
		return delegate(ctx, deps, "core", true, []string{"preflight"}, nil, maybeCLITelemetryContext("doctor", "core", false, deps, telemetryOverride))
	}
	return delegate(ctx, deps, "infra", false, append([]string{"doctor"}, args...), nil, maybeCLITelemetryContext("doctor", "infra", false, deps, telemetryOverride))
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
	return delegate(ctx, deps, "core", true, append([]string{"apply"}, filtered...), nil, maybeCLITelemetryContext("install", "core", false, deps, telemetryOverride))
}

func runValidate(ctx context.Context, args []string, deps Dependencies, telemetryOverride *bool) int {
	if contains(args, "--core") || len(args) == 0 {
		if !platform.SupportsCoreHost(deps.GOOS, deps.GOARCH) {
			return unsupportedCoreHost(deps)
		}
		filtered := removeFlag(args, "--core")
		return delegate(ctx, deps, "core", true, append([]string{"validate"}, filtered...), nil, maybeCLITelemetryContext("validate", "core", false, deps, telemetryOverride))
	}
	profile, code := resolveProfileArg(args, deps)
	if code != 0 {
		return code
	}
	return delegate(ctx, deps, "infra", false, []string{"validate", "--profile", profile}, nil, maybeCLITelemetryContext("validate", "infra", true, deps, telemetryOverride))
}

func runBackup(ctx context.Context, args []string, deps Dependencies, telemetryOverride *bool) int {
	if !platform.SupportsCoreHost(deps.GOOS, deps.GOARCH) {
		return unsupportedCoreHost(deps)
	}
	filtered := removeFlag(args, "--core")
	return delegate(ctx, deps, "core", true, append([]string{"backup"}, filtered...), nil, maybeCLITelemetryContext("backup", "core", false, deps, telemetryOverride))
}

func maybeCLITelemetryContext(commandName, bundleKind string, legacyProfile bool, deps Dependencies, override *bool) *cliTelemetryContext {
	if !cliCommandEmitsTelemetry(commandName) {
		return nil
	}
	return newCLITelemetryContext(commandName, bundleKind, legacyProfile, deps, mustResolveTelemetry(override, deps))
}

func cliCommandEmitsTelemetry(commandName string) bool {
	switch commandName {
	case "install", "profile-install", "infra-install", "infra-apply", "infra-destroy", "apply", "destroy", "addon-install":
		return true
	default:
		return false
	}
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

func resolveTGZArg(args []string, deps Dependencies) (string, int, bool) {
	for i := 0; i < len(args); i++ {
		if args[i] == "--tgz" {
			if i+1 >= len(args) {
				fmt.Fprintln(deps.Stderr, "missing value for --tgz")
				return "", 2, true
			}
			return resolveTGZPath(args[i+1], deps)
		}
	}
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		return resolveTGZPath(args[0], deps)
	}
	return "", 0, false
}

func resolveTGZPath(value string, deps Dependencies) (string, int, bool) {
	if isHTTPAssetURL(value) {
		path, err := downloadAsset(ctxless(), value, deps, "tgz")
		if err != nil {
			fmt.Fprintf(deps.Stderr, "invalid tgz URL: %v\n", err)
			return "", 2, true
		}
		return path, 0, true
	}
	if statFile(value) {
		return value, 0, true
	}
	fmt.Fprintf(deps.Stderr, "tgz not found: %s\n", value)
	return "", 2, true
}

func maybeResolveCatalogName(ctx context.Context, deps Dependencies, kind string, args []string, tgz string, code int, ok bool) (string, bool, int) {
	value, found := firstNonFlagArg(args)
	if !found {
		return "", false, 0
	}
	if contains(args, "--tgz") || isHTTPAssetURL(value) || statFile(value) || looksLikeLocalTGZPath(value) {
		return "", false, 0
	}
	resolved, err := resolveCatalogTGZ(ctx, deps, kind, value)
	if err != nil {
		fmt.Fprintln(deps.Stderr, err.Error())
		return "", true, 2
	}
	return resolved, true, 0
}

func looksLikeLocalTGZPath(value string) bool {
	return strings.Contains(value, "/") || strings.HasSuffix(value, ".tgz")
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
	if profile, ok := lookupProfilePath(value, deps); ok {
		return profile, 0
	}
	fmt.Fprintf(deps.Stderr, "profile not found: %s\n", value)
	return "", 2
}

func lookupProfilePath(value string, deps Dependencies) (string, bool) {
	if statFile(value) {
		return value, true
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
				return candidate, true
			}
			matches, _ := filepath.Glob(filepath.Join(root, "*", name))
			for _, match := range matches {
				if statFile(match) {
					return match, true
				}
			}
		}
	}
	return "", false
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
		return bundles.ResolveLocalBundle(kind, deps.WorkingDir, deps.CacheDir)
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

func resolveDelegatedBOM(ctx context.Context, deps Dependencies, kind string) (map[string]any, error) {
	ref, err := resolveBundle(ctx, deps, kind)
	if err != nil {
		return nil, err
	}

	extraEnv := map[string]string{}
	if ref.Source == "remote" {
		extraEnv["PRODUCTIVE_K3S_SOURCE"] = "remote"
		extraEnv["PRODUCTIVE_K3S_VERSION"] = bundles.DefaultReleaseManifest().CoreVersion
		if kind == "infra" {
			extraEnv["PRODUCTIVE_K3S_INFRA_VERSION"] = bundles.DefaultReleaseManifest().InfraVersion
		}
	}

	out, err := deps.RunOutput(ctx, Invocation{
		Path: ref.Entrypoint,
		Args: []string{"bom", "--json"},
		Dir:  ref.Root,
		Env:  mergedEnv(extraEnv),
	})
	if err == nil {
		var resolved map[string]any
		if err := json.Unmarshal(out, &resolved); err != nil {
			return nil, err
		}
		resolved["resolution"] = map[string]any{
			"source":     ref.Source,
			"root":       ref.Root,
			"entrypoint": ref.Entrypoint,
		}
		return resolved, nil
	}

	bundleInfoOut, fallbackErr := deps.RunOutput(ctx, Invocation{
		Path: ref.Entrypoint,
		Args: []string{"bundle", "info", "--json"},
		Dir:  ref.Root,
		Env:  mergedEnv(extraEnv),
	})
	if fallbackErr != nil {
		return nil, err
	}

	var bundleInfo map[string]any
	if err := json.Unmarshal(bundleInfoOut, &bundleInfo); err != nil {
		return nil, err
	}

	resolved := map[string]any{
		"schema_version": "1",
		"bom_type":       "productive-k3s-cli-bom/v1",
		"cli": map[string]any{
			"name":       bundleInfo["bundle_name"],
			"version":    bundleInfo["bundle_version"],
			"entrypoint": bundleInfo["cli_entrypoint"],
		},
		"implementation": map[string]any{
			"language": "unknown",
		},
		"bundle": bundleInfo,
		"fallback": map[string]any{
			"mode":   "bundle-info",
			"reason": "delegated bundle does not expose bom --json",
		},
		"resolution": map[string]any{
			"source":     ref.Source,
			"root":       ref.Root,
			"entrypoint": ref.Entrypoint,
		},
	}
	resolved["resolution"] = map[string]any{
		"source":     ref.Source,
		"root":       ref.Root,
		"entrypoint": ref.Entrypoint,
	}
	return resolved, nil
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

func osExecOutput(ctx context.Context, invocation Invocation) ([]byte, error) {
	cmd := exec.CommandContext(ctx, invocation.Path, invocation.Args...)
	cmd.Dir = invocation.Dir
	cmd.Env = invocation.Env
	return cmd.Output()
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
  bom
  config
  doctor
  install
  validate
  backup
  bundle
  profile
  infra
  addon
  plan
  apply
  destroy
  status

Use:
  pk3s help profile
  pk3s help infra
  pk3s help addon
  pk3s help plan
  pk3s help bundle
`)
}

func helpTopics() map[string]string {
	return map[string]string{
		"profile": `Profile commands

Usage:
  pk3s profile <list|show|validate|install> [flags]

Subcommands:
  list
  show <name>
  validate --tgz <file|url>
  install --tgz <file|url> [--env-file <file>]

Examples:
  pk3s profile list
  pk3s profile show aws-single-node-basic
  pk3s profile validate --tgz ./multipass-profile.tgz
  pk3s profile install --tgz ./multipass-profile.tgz
  pk3s profile install aws-single-node-basic --env-file ./aws.env

Notes:
  Public profile package workflows are package-oriented and delegate to the Infra runtime.
  The embedded profile.env inside the tgz is treated as base/defaults.
  Catalog-backed profiles may declare local-override inputs; pk3s will require --env-file before runtime for those entries.
  Use --env-file for installation-specific values, especially for cloud and on-prem profiles.
`,
		"infra": `Infra commands

Usage:
  pk3s infra <install|plan|apply|destroy|status> --tgz <file|url> [--env-file <file>]

Examples:
  pk3s infra install --tgz ./aws-single-node-basic.tgz
  pk3s infra status --tgz ./aws-single-node-basic.tgz
  pk3s infra apply aws-single-node-basic --env-file ./aws.env

Notes:
  Public Infra installation is package-oriented and delegates to Productive K3S Infra.
  The embedded profile.env inside the tgz is treated as base/defaults.
  Catalog-backed profiles may declare local-override inputs; pk3s will require --env-file before runtime for those entries.
  Use --env-file for installation-specific values, especially for cloud and on-prem profiles.
`,
		"addon": `Addon commands

Usage:
  pk3s addon list
  pk3s addon validate --tgz <file|url>
  pk3s addon install --tgz <file|url> [--public-host <fqdn>] (--kubeconfig <file> | --cluster-context <name>)

Examples:
  pk3s addon list
  pk3s addon validate --tgz ./longhorn-addon.tgz
  pk3s addon install --tgz ./longhorn-addon.tgz --cluster-context default
  pk3s addon install nginx --kubeconfig ~/.kube/config
  pk3s addon install nginx --profile multipass-1-server-2-agents --public-host nginx-01.k3s.lab.internal

Notes:
  Public add-on workflows are package-oriented and delegate to Productive K3S Core.
  Add-on installation requires an explicit Kubernetes target and does not implicitly use the host cluster.
  Use --profile after the target profile has been installed or inspected so pk3s can derive the cluster access details.
  --public-host only works for add-ons that declare the basic Core-managed ingress contract.
  Advanced ingress customization remains an add-on responsibility, not a generic Core feature.
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
  This delegates to Productive K3S Core apply on a supported Linux host.
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
		"bom": `BOM command

Usage:
  pk3s bom --json

Examples:
  pk3s bom --json

Notes:
  Emits a recursive bill of materials for pk3s, including bundle resolution details and delegated core/infra BOM metadata.
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

func isHTTPAssetURL(value string) bool {
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

func downloadAsset(ctx context.Context, rawURL string, deps Dependencies, ext string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
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

	hash := sha256.Sum256([]byte(rawURL))
	filename := hex.EncodeToString(hash[:]) + "." + ext
	targetDir := filepath.Join(deps.CacheDir, ext, "downloads")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", err
	}
	targetPath := filepath.Join(targetDir, filename)
	if err := os.WriteFile(targetPath, body, 0o600); err != nil {
		return "", err
	}
	return targetPath, nil
}

type catalogEntry struct {
	Kind         string
	ID           string
	Name         string
	MetadataName string
	Category     string
	Version      string
	Description  string
	Visibility   string
	ArtifactType string
	ArtifactURL  string
	Install      catalogInstallSummary
}

type catalogInstallSummary struct {
	RequiresLocalOverrides bool
	Inputs                 []catalogInput
}

type catalogInput struct {
	Name        string
	Required    bool
	Sensitive   bool
	Source      string
	Description string
}

func resolveCatalogTGZ(ctx context.Context, deps Dependencies, kind string, name string) (string, error) {
	entry, err := findCatalogEntry(ctx, deps, kind, name)
	if err != nil {
		return "", err
	}
	return downloadCatalogEntryTGZ(ctx, deps, entry)
}

func loadCatalogEntriesFromConfiguredSources(ctx context.Context, deps Dependencies) ([]catalogEntry, string, error) {
	urls := catalogURLsFromEnv()
	if len(urls) == 0 {
		return nil, "", fmt.Errorf("no catalog sources configured")
	}
	var lastErr error
	for _, source := range urls {
		entries, err := loadCatalogEntries(ctx, deps, source)
		if err != nil {
			lastErr = err
			continue
		}
		return entries, source, nil
	}
	if lastErr != nil {
		return nil, "", fmt.Errorf("could not load any configured catalog source: %v", lastErr)
	}
	return nil, "", fmt.Errorf("no catalog sources configured")
}

type profileState struct {
	ServerURL string `json:"server_url"`
	SSH       struct {
		User    string `json:"user"`
		Port    int    `json:"port"`
		KeyPath string `json:"key_path"`
	} `json:"ssh"`
	Server struct {
		IPv4     string `json:"ipv4"`
		PublicIP string `json:"public_ip"`
		Name     string `json:"name"`
	} `json:"server"`
}

func resolveAddonInstallEnv(ctx context.Context, deps Dependencies, target addonInstallTarget) (map[string]string, func(), error) {
	if target.profileName != "" {
		kubeconfig, cleanup, err := kubeconfigForProfileState(ctx, deps, target.profileName)
		if err != nil {
			return nil, func() {}, err
		}
		return map[string]string{"KUBECONFIG": kubeconfig}, cleanup, nil
	}

	env := map[string]string{}
	if target.kubeconfig != "" {
		env["KUBECONFIG"] = target.kubeconfig
	}
	if target.clusterContext != "" {
		env["PK3S_KUBE_CONTEXT"] = target.clusterContext
	}
	return env, func() {}, nil
}

func kubeconfigForProfileState(ctx context.Context, deps Dependencies, profileName string) (string, func(), error) {
	state, err := loadProfileState(profileName)
	if err != nil {
		return "", func() {}, err
	}
	content, err := fetchRemoteKubeconfig(ctx, deps, state)
	if err != nil {
		return "", func() {}, err
	}
	content = rewriteKubeconfigServer(content, state.ServerURL)
	tmpFile, err := os.CreateTemp("", "pk3s-addon-kubeconfig-*.yaml")
	if err != nil {
		return "", func() {}, err
	}
	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
		return "", func() {}, err
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpFile.Name())
		return "", func() {}, err
	}
	return tmpFile.Name(), func() { _ = os.Remove(tmpFile.Name()) }, nil
}

func loadProfileState(profileName string) (profileState, error) {
	path := profileStatePath(profileName)
	body, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return profileState{}, fmt.Errorf("profile state not found for %q; run `pk3s infra install %s [--env-file <file>]` or `pk3s infra status %s [--env-file <file>]` first", profileName, profileName, profileName)
		}
		return profileState{}, err
	}
	var state profileState
	if err := json.Unmarshal(body, &state); err != nil {
		return profileState{}, fmt.Errorf("could not parse profile state %q: %w", path, err)
	}
	if strings.TrimSpace(state.ServerURL) == "" {
		return profileState{}, fmt.Errorf("profile state for %q is missing server_url", profileName)
	}
	return state, nil
}

func profileStatePath(profileName string) string {
	if dir := strings.TrimSpace(os.Getenv("PK3S_PROFILE_STATE_DIR")); dir != "" {
		return filepath.Join(dir, profileName+".json")
	}
	if dir := strings.TrimSpace(os.Getenv("XDG_CACHE_HOME")); dir != "" {
		return filepath.Join(dir, "pk3s", "profiles", profileName+".json")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "pk3s", "profiles", profileName+".json")
}

func fetchRemoteKubeconfig(ctx context.Context, deps Dependencies, state profileState) (string, error) {
	host := strings.TrimSpace(state.Server.PublicIP)
	if host == "" {
		host = strings.TrimSpace(state.Server.IPv4)
	}
	if host == "" {
		return "", fmt.Errorf("profile state is missing a reachable server IP")
	}
	if strings.TrimSpace(state.SSH.User) == "" {
		return "", fmt.Errorf("profile state is missing ssh.user")
	}
	if strings.TrimSpace(state.SSH.KeyPath) == "" {
		return "", fmt.Errorf("profile state is missing ssh.key_path")
	}
	port := "22"
	if state.SSH.Port > 0 {
		port = fmt.Sprintf("%d", state.SSH.Port)
	}
	out, err := deps.RunOutput(ctx, Invocation{
		Path: "ssh",
		Args: []string{
			"-i", state.SSH.KeyPath,
			"-p", port,
			"-o", "StrictHostKeyChecking=no",
			"-o", "UserKnownHostsFile=/dev/null",
			fmt.Sprintf("%s@%s", state.SSH.User, host),
			"sudo", "cat", "/etc/rancher/k3s/k3s.yaml",
		},
	})
	if err != nil {
		return "", fmt.Errorf("could not fetch kubeconfig for profile target: %w", err)
	}
	return string(out), nil
}

func rewriteKubeconfigServer(content string, serverURL string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "server: ") {
			prefix := line[:len(line)-len(strings.TrimLeft(line, " "))]
			lines[i] = prefix + "server: " + serverURL
			break
		}
	}
	return strings.Join(lines, "\n")
}

func catalogURLsFromEnv() []string {
	raw := strings.TrimSpace(os.Getenv("PK3S_CATALOG_URLS"))
	if raw == "" {
		return []string{defaultCatalogURL}
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func loadCatalogEntries(ctx context.Context, deps Dependencies, source string) ([]catalogEntry, error) {
	body, err := readCatalogSource(ctx, deps, source)
	if err != nil {
		return nil, err
	}
	return parseCatalogEntries(body), nil
}

func findCatalogEntry(ctx context.Context, deps Dependencies, kind string, name string) (catalogEntry, error) {
	entries, _, err := loadCatalogEntriesFromConfiguredSources(ctx, deps)
	if err != nil {
		return catalogEntry{}, err
	}
	for _, entry := range entries {
		if entry.Kind != kind {
			continue
		}
		if catalogEntryMatchesName(entry, name) {
			return entry, nil
		}
	}
	return catalogEntry{}, fmt.Errorf("catalog entry not found for %s %q", kind, name)
}

func downloadCatalogEntryTGZ(ctx context.Context, deps Dependencies, entry catalogEntry) (string, error) {
	name := entry.Name
	if name == "" {
		name = entry.ID
	}
	if entry.ArtifactType != "tgz" {
		return "", fmt.Errorf("catalog entry %q does not expose a tgz artifact", name)
	}
	if strings.TrimSpace(entry.ArtifactURL) == "" {
		return "", fmt.Errorf("catalog entry %q does not expose a downloadable tgz URL", name)
	}
	return downloadAsset(ctx, entry.ArtifactURL, deps, "tgz")
}

func profileCatalogInstallPreflight(entry catalogEntry, env map[string]string, stderr io.Writer) int {
	if !entry.Install.RequiresLocalOverrides {
		return 0
	}
	if strings.TrimSpace(env["PK3S_PROFILE_OVERRIDE_ENV_FILE"]) != "" {
		return 0
	}
	name := entry.Name
	if name == "" {
		name = entry.ID
	}
	fmt.Fprintf(stderr, "profile %q requires installation-specific local overrides; use --env-file <file>\n", name)
	localInputs := make([]string, 0, len(entry.Install.Inputs))
	for _, input := range entry.Install.Inputs {
		if input.Source == "local-override" {
			localInputs = append(localInputs, input.Name)
		}
	}
	if len(localInputs) > 0 {
		fmt.Fprintf(stderr, "Required local inputs: %s\n", strings.Join(localInputs, ", "))
	}
	return 2
}

func renderProfileCatalogEntry(w io.Writer, entry catalogEntry) {
	fmt.Fprintf(w, "Name: %s\n", entry.Name)
	fmt.Fprintf(w, "Kind: %s\n", entry.Kind)
	if entry.Version != "" {
		fmt.Fprintf(w, "Version: %s\n", entry.Version)
	}
	if entry.Category != "" {
		fmt.Fprintf(w, "Category: %s\n", entry.Category)
	}
	if entry.Description != "" {
		fmt.Fprintf(w, "Description: %s\n", entry.Description)
	}
	fmt.Fprintf(w, "Requires local overrides: %t\n", entry.Install.RequiresLocalOverrides)
	if len(entry.Install.Inputs) == 0 {
		return
	}
	fmt.Fprintln(w, "Install inputs:")
	for _, input := range entry.Install.Inputs {
		line := "  - " + input.Name
		if input.Source != "" {
			line += " [" + input.Source + "]"
		}
		if input.Required {
			line += " required"
		}
		if input.Sensitive {
			line += " sensitive"
		}
		if input.Description != "" {
			line += " - " + input.Description
		}
		fmt.Fprintln(w, line)
	}
}

func readCatalogSource(ctx context.Context, deps Dependencies, source string) ([]byte, error) {
	switch {
	case strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://"):
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
		if err != nil {
			return nil, err
		}
		resp, err := deps.HTTPClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("catalog download failed: %s", resp.Status)
		}
		return io.ReadAll(resp.Body)
	case strings.HasPrefix(source, "file://"):
		return os.ReadFile(strings.TrimPrefix(source, "file://"))
	default:
		return os.ReadFile(source)
	}
}

func parseCatalogEntries(body []byte) []catalogEntry {
	lines := strings.Split(string(body), "\n")
	entries := []catalogEntry{}
	var current *catalogEntry
	section := ""
	subsection := ""
	var currentInput *catalogInput
	flushInput := func() {
		if current != nil && currentInput != nil {
			current.Install.Inputs = append(current.Install.Inputs, *currentInput)
			currentInput = nil
		}
	}
	flush := func() {
		if current != nil {
			flushInput()
			entries = append(entries, *current)
		}
	}
	for _, raw := range lines {
		line := strings.TrimRight(raw, " \t\r")
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		if strings.TrimSpace(line) == "entries:" {
			continue
		}
		if strings.HasPrefix(line, "  - ") {
			flush()
			current = &catalogEntry{}
			section = ""
			subsection = ""
			rest := strings.TrimSpace(strings.TrimPrefix(line, "  - "))
			if rest != "" && strings.Contains(rest, ":") {
				key, value, ok := yamlKV(rest)
				if ok {
					assignCatalogField(current, section, key, value)
				}
			}
			continue
		}
		if current == nil {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		trimmed := strings.TrimSpace(line)
		if section == "install" && subsection == "inputs" && strings.HasPrefix(line, "        - ") {
			flushInput()
			currentInput = &catalogInput{}
			rest := strings.TrimSpace(strings.TrimPrefix(line, "        - "))
			key, value, ok := yamlKV(rest)
			if ok {
				assignCatalogInputField(currentInput, key, value)
			}
			continue
		}
		if section == "install" && subsection == "inputs" && currentInput != nil && indent >= 10 {
			key, value, ok := yamlKV(trimmed)
			if ok {
				assignCatalogInputField(currentInput, key, value)
			}
			continue
		}
		if strings.HasSuffix(trimmed, ":") {
			if indent <= 6 {
				flushInput()
			}
			if indent == 4 {
				section = strings.TrimSuffix(trimmed, ":")
				subsection = ""
			}
			if indent == 6 {
				subsection = strings.TrimSuffix(trimmed, ":")
			}
			continue
		}
		key, value, ok := yamlKV(trimmed)
		if !ok {
			continue
		}
		assignCatalogField(current, sectionForIndent(section, indent), key, value)
	}
	flush()
	return entries
}

func sectionForIndent(section string, indent int) string {
	if indent >= 6 {
		return section
	}
	return ""
}

func yamlKV(line string) (string, string, bool) {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])
	value = strings.Trim(value, "\"")
	if value == "null" {
		value = ""
	}
	return key, value, true
}

func assignCatalogField(entry *catalogEntry, section string, key string, value string) {
	switch section {
	case "metadata":
		if key == "name" {
			entry.MetadataName = value
		}
	case "artifact":
		if key == "type" {
			entry.ArtifactType = value
		}
		if key == "url" {
			entry.ArtifactURL = value
		}
	case "install":
		if key == "requiresLocalOverrides" {
			entry.Install.RequiresLocalOverrides = strings.EqualFold(value, "true")
		}
	default:
		switch key {
		case "kind":
			entry.Kind = value
		case "id":
			entry.ID = value
		case "name":
			entry.Name = value
		case "category":
			entry.Category = value
		case "version":
			entry.Version = value
		case "description":
			entry.Description = value
		case "visibility":
			entry.Visibility = value
		}
	}
}

func assignCatalogInputField(input *catalogInput, key string, value string) {
	switch key {
	case "name":
		input.Name = value
	case "required":
		input.Required = strings.EqualFold(value, "true")
	case "sensitive":
		input.Sensitive = strings.EqualFold(value, "true")
	case "source":
		input.Source = value
	case "description":
		input.Description = value
	}
}

func catalogEntryMatchesName(entry catalogEntry, name string) bool {
	candidates := []string{entry.MetadataName, entry.Name, entry.ID}
	return slices.Contains(candidates, name)
}

func firstNonFlagArg(args []string) (string, bool) {
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			return arg, true
		}
	}
	return "", false
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
