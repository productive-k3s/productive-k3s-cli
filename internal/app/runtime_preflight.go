package app

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type profileRuntimeRequirements struct {
	Name     string
	Scenario string
	Engine   string
}

func packagedProfileRuntimePreflight(tgzPath string, deps Dependencies) int {
	requirements, found, err := readPackagedProfileRuntime(tgzPath)
	if err != nil {
		fmt.Fprintf(deps.Stderr, "could not inspect profile runtime requirements: %v\n", err)
		return 2
	}
	if !found {
		return 0
	}
	return profileRuntimePreflight(requirements, deps)
}

func profileEnvRuntimePreflight(profilePath string, deps Dependencies) int {
	body, err := os.ReadFile(profilePath)
	if err != nil {
		fmt.Fprintf(deps.Stderr, "could not inspect profile runtime requirements: %v\n", err)
		return 2
	}
	return profileRuntimePreflight(profileRuntimeFromEnv(body), deps)
}

func readPackagedProfileRuntime(tgzPath string) (profileRuntimeRequirements, bool, error) {
	file, err := os.Open(tgzPath)
	if err != nil {
		return profileRuntimeRequirements{}, false, err
	}
	defer file.Close()

	gz, err := gzip.NewReader(file)
	if err != nil {
		return profileRuntimeRequirements{}, false, nil
	}
	defer gz.Close()

	reader := tar.NewReader(gz)
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return profileRuntimeRequirements{}, false, nil
		}
		if err != nil {
			return profileRuntimeRequirements{}, false, nil
		}
		if header.Typeflag != tar.TypeReg || filepath.Base(filepath.Clean(header.Name)) != "profile.yaml" {
			continue
		}
		if header.Size > 1024*1024 {
			return profileRuntimeRequirements{}, false, fmt.Errorf("profile.yaml exceeds the 1 MiB inspection limit")
		}
		body, err := io.ReadAll(io.LimitReader(reader, 1024*1024+1))
		if err != nil {
			return profileRuntimeRequirements{}, false, err
		}
		var document struct {
			Metadata struct {
				Name string `yaml:"name"`
			} `yaml:"metadata"`
			Spec struct {
				Scenario struct {
					Type string `yaml:"type"`
				} `yaml:"scenario"`
				Engine struct {
					Type string `yaml:"type"`
				} `yaml:"engine"`
			} `yaml:"spec"`
		}
		if err := yaml.Unmarshal(body, &document); err != nil {
			return profileRuntimeRequirements{}, false, nil
		}
		return profileRuntimeRequirements{
			Name:     strings.TrimSpace(document.Metadata.Name),
			Scenario: strings.ToLower(strings.TrimSpace(document.Spec.Scenario.Type)),
			Engine:   strings.ToLower(strings.TrimSpace(document.Spec.Engine.Type)),
		}, true, nil
	}
}

func profileRuntimeFromEnv(body []byte) profileRuntimeRequirements {
	values := map[string]string{}
	scanner := bufio.NewScanner(bytes.NewReader(body))
	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(scanner.Text()), "export "))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			values[strings.TrimSpace(parts[0])] = strings.Trim(strings.TrimSpace(parts[1]), `"'`)
		}
	}
	return profileRuntimeRequirements{
		Name:     values["PK3S_INFRA_PROFILE_NAME"],
		Scenario: strings.ToLower(values["PK3S_INFRA_SCENARIO"]),
		Engine:   strings.ToLower(values["PK3S_INFRA_ENGINE"]),
	}
}

func profileRuntimePreflight(requirements profileRuntimeRequirements, deps Dependencies) int {
	disabled := commaSeparatedSet(os.Getenv("PK3S_RUNTIME_DISABLED_SCENARIOS"))
	if disabled[requirements.Scenario] {
		fmt.Fprintf(deps.Stderr, "profile %q cannot run in this environment: scenario %q is disabled by runtime policy\n", profileRuntimeName(requirements), requirements.Scenario)
		return 2
	}

	requiredCommands := []string{}
	switch requirements.Engine {
	case "opentofu":
		requiredCommands = append(requiredCommands, "jq", "tofu")
	case "ansible":
		requiredCommands = append(requiredCommands, "ansible-playbook", "curl", "jq", "python3", "scp", "sha256sum", "ssh", "tar")
	}
	if requirements.Scenario == "multipass" {
		requiredCommands = append(requiredCommands, "curl", "multipass", "sha256sum", "ssh", "ssh-keygen", "tar", "timeout")
	}
	sort.Strings(requiredCommands)
	requiredCommands = slices.Compact(requiredCommands)

	missing := make([]string, 0, len(requiredCommands))
	for _, command := range requiredCommands {
		if _, err := deps.LookPath(command); err != nil {
			missing = append(missing, command)
		}
	}
	if len(missing) == 0 {
		return 0
	}
	fmt.Fprintf(deps.Stderr, "profile %q cannot run in this environment: missing required command(s): %s\n", profileRuntimeName(requirements), strings.Join(missing, ", "))
	return 2
}

func commaSeparatedSet(value string) map[string]bool {
	result := map[string]bool{}
	for _, item := range strings.Split(value, ",") {
		if item = strings.ToLower(strings.TrimSpace(item)); item != "" {
			result[item] = true
		}
	}
	return result
}

func profileRuntimeName(requirements profileRuntimeRequirements) string {
	if strings.TrimSpace(requirements.Name) != "" {
		return requirements.Name
	}
	return "unknown"
}
