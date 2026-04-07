# TUI History Suggestion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add fish-style inline history suggestion and prefix-filtered history navigation to the TUI input in both menu and shell contexts without changing existing history storage.

**Architecture:** Extend `internal/tui/input.go` with transient suggestion and filtered-history state that sits on top of the existing context-sensitive history sources. Keep `Tab` completion unchanged in `internal/tui/app.go`, add explicit `right` acceptance for inline suggestions, and verify the behavior with focused input and app tests.

**Tech Stack:** Go, Bubble Tea, Bubbles `textinput`, Lip Gloss, Go test suite.

---

## File Map

- Modify: `internal/tui/input.go`
  - add helper logic for prefix matches, inline suggestion state, filtered history navigation, and custom rendering
- Modify: `internal/tui/app.go`
  - accept inline suggestion on `right` without changing `Tab` completion behavior
- Modify: `internal/tui/inputedit_test.go`
  - add app-level key routing tests for the new `right` behavior and `Tab` regression protection if it fits existing coverage
- Create: `internal/tui/input_history_test.go`
  - add focused tests for suggestion selection and filtered history navigation

## Task 1: Lock In Suggestion And Filter Rules With Tests

**Files:**
- Create: `internal/tui/input_history_test.go`
- Modify: `internal/tui/inputedit_test.go`

- [ ] **Step 1: Write failing tests for prefix suggestion selection in `internal/tui/input_history_test.go`**

```go
package tui

import "testing"

func TestInputSuggestionUsesMostRecentPrefixMatch(t *testing.T) {
	input := NewInput()
	input.menuHistory = []string{
		"help",
		"run whoami",
		"run rev bash",
		"sessions",
	}
	input.SetValue("run")

	got, ok := input.Suggestion()
	if !ok {
		t.Fatal("expected suggestion for prefix run")
	}
	if got != "run rev bash" {
		t.Fatalf("expected most recent prefix match, got %q", got)
	}
}

func TestInputSuggestionHiddenWhenExactMatchOrEmpty(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		input := NewInput()
		input.menuHistory = []string{"run whoami"}

		if _, ok := input.Suggestion(); ok {
			t.Fatal("expected no suggestion for empty input")
		}
	})

	t.Run("exact match", func(t *testing.T) {
		input := NewInput()
		input.menuHistory = []string{"run whoami"}
		input.SetValue("run whoami")

		if _, ok := input.Suggestion(); ok {
			t.Fatal("expected no suggestion when input already equals match")
		}
	})
}
```

- [ ] **Step 2: Write failing tests for filtered history navigation in `internal/tui/input_history_test.go`**

```go
func TestInputHistoryNavigationFiltersByPrefixAndRestoresTypedPrefix(t *testing.T) {
	input := NewInput()
	input.menuHistory = []string{
		"help",
		"run whoami",
		"sessions",
		"run rev bash",
		"use 2",
	}
	input.SetValue("run")

	input.HistoryUp()
	if got := input.Value(); got != "run rev bash" {
		t.Fatalf("expected newest run* entry, got %q", got)
	}

	input.HistoryUp()
	if got := input.Value(); got != "run whoami" {
		t.Fatalf("expected older run* entry, got %q", got)
	}

	input.HistoryDown()
	if got := input.Value(); got != "run rev bash" {
		t.Fatalf("expected newer run* entry, got %q", got)
	}

	input.HistoryDown()
	if got := input.Value(); got != "run" {
		t.Fatalf("expected original typed prefix restored, got %q", got)
	}
}

func TestInputHistoryNavigationUsesSessionHistoryInShellContext(t *testing.T) {
	input := NewInput()
	input.SetContext(ContextShell)
	input.SetSessionID(7)
	input.sessionHistory[7] = []string{
		"pwd",
		"ps aux",
		"python -c 'print(1)'",
		"ps -ef",
	}
	input.SetValue("ps")

	input.HistoryUp()
	if got := input.Value(); got != "ps -ef" {
		t.Fatalf("expected newest shell prefix match, got %q", got)
	}
}
```

- [ ] **Step 3: Write failing app-routing tests for `right` acceptance and `tab` preservation in `internal/tui/inputedit_test.go`**

