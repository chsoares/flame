# Flame TUI Plan - Bubble Tea Implementation

## Context
Flame is a Go reverse shell handler for CTF. Currently uses a readline-based CLI (StartMenu() in session.go). We want a full Bubble Tea TUI inspired by Charmbracelet Crush's architecture, with:

- Live session sidebar
- Shell output viewport with bidirectional I/O relay
- Menu mode for flame commands (upload, modules, config)
- Background session notifications
- Transfer progress in status bar

**Key insight from Crush:** All async I/O (network, processes) flows through a generic pubsub.Broker[T] → bridge goroutines → single chan tea.Msg → program.Send() → Update(). The TUI never blocks on I/O.

**Key UX decision:** Line-buffered input (p0wny-shell style), NOT raw keystroke relay. The input bar is always visible at the bottom. The user types a complete command, presses Enter, and the whole line is sent to the remote shell. Output appears as scrolling history above. This eliminates the keyToBytes problem entirely and enables native history + tab completion.

## Architecture Overview

```
                    ┌─────────────────────────┐
                    │   tea.Program (main)     │
                    │   Init / Update / View   │
                    └────────┬────────────────┘
                             │ tea.Msg
                    ┌────────┴────────────────┐
                    │     Event Bridge         │
                    │  single chan tea.Msg      │
                    │  (buffer 256)             │
                    └────────┬────────────────┘
                             │ goroutines forward
              ┌──────────────┼──────────────────┐
              │              │                  │
    Broker[SessionEvent]  Broker[ShellOutput]  Broker[TransferEvent]
              │              │                  │
         Listener.go    ShellAdapter        Transferer
         Manager.go     (conn.Read loop)    (chunk progress)
```

## Focus State Machine

```
              Tab                    Tab
  FocusInput ◄────► FocusSidebar ────► FocusInput
                         │
                         │ Enter (select session)
                         ▼
                    FocusInput (session changed)
```

There are only two focus modes now (simplified from the raw-relay design):

- **FocusInput** — The input bar is active. User types commands. Behavior depends on the context:
  - Shell context (session selected + shell active): Enter sends command to remote net.Conn + \n. Prompt shows user@host:/path$. Output from remote appears in main pane.
  - Menu context (no shell active, or detached): Enter executes flame command. Prompt shows flame [N] >. Output from flame appears in main pane.

- **FocusSidebar** — Session list highlighted. j/k navigate, Enter selects session.

Context switching:
- `shell` command (in menu context) → enters shell context for selected session
- `F12` / `Ctrl+]` / `detach` → exits shell context back to menu context
- `Ctrl+1..9` → switches to session N's shell context directly
- `Ctrl+S` → toggles sidebar focus

The input bar is always visible in both contexts. Only the prompt and command routing change.

## Layout

```
┌─────────────────────────────────────────────────┐
│ HEADER (1 line fixed)                           │
│ flame 󰗣 │ 10.10.14.5:4444 │ 3 sessions │ SHELL│
├─────────────────────────────────┬───────────────┤
│                                 │ SIDEBAR       │
│ MAIN PANE (flex, scrollable)    │ (28 cols)     │
│                                 │               │
│ www@target:/tmp$ ls -la         │ [1] root@lin  │
│ total 24                        │ [2] user@win  │
│ drwxrwxrwt  6 root root ...    │▶[3] www@lin   │
│ -rw-r--r--  1 www  www  ...    │               │
│ www@target:/tmp$ whoami         │ ───────────── │
│ www-data                        │ Modules:      │
│ www@target:/tmp$ _              │  peas ✓       │
│                                 │               │
├─────────────────────────────────┤               │
│ INPUT (always visible, 1 line)  │               │
│ www@target:/tmp$ _              │               │
├─────────────────────────────────┴───────────────┤
│ STATUS BAR (1 line fixed)                       │
│ [SHELL] F12:menu Tab:sidebar │ upload 45% lp.sh│
└─────────────────────────────────────────────────┘
```

## Shell I/O Strategy: "Line-Buffered Web Shell" (p0wny-shell pattern)

### Input Flow
1. User types command in input bar (with full editing: arrow keys, home/end, history, tab completion)
2. Presses Enter → complete line sent as command + "\n" to net.Conn
3. Command is echoed into the main pane log: `user@host:/path$ command`
4. Input bar clears, ready for next command

### Output Flow
1. Goroutine reads from net.Conn continuously → publishes ShellOutputMsg via broker
2. ShellOutputMsg arrives at Update() → appended to main pane viewport
3. Viewport auto-scrolls to bottom (follow mode)
4. ANSI color codes are preserved and rendered by the viewport

### Why This Works Better Than Raw Relay
- No keyToBytes problem — we send plain strings, not reconstructed escape sequences
- Native editing — Bubble Tea's text input handles cursor movement, selection, etc.
- History — Up/Down arrow navigates command history per session
- Tab completion — Can be implemented client-side (send compgen to remote) or locally
- Consistent UX — Same input bar pattern in shell mode and menu mode
- Ctrl+C — Sends \x03 to remote connection (interrupts running command)

### Limitations (Accepted)
- No interactive programs (vim, htop, nano) — same as p0wny-shell
- No raw PTY features (job control with Ctrl+Z is limited)
- Password prompts that disable echo will show the typed text in the input bar (not on remote)
- For these use cases, users can use a separate terminal with a traditional shell handler

