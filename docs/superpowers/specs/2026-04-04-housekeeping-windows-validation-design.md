# Housekeeping and Windows Validation Design

**Goal:** Refresh `docs/` into a reliable handoff source, remove low-risk dead code, validate Linux `run py`, and establish a real Windows TUI baseline before refactoring Windows module runners.

## Scope

This design covers five linked workstreams:

1. Documentation reorganization with minimal churn.
2. Housekeeping pass for dead code that is truly stale or unused.
3. Linux validation for `run py` in the current worker-session model.
4. Windows baseline testing in the TUI, with PowerShell first and `cmd` observed second.
5. Follow-up planning for Windows module refactoring and the TUI help system.

## Documentation Layout

`docs/current-status.md` remains the handoff source of truth.

New structure:

- `docs/architecture/tui-plan.md` — moved copy of the original TUI architecture/evolution plan.
- `docs/testing/linux-run-py.md` — real validation notes, commands, outcomes, fixes.
- `docs/testing/windows-baseline.md` — Windows shell, transfer, spawn, and UX validation notes.
- `docs/superpowers/specs/` — design records for larger efforts.
- `docs/superpowers/plans/` — executable implementation plans.

Existing topical docs stay in place and are updated in place:

- `docs/current-status.md`
- `docs/modules.md`
- `docs/sessions.md`
- `docs/transfers.md`
- `docs/configuration.md`

## Execution Order

### Phase 1: Housekeeping and docs sync

- Move `TUI_PLAN.md` into `docs/architecture/tui-plan.md`.
- Reconcile docs with current code and recent commits.
- Remove dead code only when behavior is unaffected and verification is straightforward.
- Keep the current worker-session module model unchanged.

### Phase 2: Linux `run py` validation

- Test `run py` using the current TUI and worker-session flow.
- Document the exact outcome before changing implementation.
- If a bug appears, fix the minimum code needed and record the reason.

### Phase 3: Windows baseline validation

- Validate TUI behavior against a real Windows target.
- Prioritize PowerShell because Windows modules depend on it.
- Also capture `cmd` behavior if available from the same test environment.

Checklist:

- session detection
- prompt behavior
- attach/detach
- command relay
- Ctrl+C and history behavior
- upload/download
- spawn
- worker-session spawn and cleanup visibility

### Phase 4: Windows module refactor

Only after baseline validation is recorded.

- Refactor `RunPowerShellInMemory`
- Refactor `RunDotNetInMemory`
- Refactor `RunPythonInMemory`

Target state: same blocking model as `RunScriptInMemory` and `RunBinary`, using `ExecuteWithStreamingCtx`.

### Phase 5: Help system

Design and implement after Linux and Windows command behavior is stable enough to document accurately.

## Windows Shell Upgrade Decision

If a shell is detected as `cmd`, upgrading into PowerShell makes sense as a future shell-quality improvement, but not as an automatic behavior right now.

Decision:

- validate raw `cmd` and PowerShell behavior first
- do not auto-upgrade `cmd` during baseline work
- after validation, consider an explicit command such as `!psupgrade`
- only consider auto-suggestion later if testing shows clear operator benefit

This keeps detection and transport issues visible during baseline testing and avoids changing the target shell unexpectedly.

## Dead Code Policy

The housekeeping pass should remove only code that meets one of these standards:

- no remaining callers in the codebase
- replaced by current architecture and not needed for near-term Windows work
- safe to remove with build verification and no runtime behavior change

Code still needed for upcoming Windows runner refactors should be left in place until those changes land.

## Verification Strategy

- After documentation and housekeeping changes: `go build -o gummy .`
- After any behavior change: reproduce with the narrowest practical validation step first, then rebuild.
- Testing evidence is written back into `docs/testing/*.md`, not left only in conversation history.

## Deliverables

- synced handoff docs
- architecture doc under `docs/architecture/`
- testing logs for Linux `run py` and Windows baseline
- reduced dead code where safe
- follow-up plan for Windows modules and help system
