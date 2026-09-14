package tui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestParseCatalogList(t *testing.T) {
	items := parseCatalogList("development\t0.1.0\tlocal\nproduction\t0.2.0\tcloud\tneeds-env\n")
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Name != "development" || items[0].Version != "0.1.0" || items[0].Category != "local" {
		t.Fatalf("unexpected first item: %#v", items[0])
	}
	if items[1].Flags != "needs-env" {
		t.Fatalf("unexpected flags: %#v", items[1])
	}
}

func TestModelLoadsProfilesAndStartsValidation(t *testing.T) {
	runner := func(_ context.Context, args []string) CommandResult {
		joined := strings.Join(args, " ")
		switch joined {
		case "profile list":
			return CommandResult{Args: args, Stdout: "development\t0.1.0\tlocal\n"}
		case "profile show development":
			return CommandResult{Args: args, Stdout: "Name: development\nKind: profile\n"}
		case "profile validate development":
			return CommandResult{Args: args, Stdout: "validated\n"}
		default:
			t.Fatalf("unexpected command: %s", joined)
			return CommandResult{Args: args, Code: 2}
		}
	}

	model := NewModel(context.Background(), runner)
	msg := model.Init()().(catalogLoadedMsg)
	updated, _ := model.Update(msg)
	model = updated.(Model)
	if len(model.currentItems()) != 1 {
		t.Fatalf("expected one loaded profile, got %#v", model.currentItems())
	}

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	model = updated.(Model)
	if model.mode != modeRunning {
		t.Fatalf("expected running mode, got %#v", model.mode)
	}
	result := cmd().(commandResultMsg)
	updated, _ = model.Update(result)
	model = updated.(Model)
	if model.mode != modeLogs || !strings.Contains(model.logs, "validated") {
		t.Fatalf("expected logs with command output, got mode=%#v logs=%q", model.mode, model.logs)
	}
}

func TestModelLoadsStacksAndShowsDetails(t *testing.T) {
	runner := func(_ context.Context, args []string) CommandResult {
		joined := strings.Join(args, " ")
		switch joined {
		case "stack list":
			return CommandResult{Args: args, Stdout: "cluster-health\t0.1.0\toperations\n"}
		case "stack show cluster-health":
			return CommandResult{Args: args, Stdout: "Name: cluster-health\nKind: stack\nArtifact URL: https://downloads.productive-k3s.io/addons/cluster-health-0.1.0.tgz\n"}
		default:
			t.Fatalf("unexpected command: %s", joined)
			return CommandResult{Args: args, Code: 2}
		}
	}

	model := NewModel(context.Background(), runner)
	model.section = sectionStacks
	msg := model.loadSection(sectionStacks)().(catalogLoadedMsg)
	updated, _ := model.Update(msg)
	model = updated.(Model)
	if len(model.currentItems()) != 1 || model.currentItems()[0].Name != "cluster-health" {
		t.Fatalf("expected one loaded stack, got %#v", model.currentItems())
	}

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	updated, _ = model.Update(cmd())
	model = updated.(Model)
	if !strings.Contains(model.detail, "Artifact URL: https://downloads.productive-k3s.io/addons/cluster-health-0.1.0.tgz") {
		t.Fatalf("expected stack details from CLI show, got %q", model.detail)
	}
}

func TestOperationProgressAndScrollableLogs(t *testing.T) {
	runner := func(_ context.Context, args []string) CommandResult {
		switch strings.Join(args, " ") {
		case "profile list":
			return CommandResult{Args: args, Stdout: "development\t0.1.0\tlocal\n"}
		case "profile show development":
			return CommandResult{Args: args, Stdout: "Name: development\nKind: profile\n"}
		case "profile install development":
			return CommandResult{Args: args, Stdout: strings.Repeat("line\n", 40)}
		default:
			t.Fatalf("unexpected command: %s", strings.Join(args, " "))
			return CommandResult{Args: args, Code: 2}
		}
	}

	model := NewModel(context.Background(), runner)
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	model = updated.(Model)
	updated, _ = model.Update(model.Init()().(catalogLoadedMsg))
	model = updated.(Model)

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	model = updated.(Model)
	progress := model.operationView()
	if !strings.Contains(progress, "Resolve catalog entry") || !strings.Contains(progress, "Run profile install") {
		t.Fatalf("operation view missing steps: %s", progress)
	}

	updated, _ = model.Update(cmd())
	model = updated.(Model)
	if model.mode != modeLogs {
		t.Fatalf("expected logs mode, got %#v", model.mode)
	}
	if model.logViewport.Height != 7 {
		t.Fatalf("unexpected viewport height: %d", model.logViewport.Height)
	}
	if !strings.Contains(model.logsView(), "line") {
		t.Fatalf("expected viewport to render log content")
	}
}

func TestParseOperationEventLine(t *testing.T) {
	event, ok := ParseOperationEventLine(`{"schema_version":"productive-k3s-operation-event/v1","component":"infra","operation":"profile.install","step":"profile.install.run","status":"running","message":"Executing packaged profile installer","subject":"demo","emitted_at":"2026-09-07T10:00:00-03:00"}`)
	if !ok {
		t.Fatal("expected operation event to parse")
	}
	if event.Component != "infra" || event.Step != "profile.install.run" || event.Subject != "demo" {
		t.Fatalf("unexpected event: %#v", event)
	}
	if _, ok := ParseOperationEventLine("human log line"); ok {
		t.Fatal("human log line parsed as an operation event")
	}
}

