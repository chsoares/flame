# Download Indeterminate Status Bar Animation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Animate the download transfer bar in the TUI with an indeterminate marquee-style motion while preserving the existing download bytes text and the current transfer overlay model.

**Architecture:** Keep the change inside the existing transfer overlay path. Extend `StatusBar` with lightweight transient animation state for downloads only, drive it from a timer while a download is active, and keep uploads on the current percentage-based bar. The download overlay should still render the same left label and right-side bytes text, but the center bar should animate continuously instead of relying on a known percentage.

**Tech Stack:** Go, Bubble Tea, Lip Gloss, Go test suite.

---

### Task 1: Add tests for indeterminate download rendering and animation ticks

**Files:**
- Modify: `internal/tui/statusbar_test.go`
- Modify: `internal/tui/app.go`

- [ ] **Step 1: Add a status bar test that asserts download rendering keeps the bytes text and uses an animated bar shape**

```go
func TestStatusBarDownloadViewKeepsBytesAndBar(t *testing.T) {
	s := NewStatusBar(120)
	s.TransferPct = 0
	s.TransferMsg = "Downloading passwd..."
	s.TransferRight = "15.2 KB"
	s.TransferUpload = false
	s.TransferAnimating = true
	s.TransferAnimPhase = 0

	got := ansi.Strip(s.View())
	if !strings.Contains(got, "Downloading passwd...") {
		t.Fatalf("expected download label, got %q", got)
	}
	if !strings.Contains(got, "15.2 KB") {
		t.Fatalf("expected bytes text to remain visible, got %q", got)
	}
	if !strings.Contains(got, "/") {
		t.Fatalf("expected animated bar characters, got %q", got)
	}
}
```

- [ ] **Step 2: Add a status bar test that verifies the download animation advances on tick**

```go
func TestStatusBarDownloadAnimationAdvances(t *testing.T) {
	s := NewStatusBar(120)
	s.TransferPct = 0
	s.TransferMsg = "Downloading passwd..."
	s.TransferRight = "15.2 KB"
	s.TransferUpload = false
	s.TransferAnimating = true
	s.TransferAnimPhase = 0

	first := ansi.Strip(s.View())
	s.StepTransferAnimation()
	second := ansi.Strip(s.View())

	if first == second {
		t.Fatalf("expected animation to change between frames, got %q", first)
	}
}
```

- [ ] **Step 3: Add an app-level test that a download transfer starts animation and keeps bytes updates flowing**

```go
func TestTransferProgressDownloadEnablesAnimation(t *testing.T) {
	a := New(nil, "127.0.0.1:4444")
	a.statusBar.Width = 120

	a, _ = a.Update(transferProgressMsg{Filename: "passwd", Pct: 0, Right: "15.2 KB", Upload: false})
	app := a.(App)

	if !app.statusBar.TransferAnimating {
		t.Fatal("expected download animation to be enabled")
	}
	if app.statusBar.TransferRight != "15.2 KB" {
		t.Fatalf("expected bytes text to stay in status bar, got %q", app.statusBar.TransferRight)
	}
}
```

- [ ] **Step 4: Run the focused tests and confirm they fail before implementation**

Run: `go test ./internal/tui -run 'TestStatusBarDownloadViewKeepsBytesAndBar|TestStatusBarDownloadAnimationAdvances|TestTransferProgressDownloadEnablesAnimation' -v`

Expected: FAIL because `StatusBar` does not yet animate downloads.

### Task 2: Implement indeterminate download animation in `StatusBar`

**Files:**
- Modify: `internal/tui/statusbar.go`
- Modify: `internal/tui/messages.go`
- Modify: `internal/tui/app.go`

- [ ] **Step 1: Extend `StatusBar` with transient download animation state and a helper to advance it**