### Prompt Tracking
The remote prompt (user@host:/path$) needs to be tracked. Strategy:
1. After PTY upgrade, set a known prompt format: `export PS1='\\u@\\h:\\w\\$ '`
2. Parse shell output to detect prompt lines (regex match against known format)
3. Update the input bar prompt to match the remote prompt
4. Fallback: use the last known whoami from session detection

### Session Output Buffering
Each session gets a RingBuffer (256KB). Solves:
- Background preservation: Inactive sessions buffer output
- Session switch with history: Switching loads ring buffer into viewport
- Memory bounded: Wraps at capacity

## Event/Message Types

```go
// Session lifecycle
SessionConnectedMsg{SessionID, NumID, RemoteIP}
SessionInfoDetectedMsg{SessionID, Whoami, Platform}
SessionDisconnectedMsg{SessionID, NumID}

// Shell I/O
ShellOutputMsg{SessionID, Data []byte}

// Transfers
TransferProgressMsg{SessionID, Filename, BytesDone, BytesTotal, Done, Err}

// Modules
ModuleOutputMsg{SessionID, Data []byte}
ModuleFinishedMsg{SessionID, ModuleName, Err}

// User actions (from input bar)
SendCommandMsg{SessionID, Command string}   // Enter in shell context
ExecuteFlameMsg{Command string}             // Enter in menu context
SwitchSessionMsg{NumID}
EnterShellMsg{}
ExitShellMsg{}

// Prompt tracking
PromptDetectedMsg{SessionID, Prompt string} // e.g., "www@target:/tmp$"
```

## Package Structure

```
internal/
  tui/
    app.go              # Root tea.Model (Init/Update/View)
    layout.go           # Rectangle calculations
    focus.go            # FocusMode enum + context state
    messages.go         # All tea.Msg types
    broker.go           # Generic Broker[T] pub/sub
    bridge.go           # Bridge goroutines → program.Send()
    ringbuffer.go       # Session output ring buffer
    components/
      header.go         # Top bar (listener info, session count, mode)
      sidebar.go        # Session list with selection + unread indicators
      outputpane.go     # Scrollable output viewport (shell + menu output)
      input.go          # Text input with history + completion + dynamic prompt
      statusbar.go      # Mode indicator + progress + hotkey hints
      notification.go   # Toast overlay for async events
  shell_adapter.go      # Line-based shell adapter: sends lines, reads output (NEW)
  session.go            # MODIFIED: remove readline, expose ExecuteCommand()
  shell.go              # MODIFIED: Handler kept for PTY upgrade, adapter wraps it
  listener.go           # MODIFIED: publish events via broker
  transfer.go           # MODIFIED: publish progress via broker
  modules.go            # MOSTLY UNCHANGED
  pty.go                # MODIFIED: remove SIGWINCH handler (TUI handles)
```

Note: shellpane.go and menupane.go merged into outputpane.go — both modes use the same scrollable viewport, only the content source differs.

## Phased Implementation

### Phase 1: Skeleton TUI + Menu Mode (MVP foundation) ✅ DONE
Goal: Replace readline with Bubble Tea. All flame commands work through TUI input.
- Create: tui/app.go, tui/layout.go, tui/focus.go, tui/messages.go, tui/components/{header,outputpane,input,statusbar}.go
- Modify: main.go (launch TUI instead of StartMenu()), session.go (extract handleCommand() into ExecuteCommand(cmd) string)
- **Current sub-goal:** Polish UI to match Crush's visual quality (ASCII art, ////// fills, integrated header/sidebar, gray help text)

### Phase 2: Event Bridge + Sidebar
Goal: Live session notifications. Sidebar shows sessions in real-time.
- Create: tui/broker.go, tui/bridge.go, tui/components/{sidebar,notification}.go
- Modify: listener.go (publish via broker), session.go (publish detection/disconnect events)

### Phase 3: Shell Context (line-buffered, p0wny-shell style)
Goal: Enter shell context. Type commands in input bar, see output in outputpane.
- Create: shell_adapter.go, tui/ringbuffer.go
- Modify: session.go, tui/app.go, tui/components/input.go

### Phase 4: Session Switching + Background Buffering
Goal: Switch between sessions seamlessly. Each session has its own output history.

### Phase 5: Transfer Progress + Module Integration
Goal: Progress bars. Module output in TUI.

### Phase 6: PTY Resize + Polish
Goal: Terminal resize propagates to remote. Visual refinements.

## Risk Assessment

### HIGH
- Concurrent net.Conn access — mutex-gated mode enum
- Prompt tracking — fragile, set known PS1 after PTY upgrade

### MEDIUM
- Shell output backpressure — drop-on-full pattern + ring buffer
- ANSI in viewport — cursor-movement codes from full-screen programs produce garbage

### LOW
- PTY resize race — debounce stty commands (200ms)
- Command echo deduplication — stty -echo after PTY upgrade

## Dependencies
```
charm.land/bubbletea/v2
charm.land/bubbles/v2
charm.land/lipgloss/v2
charm.land/ultraviolet           # Evaluate stability — may use manual layout instead
```
