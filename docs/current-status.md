# TUI Current Status — 2026-04-04

## Docs Map

- `docs/current-status.md` — handoff source of truth
- `docs/architecture/tui-plan.md` — original TUI architecture and phased plan
- `docs/testing/linux-run-py.md` — Linux `run py` validation log
- `docs/testing/windows-baseline.md` — Windows baseline validation log
- `docs/superpowers/specs/2026-04-04-housekeeping-windows-validation-design.md` — approved design record
- `docs/superpowers/plans/2026-04-04-housekeeping-windows-validation-plan.md` — execution plan

## What's Done

### Core TUI
- Bubble Tea TUI with 3-tier responsive layout (Full/Medium/Compact)
- F11 sidebar toggle, F12 attach/detach toggle
- Per-session output buffers and command history
- Shell relay with PTY upgrade, SIGWINCH resize
- Mouse text selection, clipboard, scrollbar, notification bar, spinner
- Quit confirmation, ANSI-aware rendering
- Persistent menu history, session auto-select
- Exit banner with session log path
- Compact attach/detach messages (dedicated icons)

### Transfers
- Async fire-and-forget via `transferDoneFunc` callback
- Status bar progress overlay (hatching `/` for uploads, size counter for downloads)
- SmartUpload (HTTP binbag → b64 fallback), SmartDownload (HTTP POST → b64 fallback)
- Stop/restart relay + stty -echo + 1KB chunks for PTY shells
- Tilde expansion, Ctrl+C cancel, Enter/F12 blocked during transfer
- Windows transfer baseline now validated for binbag HTTP and small-file b64 fallback

### Bang Mode (`!` prefix in shell)
- `!upload`, `!download`, `!spawn`, `!kill`, `!run` from shell context
- Tab completion, menu history, contextual placeholder
- `!use <id>` now detaches safely before switching sessions

### Spawn
- Async with TUI spinner, payload echo suppressed via dual-mode relay suppression
- Visible session numbering no longer skips after hidden worker sessions

### Windows baseline progress
- PowerShell shell attach/render/command echo now works well for short commands
- `Ctrl+C` on Windows is not working yet
- Long-running PowerShell command output is still buffered and arrives in blocks, not as true streaming
- Current evidence points to the Windows reverse-shell payload architecture, not the TUI renderer, as the main limitation
- Upload/download on Windows now work with binbag HTTP and small-file non-binbag fallback
- Transfer cancel via `Ctrl+C` now works for Windows uploads/downloads too
- Detached Windows launcher now keeps the original shell responsive after worker creation and `spawn`

### Binbag + Pivot + Config
- `binbag ls/on/off/path/port`, auto-persist, upload path fallback CWD → binbag
- `pivot <ip>` / `pivot off` — IP-only, affects all URLs/payloads
- Tab completion merges CWD + binbag files

### Module System

**Architecture (Penelope-inspired worker session model):**

The module system spawns an invisible "worker session" to execute modules. This keeps the main session 100% free for shell interaction, uploads, spawns, etc.

**Flow:**
1. `run <module>` → spinner "Spawning worker shell for <module>..."
2. Flame sends spawn payload from main session (suppressed from viewport)
3. `pendingWorker` atomic flag marks the next `AddSession` as a worker
4. Worker session connects — completely invisible:
   - Not shown in list/sidebar/session count
   - No "reverse shell received" notification
   - No detection spinner, stdout/stderr suppressed during detection
   - No "session closed" notification when removed
5. Module executes on worker (BLOCKING):
   - `RunScriptInMemory`: HTTP binbag (`curl -s URL | bash -s -- args`) or b64 fallback
   - `RunBinary`: SmartUpload → `chmod +x` → execute, with `trap EXIT` for cleanup (shred)
   - `ExecuteWithStreamingCtx` captures output to local file
   - Terminal window opens with `tail -f` of local output file
   - Done marker (`__FLAME_DONE_*__`) filtered from output file
6. Spinner stops immediately after setup — notification in viewport
7. When module finishes, worker session is closed and removed (cleanup via trap for binaries)

