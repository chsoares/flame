# TUI Help Overlay Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an `F1` help overlay to the Bubble Tea TUI, backed by the existing help registry, and migrate the quit dialog to the same transparent overlay shell without changing terminal `help` behavior.

**Architecture:** Build a small shared overlay renderer in `internal/tui` and keep modal-specific logic in dedicated models. Extend `internal/help.go` with a structured accessor for canonical help topics so the TUI can browse the existing registry directly while the CLI keeps its current renderers unchanged.

**Tech Stack:** Go, Bubble Tea, Lip Gloss, existing `internal/help.go` registry, Go test suite.

---

## File Map

- Modify: `internal/help.go`
- Modify: `internal/help_test.go`
- Create: `internal/tui/overlay.go`
- Create: `internal/tui/overlay_test.go`
- Create: `internal/tui/helpmodal.go`
- Create: `internal/tui/helpmodal_test.go`
- Create: `internal/tui/app_help_test.go`
- Modify: `internal/tui/dialog.go`
- Modify: `internal/tui/app.go`
- Modify: `internal/tui/statusbar.go`
- Create: `internal/tui/statusbar_test.go`
- Modify: `docs/current-status.md`

### Task 1: Expose help registry data for the TUI

**Files:**
- Modify: `internal/help.go`
- Modify: `internal/help_test.go`

- [ ] **Step 1: Write the failing tests for canonical TUI topics**

Add tests near the end of `internal/help_test.go` for a new helper that exposes canonical topics without aliases as separate rows.

```go
func TestHelpTopicsForTUIUsesCanonicalTopics(t *testing.T) {
	topics := HelpTopicsForTUI()
	checks := []string{"rev", "run", "run ps1", "binbag", "pivot", "exit"}
	for _, check := range checks {
		found := false
		for _, entry := range topics {
			if entry.Topic == check {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected canonical topic %q in TUI topics: %#v", check, topics)
		}
	}
}

func TestHelpTopicsForTUIDoesNotDuplicateAliases(t *testing.T) {
	topics := HelpTopicsForTUI()
	for _, alias := range []string{"rev bash", "rev sh", "run bash", "list"} {
		for _, entry := range topics {
			if entry.Topic == alias {
				t.Fatalf("did not expect alias %q as standalone TUI topic", alias)
			}
		}
	}
}
```

- [ ] **Step 2: Run the focused help tests and verify they fail**

Run: `go test ./internal -run 'TestHelpTopicsForTUI|TestHelpTopicsForTUIDoesNotDuplicateAliases'`

Expected: FAIL with `undefined: HelpTopicsForTUI`.

- [ ] **Step 3: Implement the minimal structured accessor in `internal/help.go`**

Add a helper that returns canonical entries in registry order and leaves the existing render functions alone.

```go
func HelpTopicsForTUI() []HelpEntry {
	topics := make([]HelpEntry, 0, len(helpEntries))
	for _, entry := range helpEntries {
		topics = append(topics, entry)
	}
	return topics
}
```

If later tasks need stronger isolation, add a small copy helper so callers cannot mutate shared slices:

```go
func copyHelpEntry(entry HelpEntry) HelpEntry {
	cloned := entry
	cloned.Aliases = append([]string(nil), entry.Aliases...)
	cloned.Usage = append([]string(nil), entry.Usage...)
	cloned.Details = append([]string(nil), entry.Details...)
	cloned.Examples = append([]string(nil), entry.Examples...)
	return cloned
}
```

Then return copies:

```go
func HelpTopicsForTUI() []HelpEntry {
	topics := make([]HelpEntry, 0, len(helpEntries))
	for _, entry := range helpEntries {
		topics = append(topics, copyHelpEntry(entry))
	}
	return topics
}
```

- [ ] **Step 4: Re-run the focused help tests and verify they pass**

Run: `go test ./internal -run 'TestHelpTopicsForTUI|TestHelpTopicsForTUIDoesNotDuplicateAliases'`

Expected: PASS.

- [ ] **Step 5: Checkpoint the task without committing**

