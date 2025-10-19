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
	transferListeners map[string]chan TransferProgress // Channels to notify transfer progress
	listenerMu        sync.Mutex
}

// NewFileServer creates a new file server
func NewFileServer(binbagPath string, port int) *FileServer {
	return &FileServer{
		binbagPath:        binbagPath,
		port:              port,
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

// ServeHTTP handles HTTP requests
func (fs *FileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract filename from URL path (Base prevents directory traversal)
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
	// Create channel for this transfer
	ch := make(chan TransferProgress, 10)

	fs.listenerMu.Lock()
	fs.transferListeners[filename] = ch
	fs.listenerMu.Unlock()

	// Cleanup function
	cleanup := func() {
		fs.listenerMu.Lock()
		delete(fs.transferListeners, filename)
		fs.listenerMu.Unlock()
	}

	// Use a timer that resets on every progress update
	timer := time.NewTimer(inactivityTimeout)
	defer timer.Stop()

	for {
		select {
		case progress := <-ch:
			// Reset timer on every progress update (activity detected)
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(inactivityTimeout)

			// Call progress callback
			if progressCallback != nil {
				progressCallback(progress)
			}

			// Check if done
			if progress.Done {
				cleanup()
				return progress.Success
			}

		case <-timer.C:
			// Inactivity timeout reached
			cleanup()
			return false
		}
	}
}

// notifyProgress notifies listeners about transfer progress
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
