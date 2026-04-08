# AGENTS.md

Flame is a Go TUI for handling reverse shells during CTF work. The shipping product is the TUI, not the older CLI-oriented workflow.

## Project Overview

Flame listens for reverse shells, tracks multiple sessions, lets the operator switch between them, exposes an interactive shell view, supports upload and download, runs built-in modules in worker sessions, generates payloads, and can bootstrap a callback through SSH. `v0.9.1` is the current patch release and is followed by a planned pre-`1.0` architecture cleanup.

## Build And Test

- Build: `go build -o flame .`
- Test: `go test ./... -coverprofile=coverage.out`

## Development Workflow

- There are no pull requests in the normal solo workflow. Changes are committed on the working branch and merged to `main` when stable.
- Versioning follows Semantic Versioning.
- PATCH `x.x.1`: bug fixes without intended behavior change.
- MINOR `x.1.x`: new backward-compatible functionality.
- MAJOR `1.x.x`: breaking changes.
- The current released version is tracked by the latest git tag. Check the latest tag before proposing or applying a version bump.
- Branch names should use `feat/`, `fix/`, or `chore/` prefixes.
- Any commit that changes the version must update the in-app version string in `internal/version.go` in the same change. Do not let docs, tags, and the app banner drift.

## Architecture

- `main.go` parses flags, initializes runtime config, starts the listener, and launches the TUI.
- `internal/listener.go` owns the listener lifecycle and session intake.
- `internal/session.go` currently holds session management, command routing, module orchestration, worker flows, and some TUI-facing execution behavior. It is still broader than it should be.
- `internal/modules.go` defines the module registry and built-in module implementations.
- `internal/payloads.go` builds reverse-shell payloads and optional C# output files.
- `internal/ssh.go` handles SSH-assisted reverse-shell handoff.
- `internal/config.go`, `internal/runtime_config.go`, and `internal/paths.go` define persisted config and app data behavior under `~/.flame/`.
- `internal/help.go` is the canonical command/help data source.
- `internal/tui/` contains the Bubble Tea application, layout, modals, output pane, clipboard support, and shell-facing UI.

Important intentional direction:

- TUI is the product.
- Shared backend logic should exist to serve the TUI.
- Do not preserve CLI-era structure or behavior at the expense of a cleaner TUI architecture.

## Testing

- Tests live next to source files using Go's normal `_test.go` pattern.
- Run the full suite with `go test ./... -coverprofile=coverage.out` after any non-trivial change.
- Do not break existing tests.
- New behavior requires a test.
- Prefer table-driven tests when a function has multiple input cases.
- When refactoring risky seams, add tests around the new seam instead of waiting for a later broad coverage pass.

## TODO.md Maintenance

- `TODO.md` is the canonical list of planned work and deferred ideas.
- If something is discussed but not implemented in the current session, add it to the right section of `TODO.md` before closing the work.
- Future work is grouped under `Refactor` and `Feature` to keep triage simple.
- When a TODO item is completed, mark it `[x]` and move it to a `## Done` section with the version that shipped it.
- Never delete TODO history; archive completed items instead.

## Pre-1.0 Refactor Guidance

The active pre-`1.0` direction is documented in:

- `docs/superpowers/specs/2026-04-07-pre-1.0-tui-refactor-roadmap.md`
- `docs/superpowers/plans/2026-04-07-pre-1.0-tui-refactor-plan.md`

Those docs exist because early parts of the codebase were written before the current level of TDD and review discipline. Keep these points in mind when changing core architecture:

- Fix correctness risks first, then improve seams, then split files.
- Do not do broad cleanup before risky architectural seams are corrected.
- Avoid stdout capture or print-driven internals as a control boundary for the TUI. Prefer explicit return values or structured results.
- Unify module execution around one TUI-first runtime path instead of preserving parallel CLI and TUI behavior.
- Remove or collapse legacy CLI-only assumptions when they conflict with a cleaner TUI design.
- Add tests for newly-isolated seams during refactor work, but do not turn a focused refactor into an open-ended whole-repo coverage campaign.

## What Not To Touch Without Discussion

- `internal/session.go`: it is a known god file and a planned refactor target. Avoid incidental rewrites unless the current task is explicitly about that area.
- `internal/tui/app.go`: same constraint as `internal/session.go`; avoid mechanical file moves without fixing the underlying seam first.
- Session attribution and worker/SSH pending-state logic: this area has known correctness risk and should be changed deliberately.
- TUI output and command execution flows that still depend on stdout-oriented behavior: do not deepen that pattern.
- Versioning behavior: if you change the release version, update the app string, docs, and tag plan together.