Run: `go test ./internal -run 'TestHelpTopicsForTUI|TestHelpTopicsForTUIDoesNotDuplicateAliases|TestLookupHelpTopic|TestRenderHelpTopic'`

Expected: PASS. Do not create a git commit unless the user explicitly asks.

### Task 2: Build the shared transparent overlay shell and restyle quit

**Files:**
- Create: `internal/tui/overlay.go`
- Create: `internal/tui/overlay_test.go`
- Modify: `internal/tui/dialog.go`

- [ ] **Step 1: Write the failing overlay-shell tests**

Create `internal/tui/overlay_test.go` with focused rendering checks for the shared shell and the quit dialog title.

```go
package tui

import (
	"strings"
	"testing"
)

func TestRenderOverlayKeepsUnderlyingScreenVisible(t *testing.T) {
	base := strings.Repeat("x", 40) + "\n" + strings.Repeat("y", 40)
	got := RenderOverlay(40, 8, base, OverlayBox{
		Title: "help",
		Body:  "body",
		Width: 20,
	})
	if !strings.Contains(got, "help") {
		t.Fatal("expected overlay title in rendered output")
	}
	if !strings.Contains(got, "x") || !strings.Contains(got, "y") {
		t.Fatal("expected overlay to preserve visible base screen around modal")
	}
}

func TestConfirmQuitDialogUsesLowercaseTitle(t *testing.T) {
	d := confirmQuitDialog(2)
	if d.Title != "quit" {
		t.Fatalf("expected lowercase quit title, got %q", d.Title)
	}
}
```

- [ ] **Step 2: Run the focused overlay tests and verify they fail**

Run: `go test ./internal/tui -run 'TestRenderOverlayKeepsUnderlyingScreenVisible|TestConfirmQuitDialogUsesLowercaseTitle'`

Expected: FAIL with `undefined: RenderOverlay` and the quit-title assertion failing.

- [ ] **Step 3: Implement the shared overlay shell in `internal/tui/overlay.go`**

Create a small overlay renderer with a shared title bar and body composition.

```go
package tui

import "github.com/charmbracelet/lipgloss"

type OverlayAlign int

const (
	OverlayAlignLeft OverlayAlign = iota
	OverlayAlignCenter
)

type OverlayBox struct {
	Title string
	Body  string
	Width int
	Align OverlayAlign
}

func RenderOverlay(termW, termH int, base string, box OverlayBox) string {
	if termW <= 0 || termH <= 0 {
		return base
	}
	width := box.Width
	if width <= 0 {
		width = 72
	}
	if width > termW-4 {
		width = termW - 4
	}
	if width < 16 {
		width = termW
	}

	bodyWidth := width - 4
	if bodyWidth < 1 {
		bodyWidth = 1
	}
	bodyStyle := lipgloss.NewStyle().Width(bodyWidth)
	if box.Align == OverlayAlignCenter {
		bodyStyle = bodyStyle.Align(lipgloss.Center)
	}

	header := overlayTitleBar(box.Title, bodyWidth)
	body := bodyStyle.Render(box.Body)
	panel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorMagenta).
		Padding(0, 1).
		Width(bodyWidth).
		Render(header + "\n" + body)

	placed := lipgloss.Place(termW, termH, lipgloss.Center, lipgloss.Center, panel)
	return lipgloss.PlaceOverlay(0, 0, placed, base)
}

func overlayTitleBar(title string, width int) string {
	left := styleMagentaBold.Render(title)
	fillWidth := width - lipgloss.Width(left) - 1
	if fillWidth < 0 {
		fillWidth = 0
	}
	return left + " " + hatching(fillWidth)
}
```

Use existing `max`/`min` helpers if they already exist in the package. If they do not, add tiny local clamps in this file instead of broad utility extraction.

- [ ] **Step 4: Migrate `internal/tui/dialog.go` to the shared shell**

Keep the current selection behavior, but switch the view path to the shared renderer, lowercase title, and centered body.

