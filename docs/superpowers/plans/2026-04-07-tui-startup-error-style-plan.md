# TUI Startup Error Style Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the legacy bordered startup banner with the splash banner and restyle startup/PTy errors through the same notification/output pattern the TUI already uses for command errors.

**Architecture:** Add one shared error presentation helper in `internal/ui` so startup code and TUI runtime code can format the same message consistently. Replace legacy bordered banner prints in `main.go` with the splash banner, keep bootstrap failures as printed terminal output rendered with TUI styling, and move PTY fallback messaging into `internal/tui` so it can produce both notification and output-pane entries.

**Tech Stack:** Go, Bubble Tea, Lip Gloss, Go test suite.

---

## File Map

- Modify: `internal/ui/colors.go`
  - replace the legacy PTY fallback string helper with a TUI-styled error helper that can be reused by startup and runtime code
- Modify: `main.go`
  - swap legacy `ui.Banner()` / `ui.SubBanner()` prints for the splash renderer and format startup/bootstrap errors through the shared UI helper instead of raw `fmt.Println(ui.Error(...))`
- Modify: `internal/tui/app.go`
  - convert PTY upgrade failures into a notification plus output-pane error, matching the upload/spawn pattern
- Modify: `internal/tui/statusbar.go`
  - if needed, add a small helper for error notification text so PTY fallback can reuse the existing notification style cleanly
- Add/modify tests beside the touched files

## Task 1: Lock Down The Current Error Surface With Tests

**Files:**
- Modify: `internal/ui/colors.go`
- Modify: `internal/tui/app.go`
- Modify: `main.go`

- [ ] **Step 1: Write a failing test that asserts the legacy PTY fallback string no longer appears as plain text**

```go
func TestPTYFailedUsesTUIStyle(t *testing.T) {
	got := PTYFailed()
	if strings.Contains(got, "using raw shell") {
		t.Fatalf("expected PTY failure text to be restyled, got %q", got)
	}
	if !strings.Contains(got, "PTY") {
		t.Fatalf("expected PTY failure helper to still communicate PTY context, got %q", got)
	}
}
```

- [ ] **Step 2: Write a failing test that asserts startup errors go through the shared helper**

```go
func TestStartupErrorStyleHelper(t *testing.T) {
	got := ui.Error("Failed to start listener: boom")
	if !strings.Contains(got, "Failed to start listener: boom") {
		t.Fatalf("expected startup error body, got %q", got)
	}
	if strings.Contains(got, "TUI error:") {
		t.Fatalf("expected caller to provide context, not helper, got %q", got)
	}
}
```

- [ ] **Step 3: Write a failing test that PTY failures reach both notification and output-pane paths when handled in TUI**

```go
func TestPTYFailureUsesNotificationAndOutput(t *testing.T) {
	app := New(nil, "127.0.0.1:4444")
	app.output.Clear()
	app.statusBar.Notify = nil

	app.handlePTYFailure("PTY upgrade failed")

	if app.statusBar.Notify == nil {
		t.Fatal("expected PTY failure notification")
	}
	if got := app.output.GetContent(); !strings.Contains(got, "PTY upgrade failed") {
		t.Fatalf("expected PTY failure in output pane, got %q", got)
	}
}
```

- [ ] **Step 4: Run the focused tests and confirm they fail for the current behavior**

Run: `go test ./internal/ui ./internal/tui -run 'TestPTYFailedUsesTUIStyle|TestStartupErrorStyleHelper|TestPTYFailureUsesNotificationAndOutput' -v`

Expected: FAIL because the PTY fallback helper, TUI PTY failure path, and startup banner still use legacy/plain behavior.

- [ ] **Step 5: Write a failing test that startup uses the splash banner instead of the legacy bordered banner**

```go
func TestStartupBannerUsesSplash(t *testing.T) {
	got := ui.Banner()
	if strings.Contains(got, "╭") || strings.Contains(got, "╰") {
		t.Fatalf("expected splash-style banner, got %q", got)
	}
}
```

## Task 2: Implement Shared Error Styling And PTY Runtime Handling

**Files:**
- Modify: `internal/ui/colors.go`
- Modify: `internal/tui/app.go`

- [ ] **Step 1: Replace the legacy PTY fallback helper with a TUI-style error helper**

```go
func PTYFailed() string {
	return Error("PTY upgrade failed - using raw shell")
}
```

- [ ] **Step 2: Add a small TUI PTY failure handler that emits both notification and output-pane text**

```go
func (a *App) handlePTYFailure(msg string) {
	a.statusBar.Notify = &Notification{Message: msg, Level: NotifyError}
	a.output.Append(ui.Error(msg) + "\n\n")
}
```

- [ ] **Step 3: Route the PTY upgrade failure path to the new handler**

```go
if !ptySuccess {
	a.handlePTYFailure(ui.PTYFailed())
}
```

- [ ] **Step 4: Update startup errors in `main.go` to use the shared helper consistently**

```go
if err != nil {
	fmt.Println(ui.Error(fmt.Sprintf("Config initialization failed: %v (using defaults)", err)))
}

if err := l.Start(); err != nil {
	fmt.Println(ui.Error(fmt.Sprintf("Failed to start listener: %v", err)))
	os.Exit(1)
}

if err := tui.Run(manager, listenerAddr); err != nil {
	fmt.Println(ui.Error(fmt.Sprintf("TUI error: %v", err)))
	os.Exit(1)
}
```

Replace the legacy `ui.Banner()` and `ui.SubBanner()` startup prints with `tui.RenderExitBanner(...)` or the existing splash renderer used by the TUI.

- [ ] **Step 5: Run the focused tests and verify they pass**

Run: `go test ./internal/ui ./internal/tui -run 'TestPTYFailedUsesTUIStyle|TestStartupErrorStyleHelper|TestPTYFailureUsesNotificationAndOutput' -v`

Expected: PASS.

## Task 3: Broader Verification And Cleanup

**Files:**
- Modify: `docs/current-status.md`

- [ ] **Step 1: Add a short note to `docs/current-status.md` describing the TUI-style error cleanup**

```md
- Startup and PTY fallback errors now use the same TUI-style error presentation.
- PTY failures surface as both a status notification and an output-pane error line.
- The legacy bordered startup banner is gone; startup now uses the splash banner.
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

- [ ] **Step 5: Commit the cleanup**

```bash
git add main.go internal/ui/colors.go internal/tui/app.go docs/current-status.md
git commit -m "fix: restyle startup and pty errors"
```
