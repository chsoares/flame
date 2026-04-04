# Full Flame Rebrand Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rebrand the project from `flame` to `flame` across the user-facing product, local config/data paths, TUI/CLI branding, docs, and supporting copy.

**Architecture:** Keep the existing functionality intact while renaming the user-visible product identity and default storage paths. Treat this as a product rename, not a repository/module-path migration: update the visual branding, prompts, banners, docs, and runtime directories, but do not rename the Go module import path yet.

**Tech Stack:** Go, Bubble Tea TUI, Lip Gloss, markdown docs, local filesystem path migration, targeted Go tests, full project build verification.

---

### Task 1: Lock the approved `flame` visual branding

**Files:**
- Modify: `internal/tui/logo.go`
- Modify: `internal/tui/input.go`
- Modify: `internal/tui/header.go`
- Modify: `internal/tui/statusbar.go`
- Modify: `internal/tui/layout.go`
- Modify: `internal/ui/colors.go`
- Test: `internal/tui/logo_test.go`

- [ ] Keep the approved `flame` ASCII art and spacing.
- [ ] Use the fire icon as the primary logo symbol.
- [ ] Ensure the sidebar width stays on the approved value (`34`).
- [ ] Run the branding tests and keep them green.

### Task 2: Rename user-facing paths and local storage defaults

**Files:**
- Modify: `internal/tui/input.go`
- Modify: config/history/log path code in `internal/`
- Modify: any docs mentioning `~/.flame`

- [ ] Rename default runtime storage from `~/.flame` to `~/.flame`.
- [ ] Keep behavior safe for fresh runs.
- [ ] If migration logic is simple and low-risk, move existing `~/.flame` to `~/.flame`; otherwise document the manual move path clearly.

### Task 3: Rename visible product strings everywhere practical

**Files:**
- Modify: `internal/tui/*.go`
- Modify: `internal/ui/colors.go`
- Modify: `README.md`
- Modify: `CLAUDE.md`
- Modify: docs under `docs/`

- [ ] Replace user-facing `flame` branding with `flame` where it describes the product/UI.
- [ ] Avoid changing repository/module-path references that would break imports.
- [ ] Rename the old `SymbolDroplet` concept to a more neutral internal name such as `SymbolLogo`.

### Task 4: Verify and finalize

**Files:**
- Modify: docs as needed to reflect final naming

- [ ] Run targeted TUI tests.
- [ ] Run a broader grep pass for lingering user-facing `flame` references.
- [ ] Build the project with `go build -o flame .`.
- [ ] Commit the rebrand as a dedicated block.
