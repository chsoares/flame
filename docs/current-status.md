# Current Status

## Help Modal

- The help modal now opens a detail view from the selected topic.
- The modal shell is shared with the quit dialog.
- The help detail uses the same `HelpEntry` data as the terminal help output.
- Enter opens detail, Backspace returns to the list, and Esc closes the modal.

## Notes for Next Session

- Keep the shared modal shell as the single source for modal framing.
- Keep help content sourced from `internal/help.go`.
- The terminal help and modules output no longer use box borders.
- Before the v0.9.0 release, finish only:
  - improved autocomplete/suggestion in the input bar
  - add `v0.9.0` to the banner/splash
- Everything else waits until after `1.0`.

## Release Recommendation

- Recommended path: ship `v0.9.0` first with the current scope, then do the larger TUI-first architecture refactor for `1.0`.
- The pre-`1.0` refactor roadmap is documented in `docs/superpowers/specs/2026-04-07-pre-1.0-tui-refactor-roadmap.md`.
- The phased execution plan with mandatory user checkpoints is documented in `docs/superpowers/plans/2026-04-07-pre-1.0-tui-refactor-plan.md`.
- The most important workflow rule in that plan: every phase must be implemented, tested, and manually verified in the live app before continuing.
