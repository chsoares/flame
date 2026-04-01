# TUI Current Status — 2026-04-01

## What's Done

### Phase 1 — Menu Mode + Visual Polish (earlier session)
- Bubble Tea TUI replaces readline REPL
- All menu commands work via stdout capture (`ExecuteCommand`)
- Background notifications route to TUI (`notifyTUI` callback)
- Word-wrap, scroll, command history
- Visual redesign: ASCII art banner, 3-tier responsive layout, sidebar, splash screen
- Color palette: base(253)/muted(245)/subtle(240), magenta(5)/cyan(6) accents

### Phase 1.5 — Mouse, Scrollbar, Notifications, Spinner (this session)
- **Mouse text selection** — click-drag, double-click word, triple-click line
- **Clipboard copy** — OSC 52 (works over SSH) + native (wl-copy/xclip/xsel)
- **Smart word-wrap copy** — joins artificially wrapped lines without `\n`
- **Transient scrollbar** — appears on scroll, fades after 1s, half-block `▐` in dim
- **Notification bar** — full-width overlay on status bar, 3 levels:
  - Info (blue bg, ✹): clipboard copy — 2s duration
  - Important (cyan bg, 🔥): reverse shell received — 4s duration
  - Error (red bg, 💀): session closed — 4s duration
- **Inline spinner** — animated `⠋⠙⠹...` in viewport for async operations
- **Session list styling** — `[]` subtle, selected num cyan, arrow `▶` magenta

### Phase 3 — Shell Relay (this session)
- **PTY upgrade on shell entry** — sized to viewport dimensions (not full terminal)
- **Bidirectional I/O relay** — background goroutine reads `net.Conn` → viewport
- **User input** → `conn.Write(cmd + "\n")` on Enter
- **Ctrl+C** → sends `\x03` (SIGINT) to remote
- **F12** detaches (relay keeps running), re-attach shows accumulated output
- **ANSI sanitization** — keeps SGR (colors), strips cursor movement/screen clear/OSC
- **Async PTY upgrade** — runs in background with spinner, doesn't freeze TUI
- **SIGWINCH disabled** in TUI mode (Bubble Tea manages layout)
- **Session disconnect** → auto-return to menu + error notification
- **Shell lifecycle messages** — `✹ Entering interactive shell` / `➜ Press F12 to detach` / `✹ Exiting interactive shell`

### Other Fixes (this session)
- Shell received color → cyan, session info (✹) → yellow (swapped)
- `UseSession` error now displayed (was silently ignored)
- Kill session deadlock fixed (notify outside mutex)
- Splash screen pushed down ~1/3 of screen
- "Session not found" capitalized

## What's Next

### Priority 1: Per-Session Buffers + History (Architectural)

**This is the most important next step.** Currently there's a single `OutputPane` with one `strings.Builder`. All output (menu commands, shell I/O, notifications) goes into the same buffer. Switching sessions or re-attaching loses context.

#### Target Architecture

Each session gets its own output buffer and command history. The menu gets its own too.

```
App
├── menuBuffer      (strings.Builder)  — gummy menu output
├── menuHistory     ([]string)         — menu command history (up/down)
├── sessionBuffers  (map[int]*Buffer)  — per-session output
├── sessionHistory  (map[int][]string) — per-session command history
└── viewport        (OutputPane)       — single viewport, swaps content on context switch
```

#### Behavior

**In menu mode:**
- Viewport shows `menuBuffer`
- Up/down arrows navigate `menuHistory`
- Background shell output accumulates silently in `sessionBuffers[N]`
- When `shell` typed: last ~3 commands from the session are shown inline as preview:
  ```
  ❯ shell
  ✹ Entering interactive shell
  ➜ Press F12 to detach

  [user@host ~]$ whoami
  root
  [user@host ~]$ cd /tmp
  [user@host /tmp]$
  ```
  Then viewport switches to `sessionBuffers[N]` with full scrollable history

