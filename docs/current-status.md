# Current Status

## Help Modal

- The help modal now opens a detail view from the selected topic.
- The modal shell is shared with the quit dialog.
- The help detail uses the same `HelpEntry` data as the terminal help output.
- Enter opens detail, Backspace returns to the list, and Esc closes the modal.

## Input History

- The TUI input now shows the most recent prefix-matching command as a subtle inline suggestion.
- Right arrow accepts the inline suggestion without changing `Tab` completion behavior.
- Up/down now filter history by the typed prefix in both menu and shell inputs while keeping their existing separate history stores.

## Notes for Next Session

- Keep the shared modal shell as the single source for modal framing.
- Keep help content sourced from `internal/help.go`.
- The terminal help and modules output no longer use box borders.
- Before the v0.9.0 release, finish only:
  - improved autocomplete/suggestion in the input bar
  - add `v0.9.0` to the banner/splash
  - review pre-launch/startup error output and adapt it to the TUI-era UI style:
    - replace the old boxed CLI banner with the splash/banner style already used by the TUI
    - remove legacy padded text formatting in those error screens
    - use `Interface` in title case
- Everything else waits until after `1.0`.

## Release Recommendation

- Recommended path: ship `v0.9.0` first with the current scope, then do the larger TUI-first architecture refactor for `1.0`.
- The pre-`1.0` refactor roadmap is documented in `docs/superpowers/specs/2026-04-07-pre-1.0-tui-refactor-roadmap.md`.
- The phased execution plan with mandatory user checkpoints is documented in `docs/superpowers/plans/2026-04-07-pre-1.0-tui-refactor-plan.md`.
- The most important workflow rule in that plan: every phase must be implemented, tested, and manually verified in the live app before continuing.
