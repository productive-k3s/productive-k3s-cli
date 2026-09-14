package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type CommandResult struct {
	Args   []string
	Code   int
	Stdout string
	Stderr string
	Events []OperationEvent
}

type CommandRunner func(context.Context, []string) CommandResult
type StreamCommandRunner func(context.Context, []string, func(OperationEvent), func(string)) CommandResult
type InteractiveRunner func(context.Context, []string) tea.Cmd

type commandResultMsg CommandResult
type operationEventMsg OperationEvent
type operationLogMsg string
type interactiveFinishedMsg struct {
	args []string
	err  error
}

type catalogLoadedMsg struct {
	section section
	items   []catalogItem
	detail  string
	err     string
}

type section int

const (
	sectionProfiles section = iota
	sectionStacks
	sectionAddons
	sectionClusters
)

type mode int

const (
	modeCatalog mode = iota
	modeRunning
	modeLogs
	modeHelp
)

type catalogItem struct {
	Name     string
	Version  string
	Category string
	Flags    string
}

type operationStep struct {
	Name   string
	Status string
}

type OperationEvent struct {
	SchemaVersion string `json:"schema_version"`
	Component     string `json:"component"`
	Operation     string `json:"operation"`
	Step          string `json:"step"`
	Status        string `json:"status"`
	Message       string `json:"message"`
	Subject       string `json:"subject"`
	EmittedAt     string `json:"emitted_at"`
}

type Model struct {
	ctx              context.Context
	runner           CommandRunner
	streamRunner     StreamCommandRunner
	interactive      InteractiveRunner
	width            int
	height           int
	section          section
	cursor           int
	mode             mode
	items            map[section][]catalogItem
	detail           string
	status           string
	logs             string
	logViewport      viewport.Model
	loaded           map[section]bool
	loading          bool
	lastResult       CommandResult
	operationTitle   string
	operationArgs    []string
	operationSteps   []operationStep
	operationStarted string
	operationEvents  chan tea.Msg
	eventedOperation bool
}

func Run(ctx context.Context, stdout io.Writer, stderr io.Writer, runner CommandRunner) error {
	return RunInteractive(ctx, stdout, stderr, runner, nil)
}

func RunInteractive(ctx context.Context, stdout io.Writer, stderr io.Writer, runner CommandRunner, interactive InteractiveRunner) error {
	if runner == nil {
		return fmt.Errorf("missing TUI command runner")
	}
	return RunModel(stdout, NewModelInteractive(ctx, runner, interactive))
}

