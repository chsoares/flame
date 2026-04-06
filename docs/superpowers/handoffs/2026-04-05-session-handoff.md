# Session Handoff - 2026-04-05

## Done

- SSH background handoff stays on PTY, but the PTY probe is now silent.
- The `PTY_TEST_OK` marker was removed from the probe path.
- Shared TUI input editing and SSH password normalization remain in place.

## Verified

- `go test ./...`
- Focused SSH/PTY tests pass.

## Notes for Next Session

- If SSH behavior regresses, check `internal/pty.go` first.
- Keep ignoring the unrelated deleted docs already present in the worktree.
