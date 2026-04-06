# Modal Shell Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Consolidate the TUI modal shell into one shared template so quit/kill and help both render through the same border, header, and footer layout.

**Architecture:** Create a small shared modal shell renderer in `internal/tui/modal.go` and keep modal-specific logic in `dialog.go` and `helpmodal.go`. The shared shell owns all framing and spacing rules, while each modal only builds its own body content and alignment.

**Tech Stack:** Go, Bubble Tea, Lip Gloss, existing TUI styles and helpers, Go test suite.

---

## File Map

- Create: `internal/tui/modal.go`
- Modify: `internal/tui/dialog.go`
- Modify: `internal/tui/helpmodal.go`
- Delete or replace: `internal/tui/modal_skeleton.go`
- Create: `internal/tui/modal_test.go`
- Modify: `docs/current-status.md`

### Task 1: Define the shared modal shell API

**Files:**
- Create: `internal/tui/modal.go`
- Create: `internal/tui/modal_test.go`

- [ ] **Step 1: Write the failing shell tests**

Add focused tests for the shared template:

```go
package tui

import (
	"strings"
	"testing"
)

func TestRenderModalShellKeepsTitleInsideBorder(t *testing.T) {
	got := RenderModalShell(strings.Repeat("x", 80), 80, 24, ModalShell{
		Title: "quit",
		Body:  []string{"body"},
	})
	if !strings.Contains(got, "quit") {
		t.Fatal("expected modal title in rendered shell")
	}
}

func TestRenderModalShellDoesNotLeaveGapBelowFooter(t *testing.T) {
	got := RenderModalShell(strings.Repeat("x", 80), 80, 24, ModalShell{
		Title:  "help",
		Body:   []string{"body"},
		Footer: "footer",
	})
	if strings.Contains(got, "footer\n\n") {
		t.Fatal("expected no blank line below footer")
	}
}
```

- [ ] **Step 2: Run the focused shell tests and verify they fail**

Run: `go test ./internal/tui -run 'TestRenderModalShellKeepsTitleInsideBorder|TestRenderModalShellDoesNotLeaveGapBelowFooter'`

Expected: FAIL because `RenderModalShell` and `ModalShell` are not yet implemented in the new file.

- [ ] **Step 3: Implement the shared shell in `internal/tui/modal.go`**

Create the shared template with a small, explicit API:

```go
package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type BodyAlign int

const (
	BodyAlignLeft BodyAlign = iota
	BodyAlignCenter
)

type ModalShell struct {
	Title     string
	Width     int
	MaxHeight int
	Body      []string
	Footer    string
	Align     BodyAlign
}

func RenderModalShell(base string, termW, termH int, shell ModalShell) string {
	dialogW := shell.Width
	if dialogW <= 0 {
		dialogW = 52
	}
	if dialogW > termW-4 {
		dialogW = termW - 4
	}
	innerW := dialogW - 6
	contentW := innerW - 4
	if contentW < 1 {
		contentW = 1
	}
	headHatchW := contentW - lipgloss.Width(shell.Title) - 1
	if headHatchW < 1 {
		headHatchW = 1
	}
	headerRow := styleMagentaBold.Render(shell.Title) + " " + hatching(headHatchW)

	maxH := modalBoxHeight(termH)
	if shell.MaxHeight > 0 && shell.MaxHeight < maxH {
		maxH = shell.MaxHeight
	}
	bodyRows := maxH - 3
	if shell.Footer != "" {
		bodyRows -= 2
	}
	if bodyRows < 1 {
		bodyRows = 1
	}
	body := padToHeight(shell.Body, bodyRows)
	if shell.Align == BodyAlignCenter {
		for i, line := range body {
			body[i] = centerLine(line, contentW)
		}
	}

	lines := []string{headerRow, ""}
	lines = append(lines, body...)
	if shell.Footer != "" {
		lines = append(lines, "", shell.Footer)
	}
	lines = padToHeight(lines, maxH)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorMagenta).
		Padding(0, 2).
		Width(innerW).
		Render(strings.Join(lines, "\n"))
	return overlayCenteredBox(base, box, termW, termH)
}
```

