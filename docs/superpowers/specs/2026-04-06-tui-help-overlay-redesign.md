# TUI Help Overlay Redesign

Date: 2026-04-06

## Goal

Rebuild the TUI modal work in small validated phases so the final `help` experience matches the approved visual reference and the existing `quit` dialog becomes the canonical modal shell.

## Why The First Attempt Failed

The previous implementation bundled too many concerns into one delivery:

- modal shell styling
- quit restyling
- `F1` wiring
- help list rendering
- filter interaction
- detail navigation
- short-terminal behavior

That made review expensive, hid visual mistakes until late, and drifted away from the reference UI.

## Lessons Learned

- Build the modal shell from the existing `quit` dialog first, then reuse that shell.
- Validate in the live app after each phase instead of relying only on unit tests.
- Keep the modal size stable while filtering.
- Show only command names in the list view. No inline summaries, aliases, or extra help text.
- Match the Flame input style inside the modal: prompt symbol plus dark-gray `Type to filter` placeholder.
- Use a filled selection bar like the Crush command dialog, not a `>` cursor prefix.
- Reuse terminal help content exactly in detail view. Do not invent labels or metadata that terminal help does not show.
- Keep footer help text in the same visual language as Flame's status bar and existing UI conventions.
- Treat `quit` and `help` as separate phases even if they share rendering primitives.

## Scope

This redesign covers:

- restoring the worktree to a clean baseline before reimplementation
- rebuilding the modal shell through the `quit` dialog first
- implementing `help` in four reviewed phases
- live-app validation after each phase

This redesign does not cover:

- changing terminal `help` semantics or output
- changing the input-line `help` command behavior
- adding new metadata to help topics
- broad TUI layout refactors outside the modal work

## Non-Negotiable UX Requirements

- `quit` becomes the reference modal shell.
- The first line of the modal is inside the border, not above it.
- That first line contains the lowercase modal name and hatch fill, in the visual style of the compact header / Crush reference.
- The modal background remains transparent outside the bordered box.
- The `quit` content remains centered.
- The `help` modal keeps one stable size while filtering and while moving between topics.
- The `help` list view shows only command names.
- The selected command is shown with a full-width colored selection bar.
- The filter row uses Flame's prompt symbol and a dark-gray `Type to filter` placeholder.
- The footer actions follow Flame's existing help/status visual language.
- The detail view uses terminal help formatting as the content source and must not add aliases, summaries, or labels that are absent from terminal help.

## Reference Alignment

The visual target is the Crush commands dialog behavior adapted to Flame's existing TUI language:

- medium-sized centered modal, not full-height
- constant modal size during list filtering
- scroll within the list when items exceed visible rows
- input row at the top, list below, footer at the bottom
- title and hatch line inside the modal border

The `quit` modal should match the same shell but keep its own centered content layout.

## Phased Delivery

### Phase 0: Baseline reset

Before any new implementation work, reset the isolated worktree to the clean branch baseline. Do not carry forward code from the failed attempt.

Completion criteria:

- worktree is clean
- previous experimental modal files are gone
- baseline tests still pass before new work starts

### Phase 1: Rebuild `quit` as the canonical modal shell

This phase changes only the existing `quit` dialog.

Goals:

- keep current `quit` behavior intact
- move the title row inside the border
- keep the modal transparent outside the box
- preserve the old good proportions: centered, compact, medium width, not oversized vertically
- keep content centered

Validation for this phase:

- unit tests for render structure where practical
- live-app check of the real `quit` modal before moving on

This phase is the visual baseline for everything that follows.

### Phase 2: Mock `F1` help modal shell with static list only

This phase intentionally does not implement filtering or detail navigation.

Goals:

- open a help modal with `F1`
- render the new shell based on the approved `quit` shell
- show a fixed-size modal with a static list of command names
- use a selection bar, not a cursor prefix
- add internal scrolling for commands beyond the visible window

Validation for this phase:

- live-app review focused only on size, spacing, border/title placement, selection style, and list viewport behavior

The phase is complete only if the modal visually matches the target shape and proportions.

### Phase 3: Add filter input to the list modal

This phase adds the input row only after the shell and list are approved.

Goals:

- keep the exact same modal dimensions from Phase 2
- add prompt symbol plus dark-gray `Type to filter` placeholder
- filter the command-name list only
- preserve list scrolling and selection styling

Validation for this phase:

- unit tests for filtering logic
- live-app review focused on input appearance and modal size stability while filtering

This phase is not complete if the modal resizes as the query changes.

### Phase 4: Add topic detail navigation

This phase adds `Enter` into detail and `Backspace` back to list.

Goals:

- keep the same outer modal dimensions as the approved list modal
- replace inner content only
- render terminal help content without adding extra metadata
- keep footer/back navigation readable and positioned correctly
- keep overflow inside the modal via internal scrolling instead of growing the box

Validation for this phase:

- unit tests for state transitions
- unit tests for using terminal help content without extra adornment
- live-app review focused on layout stability and detail readability

### Phase 5: Final polish and integration checks

After all prior phases are individually approved:

- verify `F1` open/close behavior across splash, menu, and shell
- verify `quit` and `help` share the same modal shell language without breaking each other
- verify status-bar wording follows Flame's existing patterns
- update handoff docs only after the final UI is approved

## Architecture Guidance

- Keep the shared modal shell minimal and driven by the approved `quit` phase.
- Do not create a broad modal framework before the shell is visually correct.
- Keep help list/detail state in its own model rather than pushing all logic into `app.go`.
- Use the existing help registry as the source of truth for topics and detail content.
- If a helper is needed for structured topic access, keep it minimal and do not change terminal rendering behavior.

## Testing Strategy

Use both automated and manual validation.

Automated:

- focused tests for modal render invariants that are stable and meaningful
- focused tests for list filtering and navigation logic
- focused tests for terminal-help sourcing

Manual after each phase:

- run the TUI in the real app
- inspect proportions, spacing, footer position, title row placement, selection rendering, and overflow behavior
- do not proceed to the next phase until the current phase is visually approved

## Success Criteria

- `quit` is visually correct and becomes the approved modal reference.
- `help` is delivered incrementally in approved phases.
- The `help` modal matches the intended Crush-like structure without copying irrelevant Crush-specific behavior.
- The list view is visually clean: prompt row, command-name list, colored selection bar, fixed modal size.
- The detail view stays inside the same modal shell and uses terminal help content only.
- Terminal `help` remains unchanged.
- Final verification still includes `go test ./...` and `go build -o flame .` once all phases are approved.
