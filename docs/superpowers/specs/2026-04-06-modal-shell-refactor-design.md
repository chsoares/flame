# Modal Shell Refactor Design

Date: 2026-04-06

## Goal

Consolidate the TUI modal rendering into one shared template so changes to the shell automatically apply to every modal, while keeping each modal responsible only for its own content and behavior.

## Current State

The TUI currently has three modal-related files:

- `internal/tui/dialog.go`
- `internal/tui/helpmodal.go`
- `internal/tui/modal_skeleton.go`

The current setup duplicates shell concerns across modal implementations. That makes small visual fixes easy to miss in one modal while appearing in another.

## Scope

This refactor covers:

- a single shared modal shell template
- refactoring quit/kill and help to use the shared shell
- keeping modal-specific body construction inside each modal file
- removing extra spacing between modal content and the bottom border/footer area

This refactor does not cover:

- changing help modal behavior
- changing quit/kill key handling
- adding new modal types
- redesigning the rest of the TUI layout

## File Layout

Use three modal files, each with one job:

- `internal/tui/modal.go` for the shared shell renderer
- `internal/tui/dialog.go` for quit/kill modal state and body content
- `internal/tui/helpmodal.go` for help modal state, filtering, and body content

If the existing `modal_skeleton.go` is still present after the refactor, it should be renamed or replaced by `modal.go` so there is only one shared shell implementation.

## Shared Shell API

Introduce a small shell template that accepts the common rendering inputs:

- terminal width and height
- base screen content
- modal title
- modal width and/or max height
- body content built by the modal
- optional footer text
- optional body alignment

Suggested shape:

```go
type ModalShell struct {
    Title      string
    Width      int
    MaxHeight  int
    Body       []string
    Footer     string
    BodyAlign  BodyAlign
}

func RenderModalShell(base string, termW, termH int, shell ModalShell) string
```

The shared template is responsible for:

- drawing the border
- drawing the title row inside the border
- reserving the footer area consistently
- keeping the overlay transparent outside the box
- preserving one rendering path for all modals

## Modal Responsibilities

### `dialog.go`

Keep dialog-specific behavior here:

- selecting confirm/cancel
- building the centered quit/kill content
- choosing the modal title
- calling the shared shell renderer

### `helpmodal.go`

Keep help-specific behavior here:

- filtering topics
- tracking list/detail state
- building the help list or detail body
- choosing left-aligned body rendering
- calling the shared shell renderer

## Rendering Rules

- The shell template must not leave a blank line between the footer and the bottom border.
- The shell template must be the only place that draws the common frame.
- Modal-specific code should only produce body content and footer text.
- A shell change should affect every modal automatically.

## Preferred Behavior

- Quit/kill content stays centered.
- Help content stays left-aligned.
- The footer/help hint area remains part of the shared shell, not duplicated in each modal.
- The shell stays visually stable when the body content changes.

## Testing Strategy

Add focused tests that verify:

- the shell renders the title inside the border
- the shell does not add stray blank padding below the footer
- quit and help both use the same shell renderer
- changing the shell affects both modal views

Existing modal behavior tests should remain valid unless they were asserting duplicated shell details.

## Success Criteria

- There is one shared modal shell implementation.
- `dialog.go` and `helpmodal.go` no longer own border/header/footer layout.
- Updating the shell changes both modals.
- The extra blank line below the modal footer is gone.
- The code is easier to reason about because shell concerns and modal content concerns are separated.
