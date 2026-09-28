package tools

import (
	"os"
	"os/exec"
	"strings"
)

type Status struct {
	Name      string
	Found     bool
	Path      string
	ErrorText string
}

func Detect(name string) Status {
	path, err := exec.LookPath(name)
	if err != nil {
		return Status{Name: name, Found: false, ErrorText: err.Error()}
	}
	return Status{Name: name, Found: true, Path: path}
}

func EnvWithKubeconfig(base []string, kubeconfig string) []string {
	env := map[string]string{}
	for _, item := range base {
		key, value, ok := strings.Cut(item, "=")
		if ok {
			env[key] = value
		}
	}
	env["KUBECONFIG"] = kubeconfig
	result := make([]string, 0, len(env))
	for key, value := range env {
		result = append(result, key+"="+value)
	}
	return result
}

func CurrentEnvWithKubeconfig(kubeconfig string) []string {
	return EnvWithKubeconfig(os.Environ(), kubeconfig)
}