func RunModel(stdout io.Writer, model Model) error {
	p := tea.NewProgram(model, tea.WithOutput(stdout), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func NewModel(ctx context.Context, runner CommandRunner) Model {
	return NewModelInteractive(ctx, runner, nil)
}

func NewModelInteractive(ctx context.Context, runner CommandRunner, interactive InteractiveRunner) Model {
	if ctx == nil {
		ctx = context.Background()
	}
	return Model{
		ctx:         ctx,
		runner:      runner,
		interactive: interactive,
		section:     sectionProfiles,
		items:       map[section][]catalogItem{},
		loaded:      map[section]bool{},
		logViewport: viewport.New(80, 20),
		status:      "Loading catalog",
	}
}

func (m Model) WithStreamRunner(runner StreamCommandRunner) Model {
	m.streamRunner = runner
	return m
}

func InteractiveFinished(args []string, err error) tea.Msg {
	return interactiveFinishedMsg{args: append([]string(nil), args...), err: err}
}

func (m Model) Init() tea.Cmd {
	return m.loadSection(sectionProfiles)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeViewport()
	case tea.KeyMsg:
		return m.updateKey(msg)
	case catalogLoadedMsg:
		m.loading = false
		m.loaded[msg.section] = true
		if msg.err != "" {
			m.status = msg.err
			m.items[msg.section] = nil
			m.detail = msg.err
			return m, nil
		}
		m.items[msg.section] = msg.items
		m.detail = msg.detail
		m.status = fmt.Sprintf("%s loaded", sectionTitle(msg.section))
	case commandResultMsg:
		m.loading = false
		m.mode = modeLogs
		m.operationEvents = nil
		m.lastResult = CommandResult(msg)
		for _, event := range msg.Events {
			m.applyOperationEvent(event)
		}
		m.logs = renderCommandResult(CommandResult(msg))
		m.setLogs(m.logs)
		if msg.Code == 0 {
			m.status = "Command completed"
			if !m.eventedOperation {
				m.finishOperation("success")
			}
		} else {
			m.status = "Command failed"
			if !m.eventedOperation {
				m.finishOperation("failed")
			}
		}
	case operationEventMsg:
		m.applyOperationEvent(OperationEvent(msg))
		if m.operationEvents != nil {
			return m, m.waitOperationMsg(m.operationEvents)
		}
	case operationLogMsg:
		m.appendOperationLog(string(msg))
		if m.operationEvents != nil {
			return m, m.waitOperationMsg(m.operationEvents)
		}
	case interactiveFinishedMsg:
		m.loading = false
		m.mode = modeCatalog
		if msg.err != nil {
			m.status = "K9s failed"
			m.logs = fmt.Sprintf("$ pk3s %s\n\nError: %v", strings.Join(msg.args, " "), msg.err)
			m.setLogs(m.logs)
			m.mode = modeLogs
		} else {
			m.status = "K9s closed"
		}
	}
	return m, nil
}

func (m Model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		if m.mode == modeCatalog {
			return m, tea.Quit
		}
		m.mode = modeCatalog
		return m, nil
	case "?":
		if m.mode == modeHelp {
			m.mode = modeCatalog
		} else {
			m.mode = modeHelp
		}
		return m, nil
	case "esc":
		m.mode = modeCatalog
		return m, nil
	}
	if m.mode == modeLogs {
		var cmd tea.Cmd
		m.logViewport, cmd = m.logViewport.Update(msg)
		return m, cmd
	}
	if m.mode != modeCatalog {
		return m, nil
	}
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			return m, m.loadDetail()
		}
	case "down", "j":
		if m.cursor < len(m.currentItems())-1 {
			m.cursor++
			return m, m.loadDetail()
		}
	case "left", "h":
		if m.section > sectionProfiles {
			m.section--
			m.cursor = 0
			return m, m.ensureLoaded()
		}
	case "right", "l", "tab":
		if m.section < sectionClusters {
			m.section++
			m.cursor = 0
			return m, m.ensureLoaded()
		}
	case "r":
		m.loaded[m.section] = false
		m.status = "Refreshing"
		return m, m.loadSection(m.section)
	case "enter", "d":
		return m, m.loadDetail()
	case "i":
		return m.startInstall()
	case "v":
		return m.startValidate()
	case "t":
		return m.startClusterTest()
	case "K":
		return m.startK9s()
	}
	return m, nil
}

func (m Model) View() string {
	if m.width == 0 {
		m.width = 96
	}
	if m.height == 0 {
		m.height = 28
	}
	switch m.mode {
	case modeRunning:
		return m.frame("Running", m.operationView())
	case modeLogs:
		return m.frame("Logs", m.logsView())
	case modeHelp:
		return m.frame("Help", helpView())
	default:
		return m.frame(sectionTitle(m.section), m.catalogView())
	}
}

