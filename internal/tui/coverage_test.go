package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func coverageRunner(_ context.Context, args []string) CommandResult {
	command := strings.Join(args, " ")
	switch command {
	case "profile list":
		return CommandResult{Args: args, Stdout: "dev\t1.0.0\tlocal\n"}
	case "profile show dev":
		return CommandResult{Args: args, Stdout: "profile detail"}
	case "stack list":
		return CommandResult{Args: args, Stdout: "base\t1.0.0\tplatform\n"}
	case "stack show base":
		return CommandResult{Args: args, Stdout: "stack detail"}
	case "addon list":
		return CommandResult{Args: args, Stdout: "nginx\t1.0.0\tweb\n"}
	case "addon show nginx":
		return CommandResult{Args: args, Stdout: "addon detail"}
	case "cluster list":
		return CommandResult{Args: args, Stdout: "local\tReady\tlocal\n"}
	case "cluster show local":
		return CommandResult{Args: args, Stdout: "cluster detail"}
	case "cluster tools":
		return CommandResult{Args: args, Stdout: "kubectl\tavailable\n"}
	default:
		return CommandResult{Args: args, Stdout: "done"}
	}
}

func TestViewsAndNavigationModes(t *testing.T) {
	model := NewModel(nil, coverageRunner)
	model.width = 60
	model.height = 20
	model.items[sectionProfiles] = []catalogItem{{Name: "dev"}, {Name: "prod"}}
	model.detail = "details"
	model.activeCluster = "local"

	for _, current := range []mode{modeCatalog, modeRunning, modeLogs, modeHelp} {
		model.mode = current
		if view := model.View(); !strings.Contains(view, "Productive K3S") {
			t.Fatalf("mode %v did not render frame: %q", current, view)
		}
	}

	model.mode = modeCatalog
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(Model)
	if model.cursor != 1 || cmd == nil {
		t.Fatalf("down navigation failed: cursor=%d cmd=%v", model.cursor, cmd)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model = updated.(Model)
	if model.cursor != 0 {
		t.Fatalf("up navigation failed: %d", model.cursor)
	}
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRight})
	model = updated.(Model)
	if model.section != sectionStacks || cmd == nil {
		t.Fatalf("right navigation failed: section=%v", model.section)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyLeft})
	model = updated.(Model)
	if model.section != sectionProfiles {
		t.Fatalf("left navigation failed: section=%v", model.section)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	model = updated.(Model)
	if model.mode != modeHelp || !strings.Contains(model.View(), "Navigation") {
		t.Fatal("help mode did not render")
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	if model.mode != modeCatalog {
		t.Fatal("escape did not restore catalog")
	}
}

func TestCatalogAndClusterEmptyViews(t *testing.T) {
	model := NewModel(context.Background(), coverageRunner)
	model.width = 90
	model.loading = true
	if view := model.sidebar(); !strings.Contains(view, "loading") {
		t.Fatalf("missing loading state: %q", view)
	}
	model.loading = false
	if view := model.catalogView(); !strings.Contains(view, "Select an item") {
		t.Fatalf("missing empty detail: %q", view)
	}
	model.section = sectionClusters
	if view := model.catalogView(); !strings.Contains(view, "No registered clusters") {
		t.Fatalf("missing empty cluster guidance: %q", view)
	}

	unknown := model.loadSection(section(99))()
	if unknown.(catalogLoadedMsg).section != section(99) {
		t.Fatalf("unexpected unknown section result: %#v", unknown)
	}
}

func TestActionGuardStates(t *testing.T) {
	model := NewModel(context.Background(), coverageRunner)
	for name, action := range map[string]func() (tea.Model, tea.Cmd){
		"install":  model.startInstall,
		"export":   model.startExport,
		"validate": model.startValidate,
	} {
		updated, cmd := action()
		got := updated.(Model)
		if got.status != "No item selected" || cmd != nil {
			t.Fatalf("%s guard failed: status=%q", name, got.status)
		}
	}

	updated, _ := model.selectClusterTarget()
	if updated.(Model).status != "Cluster target selection is only available in Clusters" {
		t.Fatal("cluster selection section guard failed")
	}
	updated, _ = model.startClusterTest()
	if updated.(Model).status != "Cluster test is only available in Clusters" {
		t.Fatal("cluster test section guard failed")
	}
	updated, _ = model.startK9s()
	if updated.(Model).status != "K9s is only available in Clusters" {
		t.Fatal("k9s section guard failed")
	}

	model.section = sectionClusters
	updated, _ = model.selectClusterTarget()
	if updated.(Model).status != "No cluster selected" {
		t.Fatal("empty cluster selection guard failed")
	}
	updated, _ = model.startClusterTest()
	if updated.(Model).status != "No cluster selected" {
		t.Fatal("empty cluster test guard failed")
	}
	updated, _ = model.startK9s()
	if updated.(Model).status != "K9s launch is not available in this TUI session" {
		t.Fatal("missing interactive runner guard failed")
	}
}

func TestLoadersExposeFailuresAndEmptyClusters(t *testing.T) {
	failing := func(_ context.Context, args []string) CommandResult {
		if strings.Join(args, " ") == "cluster tools" {
			return CommandResult{Args: args, Code: 1, Stderr: "no tools"}
		}
		return CommandResult{Args: args, Code: 2, Stderr: "failed"}
	}
	if msg := loadCatalog(context.Background(), failing, sectionProfiles, []string{"profile", "list"}); msg.err != "failed" {
		t.Fatalf("catalog failure not propagated: %#v", msg)
	}
	if msg := loadClusters(context.Background(), failing); msg.err != "failed" {
		t.Fatalf("cluster failure not propagated: %#v", msg)
	}

	empty := func(_ context.Context, args []string) CommandResult {
		if strings.Join(args, " ") == "cluster tools" {
			return CommandResult{Args: args, Stdout: "k9s available"}
		}
		return CommandResult{Args: args}
	}
	msg := loadClusters(context.Background(), empty)
	if !strings.Contains(msg.detail, "No registered clusters") || !strings.Contains(msg.detail, "k9s available") {
		t.Fatalf("unexpected empty cluster detail: %q", msg.detail)
	}
}

func TestOperationHelpersAndInteractiveFailure(t *testing.T) {
	model := NewModel(context.Background(), coverageRunner)
	model.startOperation("Work", []string{"profile", "install", "dev"}, []string{"one", "two"})
	model.finishOperation("failed")
	if model.operationSteps[0].Status != "failed" || model.operationSteps[1].Status != "failed" {
		t.Fatalf("failure states not recorded: %#v", model.operationSteps)
	}
	model.appendOperationLog("first\n")
	model.appendOperationLog("second")
	model.appendOperationLog("   ")
	if model.logs != "first\nsecond" {
		t.Fatalf("unexpected logs: %q", model.logs)
	}
	model.applyOperationEvent(OperationEvent{})
	model.applyOperationEvent(OperationEvent{Step: "operation.started"})
	model.applyOperationEvent(OperationEvent{Step: "operation.completed", Status: "failed"})
	if model.status != "Command failed" {
		t.Fatalf("completion failure not applied: %q", model.status)
	}

	model.mode = modeRunning
	updated, _ := model.Update(InteractiveFinished([]string{"cluster", "k9s", "local"}, errors.New("boom")))
	model = updated.(Model)
	if model.mode != modeLogs || !strings.Contains(model.logs, "boom") {
		t.Fatalf("interactive error not rendered: mode=%v logs=%q", model.mode, model.logs)
	}
}

func TestFormattingHelpersCoverEverySection(t *testing.T) {
	item := catalogItem{Name: "demo", Version: "1", Category: "test", Flags: "public"}
	for _, sec := range []section{sectionProfiles, sectionAddons, sectionStacks, sectionClusters, section(99)} {
		if detail := formatItemDetail(sec, item); !strings.Contains(detail, "Name: demo") {
			t.Fatalf("section %v detail missing name: %q", sec, detail)
		}
		_ = footerText(modeCatalog, sec)
		_ = sectionTitle(sec)
	}
	for _, status := range []string{"success", "failed", "warning", "running", "pending"} {
		if statusSymbol(status) == "" {
			t.Fatalf("empty symbol for %q", status)
		}
	}
	if got := tailLines("one\ntwo\nthree", 2); len(got) != 2 || got[0] != "two" {
		t.Fatalf("unexpected tail: %#v", got)
	}
	if label := operationEventLabel(OperationEvent{Step: "profile.install.run"}); label != "profile install run" {
		t.Fatalf("unexpected fallback label: %q", label)
	}
}
