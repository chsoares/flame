# TUI Current Status — 2026-04-03

## What's Done

### Phase 1 — Menu Mode + Visual Polish
- Bubble Tea TUI replaces readline REPL
- All menu commands work via stdout capture (`ExecuteCommand`)
- Background notifications route to TUI (`notifyTUI` callback)
- Word-wrap, scroll, command history
- Visual redesign: ASCII art banner, 3-tier responsive layout, sidebar, splash screen
- Color palette: base(253)/muted(245)/subtle(240), magenta(5)/cyan(6) accents

### Phase 1.5 — Mouse, Scrollbar, Notifications, Spinner
- Mouse text selection, clipboard copy (OSC 52 + native)
- Transient scrollbar, notification bar (3 levels), inline spinner
- Session list styling in sidebar

### Phase 3 — Shell Relay
- PTY upgrade on shell entry, bidirectional I/O relay
- F12 detach/re-attach, ANSI sanitization, async PTY upgrade
- Session disconnect detection

### Quit Confirmation, Rendering & Shell Fixes
- Ctrl+D double-press, exit/quit modal dialog
- ANSI-aware word-wrap and line truncation
- Shell viewport isolation, PTY resize on re-attach, splash auto-dismiss

### Per-Session Buffers + History (2026-04-02)
- Per-session output buffers and command history
- Menu buffer separate from session buffers
- Save/restore on context switch (shell ↔ menu)

### SIGWINCH → PTY Resize (2026-04-02)
- Debounce (150ms) + relay-level suppression (atomic timestamp, 500ms window)
- `isSttyEcho()` filter, goroutine-safe via `sync/atomic`

### Stealth/Speed Toggle Removal (2026-04-02)
- Removed entirely — each module defines its own fixed execution mode

### Upload/Download via TUI (2026-04-02 → 2026-04-03)
- Async transfers via goroutines with fire-and-forget pattern
- `transferDoneFunc` callback (no blocking tea.Cmd)
- All `p.Send` wrapped in goroutines to prevent deadlock
- Status bar progress overlay (full-width, blue background, `/` hatching for uploads, size counter for downloads)
- Notification z-depth: error/important > progress > info
- Stop/restart relay during transfers for exclusive conn access
- `stty -echo` before PTY transfers to prevent backpressure
- 1KB chunks when PTY active (vs 32KB raw shell), based on `wasRelaying` not `ptyUpgraded`
- Tilde expansion (`~/path`) in upload/download paths
- Input blocked (Enter) during active transfer/spinner
- F12 blocked during transfer
- Ctrl+C/Esc cancels active transfer via context cancellation

### Bang Mode — `!` Command Prefix in Shell (2026-04-03)
- `!` as first char in shell enters gummy command mode
- Magenta prompt, contextual placeholder, menu history
- Tab completion works (same as menu)
- Backspace on empty / Ctrl+C / Ctrl+D exits bang mode
- Output goes to menuBuffer (visible on detach)

### Binbag Command System (2026-04-03)
- `binbag` / `binbag ls` — multi-column file listing
- `binbag on/off` — enable/disable HTTP server
- `binbag path <dir>` — set directory (tilde expansion)
- `binbag port <N>` — set HTTP port
- Auto-persist config on every change (removed `config save`)
- Removed `set` command entirely
- Upload path fallback: CWD → binbag (automatic)
- Always uses SmartUpload (HTTP + b64 fallback)
- Tab completion merges CWD + binbag files

### HTTP POST Download (2026-04-03)
- FileServer accepts POST/PUT: remote `curl --data-binary @file URL` → saved to CWD
- SmartDownload: tries HTTP POST first, falls back to b64 markers
- 10s inactivity timeout (resets on data received)
- Temp file in binbag (`.dl_` prefix) → moved to CWD on completion

### Pivot Refactor (2026-04-03)
- Pivot is IP-only (ports preserved from original services)
- `pivot <ip>` — all URLs/payloads use this IP (binbag, rev, spawn, ssh)
- `pivot off` — revert to listener IP
- `GetPivotIP()` used by rev, spawn, ssh for payload generation
- Not persisted (session-specific)
- Sidebar: "pivoting enabled" / "via ip" (Full mode only)

### Tab Completion (2026-04-02 → 2026-04-03)
- Commands, local paths, binbag files, binbag/pivot subcommands
- `CompleteInput()` on Manager reuses `GummyCompleter`

### Sidebar Reorganization (2026-04-03)
- Listener IP:port first, then pivot (if active), CWD, binbag
- Medium layout: only listener + sessions (saves space)
- Full layout: all info sections
- Session counter hidden in medium mode
- Binbag status in splash screen

### Status Bar Hints (2026-04-03)
- Shell: `! gummy cmd • F12 detach • F11 sidebar • PgUp/PgDn scroll • Ctrl+C interrupt • Ctrl+D quit`
- Menu: `Tab complete • F11 sidebar • PgUp/PgDn scroll • Ctrl+C cancel • Ctrl+D quit`

## What's Next

### Priority 0: Fix Spawn Command
- Spawn currently uses `listenerIP` directly — needs to use `GetPivotIP()`  ✓ (already done)
- But spawn may have other issues — needs testing and review

### Priority 1: Per-Command Help
- `help upload` — path resolution, binbag fallback, remote destination
- `help download` — HTTP POST vs b64, local destination
- `help binbag` — what it is, subcommands
- `help run` — module list, execution modes

### Priority 2: Modules via TUI
- Same async pattern as transfers
- Review each module: keep/remove/modify

### Priority 3: Windows Testing
- Shell relay in readline mode (no PTY)
- Transfers with/without binbag
- Modules (WinPEAS, PowerUp, etc.)

### Priority 4: Upload/Download Test Matrix
- Local file: CWD-relative, absolute, ~/tilde
- With binbag: filename resolution order (CWD → binbag)
- URL source, remote destination default
- With/without binbag, Linux and Windows

### Future
- F11 sidebar toggle
- Session switching shortcuts (Ctrl+1/2/3)
- Remote tab completion (p0wny-shell style)

## Important Notes for Handoff

- **Two-computer workflow**: handoff context must be in docs/ (not .claude/ memory)
- **Build the binary**: always `go build -o gummy .` before asking user to test
- **UI consistency**: always use `ui/colors.go` helpers (ui.Success, ui.Error, etc.), never raw styles
- **Naming**: objective/descriptive names, no marketing. Emdash instead of parentheses.
- **p.Send deadlock**: ALL `p.Send` calls in callbacks MUST use `go p.Send(...)` to prevent deadlock when called from within Update
- **PTY transfers**: stop relay + stty -echo + 1KB chunks when relay was active
- **Pivot is IP-only**: ports preserved from original services (listener port, HTTP port)

## File Map

```
internal/tui/
├── app.go          # Root model, Update/View, shell relay, per-session buffers, bang mode, transfers
├── styles.go       # Color palette, text styles
├── logo.go         # ASCII art banner
├── layout.go       # 3-tier layout (Full/Medium/Compact)
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
├── session.go      # Manager, commands, binbag/pivot, upload/download, spawn, modules
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