func (m Model) frame(title string, body string) string {
	header := titleStyle.Render("Productive K3S") + "  " + mutedStyle.Render(title+" | "+m.status)
	footer := footerStyle.Width(max(20, m.width-2)).Render(footerText(m.mode, m.section))
	contentHeight := max(4, m.height-4)
	body = lipgloss.NewStyle().Height(contentHeight).MaxHeight(contentHeight).Render(body)
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (m *Model) resizeViewport() {
	m.logViewport.Width = max(20, m.width-4)
	m.logViewport.Height = max(4, m.height-5)
}

func (m *Model) setLogs(content string) {
	m.resizeViewport()
	m.logViewport.SetContent(content)
	m.logViewport.GotoBottom()
}

func (m Model) catalogView() string {
	left := m.sidebar()
	right := m.detail
	if m.section == sectionClusters {
		if strings.TrimSpace(right) == "" {
			right = "No registered clusters.\n\nUse pk3s cluster register <id> --kubeconfig <file> to add one."
		}
	}
	if strings.TrimSpace(right) == "" {
		right = "Select an item to view details."
	}
	if m.width < 72 {
		return lipgloss.JoinVertical(lipgloss.Left, left, panelStyle.Width(max(20, m.width-2)).Render(right))
	}
	leftWidth := 28
	rightWidth := max(24, m.width-leftWidth-4)
	return lipgloss.JoinHorizontal(lipgloss.Top,
		panelStyle.Width(leftWidth).Render(left),
		panelStyle.Width(rightWidth).Render(right),
	)
}

func (m Model) sidebar() string {
	var b strings.Builder
	for _, sec := range []section{sectionProfiles, sectionStacks, sectionAddons, sectionClusters} {
		prefix := "  "
		if sec == m.section {
			prefix = "> "
		}
		fmt.Fprintf(&b, "%s%s\n", prefix, sectionTitle(sec))
		if sec == m.section {
			items := m.currentItems()
			if len(items) == 0 {
				if m.loading {
					b.WriteString("    loading...\n")
				} else {
					b.WriteString("    no items\n")
				}
			}
			for i, item := range items {
				row := "    " + item.Name
				if i == m.cursor {
					row = selectedStyle.Render("  > " + item.Name)
				}
				b.WriteString(row + "\n")
			}
		}
	}
	return b.String()
}

func (m Model) currentItems() []catalogItem {
	return m.items[m.section]
}

func (m Model) selectedItem() (catalogItem, bool) {
	items := m.currentItems()
	if m.cursor < 0 || m.cursor >= len(items) {
		return catalogItem{}, false
	}
	return items[m.cursor], true
}

func (m Model) ensureLoaded() tea.Cmd {
	if m.loaded[m.section] {
		return m.loadDetail()
	}
	m.loading = true
	return m.loadSection(m.section)
}

func (m Model) loadSection(sec section) tea.Cmd {
	m.loading = true
	return func() tea.Msg {
		switch sec {
		case sectionProfiles:
			return loadCatalog(m.ctx, m.runner, sec, []string{"profile", "list"})
		case sectionAddons:
			return loadCatalog(m.ctx, m.runner, sec, []string{"addon", "list"})
		case sectionStacks:
			return loadCatalog(m.ctx, m.runner, sec, []string{"stack", "list"})
		case sectionClusters:
			return loadClusters(m.ctx, m.runner)
		default:
			return catalogLoadedMsg{section: sec}
		}
	}
}

func (m Model) loadDetail() tea.Cmd {
	item, ok := m.selectedItem()
	if !ok {
		return nil
	}
	if m.section == sectionClusters {
		return func() tea.Msg {
			res := m.runner(m.ctx, []string{"cluster", "show", item.Name})
			if res.Code != 0 {
				return catalogLoadedMsg{section: m.section, items: m.currentItems(), err: strings.TrimSpace(res.Stderr)}
			}
			tools := m.runner(m.ctx, []string{"cluster", "tools"})
			detail := strings.TrimSpace(res.Stdout)
			if tools.Code == 0 && strings.TrimSpace(tools.Stdout) != "" {
				detail += "\n\nDeveloper Tools\n" + strings.TrimSpace(tools.Stdout)
			}
			return catalogLoadedMsg{section: m.section, items: m.currentItems(), detail: detail}
		}
	}
	if m.section == sectionStacks {
		return func() tea.Msg {
			res := m.runner(m.ctx, []string{"stack", "show", item.Name})
			if res.Code != 0 {
				return catalogLoadedMsg{section: m.section, items: m.currentItems(), err: strings.TrimSpace(res.Stderr)}
			}
			return catalogLoadedMsg{section: m.section, items: m.currentItems(), detail: strings.TrimSpace(res.Stdout)}
		}
	}
	if m.section != sectionProfiles {
		detail := formatItemDetail(m.section, item)
		return func() tea.Msg { return catalogLoadedMsg{section: m.section, items: m.currentItems(), detail: detail} }
	}
	return func() tea.Msg {
		res := m.runner(m.ctx, []string{"profile", "show", item.Name})
		if res.Code != 0 {
			return catalogLoadedMsg{section: m.section, items: m.currentItems(), err: strings.TrimSpace(res.Stderr)}
		}
		return catalogLoadedMsg{section: m.section, items: m.currentItems(), detail: strings.TrimSpace(res.Stdout)}
	}
}

func (m Model) startInstall() (tea.Model, tea.Cmd) {
	item, ok := m.selectedItem()
	if !ok {
		m.status = "No item selected"
		return m, nil
	}
	switch m.section {
	case sectionProfiles:
		return m.startCommandOperation("Installing profile "+item.Name, []string{"profile", "install", item.Name}, []string{
			"Resolve catalog entry",
			"Run profile install",
			"Register cluster when profile state is available",
			"Complete",
		})
	case sectionAddons:
		m.status = "Add-on install needs a target: use pk3s addon install " + item.Name + " --profile <name>"
		return m, nil
	default:
		m.status = "Install is not supported for this section"
		return m, nil
	}
}

func (m Model) startValidate() (tea.Model, tea.Cmd) {
	item, ok := m.selectedItem()
	if !ok {
		m.status = "No item selected"
		return m, nil
	}
	switch m.section {
	case sectionProfiles:
		return m.startCommandOperation("Validating profile "+item.Name, []string{"profile", "validate", item.Name}, []string{
			"Resolve profile",
			"Run profile validation",
			"Complete",
		})
	case sectionAddons:
		return m.startCommandOperation("Validating add-on "+item.Name, []string{"addon", "validate", item.Name}, []string{
			"Resolve add-on",
			"Run add-on validation",
			"Complete",
		})
	default:
		m.status = "Validate is not supported for this section"
		return m, nil
	}
}

func (m Model) startClusterTest() (tea.Model, tea.Cmd) {
	if m.section != sectionClusters {
		m.status = "Cluster test is only available in Clusters"
		return m, nil
	}
	item, ok := m.selectedItem()
	if !ok {
		m.status = "No cluster selected"
		return m, nil
	}
	return m.startCommandOperation("Testing cluster "+item.Name, []string{"cluster", "test", item.Name}, []string{
		"Load cluster registry",
		"Run kubectl get nodes with isolated KUBECONFIG",
		"Complete",
	})
}

func (m Model) startK9s() (tea.Model, tea.Cmd) {
	if m.section != sectionClusters {
		m.status = "K9s is only available in Clusters"
		return m, nil
	}
	if m.interactive == nil {
		m.status = "K9s launch is not available in this TUI session"
		return m, nil
	}
	item, ok := m.selectedItem()
	if !ok {
		m.status = "No cluster selected"
		return m, nil
	}
	args := []string{"cluster", "k9s", item.Name}
	m.startOperation("Opening K9s for "+item.Name, args, []string{
		"Load cluster registry",
		"Launch K9s with isolated KUBECONFIG",
		"Resume Productive K3S TUI",
	})
	return m, m.interactive(m.ctx, args)
}

func (m *Model) startOperation(title string, args []string, steps []string) {
	m.mode = modeRunning
	m.status = title
	m.operationTitle = title
	m.operationArgs = append([]string(nil), args...)
	m.operationStarted = time.Now().Format(time.RFC3339)
	m.operationSteps = make([]operationStep, 0, len(steps))
	m.eventedOperation = false
	for i, step := range steps {
		status := "pending"
		if i == 0 {
			status = "running"
		}
		m.operationSteps = append(m.operationSteps, operationStep{Name: step, Status: status})
	}
}

func (m Model) startCommandOperation(title string, args []string, steps []string) (tea.Model, tea.Cmd) {
	m.startOperation(title, args, steps)
	return m, m.runCommand(args)
}

func (m *Model) finishOperation(status string) {
	if len(m.operationSteps) == 0 {
		return
	}
	for i := range m.operationSteps {
		if status == "success" {
			m.operationSteps[i].Status = "success"
			continue
		}
		if i == len(m.operationSteps)-1 {
			m.operationSteps[i].Status = "failed"
		} else if m.operationSteps[i].Status == "running" {
			m.operationSteps[i].Status = "failed"
		}
	}
}

func (m *Model) runCommand(args []string) tea.Cmd {
	if m.streamRunner != nil {
		events := make(chan tea.Msg, 64)
		m.operationEvents = events
		return tea.Batch(m.waitOperationMsg(events), m.runStreamingCommand(args, events))
	}
	return func() tea.Msg {
		return commandResultMsg(m.runner(m.ctx, args))
	}
}

func (m Model) waitOperationMsg(events <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-events
		if !ok {
			return nil
		}
		return msg
	}
}