**Key files:**
- `session.go:StartModule()` — orchestrates spawn → wait → run → cleanup worker
- `session.go:RunScriptInMemory()` — resolves source, builds command, calls ExecuteWithStreamingCtx (BLOCKING)
- `session.go:RunBinary()` — upload → chmod → execute → trap EXIT shred (BLOCKING)
- `session.go:AddSession()` — checks `pendingWorker` flag, suppresses notifications for workers
- `session.go:RemoveSession()` — suppresses disconnect notifications for workers
- `shell.go:ExecuteWithStreamingCtx()` — reads conn, writes to file, filters markers
- `modules.go` — module registry and implementations

**Tested Linux modules:**
- [x] `peas` — LinPEAS (RunScriptInMemory, HTTP binbag) ✅
- [x] `lse` — Linux Smart Enumeration (RunScriptInMemory, args with `--`) ✅
- [x] `loot` — ezpz post-exploitation (RunScriptInMemory) ✅
- [x] `linexp` — Linux Exploit Suggester (RunScriptInMemory) ✅
- [x] `pspy` — process monitor (RunBinary, disk+cleanup, 5min timeout) ✅
- [x] `py <url>` — arbitrary Python script (Linux path fixed and validated) ✅
  - trap EXIT cleanup confirmed working
  - Worker auto-closes after timeout

**Untested Linux modules:**
- [ ] `sh <url>` — arbitrary bash script (RunScriptInMemory)
- [ ] `elf <url|file>` — arbitrary Linux/native binary (RunBinary)

**Windows modules:**
- [x] `winpeas` — WinPEAS (.NET in-memory, RunDotNetInMemory; buffered output caveat) ✅
- [x] `seatbelt` — Seatbelt (.NET in-memory, RunDotNetInMemory; default args `-group=all`; buffered output caveat) ✅
- [ ] `lazagne` — removed from active module registry; native `.exe` runner is a future dedicated project
- [ ] `ps1 <url>` — arbitrary PowerShell script (RunPowerShellInMemory)

**Tested Windows custom runners:**
- [x] `ps1 <url>` — arbitrary PowerShell script (RunPowerShellInMemory) ✅
- [x] `dotnet <url>` — arbitrary .NET assembly (RunDotNetInMemory, in-memory on victim) ✅

**Windows Run* methods still needing attention:**
- `RunPythonInMemory` — Linux path now uses blocking execution, but Windows still needs a dedicated implementation/refactor
- `RunPowerShellInMemory` and `RunDotNetInMemory` are now in the blocking worker-session model and validated at the custom-runner level

## Dead Code Audit

First housekeeping pass complete:
1. `ExecuteScriptFromStdin` — removed
2. `moduleCancel` field on `SessionInfo` — removed
3. `ExecuteWithStreaming` (non-Ctx version) — still required by the current unrefactored Windows/Python runners
4. `WatchForCancel` — still referenced by legacy/CLI-era paths; leave it until those paths are removed or refactored

## Architecture Decisions

### Why Worker Sessions (Penelope Model)
- Single TCP connection cannot be shared between module output streaming and shell interaction
- Penelope solves this with "control sessions" — separate connections for command execution
- Flame spawns a reverse shell from the main session, uses it exclusively for the module
- Main session stays 100% free — shell, spawn, upload, download all work normally

### Why BLOCKING Run* Methods
- Previously launched internal goroutines — caller couldn't know when module finished
- Now the caller (StartModule goroutine) blocks until ExecuteWithStreamingCtx returns
- Clean lifecycle: spawn → detect → run (blocking) → cleanup worker

### Why trap EXIT for RunBinary
- Binaries like pspy run indefinitely — cleanup (shred) can't just be chained with `;`
- `trap 'shred -uz ...' EXIT` fires on any shell exit: natural, SIGHUP (conn close), timeout
- Worker conn close triggers SIGHUP → trap fires → binary cleaned up

### Why Timeout on pspy (not RunBinary)
- RunBinary is generic — most binaries finish on their own
- pspy specifically runs indefinitely — the 5min timeout is on the PSPYModule, not RunBinary
- Each module controls its own timeout via context

### Why No Spinner/Notification During Execution
- Spinner blocks Enter key (by design — prevents concurrent operations)
- Module execution takes minutes — blocking input defeats the purpose of worker sessions
- Instead: spinner during setup only, then viewport notification "output sent to new terminal window"
- No "completed"/"started" overlay — user sees output in the terminal window

## What's Next

### Immediate: Windows modules and customs
1. document any remaining runner/payload fixes needed after real tests
2. keep native Windows executable runner as a separate future problem
3. decide whether to attack buffered Windows module output now or after payload/rev work

