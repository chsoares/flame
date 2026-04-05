# Shared TUI Input Editing Design

Date: 2026-04-05
Status: approved for planning

## Goal

Make Flame's TUI text inputs behave more like a terminal editor by adding shared support for `Home`, `End`, `Ctrl+Backspace`, `Ctrl+Delete`, and `Ctrl+Z` across the menu input, the attached-shell input bar, and future text inputs such as the planned help modal.

## Context

Flame already has a dedicated TUI input bar in `internal/tui/input.go` backed by Bubble Tea's `textinput.Model`. Today, global shortcuts such as `F11`, `F12`, and `Ctrl+C` are handled at the root app level, while ordinary text editing behavior lives in the input component.

The current input already supports cursor movement between words with control-arrow combinations, but it is still missing several editing behaviors that feel standard in terminal-style input fields:

- `Home` to jump to the beginning
- `End` to jump to the end
- `Ctrl+Backspace` to delete the previous word
- `Ctrl+Delete` to delete the next word
- `Ctrl+Z` to clear the whole current input

The user wants these behaviors to be shared across all TUI text-entry surfaces, not implemented ad hoc per screen, and does not want them added to the status-bar shortcut hints because they should feel like obvious editor behavior rather than product-specific commands.

## Scope

In scope:

- shared text-editing behavior for TUI inputs built on `textinput.Model`
- menu input behavior
- attached-shell input bar behavior
- reusable hook point for the future help modal input
- unit coverage for cursor movement, word deletion semantics, and whole-input clearing

Out of scope:

- changing global TUI shortcuts such as `F11`, `F12`, or `Ctrl+C`
- changing remote shell behavior on the victim side
- documenting these editor-style shortcuts in the status bar
- implementing the help modal itself in this phase

## Approaches Considered

### Approach A: Implement each shortcut separately in every screen

Pros:

- quick for the first screen touched

Cons:

- duplicates logic
- almost guarantees drift between menu, shell, and future help inputs
- makes future fixes harder

Decision: rejected.

### Approach B: Add a shared editing helper around `textinput.Model`

Pros:

- keeps behavior consistent across all TUI text inputs
- minimizes changes to the current component structure
- gives the future help modal the same editing semantics for free

Cons:

- requires a small abstraction layer instead of keeping everything inline

Decision: recommended.

### Approach C: Replace the current input implementation entirely

Pros:

- maximum theoretical control

Cons:

- too much churn for a narrow behavior change
- higher regression risk in a part of the TUI that is already working well

Decision: rejected.

## Chosen Design

Flame will keep using Bubble Tea's `textinput.Model`, but will add a shared helper in `internal/tui/` that applies terminal-like editing behavior to any input backed by that model.

The helper will own editor semantics that are local to text editing:

- move cursor to start
- move cursor to end
- delete previous word
- delete next word
- clear the entire input

The root app model in `internal/tui/app.go` will continue to own only global product shortcuts and focus-level routing. It should not become responsible for text-editing internals.

The input component in `internal/tui/input.go` will remain the integration point for the main input bar, but instead of relying only on the default `textinput.Update` behavior, it will first route relevant key events through the shared helper and then fall back to the existing textinput update path for everything else.

## Editing Semantics

### `Home`

Moves the cursor to position 0 without changing the current input value.

### `End`

Moves the cursor to the end of the current input value.

### `Ctrl+Backspace`

Deletes the previous word relative to the cursor.

Expected behavior:

- if the cursor is inside or just after a word, remove that word segment back to the previous word boundary
- if there are spaces directly before the cursor, consume those spaces first, then remove the preceding word
- if already at the beginning, do nothing

### `Ctrl+Delete`

Deletes the next word relative to the cursor.

Expected behavior:

- if the cursor is inside a word, remove forward to the next word boundary
- if there are spaces directly after the cursor, consume those spaces first, then remove the next word
- if already at the end, do nothing

### `Ctrl+Z`

Clears the entire current input value and resets the cursor to the beginning.

Expected behavior:

- if the input has content, remove it all in one action
- if the input is already empty, do nothing
- this is local editor behavior only and must not trigger unrelated product actions

### Word Boundary Rule

For this phase, a word boundary is whitespace-based. That matches the current terminal-like expectation and keeps the implementation small and predictable.

## Code Boundaries

Recommended structure:

- `internal/tui/inputedit.go` — shared helper functions operating on `textinput.Model`
- `internal/tui/inputedit_test.go` — focused tests for cursor jumps and word deletions
- `internal/tui/input.go` — integrate the helper into the main input component
- `internal/tui/app.go` — unchanged in responsibility; it should continue routing global keys only

This keeps the editing behavior reusable without forcing a redesign of the current TUI component graph.

## Failure Handling

- unsupported keys still fall through to the existing `textinput` behavior
- edge positions such as start-of-line and end-of-line should no-op cleanly
- repeated key presses on empty input should remain harmless

## Validation Strategy

Minimum automated checks:

- `Home` from middle of line moves to cursor 0
- `End` from middle of line moves to end
- `Ctrl+Backspace` removes the previous word
- `Ctrl+Backspace` correctly skips adjacent spaces before deleting the word
- `Ctrl+Delete` removes the next word
- `Ctrl+Delete` correctly skips adjacent spaces before deleting the word
- `Ctrl+Z` clears the entire line
- start/end boundary behavior is stable and does not panic

Minimum manual checks:

- menu input bar
- attached-shell input bar
- verify that no new status-bar hint is added for these keys

## Final Design Decision

Add a shared text-editing helper for Bubble Tea text inputs, integrate it into the existing TUI input component, keep global shortcuts separate from line-editing behavior, and make the resulting semantics reusable for the future help modal without advertising these keys in the status bar.
