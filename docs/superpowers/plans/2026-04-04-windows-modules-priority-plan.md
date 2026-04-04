# Windows Modules Priority Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Finish the highest-value Windows module paths before returning to payload/rev polish.

**Architecture:** Treat module validation as the next operational milestone. Start with the generic runners that unlock multiple tools (`dotnet`, `bin`), then validate the concrete Windows modules that depend on them (`winpeas`, `seatbelt`, `lazagne`). Keep payload/rev work as the next project immediately after modules.

**Tech Stack:** Go, worker-session module execution, Windows PowerShell baseline already validated, binbag HTTP + b64 fallback.

---

### Priority Order

1. `run dotnet` — core path for .NET-based Windows tooling
2. `run bin` — core path for native binaries like LaZagne
3. `run winpeas`
4. `run seatbelt`
5. `run lazagne`

### Why This Order

- `run dotnet` validates the in-memory .NET path that multiple real modules depend on.
- `run bin` validates the generic disk+cleanup path for Windows native tools.
- Once the generic runners are trusted, failures in named modules are more likely to be module-specific, not architectural.

### Payload / `rev` Follow-Up (Next Project)

After the Windows modules block:

1. Revisit the PowerShell one-liner only if real usage still hurts.
2. Reimplement `rev csharp` / compiled `shell.exe`.
3. Decide whether printing raw C# source still has value; current leaning is no.
4. Consider clipboard-oriented subcommands:
   - `rev bash`
   - `rev ps1`
   - `rev php`
5. Consider file-generation helpers such as `rev php shell.php`.
6. Reevaluate whether custom IP/port arguments for `rev` still matter now that pivoting exists.

### Product Opinion Captured

- `cmd` support is low priority and should stay that way unless real usage proves otherwise.
- Printing raw C# source from `rev csharp` feels low-value compared to generating/compiling a usable artifact.
- Clipboard-first payload helpers are likely a better UX improvement than broadening the current `rev` dump output.
