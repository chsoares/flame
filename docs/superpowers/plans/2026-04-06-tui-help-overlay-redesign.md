# TUI Help Overlay Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rebuild the TUI modal work in small approved phases, starting with `quit` as the canonical modal shell and only then layering the `F1` help experience.

**Architecture:** Keep the modal shell minimal and prove it visually through `quit` first. Then build the help flow in separate phases: static list, filter, detail navigation, and final integration, with a live-app checkpoint between phases.

**Tech Stack:** Go, Bubble Tea, Lip Gloss, existing Flame TUI components, existing CLI help registry, manual live-app validation.

---

## File Map

- Modify: `internal/tui/dialog.go`
- Modify: `internal/tui/app.go`
- Modify: `internal/tui/statusbar.go`
- Modify: `internal/help.go` only if a minimal structured topic accessor is still needed later
- Modify: `internal/help_test.go` only if a minimal structured topic accessor is added later
- Create later phases as needed:
  - `internal/tui/helpmodal.go`
  - `internal/tui/helpmodal_test.go`
  - `internal/tui/app_help_test.go`
  - `internal/tui/statusbar_test.go`
- Modify at the end: `docs/current-status.md`

## Phase Gates

Do not start the next phase until:

- the current phase passes its focused automated checks
- the current phase is manually reviewed in the live app
- the human explicitly approves continuing

## Lessons Learned To Enforce During Implementation

- Do not show summaries in the list view.
- Do not show aliases or any extra metadata in detail unless the terminal help already shows it.
- Do not let the help modal resize while filtering.
- Do not ship visual changes based only on unit tests; run the app and inspect it.
- Do not redesign `quit` and `help` together before `quit` alone is visually approved.

### Task 1: Reset and verify the baseline worktree

**Files:**
- No source changes required if the worktree is already clean

- [ ] **Step 1: Confirm the worktree is clean**

Run:

```bash
git status --short
```

Expected: no output.

- [ ] **Step 2: Run the full baseline tests**

Run:

```bash
go test ./...
```

Expected: PASS on the clean baseline.

- [ ] **Step 3: Record that implementation restarts from the clean baseline**

No code change. This is a handoff checkpoint before any new phase work.

### Task 2: Rebuild `quit` as the canonical modal shell

**Files:**
- Modify: `internal/tui/dialog.go`
- Create if needed: `internal/tui/dialog_test.go`

- [ ] **Step 1: Write the failing tests for the `quit` render contract**

Add focused tests for these invariants only:

- title row is inside the bordered box
- transparent outside area is preserved by the render path
- quit content remains centered
- default selection remains cancel/nope

Suggested test names:

```go
func TestConfirmQuitDialogDefaultsToCancel(t *testing.T) {}
func TestQuitDialogRendersTitleInsideBorder(t *testing.T) {}
func TestQuitDialogPreservesUnderlyingScreenOutsideBox(t *testing.T) {}
```

- [ ] **Step 2: Run the focused quit tests and watch them fail if needed**

Run a focused command for the added tests.

- [ ] **Step 3: Implement only the `quit` visual rebuild**

Requirements for this step:

- keep current `quit` behavior and keys intact
- keep medium compact size similar to the old good dialog
- place `quit ////////////////////` on the first line inside the border
- keep transparent background outside the box
- keep body content centered
- keep footer in the correct bottom position

Do not add `help` modal code in this task.

- [ ] **Step 4: Re-run the focused quit tests**

Run the same focused command.

Expected: PASS.

- [ ] **Step 5: Run the app live and validate the real `quit` modal**

Manual checklist:

```text
- modal width and height feel compact/medium
- title row sits inside the border
- transparent background works
- content is centered
- footer sits at the bottom of the modal body
- old visual quality of the quit dialog is restored
```

- [ ] **Step 6: Stop and get approval before Phase 3**

Do not continue until the human approves the live `quit` result.

### Task 3: Add the `F1` help mock with static list only

**Files:**
- Modify: `internal/tui/app.go`
- Create: `internal/tui/helpmodal.go`
- Create: `internal/tui/helpmodal_test.go`
- Create if needed: `internal/tui/app_help_test.go`

- [ ] **Step 1: Write failing tests for static help modal open/render behavior**

Cover only:

- `F1` opens the help modal
- modal keeps a fixed outer size
- list renders command names only
- selection uses a row highlight instead of a prefix cursor

Suggested test names:

```go
func TestAppF1OpensHelpModal(t *testing.T) {}
func TestHelpModalListShowsTopicNamesOnly(t *testing.T) {}
func TestHelpModalKeepsStableDimensions(t *testing.T) {}
```

- [ ] **Step 2: Run the focused tests and verify failure**

Run a focused `go test ./internal/tui -run 'TestAppF1OpensHelpModal|TestHelpModal'` command matching the new tests.

- [ ] **Step 3: Implement only a static list modal**