**In shell mode:**
- Viewport shows `sessionBuffers[selectedSession]`
- Up/down arrows navigate `sessionHistory[selectedSession]`
- All relay output goes to `sessionBuffers[selectedSession]`, not to the viewport directly
- The viewport just renders whatever buffer is active

**On detach (F12):**
- Viewport switches back to `menuBuffer`
- The `✹ Exiting interactive shell` message is appended to `menuBuffer`
- Relay goroutine keeps running, output keeps accumulating in `sessionBuffers[N]`
- **User can detach without fear of losing information**

**On re-attach (`shell` again):**
- Viewport switches to `sessionBuffers[N]`
- All output received while detached is there, scrollable
- User sees exactly where they left off + any new output

**On session switch (`use 2` then `shell`):**
- Session 1 buffer preserved, session 2 buffer loaded
- Each session's scroll position could be preserved too (nice-to-have)

#### Implementation Notes

- `OutputPane.SetContent()` already exists for swapping content
- Need a new `OutputPane.GetContent()` or similar to save current buffer
- Relay goroutine needs to write to session buffer (not viewport directly)
- `ShellOutputMsg` handler checks: if this session is active in viewport, also update viewport; otherwise just accumulate in buffer
- Menu buffer captures `ExecuteCommand` output as before
- History arrays are simple `[]string` per context

### Priority 2: SIGWINCH → PTY Resize

Currently the PTY is sized to the viewport at shell entry time. If the user resizes the terminal, the PTY doesn't know. Commands like `ls` format columns wrong.

**Fix:** In `Update()`, when `tea.WindowSizeMsg` arrives and we're in shell mode, send `stty rows H cols W` to the active session's connection. Also update `handler.viewportCols/Rows` for future reference.

This is a simple ~10 line change in `app.go`.

### Priority 3: Upload/Download via TUI

File transfers currently use `captureStdout` which captures the spinner/progress output as text. In TUI mode, we need:

- Progress shown in the notification bar (or inline spinner)
- Transfer runs async (doesn't freeze TUI)
- Success/failure shown via notification overlay
- The existing transfer code (`internal/transfer.go`) writes to stdout — need to redirect or use callbacks

### Priority 4: Modules via TUI

Same pattern as uploads — `run peas`, `run lse`, etc. need:

- Spinner while downloading/executing
- Output streaming to viewport (for modules that stream output)
- Completion notification

### Priority 5: Tab Toggle Sidebar

- Tab collapses sidebar (switches to LayoutCompact even on wide terminal)
- Tab again restores sidebar
- Simple boolean flag `sidebarCollapsed` in App

### Priority 6: Session Switching Shortcut

- `Ctrl+1`, `Ctrl+2`, etc. to switch sessions quickly
- If in shell mode: detach → use N → attach (seamless)
- If in menu mode: just `use N`

## Logging Architecture — Analysis & Proposals

### Current Structure

```
~/.gummy/
├── config.toml                          # Persistent configuration
├── history                              # Menu command history (readline, 1000 lines)
├── shell_history                        # Shell command history (liner, Windows mode)
├── 2026_04_01/                          # Date-based grouping
│   └── 172.20.2.6_chsoares_arch-workstation/  # Per-host directory
│       ├── scripts/                     # Downloaded/uploaded scripts (modules)
│       └── logs/
│           └── session_1.log            # Raw I/O log (append-only)
```

### Current Problems

1. **`session_1.log` is raw I/O dump** — contains ANSI codes, control characters, PTY setup noise. Not human-readable. Not useful for replaying in TUI.

2. **No structured session metadata** — who connected, when, whoami result, platform — only in the log header, not queryable.

3. **Menu history is global** — `~/.gummy/history` is shared across all sessions. In TUI mode with per-session history, this file is irrelevant.

4. **`shell_history` is for the old liner-based Windows mode** — may not be needed in TUI.

5. **Logs don't support the per-session buffer model** — we need to be able to *reload* a session's output history into a buffer on re-attach. Raw I/O logs with ANSI noise can't serve this purpose.

6. **No session index** — no way to list past sessions, their status, duration, etc.

### Proposed Structure

```
~/.gummy/
├── config.toml                          # Persistent configuration
├── sessions/                            # All session data
│   └── 2026-04-01/                      # Date-based grouping
│       └── 001_172.20.2.6_root/         # Sequential ID + host info
│           ├── meta.json                # Session metadata (structured)
│           ├── output.log               # Sanitized output (what the user saw)
│           ├── raw.log                  # Raw I/O (for forensics, optional)
│           ├── history.txt              # Shell command history for this session
│           └── scripts/                 # Downloaded/uploaded files
├── menu_history.txt                     # Gummy menu command history
└── gummy.log                           # Application-level log (errors, events)
```

#### `meta.json` — Structured session metadata
```json
{
  "id": 1,
  "session_id": "a1b2c3d4e5f6",
  "remote_ip": "172.20.2.6",
  "whoami": "root",
  "hostname": "target",
  "platform": "linux",
  "connected_at": "2026-04-01T14:30:00Z",
  "disconnected_at": "2026-04-01T15:45:00Z",
  "pty_upgraded": true,
  "status": "closed"
}
```

#### `output.log` — Sanitized, replayable output
- ANSI color codes **preserved** (for colored replay in TUI)
- Cursor movement, screen clear, OSC **stripped** (same sanitization as `sanitizeShellOutput`)
- Line endings normalized (`\r\n` → `\n`)
- This is what gets loaded into `sessionBuffers[N]` on re-attach
- Human-readable if opened in a terminal (`cat output.log`)

#### `raw.log` — Raw I/O for forensics
- Everything as received from `net.Conn`, no sanitization
- Useful for debugging, forensic analysis, replaying in a real terminal
- Optional (can be disabled in config to save disk)

#### `history.txt` — Per-session command history
- One command per line
- Loaded into `sessionHistory[N]` on re-attach
- Survives gummy restart (future: session persistence)

### Migration Path

1. Start writing `output.log` (sanitized) alongside the existing `session_N.log` (raw)
2. Use `output.log` as the source for per-session buffers
3. Gradually deprecate the old format
4. Add `meta.json` when structured metadata becomes useful

### How This Enables Per-Session Buffers

```
On shell attach (use N + shell):
  1. If sessionBuffers[N] exists in memory → use it (fast path)
  2. If not, check ~/.gummy/sessions/.../output.log → load last N lines
  3. If no log file → start fresh

On relay output received:
  1. Sanitize output
  2. Append to sessionBuffers[N] (in-memory)
  3. Append to output.log (on-disk, async)

On shell command entered:
  1. Append to sessionHistory[N] (in-memory)
  2. Append to history.txt (on-disk)
```

This gives us: hot buffers in memory for fast switching, cold storage on disk for persistence, and human-readable logs for review.

## File Map (Updated)

```
internal/tui/
├── app.go          # Root model, Update/View, splash, sidebar, shell relay, mouse routing
├── styles.go       # Color palette, text styles, hatching/sectionHeader helpers
├── logo.go         # ASCII art letters, renderBannerFull/Compact/Splash
├── layout.go       # 3-tier layout (Full/Medium/Compact), Rect, GenerateLayout
├── header.go       # Compact header bar (narrow mode only)
├── statusbar.go    # Hotkey hints, notification overlay (3 levels), transfer progress
├── input.go        # Text input with ❯ prompt, history, context switching
├── outputpane.go   # Scrollable viewport, word-wrap, selection highlight, inline spinner
├── selection.go    # Mouse selection state, click tracker, word boundaries, text extraction
├── clipboard.go    # OSC 52 + native clipboard (wl-copy/xclip/xsel)
├── focus.go        # FocusMode enum
└── messages.go     # All tea.Msg types (shell, notification, spinner, scrollbar, clipboard)
```

## Branch: `tui` (based on `main`)
