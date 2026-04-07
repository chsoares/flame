# Current Status

## Release State

- `v0.9.0` is the feature-complete release checkpoint for the current TUI surface.
- The app version string is `v0.9.0` and must stay aligned with tags and release docs.
- Older exploratory planning docs have been trimmed; the active handoff lives in the pre-`1.0` roadmap and plan below.

## Next Work

- The next real work is the phased pre-`1.0` TUI refactor.
- Follow `docs/superpowers/specs/2026-04-07-pre-1.0-tui-refactor-roadmap.md` for scope and ordering.
- Follow `docs/superpowers/plans/2026-04-07-pre-1.0-tui-refactor-plan.md` for the execution workflow and user checkpoints.

## Refactor Guardrails

- TUI is the product; do not preserve CLI-era structure at the expense of the shipped app.
- Fix correctness risks first, then replace risky seams, then split files.
- Do not deepen stdout-capture or print-driven internals in TUI-facing flows.
- Add tests around new seams introduced by the refactor, but do not turn the work into a whole-repo cleanup pass.
