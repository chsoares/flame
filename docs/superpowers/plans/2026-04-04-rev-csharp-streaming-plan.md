# Rev CSharp Streaming Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reintroduce `rev csharp` as a usable Windows payload path, remove the old `rev` IP/port argument behavior, and prepare the C# payload to become the candidate worker payload for better Windows streaming.

**Architecture:** Simplify `rev` so it only uses the current listener address and no longer overloads positional arguments as alternate IP/port. Add a C# reverse shell payload generator that can print source and/or compile an executable, then validate whether that payload is suitable for worker-based Windows module execution.

**Tech Stack:** Go, existing payload generator in `internal/payloads.go`, menu/TUI command handling in `internal/session.go` and `internal/tui`, optional local C# compilation, markdown docs.

---

### Task 1: Remove legacy `rev` override args

**Files:**
- Modify: `internal/session.go`
- Modify: docs mentioning `rev [ip] [port]`
- Test: payload/rev unit tests if needed

- [ ] Make `rev` use the active listener IP/port only.
- [ ] Remove the old behavior where positional args can override IP/port.
- [ ] Update help/docs accordingly.

### Task 2: Reintroduce `rev csharp`

**Files:**
- Modify: `internal/payloads.go`
- Modify: `internal/session.go`
- Test: new payload tests

- [ ] Add C# payload generation.
- [ ] Support at least `rev csharp` output in a usable form.
- [ ] If low-friction, support writing/compiling an `.exe` artifact too.

### Task 3: Prepare the streaming follow-up

**Files:**
- Modify: `docs/current-status.md`
- Modify: `docs/testing/windows-baseline.md`

- [ ] Record that the C# payload is the candidate next step for fixing buffered Windows module output.
- [ ] Leave explicit retest steps for trying it as a worker payload after generation is back.
