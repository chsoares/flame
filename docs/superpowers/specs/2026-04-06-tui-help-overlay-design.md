# TUI Help Overlay Design

Date: 2026-04-06

## Goal

Add an in-app help overlay to the Bubble Tea TUI so users can browse the same help topics available in the terminal CLI without leaving the app, while preserving existing terminal `help` behavior.

## Scope

This design covers:

- a new `F1` help overlay inside the TUI
- a shared transparent overlay shell for modal UI
- migrating the existing quit dialog to the same overlay shell
- small status bar hint updates to advertise the new shortcut

This design does not cover:

- changing terminal `help` semantics or output
- changing the input-line `help` command behavior inside the TUI
- broader TUI navigation refactors

## Requirements

- `F1` opens a help overlay from inside the TUI.
- The help overlay uses the existing help registry as the source of truth.
- The overlay shows a filter input at the top and a list of help topics below it.
- `Enter` on a selected topic opens that topic's detailed help in the same overlay.
- `Backspace` with an empty filter returns from detail view to the topic list.
- `Esc` closes the overlay.
- The input-line `help` command continues printing help into the terminal output exactly as it does today.
- The quit dialog is restyled to use the same transparent overlay system.
- Overlay titles use lowercase `help` and `quit` to match Flame's UI voice.
- The help overlay keeps its content left-aligned for scanability.
- The quit overlay keeps its content centered.
- The status bar shows `F1 help` before the other function-key hints.
- The status bar no longer shows `Tab complete`.

## Existing Context

The current TUI has a single modal type in `internal/tui/dialog.go`. It renders a centered box, but the modal path effectively replaces the visible screen instead of feeling like an overlay layer shared across multiple modal types.

The current CLI help system lives in `internal/help.go` and already exposes:

- topic lookup through `LookupHelpTopic`
- completion topics through `HelpTopicsForCompletion`
- rendered text for general help and topic detail through `RenderGeneralHelp` and `RenderHelpTopic`

That registry is already the source of truth for terminal help. The TUI design should consume the same registry rather than introducing a second help model.

## Chosen Approach

Implement a small reusable overlay framework inside `internal/tui/` with two content types:

- `help` overlay: filterable topic browser with detail view
- `quit` overlay: existing confirmation content, restyled into the same shell

The shared shell is responsible for:

- drawing a centered bordered box over the existing screen
- keeping the underlying UI visible around the modal instead of replacing it with a darkened full-screen view
- rendering a common title bar with a lowercase title and hatch fill inspired by the compact header

Each overlay content type remains responsible for its own body layout, alignment, key handling, and footer hints.

## Overlay Architecture

### Shared shell

Introduce a reusable overlay renderer that takes:

- terminal width and height
- already-rendered base screen content
- box width and optional height policy
- title string
- inner body string

The renderer centers the box on top of the existing UI and composes the result so the screen remains visible outside the box bounds. This becomes the standard rendering path for modal surfaces in the Flame TUI.

The shell header uses the same visual language for all overlays:

- lowercase title on the left
- cyan or magenta hatch fill to the right, matching the established compact-header feel
- bordered box using existing shared colors from `internal/tui/styles.go`

### Help overlay model

Add a dedicated help overlay model under `internal/tui/` with two states:

- `list` state
- `detail` state

The model tracks:

- current filter text
- filtered topic list
- selected topic index
- current state (`list` or `detail`)
- selected topic entry when showing detail

This stays independent from the existing quit dialog logic, but both use the same overlay shell.

### Registry access

The TUI should not parse terminal-rendered boxes back into data. Instead, `internal/help.go` should expose a small structured accessor for navigable topics.

The minimal addition is a helper that returns the canonical help entries the TUI should browse, while preserving the existing terminal rendering functions untouched. The TUI can then:

- list canonical topics in stable order
- filter by topic, alias, or summary text
- resolve the selected canonical topic back through the existing registry helpers

The CLI keeps using the current rendering path, so terminal help behavior remains unchanged.

## Help Overlay UX

### Open and close

- `F1` opens the help overlay from menu or shell context.
- `Esc` closes the overlay from either list or detail state.
- Opening help does not change the existing input-line `help` command behavior.

### List state

The overlay body begins with a single-line filter field and then the topic list.

Filter behavior:

- typing updates the filter immediately
- filtering matches topic name, aliases, and summary text
- matching is case-insensitive
- when the filter changes, selection resets to the first visible item unless the previous selection is still valid