### Then: Payloads / `rev`
1. improve the PowerShell oneliner only if real usage still shows pain after module work
2. validate whether the revived `rev csharp` payload can become the better Windows worker payload
3. reconsider the `rev` UX as a product feature, not just a payload dump
4. evaluate clipboard-first subcommands like `rev bash`, `rev ps1`, `rev php`
5. keep `rev` tied to the active listener/pivot instead of reintroducing custom IP/port args

### Then: Other priorities
1. per-command help (TUI modal)
2. upload/download test matrix
3. keep `cmd` support as low priority unless real usage proves it matters
4. audit legacy CLI-era input/output paths that may still leak raw stdout/spinner behavior into the TUI
5. audit hardcoded UI strings/colors/symbols that bypass shared helpers and make UI maintenance harder

## Important Notes for Handoff

- **Build the binary**: always `go build -o flame .` before testing
- **UI consistency**: always use `ui/colors.go` helpers, never raw styles
- **p.Send deadlock**: ALL callbacks MUST use `go p.Send(...)`
- **PTY transfers**: stop relay + stty -echo + 1KB chunks when relay was active
- **Worker sessions**: invisible, use `pendingWorker` atomic flag, suppress all notifications
- **Worker cleanup**: StartModule closes worker conn + RemoveSession after module.Run returns
- **Module args**: use `-- args` separator (not just `args`) to avoid bash option conflicts
- **Module output**: ExecuteWithStreamingCtx filters done markers from local file
- **Binary cleanup**: RunBinary uses `trap EXIT` for shred — fires on conn close
- **Two-computer workflow**: handoff context must live in `docs/`
- **Windows-first rule**: do not refactor Windows module runners again without fresh baseline evidence in `docs/testing/windows-baseline.md`
- **Windows payload caveat**: current PowerShell payload behaves like command/response, not a true streaming terminal; see `docs/testing/windows-baseline.md`
- **Linux gap**: `Ctrl+C` still needs explicit validation on Linux shell relay
- **Roadmap decision (2026-04-04):** modules/custom Windows execution comes before `rev`/payload product polish; payload work is the next major step after modules
- **Runner scope decision (2026-04-04):** `run elf` is explicitly scoped to Linux/native Unix targets; native Windows `.exe` execution needs a separate design later
- **Explicit runner guards (2026-04-04):** unsupported combos should fail early with clear errors (`run sh` on Windows, `run ps1`/`run dotnet` on Linux, `run elf` on Windows)
- **Windows module caveat (2026-04-04):** `run winpeas` and `run seatbelt` work, but their output is still buffered under the current Windows payload path

## File Map

```
internal/
├── session.go      # Manager, commands, StartModule, RunScriptInMemory, RunBinary, worker sessions
├── shell.go        # Handler, ExecuteWithStreaming/Ctx, PTY modes
├── modules.go      # Module registry: peas, lse, loot, linexp, pspy, winpeas, seatbelt, lazagne, elf, sh, ps1, dotnet, py
├── transfer.go     # SmartUpload/SmartDownload, HTTP + b64
├── fileserver.go   # HTTP server (GET=serve, POST=receive)
├── config.go       # TOML config
├── runtime_config.go # Auto-persist, pivot, binbag management
├── payloads.go     # Reverse shell generators (used by spawn + module worker)
├── listener.go     # TCP listener
├── pty.go          # PTY upgrade
├── ssh.go          # SSH + auto revshell
├── netutil.go      # Network utilities
├── downloader.go   # HTTP file downloader
├── terminal.go     # Terminal window opener
└── ui/colors.go    # UI helpers, symbols, color functions

internal/tui/
├── app.go          # Root model, relay, buffers, bang mode, transfers, spawn, modules, kill
├── input.go        # Text input, bang mode, persistent history
├── outputpane.go   # Viewport, spinner
├── statusbar.go    # Hints, notifications, progress bar
├── layout.go       # 3-tier layout + F11 toggle
├── messages.go     # All tea.Msg types
├── styles.go       # Colors
├── logo.go         # Banner + exit banner
├── header.go       # Compact header
├── selection.go    # Mouse selection
├── clipboard.go    # OSC 52 + native
├── focus.go        # FocusMode
└── dialog.go       # Modal dialogs
```

## Branch: `tui` (based on `main`)