func (m Model) runStreamingCommand(args []string, events chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		defer close(events)
		result := m.streamRunner(
			m.ctx,
			args,
			func(event OperationEvent) { events <- operationEventMsg(event) },
			func(line string) { events <- operationLogMsg(line) },
		)
		return commandResultMsg(result)
	}
}

func (m *Model) applyOperationEvent(event OperationEvent) {
	if strings.TrimSpace(event.Step) == "" {
		return
	}
	if !m.eventedOperation {
		m.operationSteps = nil
		m.eventedOperation = true
	}
	if event.Step == "operation.started" {
		m.status = "Command running"
		return
	}
	if event.Step == "operation.completed" {
		m.upsertOperationStep("Complete", event.Status)
		if event.Status == "success" {
			m.status = "Command completed"
		} else if event.Status == "failed" {
			m.status = "Command failed"
		}
		return
	}
	label := operationEventLabel(event)
	m.upsertOperationStep(label, event.Status)
}

func (m *Model) upsertOperationStep(name string, status string) {
	if status == "" {
		status = "running"
	}
	for i := range m.operationSteps {
		if m.operationSteps[i].Name == name {
			m.operationSteps[i].Status = status
			return
		}
	}
	m.operationSteps = append(m.operationSteps, operationStep{Name: name, Status: status})
}

