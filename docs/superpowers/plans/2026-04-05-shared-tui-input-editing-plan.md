# Shared TUI Input Editing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add shared terminal-like line editing to Flame's TUI inputs with `Home`, `End`, `Ctrl+Backspace`, `Ctrl+Delete`, and `Ctrl+Z`.

**Architecture:** Keep Bubble Tea's `textinput.Model`, but add a focused helper in `internal/tui/` that applies line-edit actions to that model. `internal/tui/input.go` integrates the helper for the current menu and attached-shell input, while future inputs such as the help modal can reuse the same helper without duplicating behavior.

**Tech Stack:** Go, Bubble Tea `textinput.Model`, existing TUI input component, Go unit tests

---

## File Structure Map

- Create: `internal/tui/inputedit.go` - shared line-edit helpers for cursor jumps, word deletion, and input clearing
- Create: `internal/tui/inputedit_test.go` - unit tests for Home/End, word deletion semantics, and input clearing
- Modify: `internal/tui/input.go` - intercept relevant key events and delegate to the shared helper before normal textinput updates
- Modify: `internal/tui/app.go` - only if needed to keep global key routing from stealing input-level editing keys

### Task 1: Add shared line-edit helper with test coverage

**Files:**
- Create: `internal/tui/inputedit.go`
- Create: `internal/tui/inputedit_test.go`

- [ ] **Step 1: Write a failing test for `Home` and `End` cursor jumps**

Create `internal/tui/inputedit_test.go` with:

```go
package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
)

func newTestInput(value string, cursor int) textinput.Model {
	ti := textinput.New()
	ti.SetValue(value)
	for range cursor {
		ti.CursorRight()
	}
	return ti
}

func TestApplyLineEditHomeAndEnd(t *testing.T) {
	t.Run("home", func(t *testing.T) {
		ti := newTestInput("hello world", 5)
		if !applyLineEdit(&ti, "home") {
			t.Fatal("expected home to be handled")
		}
		if ti.Position() != 0 {
			t.Fatalf("expected cursor 0, got %d", ti.Position())
		}
	})

	t.Run("end", func(t *testing.T) {
		ti := newTestInput("hello world", 2)
		if !applyLineEdit(&ti, "end") {
			t.Fatal("expected end to be handled")
		}
		if ti.Position() != len("hello world") {
			t.Fatalf("expected cursor at end, got %d", ti.Position())
		}
	})
}
```

- [ ] **Step 2: Run the focused test and verify it fails**

Run: `go test ./internal/tui -run TestApplyLineEditHomeAndEnd -v`

Expected: FAIL with `undefined: applyLineEdit`.

- [ ] **Step 3: Add failing tests for word deletion semantics**

Extend `internal/tui/inputedit_test.go` with:

```go
func TestApplyLineEditDeletePreviousWord(t *testing.T) {
	ti := newTestInput("alpha beta gamma", len("alpha beta "))
	if !applyLineEdit(&ti, "ctrl+backspace") {
		t.Fatal("expected ctrl+backspace to be handled")
	}
	if got := ti.Value(); got != "alpha gamma" {
		t.Fatalf("expected previous word deleted, got %q", got)
	}
}

func TestApplyLineEditDeleteNextWord(t *testing.T) {
	ti := newTestInput("alpha beta gamma", len("alpha "))
	if !applyLineEdit(&ti, "ctrl+delete") {
		t.Fatal("expected ctrl+delete to be handled")
	}
	if got := ti.Value(); got != "alpha gamma" {
		t.Fatalf("expected next word deleted, got %q", got)
	}
}
```

- [ ] **Step 4: Run the focused tests and verify they fail**

Run: `go test ./internal/tui -run 'TestApplyLineEdit(HomeAndEnd|DeletePreviousWord|DeleteNextWord)' -v`

Expected: FAIL with `undefined: applyLineEdit`.

- [ ] **Step 5: Implement the minimal helper**

Create `internal/tui/inputedit.go` with:

```go
package tui

import (
	"unicode"

	"github.com/charmbracelet/bubbles/textinput"
)

func applyLineEdit(input *textinput.Model, key string) bool {
	switch key {
	case "home":
		input.SetCursor(0)
		return true
	case "end":
		input.CursorEnd()
		return true
	case "ctrl+backspace":
		deletePreviousWord(input)
		return true
	case "ctrl+delete":
		deleteNextWord(input)
		return true
	case "ctrl+z":
		input.SetValue("")
		input.SetCursor(0)
		return true
	default:
		return false
	}
}

func deletePreviousWord(input *textinput.Model) {
	value := []rune(input.Value())
	pos := input.Position()
	if pos == 0 {
		return
	}
	start := pos
	for start > 0 && unicode.IsSpace(value[start-1]) {
		start--
	}
	for start > 0 && !unicode.IsSpace(value[start-1]) {
		start--
	}
	input.SetValue(string(append(value[:start], value[pos:]...)))
	input.SetCursor(start)
}

func deleteNextWord(input *textinput.Model) {
	value := []rune(input.Value())
	pos := input.Position()
	if pos >= len(value) {
		return
	}
	end := pos
	for end < len(value) && unicode.IsSpace(value[end]) {
		end++
	}
	for end < len(value) && !unicode.IsSpace(value[end]) {
		end++
	}
	input.SetValue(string(append(value[:pos], value[end:]...)))
	input.SetCursor(pos)
}
```