List behavior:

- arrow keys move selection
- `Tab` and `Shift+Tab` may also move selection if convenient within the current input handling model, but this is optional and not required for v1
- `Enter` opens the selected topic detail
- the list should show canonical topic names, with enough summary context to make scanning easy

The list layout should stay compact and readable in smaller terminals. If the viewport is short, the list may clip rather than trying to render every item at once.

### Detail state

The detail view renders the help topic content for the selected topic using the same registry-backed help data.

Behavior:

- `Enter` from list enters detail
- `Backspace` with an empty filter returns from detail to list
- the previously selected topic remains selected after returning to the list
- detail text stays left-aligned for readability

If the topic content is longer than the modal body allows, the first version may use a clipped static view or basic viewport scrolling, depending on which path fits the current TUI architecture with less risk. The preference is a minimal scrollable detail area if it can be implemented without introducing broader viewport complexity.

## Quit Overlay UX

The quit confirmation remains functionally the same:

- it still blocks the rest of the UI while open
- `Tab` toggles buttons
- `Enter` confirms the selected action
- `Esc` cancels

The changes are visual and structural:

- use the shared transparent overlay shell
- use lowercase title `quit`
- center the dialog content inside the modal body
- preserve the current safe default of selecting cancel

This makes quit and help feel like the same modal system rather than two unrelated UI patterns.

## Layout and Sizing

The overlay box should be wide enough to feel like a command palette rather than a small confirmation popup.

Guidelines:

- prefer a width around the current dialog width or slightly larger on wide terminals
- clamp width to fit small terminals safely with consistent margins
- help can use a taller body than quit
- quit can remain compact vertically

The overlay should not attempt to reshape the underlying layout. It only draws on top of the rendered screen.

## Status Bar Changes

Update the normal status bar hints so they read in this order:

- `F1 help`
- `F11 sidebar`
- `F12 attach` or `F12 detach`
- existing paging and quit/cancel hints

Remove `Tab complete` from the status bar in non-shell mode.

No other status bar semantics should change.

## Testing Strategy

Add focused tests around the new behavior rather than broad snapshot coverage.

Expected test areas:

- help registry accessor returns stable canonical topics without altering existing lookup behavior
- help overlay filtering and selection state transitions
- help overlay detail navigation (`Enter` into detail, `Backspace` back to list)
- overlay shell rendering keeps title styling and compositing rules stable enough for regression checks
- quit dialog rendering still centers content and preserves existing default selection
- status bar hints include `F1 help` and omit `Tab complete`

Existing help tests in `internal/help_test.go` should remain valid without semantic changes.

## Risk Management

### Risk: accidental CLI help regression

Mitigation:

- keep `RenderGeneralHelp` and `RenderHelpTopic` unchanged unless a shared helper truly requires an internal refactor
- add TUI-facing structured accessors instead of repurposing terminal render output
- keep the TUI help feature behind the `F1` overlay path only

### Risk: overlay rendering complexity

Mitigation:

- keep the overlay shell small and purpose-built for centered modal composition
- avoid a general window manager abstraction
- reuse current Lip Gloss composition patterns and existing TUI color helpers

### Risk: detail view overflows on smaller terminals

Mitigation:

- start with strict clipping or minimal scrolling inside the modal body
- do not expand scope into a large reusable viewport framework unless the current content makes it necessary

## Implementation Notes

- Preserve the current `help` command output path in the input flow.
- Prefer minimal changes in `internal/tui/app.go`: route `F1` to open the help overlay and route key handling to the active overlay while it is open.
- Keep overlay-specific rendering and state outside `app.go` where practical so the root model does not accumulate more modal-specific branching than necessary.
- Reuse existing style constants from `internal/tui/styles.go` and existing UI helper conventions.

## Acceptance Criteria

- Users can press `F1` inside the TUI to open a help overlay.
- The help overlay lists help topics from the same registry used by terminal help.
- Users can filter topics, open a topic detail view, and return to the list with `Backspace` on an empty filter.
- The input-line `help` command still prints help into the terminal output exactly as before.
- The quit dialog uses the same transparent overlay shell and shared title-bar style.
- Overlay titles use lowercase `help` and `quit`.
- Help content stays left-aligned; quit content is centered.
- The status bar shows `F1 help` first and no longer shows `Tab complete`.
- `go test ./...` and `go build -o flame .` pass after implementation.