```go
func TestAppInputModeRightAcceptsSuggestion(t *testing.T) {
	app := New(nil, "127.0.0.1:4444")
	app.splash = false
	app.context = ContextMenu
	app.input.menuHistory = []string{"run whoami", "run rev bash"}
	app.input.SetValue("run")

	app.updateInputMode(tea.KeyMsg{Type: tea.KeyRight})

	if got := app.input.Value(); got != "run rev bash" {
		t.Fatalf("expected right to accept suggestion, got %q", got)
	}
}

type tabPreservingExecutor struct{}

func (tabPreservingExecutor) ExecuteCommand(string) string                              { return "" }
func (tabPreservingExecutor) GetSelectedSessionID() int                                 { return 0 }
func (tabPreservingExecutor) SessionCount() int                                         { return 0 }
func (tabPreservingExecutor) GetSessionsForDisplay() string                             { return "" }
func (tabPreservingExecutor) GetActiveSessionDisplay() (string, string, string, bool)   { return "", "", "", false }
func (tabPreservingExecutor) GetSelectedSessionFlavor() string                          { return "" }
func (tabPreservingExecutor) SetSilent(bool)                                            {}
func (tabPreservingExecutor) SetNotifyFunc(func(string))                                {}
func (tabPreservingExecutor) SetNotifyBarFunc(func(string, int))                        {}
func (tabPreservingExecutor) SetSpinnerFunc(func(int, string), func(int), func(int, string)) {}
func (tabPreservingExecutor) SetShellOutputFunc(func(string, int, []byte))              {}
func (tabPreservingExecutor) SetSessionDisconnectFunc(func(int, string))                {}
func (tabPreservingExecutor) StartShellRelay(int, int) error                            { return nil }
func (tabPreservingExecutor) StopShellRelay()                                           {}
func (tabPreservingExecutor) WriteToShell(string) error                                 { return nil }
func (tabPreservingExecutor) ResizePTY(int, int)                                        {}
func (tabPreservingExecutor) CompleteInput(line string) string                          { return line + "/completed" }
func (tabPreservingExecutor) SetTransferProgressFunc(func(string, int, string, bool))   {}
func (tabPreservingExecutor) SetTransferDoneFunc(func(string, bool, error))             {}
func (tabPreservingExecutor) StartUpload(context.Context, string, string)               {}
func (tabPreservingExecutor) StartDownload(context.Context, string, string)             {}
func (tabPreservingExecutor) StartSpawn()                                               {}
func (tabPreservingExecutor) StartModule(string, []string)                              {}

func TestAppInputModeTabStillUsesExecutorCompletion(t *testing.T) {
	app := New(tabPreservingExecutor{}, "127.0.0.1:4444")
	app.splash = false
	app.context = ContextMenu
	app.input.SetValue("download path/to/f")

	app.updateInputMode(tea.KeyMsg{Type: tea.KeyTab})

	if got := app.input.Value(); got != "download path/to/f/completed" {
		t.Fatalf("expected tab completion path preserved, got %q", got)
	}
}
```

- [ ] **Step 4: Run the focused tests and verify they fail for the expected missing behavior**

Run: `go test ./internal/tui -run 'TestInputSuggestion|TestInputHistoryNavigation|TestAppInputModeRightAcceptsSuggestion|TestAppInputModeTabStillUsesExecutorCompletion' -v`

Expected: FAIL because `Input` does not yet expose suggestion behavior, filtered navigation, or right-arrow acceptance.

## Task 2: Implement Inline Suggestion And Filtered Navigation In `Input`

**Files:**
- Modify: `internal/tui/input.go`

- [ ] **Step 1: Add transient filtered-history state to `Input`**

```go
type Input struct {
	textinput textinput.Model
	prompt    string
	context   ContextMode
	width     int
	sessionID int
	bangMode  bool

	menuHistory    []string
	sessionHistory map[int][]string
	histIdx        int

	historyPrefix   string
	filteredHistory []string
	filteredHistIdx int
}
```

- [ ] **Step 2: Add shared helper methods for prefix matching and reset behavior**

```go
func (i *Input) resetHistoryNavigation() {
	i.histIdx = -1
	i.historyPrefix = ""
	i.filteredHistory = nil
	i.filteredHistIdx = -1
}

func (i *Input) currentSuggestion() (string, bool) {
	prefix := i.textinput.Value()
	if prefix == "" || i.textinput.Position() != len(prefix) {
		return "", false
	}

	hist := i.history()
	for idx := len(hist) - 1; idx >= 0; idx-- {
		entry := hist[idx]
		if strings.HasPrefix(entry, prefix) && entry != prefix {
			return entry, true
		}
	}

	return "", false
}

func (i *Input) Suggestion() (string, bool) {
	return i.currentSuggestion()
}

func (i *Input) filteredMatches(prefix string) []string {
	if prefix == "" {
		return nil
	}

	hist := i.history()
	matches := make([]string, 0, len(hist))
	for _, entry := range hist {
		if strings.HasPrefix(entry, prefix) {
			matches = append(matches, entry)
		}
	}
	return matches
}
```

- [ ] **Step 3: Make clears, submits, and context changes reset transient navigation state**

```go
func (i *Input) SetContext(ctx ContextMode) {
	i.context = ctx
	i.bangMode = false
	i.resetHistoryNavigation()
	// existing placeholder and prompt logic stays in place
}

func (i *Input) Clear() {
	i.textinput.SetValue("")
	i.resetHistoryNavigation()
}

func (i *Input) EnterBangMode() {
	i.bangMode = true
	i.resetHistoryNavigation()
	// existing prompt logic stays in place
}

func (i *Input) ExitBangMode() {
	i.bangMode = false
	i.resetHistoryNavigation()
	// existing prompt logic stays in place
}
```

- [ ] **Step 4: Change `HistoryUp` and `HistoryDown` to use prefix-filtered navigation when input is non-empty**