Requirements for this step:

- open with `F1`
- same shell language approved from `quit`
- medium centered modal size, fixed while open
- static command-name list only
- internal scrolling for items that do not fit
- no filter yet
- no detail view yet
- no summaries in the list

- [ ] **Step 4: Re-run the focused help mock tests**

Expected: PASS.

- [ ] **Step 5: Run the app live and validate the static help modal**

Manual checklist:

```text
- overall proportions match the intended reference
- list contains only command names
- selected row is a colored bar
- extra items are reachable by scrolling
- modal size is fixed
```

- [ ] **Step 6: Stop and get approval before Phase 4**

Do not continue until the human approves the live mock.

### Task 4: Add filter input without changing modal size

**Files:**
- Modify: `internal/tui/helpmodal.go`
- Modify: `internal/tui/helpmodal_test.go`

- [ ] **Step 1: Write failing tests for filter-row behavior**

Cover only:

- filter row uses prompt symbol and placeholder `Type to filter`
- filtering affects the visible list
- modal outer size stays unchanged while filtering

Suggested test names:

```go
func TestHelpModalShowsPromptStyledFilterPlaceholder(t *testing.T) {}
func TestHelpModalFiltersTopicNamesWithoutResizing(t *testing.T) {}
```

- [ ] **Step 2: Run the focused filter tests and verify failure**

- [ ] **Step 3: Implement only the filter row and list filtering**

Requirements:

- use Flame prompt symbol styling
- dark-gray `Type to filter` placeholder
- keep same outer modal dimensions from Phase 3
- filter by command names only in this phase
- preserve colored selection row and internal scrolling

- [ ] **Step 4: Re-run the focused filter tests**

Expected: PASS.

- [ ] **Step 5: Run the app live and validate filter appearance**

Manual checklist:

```text
- prompt symbol matches Flame input language
- placeholder is visually muted/dark gray
- filtering feels correct
- modal does not resize while typing
```

- [ ] **Step 6: Stop and get approval before Phase 5**

Do not continue until the human approves the live filter state.

### Task 5: Add detail navigation using terminal help content only

**Files:**
- Modify: `internal/help.go` only if a tiny helper is required
- Modify: `internal/help_test.go` only if a tiny helper is required
- Modify: `internal/tui/helpmodal.go`
- Modify: `internal/tui/helpmodal_test.go`

- [ ] **Step 1: Write failing tests for detail navigation and content sourcing**

Cover only:

- `Enter` opens the selected topic detail
- `Backspace` returns to the list
- outer modal size remains fixed
- detail content does not add aliases/summaries/labels that terminal help does not show

Suggested test names:

```go
func TestHelpModalEnterOpensDetailWithoutResizing(t *testing.T) {}
func TestHelpModalBackspaceReturnsToList(t *testing.T) {}
func TestHelpModalDetailMatchesTerminalHelpContent(t *testing.T) {}
```

- [ ] **Step 2: Run the focused detail tests and verify failure**

- [ ] **Step 3: Implement only detail navigation and terminal-help rendering**

Requirements:

- keep the same outer modal dimensions
- only the inner content changes
- detail uses terminal help content source exactly
- do not add aliases or extra headings beyond the terminal help output
- add internal scroll if the help content exceeds the visible body

- [ ] **Step 4: Re-run the focused detail tests**

Expected: PASS.

- [ ] **Step 5: Run the app live and validate detail view**

Manual checklist:

```text
- no modal resize when entering detail
- detail content visually matches terminal help expectations
- no invented metadata appears
- footer/back hint remains in the correct place
```

- [ ] **Step 6: Stop and get approval before final integration**

Do not continue until the human approves the live detail state.

### Task 6: Final integration, docs, and verification

**Files:**
- Modify: `internal/tui/statusbar.go`
- Create if needed: `internal/tui/statusbar_test.go`
- Modify: `docs/current-status.md`

- [ ] **Step 1: Write failing tests for final status bar expectations**

Cover:

- `F1 help` appears first
- `Tab complete` is removed where requested
- wording matches existing Flame status-bar conventions

- [ ] **Step 2: Run the focused status bar tests and verify failure**

- [ ] **Step 3: Implement final status-bar changes only after modal phases are approved**

- [ ] **Step 4: Update `docs/current-status.md` after final visual approval**

Document the shipped phased result factually and minimally.

- [ ] **Step 5: Run final verification**

Run:

```bash
go test ./...
go build -o flame .
git status --short
```

Expected:

- tests pass
- build succeeds
- worktree contains only intended changes

## Self-Review

- Spec coverage: this plan explicitly enforces the phased delivery of `quit` shell, static help mock, filter, detail, and final integration.
- Placeholder scan: each phase has concrete files, scope, and verification checkpoints.
- Type consistency: keeps shared naming small and defers file creation until the phase where each unit is actually needed.
