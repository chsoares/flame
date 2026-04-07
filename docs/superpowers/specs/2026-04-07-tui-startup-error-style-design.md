# TUI Startup Error Style Design

Date: 2026-04-07

## Goal

Replace the legacy bordered startup banner with the TUI splash banner, and render pre-launch/startup errors using the same TUI error pattern as command failures: a transient status-bar notification plus a message in the output pane.

## Scope

This covers errors that still escape as plain terminal text or legacy helper strings before or during TUI startup.

Included:

- legacy `ui.Banner()` / `ui.SubBanner()` startup prints
- listener startup failure from `main.go`
- runtime/config bootstrap failure from `main.go`
- TUI startup failures returned by `tui.Run`
- PTY upgrade failure fallback text

Excluded:

- normal in-TUI command errors already rendered by `menuAppend(ui.Error(...))` or `output.Append(...)`
- banner/logo work
- shell command logic

## Current Behavior

- `main.go` prints the legacy bordered banner via `ui.Banner()` and `ui.SubBanner()` before startup/help flows.
- `main.go` prints startup failures directly with `fmt.Println(ui.Error(...))` and exits.
- `main.go` prints `TUI error: ...` the same way if `tui.Run` returns an error.
- PTY fallback text still lives in `internal/ui/colors.go` as a standalone helper string.
- In the TUI itself, several command failures already use the target pattern:
  - menu-side failures append `ui.Error(...)` to the output pane
  - shell write failures append a red error line to the output pane and return to menu context

## Recommended Approach

Keep the startup path split from the interactive TUI, but make both layers render errors through the same user-facing style helper and use the splash banner instead of the legacy bordered banner.

Why this approach:

- the TUI already owns the visual language for errors
- startup failures occur before the app loop exists, so they cannot use status notifications yet
- a shared helper keeps the text consistent while letting each layer present it in the right place
- the splash banner already exists and matches the TUI look, so it should replace the bordered banner everywhere it is currently printed

## Behavior Design

### Startup and Pre-launch

For errors before `tui.Run` starts:

- print the same error body text used inside the TUI
- use the TUI error color/style helper, not raw `fmt.Println`
- keep exit behavior unchanged

This includes:

- legacy banner prints in `main.go`
- config init failure
- listener start failure
- `tui.Run` error

### PTY Failure

Replace the legacy PTY fallback text helper with a TUI-consistent error message.

Desired behavior:

- status-bar notification when PTY upgrade fails
- output-pane line explaining the fallback
- text should match the TUI error tone already used for other runtime failures

### Shared Presentation

Create one shared error formatter in the UI layer for startup/runtime fallback messages.

It should return a TUI-styled string, and startup code can print it while the TUI can append it to the output pane or show it as a notification.

## Data Flow

- `main.go` handles bootstrap failures before TUI exists
- `tui.Run` and startup helpers continue returning `error`
- the caller formats the error through the shared UI helper
- the TUI uses the same helper for the PTY fallback path

## Testing Strategy

Add tests for:

- startup errors render through the shared error helper
- PTY failure message uses the same TUI style helper
- PTY failure path can surface in both notification and output-pane form when the TUI is active
- no legacy plain-text `PTY upgrade failed - using raw shell` output remains

## Implementation Boundary

Preferred files to touch:

- `main.go` for replacing legacy banner prints with the splash and formatting startup failures
- `internal/ui/colors.go` for the shared error helper and PTY message replacement
- `main.go` for startup failure formatting
- `internal/tui/app.go` for routing PTY fallback into notification + output pane
- targeted tests beside those files

The goal is a small cleanup that removes the old plain-text runtime error style without changing the control flow of startup or PTY upgrade.