- [ ] **Step 6: Re-run the helper tests**

Run: `go test ./internal/tui -run 'TestApplyLineEdit(HomeAndEnd|DeletePreviousWord|DeleteNextWord)' -v`

Expected: PASS.

### Task 2: Integrate shared editing into the current TUI input component

**Files:**
- Modify: `internal/tui/input.go`
- Test: `internal/tui/inputedit_test.go`

- [ ] **Step 1: Write a failing integration test for the input component**

Extend `internal/tui/inputedit_test.go` with:

```go
func TestInputUpdateUsesSharedLineEdit(t *testing.T) {
	input := NewInput()
	input.SetValue("alpha beta gamma")
	input.textinput.SetCursor(len("alpha "))

	updated, _ := input.Update(tea.KeyMsg{Type: tea.KeyCtrlDelete})
	if got := updated.Value(); got != "alpha gamma" {
		t.Fatalf("expected shared line edit in input component, got %q", got)
	}
}
```

Also add the import in the test file:

```go
tea "github.com/charmbracelet/bubbletea"
```

- [ ] **Step 2: Run the focused integration test and verify it fails**

Run: `go test ./internal/tui -run TestInputUpdateUsesSharedLineEdit -v`

Expected: FAIL because the input component still falls through to default textinput behavior.

- [ ] **Step 3: Wire the helper into `Input.Update`**

Update `internal/tui/input.go`:

```go
func (i *Input) Update(msg tea.Msg) (*Input, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if applyLineEdit(&i.textinput, keyMsg.String()) {
			return i, nil
		}
	}
	var cmd tea.Cmd
	i.textinput, cmd = i.textinput.Update(msg)
	return i, cmd
}
```

- [ ] **Step 4: Re-run the integration test**

Run: `go test ./internal/tui -run TestInputUpdateUsesSharedLineEdit -v`

Expected: PASS.

### Task 3: Verify boundaries and current routing behavior

**Files:**
- Modify: `internal/tui/inputedit_test.go`
- Modify: `internal/tui/app.go` only if verification shows routing interference

- [ ] **Step 1: Add failing boundary tests for no-op behavior**

Extend `internal/tui/inputedit_test.go` with:

```go
func TestApplyLineEditBoundariesNoOp(t *testing.T) {
	t.Run("ctrl+backspace at start", func(t *testing.T) {
		ti := newTestInput("alpha", 0)
		applyLineEdit(&ti, "ctrl+backspace")
		if got := ti.Value(); got != "alpha" || ti.Position() != 0 {
			t.Fatalf("expected no-op at start, got %q / %d", got, ti.Position())
		}
	})

	t.Run("ctrl+delete at end", func(t *testing.T) {
		ti := newTestInput("alpha", len("alpha"))
		applyLineEdit(&ti, "ctrl+delete")
		if got := ti.Value(); got != "alpha" || ti.Position() != len("alpha") {
			t.Fatalf("expected no-op at end, got %q / %d", got, ti.Position())
		}
	})
}
```

- [ ] **Step 2: Add a failing test for clearing the entire input**

Extend `internal/tui/inputedit_test.go` with:

```go
func TestApplyLineEditClearInput(t *testing.T) {
	ti := newTestInput("alpha beta", len("alpha"))
	if !applyLineEdit(&ti, "ctrl+z") {
		t.Fatal("expected ctrl+z to be handled")
	}
	if got := ti.Value(); got != "" || ti.Position() != 0 {
		t.Fatalf("expected cleared input, got %q / %d", got, ti.Position())
	}
}
```

- [ ] **Step 3: Run the boundary and clear tests and verify current behavior**

Run: `go test ./internal/tui -run 'TestApplyLineEdit(BoundariesNoOp|ClearInput)' -v`

Expected: PASS if helper already handles boundaries correctly; if not, FAIL and then fix helper until green.

- [ ] **Step 4: Run the full TUI package tests**

Run: `go test ./internal/tui/...`

Expected: PASS.

- [ ] **Step 5: Run the full project tests and build**

Run: `go test ./... && go build -o flame .`

Expected: PASS.