func TestModelAppliesOperationEventsAsProgress(t *testing.T) {
	model := NewModel(context.Background(), func(context.Context, []string) CommandResult {
		return CommandResult{}
	})
	model.startOperation("Installing profile demo", []string{"profile", "install", "demo"}, []string{
		"Resolve catalog entry",
		"Run profile install",
		"Complete",
	})

	model.applyOperationEvent(OperationEvent{
		SchemaVersion: "productive-k3s-operation-event/v1",
		Component:     "infra",
		Operation:     "profile.install",
		Step:          "profile.package.extract",
		Status:        "success",
		Message:       "Profile package extracted",
		Subject:       "demo",
	})
	model.applyOperationEvent(OperationEvent{
		SchemaVersion: "productive-k3s-operation-event/v1",
		Component:     "infra",
		Operation:     "profile.install",
		Step:          "profile.install.run",
		Status:        "running",
		Message:       "Executing packaged profile installer",
		Subject:       "demo",
	})

	progress := model.operationView()
	if strings.Contains(progress, "Resolve catalog entry") {
		t.Fatalf("expected evented progress to replace coarse steps: %s", progress)
	}
	if !strings.Contains(progress, "Profile package extracted") || !strings.Contains(progress, "Executing packaged profile installer") {
		t.Fatalf("expected evented progress labels: %s", progress)
	}
}

func TestCommandResultAppliesBufferedOperationEvents(t *testing.T) {
	model := NewModel(context.Background(), func(context.Context, []string) CommandResult {
		return CommandResult{}
	})
	model.startOperation("Validating add-on demo", []string{"addon", "validate", "demo"}, []string{
		"Resolve add-on",
		"Run add-on validation",
		"Complete",
	})

	updated, _ := model.Update(commandResultMsg(CommandResult{
		Args: []string{"addon", "validate", "demo"},
		Events: []OperationEvent{
			{
				SchemaVersion: "productive-k3s-operation-event/v1",
				Component:     "core",
				Operation:     "addon.validate",
				Step:          "addon.package.validate",
				Status:        "success",
				Message:       "Add-on package validation passed",
				Subject:       "demo",
			},
			{
				SchemaVersion: "productive-k3s-operation-event/v1",
				Component:     "core",
				Operation:     "addon.validate",
				Step:          "operation.completed",
				Status:        "success",
				Message:       "Operation completed",
				Subject:       "demo",
			},
		},
	}))
	model = updated.(Model)
	progress := model.operationView()
	if !strings.Contains(progress, "Add-on package validation passed") || !strings.Contains(progress, "Complete") {
		t.Fatalf("expected buffered event progress, got: %s", progress)
	}
	if strings.Contains(progress, "Resolve add-on") {
		t.Fatalf("expected buffered events to replace coarse steps: %s", progress)
	}
}

func TestStartCommandOperationSubscribesToStreamRunner(t *testing.T) {
	runner := func(_ context.Context, args []string) CommandResult {
		switch strings.Join(args, " ") {
		case "profile list":
			return CommandResult{Args: args, Stdout: "development\t0.1.0\tlocal\n"}
		case "profile show development":
			return CommandResult{Args: args, Stdout: "Name: development\nKind: profile\n"}
		default:
			t.Fatalf("unexpected command: %s", strings.Join(args, " "))
			return CommandResult{Args: args, Code: 2}
		}
	}
	streamRunner := func(_ context.Context, args []string, eventSink func(OperationEvent), logSink func(string)) CommandResult {
		return CommandResult{Args: args}
	}

	model := NewModel(context.Background(), runner).WithStreamRunner(streamRunner)
	updated, _ := model.Update(model.Init()().(catalogLoadedMsg))
	model = updated.(Model)

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	model = updated.(Model)
	if cmd == nil {
		t.Fatal("expected command")
	}
	if model.operationEvents == nil {
		t.Fatal("expected model to subscribe to operation event channel")
	}
}

func TestModelLaunchesK9sWithInteractiveRunner(t *testing.T) {
	runner := func(_ context.Context, args []string) CommandResult {
		switch strings.Join(args, " ") {
		case "cluster list":
			return CommandResult{Args: args, Stdout: "local-dev\tUnknown\tpk3s-local-dev\n"}
		case "cluster show local-dev":
			return CommandResult{Args: args, Stdout: "ID: local-dev\nContext: pk3s-local-dev\n"}
		case "cluster tools":
			return CommandResult{Args: args, Stdout: "k9s\tavailable\t/bin/k9s\n"}
		default:
			t.Fatalf("unexpected command: %s", strings.Join(args, " "))
			return CommandResult{Args: args, Code: 2}
		}
	}
	var interactiveArgs []string
	interactive := func(_ context.Context, args []string) tea.Cmd {
		interactiveArgs = append([]string(nil), args...)
		return func() tea.Msg { return InteractiveFinished(args, nil) }
	}

	model := NewModelInteractive(context.Background(), runner, interactive)
	model.section = sectionClusters
	msg := model.loadSection(sectionClusters)().(catalogLoadedMsg)
	updated, _ := model.Update(msg)
	model = updated.(Model)

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'K'}})
	model = updated.(Model)
	if strings.Join(interactiveArgs, " ") != "cluster k9s local-dev" {
		t.Fatalf("unexpected interactive args: %#v", interactiveArgs)
	}
	if model.mode != modeRunning {
		t.Fatalf("expected running mode, got %#v", model.mode)
	}
	updated, _ = model.Update(cmd())
	model = updated.(Model)
	if model.mode != modeCatalog || model.status != "K9s closed" {
		t.Fatalf("expected catalog after k9s closes, got mode=%#v status=%q", model.mode, model.status)
	}
}