```go
func (d *Dialog) View(termW, termH int, base string) string {
	buttonRow := confirmBtn + "  " + cancelBtn
	hintRow := hint("Tab", "toggle") + dot + hint("Enter", "confirm") + dot + hint("Esc", "cancel")
	body := strings.Join([]string{
		"",
		styleBase.Render(d.SubMessage),
		"",
		buttonRow,
		"",
		hintRow,
		"",
	}, "\n")
	return RenderOverlay(termW, termH, base, OverlayBox{
		Title: d.Title,
		Body:  body,
		Width: 56,
		Align: OverlayAlignCenter,
	})
}

func confirmQuitDialog(sessionCount int) *Dialog {
	return &Dialog{
		Title:    "quit",
		Action:   DialogQuit,
		Selected: 1,
		// SubMessage remains the active-sessions warning.
	}
}
```

Preserve the existing confirm/cancel button visuals unless the new shell requires tiny padding adjustments.

- [ ] **Step 5: Re-run the overlay tests and verify they pass**

Run: `go test ./internal/tui -run 'TestRenderOverlayKeepsUnderlyingScreenVisible|TestConfirmQuitDialogUsesLowercaseTitle'`

Expected: PASS.

- [ ] **Step 6: Checkpoint the task without committing**

Run: `go test ./internal/tui -run 'TestRenderOverlayKeepsUnderlyingScreenVisible|TestConfirmQuitDialogUsesLowercaseTitle|TestHeader'`

Expected: PASS. Do not create a git commit unless the user explicitly asks.

### Task 3: Build the help modal model with list/detail navigation

**Files:**
- Create: `internal/tui/helpmodal.go`
- Create: `internal/tui/helpmodal_test.go`
- Modify: `internal/help.go` (only if Task 1 needs a stronger helper)

- [ ] **Step 1: Write the failing help-modal state tests**

Create `internal/tui/helpmodal_test.go` with state-transition and filtering tests.

```go
package tui

import (
	"testing"

	internal "github.com/chsoares/flame/internal"
	tea "github.com/charmbracelet/bubbletea"
)

func TestHelpModalFiltersTopicsByAliasAndSummary(t *testing.T) {
	m := NewHelpModal(internal.HelpTopicsForTUI())
	m.SetFilter("bash")
	if len(m.filtered) == 0 {
		t.Fatal("expected matches for bash alias")
	}
}

func TestHelpModalEnterOpensDetailAndBackspaceReturns(t *testing.T) {
	m := NewHelpModal(internal.HelpTopicsForTUI())
	m.selected = 0
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != helpModalDetail {
		t.Fatal("expected enter to open detail mode")
	}
	m.filter = ""
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.mode != helpModalList {
		t.Fatal("expected backspace on empty filter to return to list")
	}
}
```

- [ ] **Step 2: Run the focused help-modal tests and verify they fail**

Run: `go test ./internal/tui -run 'TestHelpModalFiltersTopicsByAliasAndSummary|TestHelpModalEnterOpensDetailAndBackspaceReturns'`

Expected: FAIL with `undefined: NewHelpModal` and related missing symbols.

- [ ] **Step 3: Implement the minimal help-modal model in `internal/tui/helpmodal.go`**

Create a model that owns filtering, selection, detail entry, and rendering.

