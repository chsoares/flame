# TUI Current Status — 2026-04-03

## What's Done

### Core TUI (Phase 1-3)
- Bubble Tea TUI with 3-tier responsive layout (Full/Medium/Compact)
- Per-session output buffers and command history
- Shell relay with PTY upgrade, F12 detach/re-attach
- Mouse text selection, clipboard copy (OSC 52 + native)
- Transient scrollbar, notification bar (3 levels), inline spinner
- Quit confirmation (Ctrl+D double-press, exit/quit modal)
- ANSI-aware word-wrap and line truncation
- SIGWINCH → PTY resize (debounced, relay-level suppression)

### Transfers (Upload/Download)
- Async fire-and-forget with `transferDoneFunc` callback
- Status bar progress overlay (hatching `/` for uploads, size counter for downloads)
- SmartUpload: HTTP via binbag (fast) → b64 chunks (fallback)
- SmartDownload: HTTP POST from remote → b64 markers (fallback)
- Stop/restart relay during transfers for exclusive conn access
- `stty -echo` + 1KB chunks for PTY-upgraded shells
- Tilde expansion, Ctrl+C/Esc cancel, Enter/F12 blocked during transfer
- Notification z-depth: error/important > progress > info

### Bang Mode (`!` prefix in shell)
- `!` enters gummy command mode with magenta prompt
- Tab completion, menu history, contextual placeholder
- `!upload`, `!download`, `!spawn` work from shell context
- Output goes to menuBuffer (visible on detach)

### Spawn
- Async with TUI spinner, non-blocking
- Uses `ReverseShellGenerator` (single source of truth for payloads)
- Payload echo suppressed via `stty -echo` + relay suppression (future timestamp mode)
- Works from menu and bang mode

### Binbag System
- `binbag ls` — multi-column file listing
- `binbag on/off/path/port` — full management with auto-persist
- Upload path fallback: CWD → binbag (automatic)
- Tab completion merges CWD + binbag files
- HTTP POST endpoint for fast downloads (remote POSTs to gummy)
- Status shown in splash screen and sidebar

### Pivot
- `pivot <ip>` — IP-only, ports preserved from original services
- Affects all URLs/payloads: binbag HTTP, rev, spawn, ssh
- `GetPivotIP()` single accessor for all consumers
- Not persisted (session-specific)

### UI/UX
- F11 toggles sidebar (collapse/expand with layout recalculation)
- F12 toggle: attach (menu) / detach (shell)
- Tab completion: commands, local paths, binbag files, subcommands
- Status bar hints contextual per mode
- Sidebar: listener, pivot, CWD, binbag (Full) / listener only (Medium)
- Suppress double blank lines in menu output

### Architecture
- All `p.Send` wrapped in `go` to prevent deadlock
- Dual-mode relay suppression (future=suppress all, past=stty filter only)
- `isSttyEcho` matches any `stty` command + bare prompts
- Stealth/speed toggle removed — modules define own execution mode

## What's Next — Priority Order

### 1. Per-Command Help
Every command needs `help <cmd>` with detailed behavior:
- `help upload` — path resolution (CWD → binbag), HTTP vs b64, remote destination
- `help download` — HTTP POST vs b64, local destination
- `help binbag` — what it is, subcommands, file listing
- `help spawn` — what it does, platform detection, pivot integration
- `help run` — module list, execution modes, platform requirements
- `help pivot` — what it affects, IP-only rationale

### 2. Modules via TUI
Same async pattern as transfers:
- Review each module: keep/remove/modify
- Async execution with spinner + streaming output
- `!run peas` from shell mode

### 3. Windows Testing
- Shell relay in readline mode (no PTY)
- `isSttyEcho` false-positive on `>` prompts — verify
- Transfers with/without binbag
- Modules (WinPEAS, PowerUp, etc.)
- Spawn with PowerShell payloads

### 4. Upload/Download Test Matrix
Systematic testing:
- Local file: CWD-relative, absolute, ~/tilde
- Binbag: filename resolution order (CWD → binbag)
- URL source: download then upload
- Remote destination: explicit vs default
- Large files: PTY mode (1KB chunks) vs raw (32KB)
- With/without binbag, Linux and Windows

### 5. Polish & Small Features
- Session switching shortcuts (Ctrl+1/2/3)
- Remote tab completion (p0wny-shell style — send completion queries to remote)
- `+N more` indicator when sessions overflow sidebar
- Sidebar scroll (if needed)

### 6. Suggestions for Discussion
- **help system**: interactive? `help` alone could show categories, `help upload` shows detail. Or a single scrollable help page?
- **session auto-select**: when only one session exists, auto-select it? Currently requires `use 1` every time.
- **persistent menu history**: save to `~/.gummy/menu_history.txt` across restarts?
- **config command**: currently shows raw config. Worth making it prettier? Or is `binbag`/`pivot`/`config` enough?
- **rev command with pivot**: `rev` should show payloads with pivot IP when enabled. Currently uses `GetPivotIP()` — verify.
- **kill from bang mode**: `!kill 1` should work but currently goes through sync ExecuteCommand. Needs testing.
- **multiple listeners**: future? listen on multiple ports simultaneously.

## Important Notes for Handoff

- **Two-computer workflow**: handoff context must be in docs/ (not .claude/ memory)
- **Build the binary**: always `go build -o gummy .` before asking user to test
- **UI consistency**: always use `ui/colors.go` helpers, never raw styles. Emdash instead of parentheses.
- **p.Send deadlock**: ALL `p.Send` calls in callbacks MUST use `go p.Send(...)`
- **PTY transfers**: stop relay + stty -echo + 1KB chunks when relay was active
- **Relay suppression**: future timestamp = suppress all (spawn), past timestamp = stty filter (resize)
- **Pivot is IP-only**: ports preserved from original services

## File Map

```
internal/tui/
├── app.go          # Root model, Update/View, shell relay, buffers, bang mode, transfers, spawn
├── styles.go       # Color palette, text styles
├── logo.go         # ASCII art banner
├── layout.go       # 3-tier layout (Full/Medium/Compact), F11 toggle
├── header.go       # Compact header bar
├── statusbar.go    # Hotkey hints, notification overlay, transfer progress bar
├── input.go        # Text input, per-context history, bang mode
├── outputpane.go   # Scrollable viewport, word-wrap, selection, spinner
├── selection.go    # Mouse selection state
├── clipboard.go    # OSC 52 + native clipboard
├── focus.go        # FocusMode enum
├── dialog.go       # Modal dialog (quit/kill confirmation)
└── messages.go     # All tea.Msg types

internal/
├── session.go      # Manager, commands, binbag/pivot, transfers, spawn, modules (~3000 LOC)
├── transfer.go     # Upload/Download/SmartUpload/SmartDownload, HTTP + b64
├── fileserver.go   # HTTP server (GET=serve, POST=receive), progress tracking
├── config.go       # TOML config structure
├── runtime_config.go # Thread-safe runtime config, auto-persist, pivot
├── shell.go        # Shell I/O handler, PTY/readline modes
├── pty.go          # PTY upgrade system
├── modules.go      # Module registry and implementations
├── listener.go     # TCP listener
├── payloads.go     # Reverse shell payload generators
├── ssh.go          # SSH + auto reverse shell
├── netutil.go      # Network utilities
├── downloader.go   # HTTP file downloader
└── terminal.go     # Terminal opener
```

## Branch: `tui` (based on `main`)
