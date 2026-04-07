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