```go
package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	internal "github.com/chsoares/flame/internal"
)

type helpModalMode int

const (
	helpModalList helpModalMode = iota
	helpModalDetail
)

type HelpModal struct {
	topics    []internal.HelpEntry
	filtered  []internal.HelpEntry
	filter    string
	selected  int
	mode      helpModalMode
	current   internal.HelpEntry
	width     int
	height    int
}

func NewHelpModal(topics []internal.HelpEntry) HelpModal {
	m := HelpModal{topics: topics}
	m.applyFilter()
	return m
}

func (m *HelpModal) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *HelpModal) SetFilter(value string) {
	m.filter = value
	m.applyFilter()
}

func (m HelpModal) Update(msg tea.KeyMsg) (HelpModal, tea.Cmd) {
	switch msg.String() {
	case "up":
		if m.mode == helpModalList && m.selected > 0 {
			m.selected--
		}
	case "down":
		if m.mode == helpModalList && m.selected < len(m.filtered)-1 {
			m.selected++
		}
	case "enter":
		if m.mode == helpModalList && len(m.filtered) > 0 {
			m.current = m.filtered[m.selected]
			m.mode = helpModalDetail
		}
	case "backspace":
		if m.mode == helpModalDetail && m.filter == "" {
			m.mode = helpModalList
			return m, nil
		}
		if m.mode == helpModalList && m.filter != "" {
			m.filter = m.filter[:len(m.filter)-1]
			m.applyFilter()
		}
	default:
		if m.mode == helpModalList && len(msg.Runes) > 0 {
			m.filter += string(msg.Runes)
			m.applyFilter()
		}
	}
	return m, nil
}

func (m *HelpModal) applyFilter() {
	query := strings.ToLower(strings.TrimSpace(m.filter))
	m.filtered = m.filtered[:0]
	for _, entry := range m.topics {
		if query == "" || helpEntryMatches(entry, query) {
			m.filtered = append(m.filtered, entry)
		}
	}
	if len(m.filtered) == 0 {
		m.selected = 0
		return
	}
	if m.selected >= len(m.filtered) {
		m.selected = 0
	}
}

func helpEntryMatches(entry internal.HelpEntry, query string) bool {
	if strings.Contains(strings.ToLower(entry.Topic), query) || strings.Contains(strings.ToLower(entry.Summary), query) {
		return true
	}
	for _, alias := range entry.Aliases {
		if strings.Contains(strings.ToLower(alias), query) {
			return true
		}
	}
	return false
}
```

Then add a `View(base string) string` path that uses Task 2's `RenderOverlay`:

```go
func (m HelpModal) View(termW, termH int, base string) string {
	body := m.renderListBody()
	if m.mode == helpModalDetail {
		body = m.renderDetailBody()
	}
	width := 84
	if width > termW-4 {
		width = termW - 4
	}
	return RenderOverlay(termW, termH, base, OverlayBox{
		Title: "help",
		Body:  body,
		Width: width,
		Align: OverlayAlignLeft,
	})
}
```

For detail rendering, reuse `internal.RenderHelpTopic(strings.Fields(m.current.Topic))` and strip nothing unless the box padding forces a tiny cleanup. The terminal renderer remains the source of truth for topic detail content.

- [ ] **Step 4: Re-run the focused help-modal tests and verify they pass**

Run: `go test ./internal/tui -run 'TestHelpModalFiltersTopicsByAliasAndSummary|TestHelpModalEnterOpensDetailAndBackspaceReturns'`

Expected: PASS.

- [ ] **Step 5: Add one render-level regression test for the help title and filter prompt**

Add this test to `internal/tui/helpmodal_test.go`:

```go
func TestHelpModalViewRendersTitleAndFilterPrompt(t *testing.T) {
	m := NewHelpModal(internal.HelpTopicsForTUI())
	got := m.View(80, 24, strings.Repeat(" ", 80*24))
	if !strings.Contains(got, "help") {
		t.Fatal("expected lowercase help title")
	}
	if !strings.Contains(got, "type to filter") {
		t.Fatal("expected filter prompt")
	}
}
```

Run: `go test ./internal/tui -run 'TestHelpModal'`

Expected: PASS.

- [ ] **Step 6: Checkpoint the task without committing**

Run: `go test ./internal/tui -run 'TestHelpModal|TestRenderOverlayKeepsUnderlyingScreenVisible'`

Expected: PASS. Do not create a git commit unless the user explicitly asks.

### Task 4: Wire the help modal into the app and update the status bar

**Files:**
- Modify: `internal/tui/app.go`
- Modify: `internal/tui/statusbar.go`
- Create: `internal/tui/statusbar_test.go`
- Create: `internal/tui/app_help_test.go`

- [ ] **Step 1: Write the failing status-bar and app wiring tests**

