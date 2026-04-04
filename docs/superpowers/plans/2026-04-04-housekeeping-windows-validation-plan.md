# Housekeeping and Windows Validation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Sync handoff docs with the real TUI state, remove safe dead code, validate Linux `run py`, and record a real Windows baseline before touching Windows module internals.

**Architecture:** Keep the current TUI and worker-session model intact. Treat documentation as a first-class deliverable, remove only low-risk dead code in the housekeeping pass, then validate real behavior before refactoring Windows runners.

**Tech Stack:** Go, Bubble Tea TUI, markdown docs, local build verification, Docker Windows VM / HTB test targets.

---

### Task 1: Reorganize docs and capture current design

**Files:**
- Create: `docs/architecture/tui-plan.md`
- Create: `docs/testing/linux-run-py.md`
- Create: `docs/testing/windows-baseline.md`
- Create: `docs/superpowers/specs/2026-04-04-housekeeping-windows-validation-design.md`
- Create: `docs/superpowers/plans/2026-04-04-housekeeping-windows-validation-plan.md`
- Modify: `docs/current-status.md`

- [ ] Move the contents of `TUI_PLAN.md` into `docs/architecture/tui-plan.md`.
- [ ] Leave a short pointer or remove `TUI_PLAN.md`, depending on whether any other local workflow still references it.
- [ ] Update `docs/current-status.md` so the priority order matches the approved design:
  1. housekeeping/docs sync
  2. Linux `run py`
  3. Windows baseline
  4. Windows runner refactor
  5. help system
- [ ] Add a short docs map to `docs/current-status.md` so handoff agents know where architecture, plans, and test evidence live.

### Task 2: Reconcile topical docs with the current code

**Files:**
- Modify: `docs/modules.md`
- Modify: `docs/sessions.md`
- Modify: `docs/transfers.md`
- Modify: `docs/configuration.md`

- [ ] Update command names and examples to match the code (`run dotnet`, current module list, worker-session behavior).
- [ ] Update session/shell docs to reflect the TUI model rather than the legacy readline CLI where needed.
- [ ] Keep the prose concise and operational for cross-machine handoff.

### Task 3: Audit and remove safe dead code

**Files:**
- Modify: `internal/session.go`
- Modify: `internal/shell.go`
- Modify: `docs/current-status.md`

- [ ] Confirm whether `ExecuteScriptFromStdin` has zero callers.
- [ ] Confirm whether `moduleCancel` is still unused in the current TUI architecture.
- [ ] Confirm whether `WatchForCancel` is still actively used and therefore should stay.
- [ ] Remove only the dead items that are not required for the upcoming Windows runner refactor.
- [ ] Update `docs/current-status.md` to reflect the new dead-code status after the audit.
- [ ] Run `go build -o flame .` and record the result.

### Task 4: Validate Linux `run py`

**Files:**
- Modify: `internal/session.go` (only if bug fixes are required)
- Modify: `internal/modules.go` (only if bug fixes are required)
- Modify: `docs/testing/linux-run-py.md`
- Modify: `docs/current-status.md`

- [ ] Test `run py` against a Linux shell in the TUI.
- [ ] Record exact setup, command used, observed output path behavior, and worker cleanup behavior.
- [ ] If it fails, reproduce the narrowest failing behavior, make the minimum fix, and rebuild.
- [ ] Update docs with the real result: works as-is, works with caveats, or needs refactor.

### Task 5: Validate Windows baseline in the TUI

**Files:**
- Modify: `docs/testing/windows-baseline.md`
- Modify: `docs/current-status.md`
- Modify: code files only if specific baseline fixes are required

- [ ] Receive a Windows PowerShell shell in the TUI.
- [ ] Validate prompt, history, relay, attach/detach, and Ctrl+C behavior.
- [ ] Validate upload/download and spawn.
- [ ] If available, also capture `cmd` behavior separately.
- [ ] Record whether an explicit future `cmd -> PowerShell` upgrade command still makes sense after real testing.

### Task 6: Write next-step plans from evidence

**Files:**
- Create: `docs/plans/windows-modules.md`
- Create: `docs/plans/help-system.md`

- [ ] Summarize the Windows runner refactor scope based on actual baseline evidence.
- [ ] Summarize the help-system scope based on stabilized command behavior.
- [ ] Keep both docs short, tactical, and easy to hand off.