```go
func (i *Input) HistoryUp() {
	prefix := i.textinput.Value()
	if prefix != "" {
		if i.filteredHistory == nil || i.historyPrefix != prefix {
			i.historyPrefix = prefix
			i.filteredHistory = i.filteredMatches(prefix)
			i.filteredHistIdx = len(i.filteredHistory)
		}
		if len(i.filteredHistory) == 0 {
			return
		}
		if i.filteredHistIdx > 0 {
			i.filteredHistIdx--
		}
		if i.filteredHistIdx >= 0 && i.filteredHistIdx < len(i.filteredHistory) {
			i.textinput.SetValue(i.filteredHistory[i.filteredHistIdx])
			i.textinput.CursorEnd()
		}
		return
	}

	// existing full-history navigation path remains here
}

func (i *Input) HistoryDown() {
	if i.filteredHistory != nil {
		if i.filteredHistIdx < len(i.filteredHistory)-1 {
			i.filteredHistIdx++
			i.textinput.SetValue(i.filteredHistory[i.filteredHistIdx])
			i.textinput.CursorEnd()
			return
		}
		i.textinput.SetValue(i.historyPrefix)
		i.textinput.CursorEnd()
		i.resetHistoryNavigation()
		return
	}

	// existing full-history navigation path remains here
}
```

- [ ] **Step 5: Add explicit suggestion acceptance and custom rendering with subtle suffix styling**

```go
func (i *Input) AcceptSuggestion() bool {
	suggestion, ok := i.currentSuggestion()
	if !ok {
		return false
	}
	i.textinput.SetValue(suggestion)
	i.textinput.CursorEnd()
	i.resetHistoryNavigation()
	return true
}

func (i *Input) View() string {
	base := i.textinput.View()
	suggestion, ok := i.currentSuggestion()
	if !ok {
		return i.prompt + base
	}
	value := i.textinput.Value()
	suffix := strings.TrimPrefix(suggestion, value)
	if suffix == "" {
		return i.prompt + base
	}
	return i.prompt + base + styleSubtle.Render(suffix)
}
```

- [ ] **Step 6: Run the focused tests and verify they pass**

Run: `go test ./internal/tui -run 'TestInputSuggestion|TestInputHistoryNavigation' -v`

Expected: PASS.

## Task 3: Accept Suggestion On Right Arrow Without Breaking `Tab`

**Files:**
- Modify: `internal/tui/app.go`
- Modify: `internal/tui/inputedit_test.go`

- [ ] **Step 1: Add explicit `right` handling to `updateInputMode` before the default textinput update path**

```go
case "right":
	if a.input.AcceptSuggestion() {
		return a, nil
	}

	var cmd tea.Cmd
	a.input, cmd = a.input.Update(msg)
	return a, cmd
```

- [ ] **Step 2: Keep the existing `tab` path unchanged and verify the regression test still describes that contract**

```go
case "tab":
	if a.context == ContextMenu || a.input.InBangMode() {
		current := a.input.Value()
		completed := a.executor.CompleteInput(current)
		if completed != current {
			a.input.SetValue(completed)
		}
	}
	return a, nil
```

- [ ] **Step 3: Run the focused app-routing tests**

Run: `go test ./internal/tui -run 'TestAppInputModeRightAcceptsSuggestion|TestAppInputModeTabStillUsesExecutorCompletion' -v`

Expected: PASS.

## Task 4: Run Broader Verification And Update Status Handoff

**Files:**
- Modify: `docs/current-status.md`

- [ ] **Step 1: Add a short handoff note to `docs/current-status.md` after implementation lands**

```md
- Input history suggestion now shows the most recent prefix-matching command in subtle gray.
- Right arrow accepts the inline suggestion.
- Up/down history navigation now filters by the typed prefix in both menu and shell inputs.
```

- [ ] **Step 2: Run the broader TUI test suite**

Run: `go test ./internal/tui/...`

Expected: PASS.

- [ ] **Step 3: Run the broader internal suite**

Run: `go test ./internal/...`

Expected: PASS.

- [ ] **Step 4: Build the app**

Run: `go build -o flame .`

Expected: PASS.

- [ ] **Step 5: Commit the feature work**

```bash
git add docs/current-status.md internal/tui/input.go internal/tui/app.go internal/tui/inputedit_test.go internal/tui/input_history_test.go
git commit -m "feat: add tui history suggestions"
```

- [ ] **Step 6: Stop for manual verification checkpoint**

Please test in the real app before any further polish:

1. Launch `./flame`, type `run` in menu mode, confirm a gray `run*` suggestion appears, then press `Right` to accept it.
2. In menu mode, type `run` and use `Up` and `Down`; confirm only `run*` commands are traversed and the bare `run` prefix is restored when you return past the newest match.
3. Attach to a shell with `F12`, run a few shell commands with a shared prefix like `ps`, then type `ps` and verify the same filtered-history and inline-suggestion behavior works there.

Expected behavior:

- `Right` accepts the gray suggestion without changing `Tab` completion behavior.
- `Tab` still performs command/path completion exactly as before.
- Menu and shell use the same prefix rules while keeping their existing separate history stores.