func (m *Model) appendOperationLog(line string) {
	line = strings.TrimRight(line, "\r\n")
	if strings.TrimSpace(line) == "" {
		return
	}
	if m.logs == "" {
		m.logs = line
	} else {
		m.logs += "\n" + line
	}
}

func operationEventLabel(event OperationEvent) string {
	if strings.TrimSpace(event.Message) != "" {
		return event.Message
	}
	return strings.ReplaceAll(event.Step, ".", " ")
}

func ParseOperationEventLine(line string) (OperationEvent, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return OperationEvent{}, false
	}
	var event OperationEvent
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		return OperationEvent{}, false
	}
	if event.SchemaVersion != "productive-k3s-operation-event/v1" || event.Step == "" {
		return OperationEvent{}, false
	}
	return event, true
}

func loadCatalog(ctx context.Context, runner CommandRunner, sec section, args []string) catalogLoadedMsg {
	res := runner(ctx, args)
	if res.Code != 0 {
		return catalogLoadedMsg{section: sec, err: strings.TrimSpace(res.Stderr)}
	}
	items := parseCatalogList(res.Stdout)
	detail := "Select an item to view details."
	if len(items) > 0 {
		detail = formatItemDetail(sec, items[0])
		if sec == sectionProfiles {
			show := runner(ctx, []string{"profile", "show", items[0].Name})
			if show.Code == 0 {
				detail = strings.TrimSpace(show.Stdout)
			}
		}
	}
	return catalogLoadedMsg{section: sec, items: items, detail: detail}
}

func loadClusters(ctx context.Context, runner CommandRunner) catalogLoadedMsg {
	res := runner(ctx, []string{"cluster", "list"})
	if res.Code != 0 {
		return catalogLoadedMsg{section: sectionClusters, err: strings.TrimSpace(res.Stderr)}
	}
	items := parseCatalogList(res.Stdout)
	toolsRes := runner(ctx, []string{"cluster", "tools"})
	toolsDetail := "Developer Tools\n" + strings.TrimSpace(toolsRes.Stdout)
	if len(items) == 0 {
		detail := "No registered clusters.\n\nUse pk3s cluster register <id> --kubeconfig <file> to add one."
		if toolsRes.Code == 0 && strings.TrimSpace(toolsRes.Stdout) != "" {
			detail += "\n\n" + toolsDetail
		}
		return catalogLoadedMsg{section: sectionClusters, items: items, detail: detail}
	}
	show := runner(ctx, []string{"cluster", "show", items[0].Name})
	detail := formatItemDetail(sectionClusters, items[0])
	if show.Code == 0 {
		detail = strings.TrimSpace(show.Stdout)
	}
	if toolsRes.Code == 0 && strings.TrimSpace(toolsRes.Stdout) != "" {
		detail += "\n\n" + toolsDetail
	}
	return catalogLoadedMsg{section: sectionClusters, items: items, detail: detail}
}

func parseCatalogList(out string) []catalogItem {
	items := []catalogItem{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		item := catalogItem{Name: strings.TrimSpace(parts[0])}
		if len(parts) > 1 {
			item.Version = strings.TrimSpace(parts[1])
		}
		if len(parts) > 2 {
			item.Category = strings.TrimSpace(parts[2])
		}
		if len(parts) > 3 {
			item.Flags = strings.Join(parts[3:], " ")
		}
		items = append(items, item)
	}
	return items
}

func formatItemDetail(sec section, item catalogItem) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Name: %s\nKind: %s\n", item.Name, strings.ToLower(sectionTitle(sec)))
	if item.Version != "" {
		fmt.Fprintf(&b, "Version: %s\n", item.Version)
	}
	if item.Category != "" {
		fmt.Fprintf(&b, "Category: %s\n", item.Category)
	}
	if item.Flags != "" {
		fmt.Fprintf(&b, "Flags: %s\n", item.Flags)
	}
	b.WriteString("\nActions\n")
	switch sec {
	case sectionProfiles:
		b.WriteString("  v Validate\n  i Install\n")
	case sectionAddons:
		b.WriteString("  v Validate\n  i Install requires --profile, --kubeconfig or --cluster-context in CLI\n")
	case sectionStacks:
		b.WriteString("  Install and export from CLI with explicit flags\n")
	case sectionClusters:
		b.WriteString("  t Test kubectl access\n  K Open K9s\n")
	default:
		b.WriteString("  No direct TUI action is currently exposed.\n")
	}
	return b.String()
}

