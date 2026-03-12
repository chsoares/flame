# Configuration

Gummy uses a combination of CLI flags, a TOML config file, and runtime commands for configuration.

## CLI Flags

```bash
./gummy -i <interface> -p <port>    # Bind to network interface
./gummy -ip <address> -p <port>     # Bind to specific IP
```

| Flag | Description | Default |
|------|-------------|---------|
| `-i`, `-interface` | Network interface to bind to (e.g., `tun0`, `eth0`) | — |
| `-ip` | IP address to bind to | — |
| `-p`, `-port` | Port to listen on | `4444` |

Either `-i` or `-ip` is required (not both).

## Config File

Persistent settings live in `~/.gummy/config.toml`:

```toml
[binbag]
# Enable HTTP file server for fast transfers
enabled = false
# Path to your binbag directory (collection of CTF tools)
path = "~/Lab/binbag"
# HTTP server port for file serving
http_port = 8080

[execution]
# Default execution mode: "stealth" or "speed"
# stealth = in-memory (curl|bash, Reflection.Load) - no disk artifacts
# speed   = disk-based (SmartUpload -> execute -> shred)
default_mode = "stealth"

[pivot]
# Use pivot point for HTTP downloads in internal networks
enabled = false
host = ""
port = 0
```

The config file is optional. Gummy works fine without it using sensible defaults.

### Creating the Config File

```
󰗣 gummy ❯ config save
```

This creates `~/.gummy/config.toml` with the current settings (or defaults if first run).

## Runtime Commands

### View Current Config

```
󰗣 gummy ❯ config
```

Shows all current settings including binbag status, execution mode, and pivot configuration.

### Binbag

Binbag is a local directory of CTF tools served via HTTP. When enabled, file transfers and module execution use HTTP instead of base64 chunking — dramatically faster.

```
# Enable binbag
󰗣 gummy ❯ set binbag ~/Lab/binbag

# Disable binbag
󰗣 gummy ❯ set binbag disable
```

When enabled, gummy starts an HTTP server on the configured port (default 8080) serving files from the binbag directory. The URL format is:

```
http://<listener-ip>:<http-port>/filename
```

### Execution Mode

Controls how modules run on the target:

```
# In-memory execution (default) - no disk artifacts
󰗣 gummy ❯ set mode stealth

# Disk-based execution - faster for large binaries
󰗣 gummy ❯ set mode speed
```

| Mode | Linux | Windows (PS1) | Windows (.NET) |
|------|-------|---------------|----------------|
| **stealth** | `curl \| bash` | `IEX DownloadString` | `DownloadData` + `Reflection.Load` |
| **speed** | Upload to disk, execute, shred | Upload to disk, execute, shred | Upload to disk, execute, shred |

### Pivot

For internal network scenarios where the target can't reach your IP directly but can reach a pivot host:

```
# Enable pivot
󰗣 gummy ❯ set pivot 172.16.0.1 8080

# Disable pivot
󰗣 gummy ❯ set pivot disable
```

When enabled, HTTP URLs use the pivot address instead of the listener IP. Useful when pivoting through compromised hosts.

### Save Settings

```
󰗣 gummy ❯ config save
```

Persists current runtime settings to `~/.gummy/config.toml` so they're loaded automatically next time.

## Data Directories

Gummy stores all session data under `~/.gummy/`:

```
~/.gummy/
├── config.toml              # Persistent configuration
├── history                  # Command history (1000 entries)
├── shell_history            # Shell mode history
└── 2026_03_12/              # Date-based session directories
    └── 10.10.11.123_www-data_victim/
        ├── scripts/         # Downloaded module scripts (cached)
        └── logs/            # Session logs, module outputs
            └── session_1.log
```

Session directories are:
- **Grouped by date** for easy cleanup
- **Shared per host** so tools aren't re-downloaded for the same target
- **Created lazily** only when a module or log needs them
