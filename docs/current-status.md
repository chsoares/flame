# Current Status

## Help Modal

- The help modal is back to the last stable checkpoint: grouped categories in the list view.
- The shell remains fixed: same header, input line, list area, and footer/status area.
- The attempted in-place detail/body swap was rolled back.
- Enter currently does not open a second page.

## Notes for Next Session

- Start from the categories-in-list shell.
- Reuse the same modal shell; only the body should ever change.
- Keep the footer/status hint area fixed.
- Avoid introducing a separate detail modal/page.
