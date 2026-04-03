# TUI Current Status — 2026-04-02

## What's Done

### Phase 1 — Menu Mode + Visual Polish
- Bubble Tea TUI replaces readline REPL
- All menu commands work via stdout capture (`ExecuteCommand`)
- Background notifications route to TUI (`notifyTUI` callback)
- Word-wrap, scroll, command history
- Visual redesign: ASCII art banner, 3-tier responsive layout, sidebar, splash screen
- Color palette: base(253)/muted(245)/subtle(240), magenta(5)/cyan(6) accents

### Phase 1.5 — Mouse, Scrollbar, Notifications, Spinner
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

### Phase 3 — Shell Relay
- **PTY upgrade on shell entry** — sized to viewport dimensions (not full terminal)
- **Bidirectional I/O relay** — background goroutine reads `net.Conn` → viewport
- **User input** → `conn.Write(cmd + "\n")` on Enter
- **Ctrl+C** → sends `\x03` (SIGINT) to remote
- **F12** detaches (relay keeps running), re-attach shows accumulated output
- **ANSI sanitization** — keeps SGR (colors), strips cursor movement/screen clear/OSC
- **Async PTY upgrade** — runs in background with spinner, doesn't freeze TUI
- **Session disconnect** → auto-return to menu + error notification
- **Shell lifecycle messages** — `✹ Entering interactive shell #N` / `➜ Press F12 to detach` / `✹ Exiting interactive shell`

### Quit Confirmation
- **Ctrl+D** → double-press: warning notification, second press within 3s quits
- **exit/quit/q** → modal dialog: centered box, Yep!/Nope buttons (Tab toggle, Enter confirm, Esc cancel)
- Reusable `Dialog` component in `dialog.go` (DialogQuit/DialogKill actions)
- Without active sessions, quit is immediate

### Rendering & Shell Fixes
- **ANSI-aware word-wrap** — `wrapContent` uses `ansi.Hardwrap()` instead of broken rune-by-rune loop
- **ANSI-aware line truncation** — `truncateLine` uses `ansi.Truncate()`
- **padViewLines** — every line in View() padded to exactly terminal width
- **Shell viewport isolation** — `CommandOutputMsg` and `spinnerStartMsg` suppressed in shell context
- **PTY resize on re-attach** — re-sends `stty rows H cols W` when relay is already active
- **Splash auto-dismiss** — splash exits on background events

### Per-Session Buffers + History (NEW — 2026-04-02)

Per-session output and command history — each session has its own state, preserved across detach/re-attach.

**Architecture:**
```
App
├── menuBuffer      (strings.Builder)        — gummy menu output
├── menuHistory     ([]string)               — menu command history (up/down)
├── sessionBuffers  (map[int]*strings.Builder) — per-session shell output
├── sessionHistory  (map[int][]string)       — per-session shell command history
└── viewport        (OutputPane)             — single pane, swaps content on context switch
```

**Behavior:**
- Menu mode: viewport shows `menuBuffer`, up/down navigates `menuHistory`
- Shell mode: viewport shows `sessionBuffers[N]`, up/down navigates `sessionHistory[N]`
- On `shell`: saves menu viewport → loads session buffer → switches context
- On F12 detach: saves session viewport → loads menu buffer → switches context
- Background shell output accumulates in `sessionBuffers[N]` silently
- Background menu output (new session, etc.) accumulates in `menuBuffer` silently
- `OutputPane.GetContent()` / `SetContent()` for save/restore
- History index resets on context switch

### SIGWINCH → PTY Resize (NEW — 2026-04-02)

Dynamic terminal resize propagated to remote PTY.

**Approach:**
- **Debounce** (TUI): `WindowSizeMsg` → 150ms debounce via `resizeID` invalidation → single `sendResizeMsg`
- **Suppress** (relay goroutine): `sttyResizeNano` (`atomic.Int64`) marks resize timestamp. Within 500ms, chunks that are stty echo or bare prompts are silently dropped.
- `isSttyEcho()` detects: stty command lines, bare prompts (ending in `$#>%`), empty lines
- No TUI-level content filtering (avoids false positives on legitimate `stty` output)
- Goroutine-safe via `sync/atomic` (no mutex contention in relay hot path)

