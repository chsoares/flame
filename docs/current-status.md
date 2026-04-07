# Current Status

## Help Modal

- The help modal now opens a detail view from the selected topic.
- The modal shell is shared with the quit dialog.
- The help detail uses the same `HelpEntry` data as the terminal help output.
- Enter opens detail, Backspace returns to the list, and Esc closes the modal.

## Startup And Banner

- The splash/banner now carries the `v0.9.0` version marker.
- Startup output and PTY fallback errors now use the TUI-era styling instead of the legacy boxed CLI look.
- The old padded startup/error presentation has been removed.

## Input History

- The TUI input now shows the most recent prefix-matching command as a subtle inline suggestion.
- Right arrow accepts the inline suggestion without changing `Tab` completion behavior.
- Up/down now filter history by the typed prefix in both menu and shell inputs while keeping their existing separate history stores.

## Transfers

- Download transfers now keep the byte count visible while the status-bar bar animates in an indeterminate marquee style.
- Uploads still use the existing percentage-based progress bar.

## Notes for Next Session

- Keep the shared modal shell as the single source for modal framing.
- Keep help content sourced from `internal/help.go`.
- The terminal help and modules output no longer use box borders.
- The v0.9.0 polish items are done.
- The next real work is the pre-1.0 TUI refactor.
- Start with phase 1 from `docs/superpowers/specs/2026-04-07-pre-1.0-tui-refactor-roadmap.md`:
  - replace fragile worker/SSH pending attribution
  - remove stdout-capture dependence from shipped TUI command execution
  - centralize module alias resolution
- After that, move to runtime consolidation, then split `internal/session.go` and `internal/tui/app.go` by responsibility.

## Release Recommendation

- Recommended path: treat `v0.9.0` as closed, then execute the pre-1.0 TUI-first refactor in phases.
- The refactor roadmap is documented in `docs/superpowers/specs/2026-04-07-pre-1.0-tui-refactor-roadmap.md`.
- The phased execution plan with mandatory user checkpoints is documented in `docs/superpowers/plans/2026-04-07-pre-1.0-tui-refactor-plan.md`.
- The key workflow rule still applies: every phase must be implemented, tested, and manually verified in the live app before continuing.
