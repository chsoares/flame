# Flame

Flame is a Go TUI for handling reverse shells during CTF work. It gives you a place to keep multiple sessions open, move between them, spawn new shells, run modules, and transfer files without living inside a bare `nc` prompt.

## Features

- Session handling with attach/detach flow, so you can leave a shell and come back without losing the session list or history
- `spawn` support for opening a new reverse shell from an existing session
- Module and script runners in a separate window, so long jobs do not block your main shell
- Fast transfers over HTTP when binbag is enabled, with base64 chunk fallback when it is not
- Pivot helper for rewriting generated payloads and HTTP URLs through an intermediate host
- Payload generation for bash, PowerShell, C#, and PHP
- SSH-assisted session bootstrap for quick callback setup
- Linux PTY upgrade and a better Windows shell flow than a plain netcat session

## Requirements

- Go 1.25.2
- A Unix-like terminal with alt-screen support
- Optional clipboard helpers: `wl-copy`, `xclip`, `xsel`, or `pbcopy`
- Optional tools: `sshpass` for password-based SSH, `mcs` for `rev csharp <file.exe>`

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

Once the TUI starts, use `help` or `F1` for the in-app command reference.

## License

Educational use for CTF competitions only.