### Session Directory Refactor (NEW — 2026-04-02)

- New format: `~/.gummy/sessions/2026-04-01/HHMMSS_IP_user/`
- Simplified log: `session.log` directly in session dir (no `logs/` subdir)
- Removed `LogsDir()` method

### Shell Output Routing Refactor (NEW — 2026-04-02)

- `ShellOutputMsg` now carries `NumID` (int) alongside `SessionID` (string)
- `shellOutputFunc` signature: `func(sessionID string, numID int, data []byte)`
- Output only updates viewport if `activeSession == msg.NumID`
- Menu output via `menuAppend()` — writes to buffer + viewport (if menu is active)

### Stealth/Speed Toggle Removal (DONE — 2026-04-02)

Removed the `stealth`/`speed` execution mode toggle entirely. Each module already defines its own fixed execution mode (`memory`, `disk-cleanup`, `disk-no-cleanup`). The toggle was never used in any logic path. Cleaned up: `ExecutionMode` field, `GetMode()`/`SetExecutionMode()`, `set mode` command, `[execution]` config section.

### Upload/Download via TUI (DONE — 2026-04-02)

Async file transfers with full TUI integration:

**Architecture:**
- `StartUpload`/`StartDownload` on Manager run transfers in goroutines
- `Transferer.progressFn` callback routes progress to TUI spinner + status bar
- `transferProgressMsg` → status bar overlay (full-width, blue background, hatching `/` progress bar)
- `transferDoneMsg` → clears status bar, shows result via `ui.Success`/`ui.Error` + notification
- Upload shows percentage in status bar hatching bar; download shows accumulated size (no total known)
- Tilde expansion (`~/path`) supported in upload/download paths

**Status bar progress overlay:**
- Same visual style as notification bar (full-width colored background)
- Upload icon () / Download icon () from `ui/colors.go`
- Hatching `/` fills proportionally for uploads; downloads show size counter
- Clears automatically on completion

### Tab Completion in Menu (DONE — 2026-04-02)

- `CompleteInput(line string) string` on Manager reuses `GummyCompleter` infrastructure
- Tab in menu mode: command completion (empty/partial) + local path completion for upload/download
- Single match auto-completes; multiple matches complete to longest common prefix
- Tilde expansion in path completion works

### Input Improvements (DONE — 2026-04-02)

- `Input.SetValue()` method for programmatic input changes (tab completion)
- `OutputPane.UpdateSpinner()` for live spinner text updates
- `spinnerUpdateMsg` message type for spinner text changes from goroutines
- Status bar hints updated: Tab→complete (menu), F11→sidebar (placeholder)

## What's Next

### Priority 1: `!` Command Prefix in Shell Mode

Run gummy menu commands without detaching from shell. `!upload file.sh /tmp/file.sh` in shell mode routes to upload logic, shows progress in status bar, viewport stays on shell output.

