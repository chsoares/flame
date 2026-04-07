package internal

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TransferProgress represents progress of a file transfer
type TransferProgress struct {
	BytesTransferred int64
	TotalBytes       int64
	Done             bool
	Success          bool
}

// FileServer serves files from binbag directory via HTTP
type FileServer struct {
	binbagPath        string
	port              int
	server            *http.Server
	mu                sync.Mutex
	running           bool
	downloadDir       string                           // Where to save received files (default: CWD)
	transferListeners map[string]chan TransferProgress // Channels to notify transfer progress
	listenerMu        sync.Mutex
}

// NewFileServer creates a new file server
func NewFileServer(binbagPath string, port int) *FileServer {
	cwd, _ := os.Getwd()
	return &FileServer{
		binbagPath:        binbagPath,
		port:              port,
		downloadDir:       cwd,
		transferListeners: make(map[string]chan TransferProgress),
	}
}

// Start starts the HTTP server
func (fs *FileServer) Start() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if fs.running {
		return fmt.Errorf("server already running")
	}

	// Validate binbagPath exists
	if _, err := os.Stat(fs.binbagPath); os.IsNotExist(err) {
		return fmt.Errorf("binbag path does not exist: %s", fs.binbagPath)
	}

	// Create HTTP server
	addr := fmt.Sprintf(":%d", fs.port)
	fs.server = &http.Server{
		Addr:    addr,
		Handler: fs, // FileServer implements http.Handler via ServeHTTP
	}

	// Start server in background goroutine
	go func() {
		if err := fs.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	fs.running = true
	return nil
}

// Stop stops the HTTP server
func (fs *FileServer) Stop() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if !fs.running {
		return nil
	}

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := fs.server.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
		return err
	}

	fs.running = false
	return nil
}

// ServeHTTP handles HTTP requests (GET = serve file, POST = receive file)
func (fs *FileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost || r.Method == http.MethodPut {
		fs.handleReceive(w, r)
		return
	}

	// GET: serve file from binbag
	filename := filepath.Base(r.URL.Path)
	filePath := filepath.Join(fs.binbagPath, filename)

	// Check if file exists
	fileInfo, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		http.NotFound(w, r)
		// Notify failure
		fs.notifyProgress(filename, TransferProgress{Done: true, Success: false})
		return
	}

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "Error opening file", http.StatusInternalServerError)
		fs.notifyProgress(filename, TransferProgress{Done: true, Success: false})
		return
	}
	defer file.Close()

	totalBytes := fileInfo.Size()

	// Set headers
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", totalBytes))

	// Copy file with progress tracking
	buf := make([]byte, 32*1024) // 32KB buffer
	bytesTransferred := int64(0)

	for {
		n, err := file.Read(buf)
		if n > 0 {
			w.Write(buf[:n])
			bytesTransferred += int64(n)

			// Notify progress
			fs.notifyProgress(filename, TransferProgress{
				BytesTransferred: bytesTransferred,
				TotalBytes:       totalBytes,
				Done:             false,
				Success:          true,
			})
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			fs.notifyProgress(filename, TransferProgress{Done: true, Success: false})
			return
		}
	}

	// Notify completion
	fs.notifyProgress(filename, TransferProgress{
		BytesTransferred: totalBytes,
		TotalBytes:       totalBytes,
		Done:             true,
		Success:          true,
	})
}

// WaitForTransfer waits for a file transfer to complete, calling progressCallback for updates
// Uses an inactivity timeout that resets on every progress update
// Returns true if successful, false if failed/timeout
func (fs *FileServer) WaitForTransfer(filename string, inactivityTimeout time.Duration, progressCallback func(TransferProgress)) bool {
	err := fs.WaitForTransferCtx(context.Background(), filename, inactivityTimeout, progressCallback)
	return err == nil
}

// WaitForTransferCtx waits for transfer completion and can be cancelled via context.
func (fs *FileServer) WaitForTransferCtx(ctx context.Context, filename string, inactivityTimeout time.Duration, progressCallback func(TransferProgress)) error {
	ch, cleanup := fs.listenForTransfer(filename)
	return fs.waitOnTransferChannel(ctx, ch, cleanup, inactivityTimeout, progressCallback)
}

// notifyProgress notifies listeners about transfer progress
// handleReceive accepts a file upload via POST (remote → local download)
func (fs *FileServer) handleReceive(w http.ResponseWriter, r *http.Request) {
	filename := filepath.Base(r.URL.Path)
	if filename == "" || filename == "." || filename == "/" {
		http.Error(w, "filename required in URL path", http.StatusBadRequest)
		return
	}

	// Write to temp file in binbag first, then move to download dir
	tmpPath := filepath.Join(fs.binbagPath, ".dl_"+filename)
	finalPath := filepath.Join(fs.downloadDir, filename)

	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		http.Error(w, "failed to create temp file", http.StatusInternalServerError)
		fs.notifyProgress(filename, TransferProgress{Done: true, Success: false})
		return
	}

	// Copy with progress tracking
	buf := make([]byte, 32*1024)
	bytesReceived := int64(0)
	totalBytes := r.ContentLength // -1 if unknown

	for {
		n, readErr := r.Body.Read(buf)
		if n > 0 {
			if _, writeErr := tmpFile.Write(buf[:n]); writeErr != nil {
				tmpFile.Close()
				os.Remove(tmpPath)
				http.Error(w, "write error", http.StatusInternalServerError)
				fs.notifyProgress(filename, TransferProgress{Done: true, Success: false})
				return
			}
			bytesReceived += int64(n)
			fs.notifyProgress(filename, TransferProgress{
				BytesTransferred: bytesReceived,
				TotalBytes:       totalBytes,
				Done:             false,
				Success:          true,
			})
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
			http.Error(w, "read error", http.StatusInternalServerError)
			fs.notifyProgress(filename, TransferProgress{Done: true, Success: false})
			return
		}
	}
	tmpFile.Close()

	// Move to final destination (try rename, fall back to copy+delete for cross-device)
	if err := os.Rename(tmpPath, finalPath); err != nil {
		// Cross-device: copy + delete
		src, _ := os.ReadFile(tmpPath)
		if writeErr := os.WriteFile(finalPath, src, 0644); writeErr != nil {
			os.Remove(tmpPath)
			http.Error(w, "failed to save file", http.StatusInternalServerError)
			fs.notifyProgress(filename, TransferProgress{Done: true, Success: false})
			return
		}
		os.Remove(tmpPath)
	}

	// Success
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK %d bytes", bytesReceived)

	fs.notifyProgress(filename, TransferProgress{
		BytesTransferred: bytesReceived,
		TotalBytes:       bytesReceived,
		Done:             true,
		Success:          true,
	})
}

func (fs *FileServer) notifyProgress(filename string, progress TransferProgress) {
	fs.listenerMu.Lock()
	defer fs.listenerMu.Unlock()

	if ch, exists := fs.transferListeners[filename]; exists {
		select {
		case ch <- progress:
		default:
			// Channel full, skip this update
		}
		if progress.Done {
			delete(fs.transferListeners, filename)
		}
	}
}
