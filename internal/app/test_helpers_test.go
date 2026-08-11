package app

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func noTelemetryDeps(t *testing.T, deps Dependencies) Dependencies {
	t.Helper()
	t.Setenv("TELEMETRY_ENABLED", "false")
	if deps.HTTPClient == nil {
		deps.HTTPClient = &http.Client{
			Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return nil, errors.New("unexpected outbound HTTP request in unit test")
			}),
		}
	}
	if deps.Exec == nil {
		deps.Exec = func(context.Context, Invocation) error { return nil }
	}
	return deps
}

func envMap(items []string) map[string]string {
	result := map[string]string{}
	for _, item := range items {
		key, value, ok := strings.Cut(item, "=")
		if ok {
			result[key] = value
		}
	}
	return result
}