**Implementation:**
- In shell input handler, detect `!` prefix
- Strip `!`, route to `executeInput` as if in menu mode
- Progress/results shown via notification bar (not viewport, since we're in shell context)

### Priority 2: Per-Command Help

Every command needs `help <cmd>` with detailed behavior explanation:
- `help upload` — path resolution (CWD-relative, absolute, ~/tilde, binbag lookup order), remote destination default
- `help download` — remote path, local destination, base64 transport
- `help config` / `help set` — each setting explained
- `help run` — module list, execution modes, platform requirements

### Priority 3: Modules via TUI

Same async pattern as transfers. Modules need:
- Spinner + streaming output to viewport
- Completion notification
- Review each module: keep/remove/modify

### Priority 4: Windows Testing

Shell relay, transfers, and modules all need testing on Windows (cmd + PowerShell).

### Priority 5: Upload/Download Test Matrix

Systematic testing of all upload scenarios (documented in memory):
- Local file: CWD-relative, absolute, ~/tilde
- With binbag: filename resolution order (binbag → CWD)
- URL source: download then upload
- Remote destination: explicit vs default (filename in remote CWD)
- With/without binbag, Linux and Windows

### Priority 3: Windows Testing

Shell relay, transfers, and modules all need testing on Windows (cmd + PowerShell):

- **Shell relay**: TUI input/output in readline mode (no PTY)
- **SIGWINCH**: should NOT fire stty on Windows sessions — verify guard works
- **`isSttyEcho`**: could false-positive on cmd prompts ending in `>` — verify
- **Transfers**: upload/download with/without binbag on Windows targets
- **Modules**: Windows modules (WinPEAS, PowerUp, etc.)

### Priority 4: Tab Toggle Sidebar

- Tab collapses sidebar (switches to LayoutCompact even on wide terminal)
- Tab again restores sidebar
- Simple boolean flag `sidebarCollapsed` in App

### Priority 5: Session Switching Shortcut

- `Ctrl+1`, `Ctrl+2`, etc. to switch sessions quickly
- If in shell mode: detach → use N → attach (seamless)
- If in menu mode: just `use N`

### Priority 6: Remote Tab Completion (p0wny-shell style)

Tab completion without raw relay — send completion queries to the remote shell and display results in the TUI.

**Approach:**
1. User types partial path/command and presses Tab
2. Gummy sends a delimited completion query to the remote:
   - Linux/bash: `echo "GUMMY_COMP_START"; compgen -f -- "partial" 2>/dev/null; echo "GUMMY_COMP_END"`
   - PowerShell: `echo "GUMMY_COMP_START"; Get-ChildItem "partial*" | Select -ExpandProperty Name; echo "GUMMY_COMP_END"`
3. Output between delimiters is captured and parsed (one match per line)
4. Results displayed inline or as popup menu below input bar
5. Single match → autocomplete. Multiple → show list for selection.

## Important Notes for Handoff

- **Two-computer workflow**: work happens on two machines via git. Handoff context must be in docs/ (not .claude/ memory).
- **User hasn't touched CLI internals in months**: transfers, modules, configs need re-explanation before modifying. Don't assume familiarity — walk through the code with the user.
- **Build the binary**: always `go build -o gummy .` before asking the user to test. Previous session lost time debugging because the binary wasn't rebuilt.
- **Naming**: user prefers objective/descriptive names. "stealth/speed" → "memory/disk" or just automatic.

## Logging Architecture — Analysis & Proposals

### Current Structure

```
~/.gummy/
├── config.toml
├── sessions/
│   └── 2026-04-01/
│       └── HHMMSS_IP_user/
│           ├── session.log         # Raw I/O log (append-only)
│           └── scripts/            # Downloaded/uploaded scripts (modules)
├── history                         # Menu command history (readline, 1000 lines)
└── shell_history                   # Shell command history (liner, Windows mode)
```

### Proposed Structure

```
~/.gummy/
├── config.toml
├── sessions/
│   └── 2026-04-01/
│       └── HHMMSS_IP_user/
│           ├── meta.json           # Session metadata (structured)
│           ├── output.log          # Sanitized output (what the user saw)
│           ├── raw.log             # Raw I/O (for forensics, optional)
│           ├── history.txt         # Shell command history for this session
│           └── scripts/            # Downloaded/uploaded files
├── menu_history.txt                # Gummy menu command history
└── gummy.log                       # Application-level log (errors, events)
```

## File Map

```
internal/tui/
├── app.go          # Root model, Update/View, splash, shell relay, per-session buffers
├── styles.go       # Color palette, text styles, hatching/sectionHeader helpers
├── logo.go         # ASCII art letters, renderBannerFull/Compact/Splash
├── layout.go       # 3-tier layout (Full/Medium/Compact), Rect, GenerateLayout
├── header.go       # Compact header bar (narrow mode only)
├── statusbar.go    # Hotkey hints, notification overlay (3 levels), transfer progress
├── input.go        # Text input with ❯ prompt, per-context history, context switching
├── outputpane.go   # Scrollable viewport, word-wrap, selection highlight, inline spinner
├── selection.go    # Mouse selection state, click tracker, word boundaries, text extraction
├── clipboard.go    # OSC 52 + native clipboard (wl-copy/xclip/xsel)
├── focus.go        # FocusMode enum
├── dialog.go       # Modal dialog component (quit/kill confirmation)
└── messages.go     # All tea.Msg types (shell, notification, spinner, scrollbar, resize)
```

## Branch: `tui` (based on `main`)
