package app

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/productive-k3s/productive-k3s-cli/internal/tui"
)

func TestRunEventedInvocationSeparatesEventsAndHumanLogs(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "productive-k3s-core.sh")
	if err := os.WriteFile(script, []byte(`#!/usr/bin/env bash
set -euo pipefail
if [[ "$1" != "--events" || "$2" != "ndjson" ]]; then
  printf 'missing events flag: %s\n' "$*" >&2
  exit 9
fi
printf '{"schema_version":"productive-k3s-operation-event/v1","component":"core","operation":"addon.validate","step":"addon.package.validate","status":"success","message":"Add-on package validation passed","subject":"demo-addon","emitted_at":"2026-09-07T10:00:00-03:00"}\n'
printf 'human log\n' >&2
`), 0o755); err != nil {
		t.Fatal(err)
	}

	var events []tui.OperationEvent
	var logs []string
	var stdout safeBuffer
	var stderr safeBuffer
	err := runEventedInvocation(
		context.Background(),
		Invocation{Path: script, Args: []string{"addon", "validate"}},
		func(event tui.OperationEvent) { events = append(events, event) },
		func(line string) { logs = append(logs, line) },
		&stdout,
		&stderr,
	)
	if err != nil {
		t.Fatalf("runEventedInvocation returned error: %v", err)
	}
	if len(events) != 1 || events[0].Step != "addon.package.validate" || events[0].Subject != "demo-addon" {
		t.Fatalf("unexpected events: %#v", events)
	}
	if stdout.String() != "" {
		t.Fatalf("expected stdout buffer to exclude event lines, got %q", stdout.String())
	}
	if stderr.String() != "human log\n" {
		t.Fatalf("unexpected stderr buffer: %q", stderr.String())
	}
	if !slices.Contains(logs, "human log") {
		t.Fatalf("expected human log to be streamed, got %#v", logs)
	}
}

func TestWithOperationEventsDoesNotDuplicateFlag(t *testing.T) {
	args := withOperationEvents([]string{"--events", "ndjson", "addon", "validate"})
	if len(args) != 4 || args[0] != "--events" {
		t.Fatalf("unexpected args: %#v", args)
	}
	args = withOperationEvents([]string{"addon", "validate"})
	if len(args) != 4 || args[0] != "--events" || args[1] != "ndjson" {
		t.Fatalf("unexpected args: %#v", args)
	}
}
