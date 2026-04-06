# Terminal Help Revamp Design

Date: 2026-04-05
Status: approved for planning

## Goal

Turn Flame's terminal help from a single summary screen into a command-centric help system that keeps `help` as a compact index and adds `help <command>` detail pages, while explicitly postponing any TUI help modal work to a separate later phase.

## Context

The current `help` output already works as a short command index, but it is not enough once command behavior becomes easy to forget after time away from the tool. Commands such as `binbag`, `pivot`, `download`, `upload`, `run`, `rev`, and `ssh` have important caveats or sub-behaviors that are not visible from a one-line description.

At the same time, the user does not want to overcomplicate the interface. The entry points should stay simple:

- `help`
- `help <command>`
- `help <command> <subtopic>` only where that extra depth is justified, such as `help run ps1`

The user also wants implementation to happen in two explicit phases:

1. terminal help first
2. TUI help modal later, with its own dedicated design/plan after terminal help is implemented and validated

## Scope

In scope for this design:

- keep the existing general `help` output as the compact top-level index
- add detailed per-command help through `help <topic>`
- support deeper topics only where the command surface truly needs it, especially `run`
- add tab completion for help topics so typo handling can stay strict
- keep unknown help topics as a plain error instead of suggestion UX
- document this roadmap in `docs/`

Out of scope for this design:

- a TUI modal, palette, sidebar, or any other interactive help browser
- help access through `-h`, `--help`, or command aliases like `spawn -h`
- task-oriented guides such as `help transfers` or `help first-shell`
- rewriting command behavior as part of the help work

## Design Principles

- keep discovery simple
- keep detail close to the command name the operator already remembers
- make help density vary by command instead of forcing every command into the same template
- avoid duplicate sources of truth so future TUI help can reuse the same content
- do not start TUI help work until terminal help has been tested and accepted

## Approaches Considered

### Approach A: One long unitary help screen

Pros:

- lowest implementation cost
- easy to print from one function

Cons:

- becomes hard to scan as command details grow
- encourages burying important caveats in a wall of text
- does not prepare the codebase for later TUI reuse

Decision: rejected.

### Approach B: Short index plus detailed help topics

Pros:

- preserves the existing quick-scan experience
- gives detailed help only when requested
- maps naturally to command names the operator already knows
- creates a reusable content source for a later TUI help browser

Cons:

- requires a small content model instead of one hardcoded print block
- needs topic-aware completion support

Decision: recommended.

### Approach C: Task-oriented help topics

Pros:

- can be friendly to new users

Cons:

- duplicates information for a command set that is still relatively small
- adds another naming system to remember

Decision: rejected for now.

## Chosen Structure

The chosen design is a command-centric help system with two layers:

### Layer 1: `help`

`help` remains the compact command index. It should still group commands by category and keep each entry to a short one-line description.

It should be updated only enough to:

- make sure the list is accurate
- make sure the descriptions are consistent
- mention that `help <command>` exists

### Layer 2: `help <topic>`

Each command or justified subtopic gets its own detailed help entry.

Examples:

- `help spawn`
- `help download`
- `help upload`
- `help ssh`
- `help rev`
- `help binbag`
- `help pivot`
- `help run`
- `help run ps1`
- `help run dotnet`
- `help run elf`

Not every topic must have the same section count. Small commands should stay small. Dense commands may include extra explanation blocks.

## Content Model

The content should be stored in a reusable internal help registry rather than scattered hardcoded strings.

Each help entry should support a flexible subset of these fields:

- title
- summary
- usage
- details
- examples
- notes

Usage of those fields should be selective:

- `use` may only need summary and usage
- `binbag` may need summary, usage variants, operational details, and notes
- `run` needs an overview plus pointers to deeper subtopics

The system should not force empty headings for fields a command does not need.

## Topic Model

Topics should be explicit strings rather than inferred from command parsing.

Representative topic keys:

- `help`
- `rev`
- `ssh`
- `sessions`
- `use`
- `kill`
- `shell`
- `upload`
- `download`
- `spawn`
- `modules`
- `run`
- `run ps1`
- `run dotnet`
- `run elf`
- `run py`
- `binbag`
- `pivot`
- `config`
- `clear`
- `exit`

Aliases may resolve to the same topic content when that helps keep the UX natural, for example `list` resolving to `sessions` and `quit` resolving to `exit`, but the visible help entry should still stay command-centric.

## Rendering Rules

### General help

- keep the existing boxed layout
- keep category grouping
- add one short footer line such as `Type 'help <command>' for details`

### Detailed help

- use the same Flame box styling so the output feels like part of the same product
- use compact section labels only when useful
- avoid giant paragraphs
- keep examples realistic and minimal
- only include notes/caveats when they actually prevent operator confusion

## Completion and Errors

- `help` topic completion should use the same tab-completion path as other commands
- completion should include multi-word topics where defined, especially `run` subtopics
- unknown help topics should return a plain error such as `Unknown help topic: run foo`
- there should be no did-you-mean suggestions in this phase

## Code Boundaries

The current `showHelp()` block in `internal/session.go` is sufficient for the top-level help today, but detailed help will become easier to maintain if content is extracted into a focused helper.

Recommended boundary:

- `internal/help.go` owns help topics, aliases, formatting helpers, and rendering functions
- `internal/help_test.go` covers topic lookup, aliases, and rendered content expectations
- `internal/session.go` keeps command dispatch and delegates rendering/lookups to the help helper
- existing completion code in `internal/session.go` extends to use the help helper for topic completion

This keeps help content reusable for the later TUI phase without designing the TUI now.

## Validation Strategy

Before any TUI planning starts, the terminal help phase must be validated manually.

Minimum manual checks:

- `help`
- `help spawn`
- `help download`
- `help binbag`
- `help pivot`
- `help run`
- `help run ps1`
- tab completion for `help d`, `help run p`, and similar partial topics
- unknown topic behavior

## Sequencing Decision

The user has already identified two command areas worth revisiting before executing this help implementation work:

- `rev` UX/content
- real testing of `ssh`

That review should happen before implementation starts, but the help revamp plan should be saved now so the work can begin as soon as those command decisions are settled.

## Future Phase Boundary

After terminal help ships and is accepted, a new design/plan should cover the TUI help modal separately.

That later phase should assume:

- terminal help content is already the source of truth
- the modal opens from a dedicated hotkey
- the modal starts from a searchable list view

But those UI decisions are not part of this implementation scope.

## Final Design Decision

Keep `help` as Flame's compact command index, introduce a reusable command-topic help registry for `help <topic>`, add tab completion for help topics, keep unknown-topic behavior strict, and explicitly defer all TUI help modal work to a separate later design and implementation cycle.
