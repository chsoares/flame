# Flame

Flame is a Go TUI for handling reverse shells during CTF work. It listens for sessions, lets you switch between them, provides an interactive shell view, supports upload and download, runs built-in modules in worker sessions, generates payloads, and can bootstrap a session through SSH.

This repository is at `v0.9.0`: feature complete for the current surface, with a planned pre-`1.0` review and refactor pass still tracked in `docs/superpowers/`.

## Requirements

- Go 1.25.2
- A Unix-like environment with a terminal that supports Bubble Tea's alt-screen TUI
- Optional clipboard helpers for local copy support: `wl-copy`, `xclip`, `xsel`, or `pbcopy`
- Optional external tools depending on workflow: `sshpass` for password-based `ssh`, `mcs` for `rev csharp <file.exe>`

## Build

```bash
go build -o flame .
```

## Run

You must provide either an interface or a direct IP.

```bash
./flame -i tun0 -p 4444
./flame -ip 10.10.14.5 -p 4444
```

Common flags:

- `-i`, `-interface`: interface to bind to
- `-ip`: direct IP address to bind to
- `-p`, `-port`: listener port, default `4444`

Once the TUI starts, use `help` or `F1` for the in-app command reference.

## Tests

```bash
go test ./... -coverprofile=coverage.out
```

## Configuration

Persistent configuration lives in `~/.flame/config.toml`. If `~/.gummy/` exists from older builds, Flame will migrate it to `~/.flame/` on access.

Current persisted settings:

- `binbag.enabled`
- `binbag.path`
- `binbag.http_port`
- `pivot.enabled`
- `pivot.host`

Default config shape:

```toml
[binbag]
enabled = false
path = "~/Lab/binbag"
http_port = 8080

[pivot]
enabled = false
host = ""
```

Runtime files under `~/.flame/` also include data such as TUI input history and generated session logs.

## Main Commands

- `sessions`, `list`, `ls`: list active sessions
- `use <id>`: select the active session
- `kill <id>`: close a session
- `shell`: enter the interactive shell for the selected session
- `upload <local> [remote]`: upload a file to the target
- `download <remote> [local]`: download a file from the target
- `spawn`: ask the current session to spawn another reverse shell
- `modules`: list built-in modules
- `run <module> [args...]`: run a module or runner in a worker session
- `rev`, `rev bash`, `rev ps1`, `rev csharp`, `rev php`: generate payloads
- `ssh user@host (-p <password> | -i <key>) [--port <port>]`: connect over SSH and trigger a reverse shell back to Flame
- `binbag ...`: manage the local HTTP tool-serving directory
- `pivot <ip>`, `pivot off`: rewrite generated URLs and payloads through a forwarder
- `config`: show current configuration
- `clear`, `help`, `exit`

## Modules And Runners

Built-in modules registered today:

- Linux: `peas`, `lse`, `loot`, `pspy`, `linexp`
- Windows: `winpeas`, `seatbelt`, `lazagne`
- Generic runners: `elf`, `sh` (`bash` alias), `ps1`, `dotnet`, `py`

Execution is split between in-memory runners and disk-backed runners with cleanup, depending on the module.

## Project Structure

- `main.go`: entry point, flag parsing, listener startup, TUI launch
- `internal/listener.go`: TCP listener lifecycle and new-session intake
- `internal/session.go`: current session manager, command dispatch, module orchestration, session selection, and shell-related flows
- `internal/shell.go` and `internal/pty.go`: shell transport and PTY upgrade handling
- `internal/transfer.go` and `internal/downloader.go`: upload and download flows
- `internal/modules.go`: module registry and built-in module implementations
- `internal/payloads.go`: reverse-shell payload generation and C# compilation helper
- `internal/ssh.go`: SSH command construction and handoff logic
- `internal/config.go`, `internal/runtime_config.go`, `internal/paths.go`: persisted config, runtime state, and app data paths
- `internal/help.go`: canonical help topics rendered in the TUI
- `internal/tui/`: Bubble Tea app, layout, modals, output pane, clipboard, and status UI
- `docs/superpowers/specs/` and `docs/superpowers/plans/`: active architecture notes and the pre-`1.0` refactor handoff

## Release Notes

`v0.9.0` is the first tagged release for this repository. It marks the current feature-complete TUI release before the planned architecture cleanup, broader test pass, and consistency review targeted for `1.0.0`.

## License

Educational use for CTF competitions only.