Create `internal/tui/statusbar_test.go` for the new hint order, and create `internal/tui/app_help_test.go` for the app-level modal wiring test and local executor stub.

```go
package tui

import (
	"strings"
	"testing"
)

func TestStatusBarShowsF1HelpFirstInMenuMode(t *testing.T) {
	bar := NewStatusBar(120)
	bar.Context = ContextMenu
	got := bar.View()
	if !strings.Contains(got, "F1") || !strings.Contains(got, "help") {
		t.Fatalf("expected F1 help hint, got %q", got)
	}
	if strings.Contains(got, "Tab") && strings.Contains(got, "complete") {
		t.Fatalf("did not expect Tab complete hint, got %q", got)
	}
}

```

Create `internal/tui/app_help_test.go` with a tiny local executor stub and the modal-open test:

```go
package tui

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

type stubExecutor struct{}

func (stubExecutor) ExecuteCommand(string) string { return "" }
func (stubExecutor) GetSelectedSessionID() int { return 0 }
func (stubExecutor) SessionCount() int { return 0 }
func (stubExecutor) GetSessionsForDisplay() string { return "" }
func (stubExecutor) GetActiveSessionDisplay() (string, string, string, bool) { return "", "", "", false }
func (stubExecutor) GetSelectedSessionFlavor() string { return "" }
func (stubExecutor) SetSilent(bool) {}
func (stubExecutor) SetNotifyFunc(func(string)) {}
func (stubExecutor) SetNotifyBarFunc(func(string, int)) {}
func (stubExecutor) SetSpinnerFunc(func(int, string), func(int), func(int, string)) {}
func (stubExecutor) SetShellOutputFunc(func(string, int, []byte)) {}
func (stubExecutor) SetSessionDisconnectFunc(func(int, string)) {}
func (stubExecutor) StartShellRelay(int, int) error { return nil }
func (stubExecutor) StopShellRelay() {}
func (stubExecutor) WriteToShell(string) error { return nil }
func (stubExecutor) ResizePTY(int, int) {}
func (stubExecutor) CompleteInput(line string) string { return line }
func (stubExecutor) SetTransferProgressFunc(func(string, int, string, bool)) {}
func (stubExecutor) SetTransferDoneFunc(func(string, bool, error)) {}
func (stubExecutor) StartUpload(context.Context, string, string) {}
func (stubExecutor) StartDownload(context.Context, string, string) {}
func (stubExecutor) StartSpawn() {}
func (stubExecutor) StartModule(string, []string) {}

func TestAppF1OpensHelpModal(t *testing.T) {
	a := New(stubExecutor{}, "127.0.0.1:4444")
	a.width = 120
	a.height = 30
	a.layout = GenerateLayout(120, 30)
	model, _ := a.Update(tea.KeyMsg{Type: tea.KeyF1})
	updated := model.(App)
	if updated.helpModal == nil {
		t.Fatal("expected F1 to open help modal")
	}
}
```

- [ ] **Step 2: Run the focused wiring tests and verify they fail**

Run: `go test ./internal/tui -run 'TestStatusBarShowsF1HelpFirstInMenuMode|TestAppF1OpensHelpModal'`

Expected: FAIL because `helpModal` is not wired and the status bar still shows `Tab complete`.

- [ ] **Step 3: Wire the help modal into `internal/tui/app.go`**

Add a dedicated app field and modal dispatch path instead of overloading `dialog`.

```go
type App struct {
	// ...existing fields...
	dialog    *Dialog
	helpModal *HelpModal
	// ...existing fields...
}
```

Intercept help-modal keys before normal input handling:

```go
case tea.KeyMsg:
	if a.helpModal != nil {
		return a.updateHelpModal(msg)
	}
	if a.dialog != nil {
		return a.updateDialog(msg)
	}
```

Handle `F1` in `updateInputMode`:

```go
case "f1":
	modal := NewHelpModal(internal.HelpTopicsForTUI())
	modal.SetSize(a.width, a.height)
	a.helpModal = &modal
	return a, nil
```