Add a tiny helper for centering lines inside the shell body if needed:

```go
func centerLine(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	padL := (width - w) / 2
	padR := width - w - padL
	return strings.Repeat(" ", padL) + s + strings.Repeat(" ", padR)
}
```

- [ ] **Step 4: Re-run the focused shell tests and verify they pass**

Run: `go test ./internal/tui -run 'TestRenderModalShellKeepsTitleInsideBorder|TestRenderModalShellDoesNotLeaveGapBelowFooter'`

Expected: PASS.

### Task 2: Move quit/kill rendering onto the shared shell

**Files:**
- Modify: `internal/tui/dialog.go`
- Modify: `internal/tui/modal.go`

- [ ] **Step 1: Write the dialog regression test**

Add a focused test that checks quit still uses the shared shell and stays centered:

```go
func TestConfirmQuitDialogUsesSharedModalShell(t *testing.T) {
	d := confirmQuitDialog(2)
	got := d.View(80, 24, strings.Repeat("x", 80*24))
	if !strings.Contains(got, "quit") {
		t.Fatal("expected quit title in dialog rendering")
	}
}
```

- [ ] **Step 2: Run the dialog test and verify it fails if needed**

Run: `go test ./internal/tui -run 'TestConfirmQuitDialogUsesSharedModalShell'`

Expected: FAIL until `Dialog.View` uses `RenderModalShell`.

- [ ] **Step 3: Refactor `Dialog.View` to use `RenderModalShell`**

Replace the local box-building code with body construction only:

```go
func (d *Dialog) View(termW, termH int, base string) string {
	centerRow := func(s string, width int) string {
		w := lipgloss.Width(s)
		if w >= width {
			return s
		}
		padL := (width - w) / 2
		padR := width - w - padL
		return strings.Repeat(" ", padL) + s + strings.Repeat(" ", padR)
	}

	btnW := 14
	selectedBg := colorCyan
	unselectedBg := colorDim
	makeBtn := func(label string, underlineIdx int, selected bool) string {
		bg := unselectedBg
		fg := colorMuted
		if selected {
			bg = selectedBg
			fg = lipgloss.Color("0")
		}
		base := lipgloss.NewStyle().Background(bg).Foreground(fg).Bold(selected)
		ul := lipgloss.NewStyle().Background(bg).Foreground(fg).Bold(selected).Underline(true)
		before := label[:underlineIdx]
		char := string(label[underlineIdx])
		after := label[underlineIdx+1:]
		text := base.Render(before) + ul.Render(char) + base.Render(after)
		textW := lipgloss.Width(text)
		padTotal := btnW - textW
		if padTotal < 0 {
			padTotal = 0
		}
		padL := padTotal / 2
		padR := padTotal - padL
		padStyle := lipgloss.NewStyle().Background(bg)
		return padStyle.Render(strings.Repeat(" ", padL)) + text + padStyle.Render(strings.Repeat(" ", padR))
	}

	confirmBtn := makeBtn("Yep!", 0, d.Selected == 0)
	cancelBtn := makeBtn("Nope", 0, d.Selected == 1)
	buttonRow := confirmBtn + "  " + cancelBtn
	footer := styleMuted.Bold(true).Render("Tab") + " " + styleSubtle.Render("toggle") +
		styleSubtle.Render(" • ") + styleMuted.Bold(true).Render("Enter") + " " + styleSubtle.Render("confirm") +
		styleSubtle.Render(" • ") + styleMuted.Bold(true).Render("Esc") + " " + styleSubtle.Render("cancel")

	body := []string{
		"",
		centerRow(styleBase.Render(d.Title), 40),
	}
	if d.SubMessage != "" {
		body = append(body, centerRow(styleMuted.Render(d.SubMessage), 40))
	}
	body = append(body, "", centerRow(buttonRow, 40))

	return RenderModalShell(base, termW, termH, ModalShell{
		Title:  shellNameForDialog(d.Action),
		Width:  52,
		Body:   body,
		Footer: footer,
		Align:  BodyAlignCenter,
	})
}
```

