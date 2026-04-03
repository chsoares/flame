# TUI Current Status — 2026-04-03

## What's Done

### Core TUI
- Bubble Tea TUI with 3-tier responsive layout (Full/Medium/Compact)
- F11 sidebar toggle (collapse/expand with layout recalculation)
- F12 toggle: attach (menu) / detach (shell)
- Per-session output buffers and command history
- Shell relay with PTY upgrade, SIGWINCH resize (debounced + relay suppression)
- Mouse text selection, clipboard copy (OSC 52 + native)
- Transient scrollbar, notification bar (3 levels with z-depth), inline spinner
- Quit confirmation (Ctrl+D double-press, exit/quit modal)
- ANSI-aware word-wrap and line truncation
- Persistent menu history (~/.gummy/menu_history.txt, 500 entries)
- Session auto-select (first session auto-selected)
- Sidebar: listener, pivot, CWD, binbag (Full) / listener only (Medium) / +N more overflow
- Exit banner with session log path (only if logs created this instance)
- Compact attach/detach messages with dedicated icons (magenta/blue)

### Transfers
- Async fire-and-forget with `transferDoneFunc` callback
- Status bar progress overlay (hatching `/` for uploads, size counter for downloads)
- SmartUpload: HTTP via binbag → b64 chunks fallback
- SmartDownload: HTTP POST from remote → b64 markers fallback
- Stop/restart relay + `stty -echo` + 1KB chunks for PTY shells
- Chunk size based on `wasRelaying` (menu=fast 32KB, shell=safe 1KB)
- Tilde expansion, Ctrl+C/Esc cancel, Enter/F12 blocked during transfer
- All `p.Send` wrapped in `go` to prevent deadlock

### Bang Mode (`!` prefix in shell)
- `!upload`, `!download`, `!spawn`, `!kill` from shell context
- Tab completion, menu history, contextual placeholder
- Output goes to menuBuffer (visible on detach)
- Kill own session: auto-detach before kill

### Spawn
- Async with TUI spinner, uses `ReverseShellGenerator` (single source of truth)
- Payload echo suppressed via dual-mode relay suppression (future timestamp)
- Works from menu and bang mode

### Binbag System
- `binbag ls/on/off/path/port` with auto-persist config
- Upload path fallback: CWD → binbag (automatic)
- Tab completion merges CWD + binbag files
- HTTP POST endpoint for fast downloads
- Splash screen shows binbag status

### Pivot
- `pivot <ip>` / `pivot off` — IP-only, ports preserved
- Affects: binbag HTTP, rev, spawn, ssh (`GetPivotIP()`)
- Not persisted (session-specific)
- Sidebar display in Full mode

### Session Logging
- Flat dir structure: `~/.gummy/sessions/YYYYMMDD-HHMMSS_IP_user/`
- No date subdirectory (avoids midnight rollover edge cases)
- Log created on first shell attach (lazy init)
- Exit shows log path only if this instance created logs

## What's Next

### Priority 1: Modules via TUI
The last major feature block. Same async pattern as transfers:
- Review each module with user: keep/remove/modify
- Async execution with spinner + streaming output to viewport
- `!run peas` from shell mode
- Module output in new terminal window (existing `terminal.go` opener)
- Current modules: peas, lse, loot, pspy (Linux), winpeas, powerup, powerview, lazagne (Windows), privesc, bin, sh, ps1, net, py (generic)

### Priority 2: Per-Command Help (TUI Modal)
User wants modal-based help, not just CLI print:
- `help upload` — path resolution, binbag fallback, HTTP vs b64
- `help download` — HTTP POST vs b64, local destination
- `help binbag` — subcommands, what binbag is
- `help spawn/run/pivot/config` etc.
- Consider: scrollable modal overlay in TUI

### Priority 3: Windows Testing
- Shell relay in readline mode (no PTY)
- `isSttyEcho` false-positive on `>` prompts
- Transfers with/without binbag
- Modules (WinPEAS, PowerUp, etc.)
- Spawn with PowerShell payloads

### Priority 4: Upload/Download Test Matrix
Systematic testing of all scenarios documented in memory.

### Priority 5: Polish
- Session switching shortcuts (Ctrl+1/2/3)
- Remote tab completion (p0wny-shell style)
- Review `config` command output format

## Important Notes for Handoff

- **Two-computer workflow**: handoff context must be in docs/
- **Build the binary**: always `go build -o gummy .` before testing
- **UI consistency**: always use `ui/colors.go` helpers, emdash not parentheses
- **p.Send deadlock**: ALL callbacks MUST use `go p.Send(...)`
- **PTY transfers**: stop relay + stty -echo + 1KB chunks when relay was active
- **Relay suppression**: future timestamp = suppress all (spawn), past = stty filter (resize)
- **Pivot is IP-only**: ports preserved from original services

## File Map

```
internal/tui/
├── app.go          # Root model, Update/View, relay, buffers, bang mode, transfers, spawn, kill
├── styles.go       # Color palette, text styles
├── logo.go         # ASCII art banner + RenderExitBanner
├── layout.go       # 3-tier layout with F11 toggle support
├── header.go       # Compact header bar
├── statusbar.go    # Hotkey hints, notification overlay, transfer progress bar
├── input.go        # Text input, per-context history, bang mode, persistent history
├── outputpane.go   # Scrollable viewport, word-wrap, selection, spinner
├── selection.go    # Mouse selection state
├── clipboard.go    # OSC 52 + native clipboard
├── focus.go        # FocusMode enum
├── dialog.go       # Modal dialog (quit/kill confirmation)
└── messages.go     # All tea.Msg types

internal/
├── session.go      # Manager, commands, binbag/pivot, transfers, spawn, modules (~3200 LOC)
├── transfer.go     # Upload/Download/SmartUpload/SmartDownload, HTTP + b64
├── fileserver.go   # HTTP server (GET=serve, POST=receive), progress tracking
├── config.go       # TOML config structure
├── runtime_config.go # Thread-safe runtime config, auto-persist, pivot
├── shell.go        # Shell I/O handler, PTY/readline modes, IsPTYUpgraded
├── pty.go          # PTY upgrade system
├── modules.go      # Module registry and implementations
├── listener.go     # TCP listener
├── payloads.go     # Reverse shell payload generators
├── ssh.go          # SSH + auto reverse shell
├── netutil.go      # Network utilities
├── downloader.go   # HTTP file downloader
├── terminal.go     # Terminal opener
└── ui/colors.go    # UI helpers, symbols (attach/detach), color functions
```

## Branch: `tui` (based on `main`)
