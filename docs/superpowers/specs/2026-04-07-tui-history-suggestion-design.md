# TUI History Suggestion Design

Date: 2026-04-07

## Goal

Add fish-style history suggestions to the TUI input without changing how history is stored today.

The feature must work in both input contexts that already exist:

- menu input with persistent history in `.flame`
- shell input with per-session ephemeral history

## Scope

This design adds two behaviors on top of the existing history implementation.

1. Inline suggestion
- While the user types, show the most recent history entry that starts with the current input.
- The typed prefix stays in the normal input color.
- The suggested suffix renders in subtle gray.
- The suggestion is accepted with the right arrow key.

2. Filtered history navigation
- When the current input is non-empty, `up` and `down` navigate only through history entries that start with the current input.
- Matching is prefix-only.
- Example: typing `run` means history navigation only walks entries shaped like `run*`.

Out of scope:

- changing persistence format or storage location
- merging menu and shell histories
- changing `Tab` completion behavior
- introducing fuzzy matching or substring matching

## Existing Context

The current TUI input already centralizes command history in `internal/tui/input.go`.

- `menuHistory` stores persistent menu commands
- `sessionHistory` stores shell history keyed by session ID
- `HistoryUp` and `HistoryDown` currently walk the full active history
- `View()` delegates directly to `textinput.View()`

The app-level key routing lives in `internal/tui/app.go`.

- `up` and `down` already call `a.input.HistoryUp()` and `a.input.HistoryDown()`
- `tab` in menu and bang mode already delegates to `executor.CompleteInput()`
- shell input does not currently use `Tab` completion

This means the smallest correct change is to keep storage as-is and extend the `Input` model with suggestion and filtered-navigation state.

## Recommended Approach

Keep all history suggestion logic inside `internal/tui/input.go`, with a small shared helper for matching history entries.

Why this approach:

- it preserves the existing history ownership model
- it gives menu and shell exactly the same matching rules
- it avoids a larger refactor of `app.go`
- it keeps `Tab` completion fully independent from history suggestion

Alternative approaches considered and rejected:

1. Duplicate the behavior in menu and shell routing
- Rejected because the rules would drift quickly.

2. Replace history with a new shared state object
- Rejected because it is larger than the problem and risks touching persistence and session behavior unnecessarily.

## Behavior Design

### Inline Suggestion

The suggestion source is the active history slice returned by the existing context-sensitive `history()` method.

Rules:

- suggestion only appears when the current input is non-empty
- suggestion only appears when there is a history item with the same prefix
- suggestion uses the most recent matching item
- if the current input already equals the matching entry, no suggestion is shown
- if the cursor is not at the end of the input, no inline suggestion is shown

Rendering:

- render the prompt normally
- render the typed value with the existing `textinput` styles
- render the remaining suffix using `styleSubtle`

Acceptance:

- pressing `right` accepts the current suggestion
- accepting a suggestion replaces the input value with the full suggested command and moves the cursor to the end
- `Tab` remains untouched and still drives command/path completion

### Filtered History Navigation

When the current input is non-empty, history navigation uses a filtered view of the active history.

Rules:

- filter by `strings.HasPrefix(entry, currentInput)`
- preserve original history order
- `up` starts at the newest matching entry
- repeated `up` walks older matching entries
- `down` walks toward newer matching entries
- navigating past the newest matching entry restores the original typed prefix

When the current input is empty:

- keep the existing full-history behavior unchanged

This restoration behavior matters because it matches shell expectations: if the user typed `run`, browsed older `run*` commands, and then returned back down past the newest match, the input should go back to `run`, not to an empty string.

## State Changes

Add minimal transient state to `Input`.

- `historyPrefix string`
  - remembers the typed prefix that started the current filtered navigation session
- `filteredHistory []string`
  - current prefix-matched history slice
- `filteredHistIdx int`
  - navigation index within `filteredHistory`

This state is UI-only and must be reset when:

- input context changes
- bang mode enters or exits
- input is cleared or submitted
- the user edits the text after navigating history

## Key Handling

`internal/tui/app.go` remains the app-level router.

Changes:

- keep `up` and `down` mapped to input methods
- add `right` handling before the default textinput update path
- on `right`, ask the input whether there is an acceptable suggestion
- if accepted, consume the key event
- otherwise, fall through so normal cursor movement still works

This keeps the acceptance logic explicit and avoids coupling it to `Tab` completion.

## Error Handling And Edge Cases

- duplicate history entries are allowed; the newest matching entry wins for inline suggestion
- whitespace is matched literally; no normalization beyond the existing input value
- if there is no suggestion, `right` should behave as it does today
- shell bang mode uses menu history, exactly like the current history implementation
- filtered navigation should not mutate stored history

## Testing Strategy

Add focused tests around `internal/tui/input.go` and the app input routing.

Required coverage:

- inline suggestion picks the most recent prefix match
- suggestion is hidden when input is empty
- suggestion is hidden when the input already equals the newest match
- suggestion is accepted by `right`
- `Tab` path remains unchanged
- `up` and `down` filter by prefix only
- leaving filtered history navigation restores the original typed prefix
- menu and shell contexts apply the same matching rules against their own history sources

## Implementation Boundary

This work should stay small.

- do not touch history persistence code except where resets are needed
- do not change `CommandExecutor`
- do not add new top-level TUI files unless testing pressure clearly requires it

The preferred implementation is a focused edit in `internal/tui/input.go`, a small key-routing edit in `internal/tui/app.go`, and tests.