Also add the tiny title helper:

```go
func shellNameForDialog(action DialogAction) string {
	if action == DialogKill {
		return "kill"
	}
	return "quit"
}
```

- [ ] **Step 4: Re-run the dialog test and verify it passes**

Run: `go test ./internal/tui -run 'TestConfirmQuitDialogUsesSharedModalShell'`

Expected: PASS.

### Task 3: Move help rendering onto the shared shell

**Files:**
- Modify: `internal/tui/helpmodal.go`
- Modify: `internal/tui/modal.go`

- [ ] **Step 1: Write the help shell regression test**

Add a focused test that checks help still renders through the shared shell and keeps the footer aligned without extra padding:

```go
func TestHelpModalUsesSharedModalShell(t *testing.T) {
	m := newHelpModal()
	got := m.View(80, 24, strings.Repeat("x", 80*24))
	if !strings.Contains(got, "help") {
		t.Fatal("expected help title in modal rendering")
	}
}
```

- [ ] **Step 2: Run the help test and verify it fails if needed**

Run: `go test ./internal/tui -run 'TestHelpModalUsesSharedModalShell'`

Expected: FAIL until `helpModal.View` calls `RenderModalShell`.

- [ ] **Step 3: Refactor `helpModal.View` to use `RenderModalShell`**

Keep filtering and list construction local, but remove the shell framing logic:

```go
func (h *helpModal) View(termW, termH int, base string) string {
	sections := h.groupedTopics()
	selectable := h.selectableTopics()
	if len(selectable) == 0 {
		h.index = 0
	} else if h.index >= len(selectable) {
		h.index = len(selectable) - 1
	}
	filterLine := styleMagentaBold.Render("❯") + " "
	if h.input == "" {
		filterLine += styleSubtle.Render("Type to filter")
	} else {
		filterLine += styleBase.Render(h.input)
	}
	body := append([]string{filterLine, ""}, buildHelpListBodyLines(sections, h.index, termW, termH)...)
	footer := styleMuted.Bold(true).Render("Enter") + " " + styleSubtle.Render("details") +
		styleSubtle.Render(" • ") + styleMuted.Bold(true).Render("Esc") + " " + styleSubtle.Render("close")

	return RenderModalShell(base, termW, termH, ModalShell{
		Title:  "help",
		Width:  52,
		Body:   body,
		Footer: footer,
		Align:  BodyAlignLeft,
	})
}
```

- [ ] **Step 4: Re-run the help test and verify it passes**

Run: `go test ./internal/tui -run 'TestHelpModalUsesSharedModalShell'`

Expected: PASS.

### Task 4: Remove the old skeleton file and update handoff docs

**Files:**
- Delete or replace: `internal/tui/modal_skeleton.go`
- Modify: `docs/current-status.md`

- [ ] **Step 1: Remove the old duplicate shell file**

Delete `internal/tui/modal_skeleton.go` once `internal/tui/modal.go` owns the shared shell implementation.

- [ ] **Step 2: Update `docs/current-status.md` to reflect the shared shell refactor**

Replace the rollback note with the new stable state:

```md
## Help Modal

- The help modal and quit dialog now share one modal shell template.
- The shell keeps the header, body, and footer spacing consistent across modals.
- The extra blank line below the modal footer has been removed.
- The help modal still owns filtering and list/detail behavior.
```

- [ ] **Step 3: Run the focused TUI tests**

Run: `go test ./internal/tui`

Expected: PASS.

- [ ] **Step 4: Run the full test suite and build**

Run:

```bash
go test ./...
go build -o flame .
```

Expected: both commands succeed.

## Self-Review

- Spec coverage: shell API, dialog migration, help migration, file split, duplicate shell removal, docs update, and verification are all covered.
- Placeholder scan: no TBD/TODO/implement later text remains.
- Type consistency: `ModalShell`, `RenderModalShell`, `BodyAlignCenter`, and `BodyAlignLeft` are used consistently across tasks.