```go
type StatusBar struct {
	Context        ContextMode
	TransferPct    int
	TransferMsg    string
	TransferRight  string
	TransferUpload bool
	TransferAnimating bool
	TransferAnimPhase int
	Width          int
	Notify         *Notification
}

func (s *StatusBar) StepTransferAnimation() {
	if s.TransferAnimating {
		s.TransferAnimPhase++
	}
}
```

- [ ] **Step 2: Render downloads with a moving marquee-style bar while preserving the bytes text**

```go
func (s StatusBar) renderTransferProgress() string {
	bg := lipgloss.Color("4")

	icon := ui.SymbolUpload
	if !s.TransferUpload {
		icon = ui.SymbolDownload
	}

	label := icon + " " + s.TransferMsg
	pctStr := s.TransferRight
	if pctStr == "" && s.TransferUpload {
		pctStr = fmt.Sprintf("%d%%", s.TransferPct)
	}

	textStyle := lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("0")).Bold(true)
	labelW := lipgloss.Width(label) + 2
	rightW := len(pctStr) + 2
	barWidth := s.Width - labelW - rightW
	if barWidth < 5 {
		barWidth = 5
	}

	barStyle := lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("0"))
	emptyStyle := lipgloss.NewStyle().Background(bg)

	var bar string
	if s.TransferUpload {
		filled := barWidth * s.TransferPct / 100
		if filled > barWidth {
			filled = barWidth
		}
		for i := 0; i < filled; i++ {
			bar += "/"
		}
		bar += fmt.Sprintf("%*s", barWidth-filled, "")
	} else {
		window := barWidth / 3
		if window < 4 {
			window = 4
		}
		if window > barWidth {
			window = barWidth
		}
		start := s.TransferAnimPhase % barWidth
		for i := 0; i < barWidth; i++ {
			pos := (i - start + barWidth) % barWidth
			if pos < window {
				bar += "/"
			} else {
				bar += " "
			}
		}
	}

	rendered := textStyle.Render(" "+label+" ") + barStyle.Render(bar) + textStyle.Render(" "+pctStr+" ")
	if contentW := lipgloss.Width(rendered); contentW < s.Width {
		rendered += lipgloss.NewStyle().Background(bg).Render(fmt.Sprintf("%*s", s.Width-contentW, ""))
	}
	return rendered
}
```

- [ ] **Step 3: Start animation on download progress, keep the bytes text unchanged, and stop animation on completion**

```go
case transferProgressMsg:
	a.statusBar.TransferPct = msg.Pct
	a.statusBar.TransferMsg = msg.Filename
	a.statusBar.TransferRight = msg.Right
	a.statusBar.TransferUpload = msg.Upload
	a.statusBar.TransferAnimating = !msg.Upload
	if !msg.Upload {
		a.statusBar.TransferAnimPhase++
	}
	return a, nil

case transferDoneMsg:
	a.statusBar.TransferPct = -1
	a.statusBar.TransferMsg = ""
	a.statusBar.TransferRight = ""
	a.statusBar.TransferAnimating = false
	...
```

- [ ] **Step 4: Run the focused tests and confirm they pass**

Run: `go test ./internal/tui -run 'TestStatusBarDownloadViewKeepsBytesAndBar|TestStatusBarDownloadAnimationAdvances|TestTransferProgressDownloadEnablesAnimation' -v`

Expected: PASS.

### Task 3: Verify the broader TUI surface and update the status handoff

**Files:**
- Modify: `docs/current-status.md`

- [ ] **Step 1: Add a short status note about the indeterminate download animation**

```md
- Download transfers now keep the byte count visible while the status-bar bar animates in an indeterminate marquee style.
```

- [ ] **Step 2: Run the broader TUI suite**

Run: `go test ./internal/tui/...`

Expected: PASS.

- [ ] **Step 3: Run the broader internal suite**

Run: `go test ./internal/...`

Expected: PASS.

- [ ] **Step 4: Build the app**

Run: `go build -o flame .`

Expected: PASS.