Add a small updater:

```go
func (a App) updateHelpModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "escape" {
		a.helpModal = nil
		return a, nil
	}
	updated, cmd := a.helpModal.Update(msg)
	a.helpModal = &updated
	return a, cmd
}
```

Render help before quit in `View()`:

```go
if a.helpModal != nil {
	return a.helpModal.View(a.width, a.height, result)
}
if a.dialog != nil {
	return a.dialog.View(a.width, a.height, result)
}
```

Also set the help modal size on `tea.WindowSizeMsg` when it is open.

- [ ] **Step 4: Update the status bar hint order**

Modify the non-shell and shell left-side hint strings in `internal/tui/statusbar.go`.

```go
if s.Context == ContextShell {
	left = hint("F1", "help") + dot +
		hint("!", "flame cmd") + dot +
		hint("F11", "sidebar") + dot +
		hint("F12", "detach") + dot +
		hint("PgUp/PgDn", "scroll") + dot +
		hint("Ctrl+C", "interrupt") + dot +
		hint("Ctrl+D", "quit")
} else {
	left = hint("F1", "help") + dot +
		hint("F11", "sidebar") + dot +
		hint("F12", "attach") + dot +
		hint("PgUp/PgDn", "scroll") + dot +
		hint("Ctrl+C", "cancel") + dot +
		hint("Ctrl+D", "quit")
}
```

- [ ] **Step 5: Re-run the focused wiring tests and verify they pass**

Run: `go test ./internal/tui -run 'TestStatusBarShowsF1HelpFirstInMenuMode|TestAppF1OpensHelpModal'`

Expected: PASS.

- [ ] **Step 6: Checkpoint the task without committing**

Run: `go test ./internal/tui -run 'TestStatusBarShowsF1HelpFirstInMenuMode|TestAppF1OpensHelpModal|TestHelpModal'`

Expected: PASS. Do not create a git commit unless the user explicitly asks.

### Task 5: Update handoff docs and run full verification

**Files:**
- Modify: `docs/current-status.md`

- [ ] **Step 1: Write the failing doc expectation as a manual checklist**

Before editing docs, confirm the implementation is complete enough that the handoff note can truthfully say the TUI help overlay exists and the quit overlay shares the same shell.

Checklist:

```text
- F1 opens help in the TUI
- help list/detail works
- input-line help still prints to terminal output
- quit uses the same overlay shell
- status bar advertises F1 help
```

If any item is false, finish code before editing docs.

- [ ] **Step 2: Update `docs/current-status.md`**

Change the help section and roadmap notes so the file reflects the completed TUI overlay work.

Use edits like:

```md
### Terminal help revamp
- [x] compact `help` index with grouped categories
- [x] `help <command>` detail pages with shared help registry
- [x] tab completion for help topics, including nested `run` topics
- [x] `binbag` and `pivot` grouped under `network` in the general help
- [x] TUI help overlay with shared registry-backed content and transparent modal shell
```

And update the roadmap text so it no longer says the TUI help phase is pending.

- [ ] **Step 3: Run package tests before full-suite verification**

Run: `go test ./internal ./internal/tui`

Expected: PASS.

- [ ] **Step 4: Run the full test suite**

Run: `go test ./...`

Expected: PASS.

- [ ] **Step 5: Build the binary**

Run: `go build -o flame .`

Expected: successful build with no errors.

- [ ] **Step 6: Final checkpoint without committing**

Run: `git status --short`

Expected: only the intended code, test, and doc changes are present. Do not create a git commit unless the user explicitly asks.

## Self-Review

- Spec coverage: covered shared overlay shell, `F1` help modal, list/detail flow, Backspace return behavior, lowercase titles, centered quit body, status bar hint changes, docs update, and final verification.
- Placeholder scan: no `TODO`, `TBD`, or "similar to previous task" shortcuts remain.
- Type consistency: plan consistently uses `HelpModal`, `RenderOverlay`, `OverlayBox`, `OverlayAlignLeft`, and `OverlayAlignCenter` across tasks.