func renderCommandResult(res CommandResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "$ pk3s %s\n\nExit code: %d\n\n", strings.Join(res.Args, " "), res.Code)
	if strings.TrimSpace(res.Stdout) != "" {
		b.WriteString("Output\n")
		b.WriteString(strings.TrimSpace(res.Stdout))
		b.WriteString("\n\n")
	}
	if strings.TrimSpace(res.Stderr) != "" {
		b.WriteString("Errors\n")
		b.WriteString(strings.TrimSpace(res.Stderr))
	}
	return b.String()
}

func (m Model) operationView() string {
	var b strings.Builder
	title := m.operationTitle
	if title == "" {
		title = m.status
	}
	fmt.Fprintf(&b, "Status: %s\n", title)
	if len(m.operationArgs) > 0 {
		fmt.Fprintf(&b, "Command: pk3s %s\n", strings.Join(m.operationArgs, " "))
	}
	if m.operationStarted != "" {
		fmt.Fprintf(&b, "Started: %s\n", m.operationStarted)
	}
	b.WriteString("\nProgress\n")
	if len(m.operationSteps) == 0 {
		b.WriteString("  ● Running command\n")
	} else {
		for _, step := range m.operationSteps {
			fmt.Fprintf(&b, "  %s %s\n", statusSymbol(step.Status), step.Name)
		}
	}
	if strings.TrimSpace(m.logs) != "" {
		b.WriteString("\nRecent Logs\n")
		for _, line := range tailLines(m.logs, 8) {
			fmt.Fprintf(&b, "  %s\n", line)
		}
	}
	b.WriteString("\nLogs will open automatically when the command finishes. Esc returns to the catalog.")
	return b.String()
}

func (m Model) logsView() string {
	if strings.TrimSpace(m.logs) == "" {
		return "No logs captured."
	}
	return m.logViewport.View()
}

func statusSymbol(status string) string {
	switch status {
	case "success":
		return "✓"
	case "failed":
		return "✗"
	case "warning":
		return "!"
	case "running":
		return "●"
	default:
		return "○"
	}
}

func tailLines(text string, limit int) []string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) <= limit {
		return lines
	}
	return lines[len(lines)-limit:]
}

func helpView() string {
	return `Navigation
  up/down or j/k    move selection
  left/right/tab    change section
  enter or d        refresh details
  v                 validate selected item
  i                 install selected profile
  t                 test selected cluster with kubectl
  K                 open K9s for selected cluster
  r                 refresh section
  pgup/pgdn         scroll logs
  home/end          jump logs
  esc               back
  q                 quit

Scope
  This TUI calls existing pk3s commands. It does not browse Kubernetes resources or replace K9s.`
}

func footerText(current mode, sec section) string {
	if current == modeLogs {
		return "↑↓ Scroll   PgUp/PgDn Scroll   Home/End Jump   Esc Back   q Back   ? Help"
	}
	if current == modeHelp {
		return "Esc Back   ? Back   q Back"
	}
	switch sec {
	case sectionProfiles:
		return "↑↓ Navigate   ←→ Section   Enter Details   v Validate   i Install   r Refresh   ? Help   q Quit"
	case sectionAddons:
		return "↑↓ Navigate   ←→ Section   Enter Details   v Validate   r Refresh   ? Help   q Quit"
	case sectionStacks:
		return "↑↓ Navigate   ←→ Section   Enter Details   r Refresh   ? Help   q Quit"
	case sectionClusters:
		return "↑↓ Navigate   ←→ Section   Enter Details   t Test   K K9s   r Refresh   ? Help   q Quit"
	default:
		return "↑↓ Navigate   ←→ Section   r Refresh   ? Help   q Quit"
	}
}

func sectionTitle(sec section) string {
	switch sec {
	case sectionProfiles:
		return "Profiles"
	case sectionStacks:
		return "Stacks"
	case sectionAddons:
		return "Add-ons"
	case sectionClusters:
		return "Clusters"
	default:
		return "Unknown"
	}
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	mutedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Background(lipgloss.Color("33"))
	panelStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).Padding(1, 2)
	footerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Border(lipgloss.NormalBorder(), true, false, false, false).BorderForeground(lipgloss.Color("240"))
)
