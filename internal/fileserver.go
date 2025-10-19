package internal

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

// FileServer serves files from binbag directory via HTTP
type FileServer struct {
	binbagPath string
	port       int
	server     *http.Server
	mu         sync.Mutex
	running    bool
}

// NewFileServer creates a new file server
func NewFileServer(binbagPath string, port int) *FileServer {
	return &FileServer{
		binbagPath: binbagPath,
		port:       port,
	}
}

// Start starts the HTTP server
func (fs *FileServer) Start() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if fs.running {
		return fmt.Errorf("server already running")
	}

	// TODO: Validate binbagPath exists
	// TODO: Create http.ServeMux
	// TODO: Register handler
	// TODO: Create http.Server
	// TODO: Start server in goroutine
	// TODO: Set running=true

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

	// TODO: Call server.Shutdown()
	// TODO: Set running=false
	// TODO: Log shutdown

	fs.running = false
	return nil
}

// ServeHTTP handles HTTP requests
func (fs *FileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// TODO: Extract filename from URL path
	// TODO: Validate filename (no directory traversal)
	// TODO: Check if file exists in binbagPath
	// TODO: Log request
	// TODO: Serve file with http.ServeFile
	// TODO: Handle errors (404, 403, 500)

	filename := filepath.Base(r.URL.Path)
	filePath := filepath.Join(fs.binbagPath, filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.NotFound(w, r)
		log.Printf("File not found: %s", filename)
		return
	}

	// Log request
	log.Printf("HTTP: %s requested %s", r.RemoteAddr, filename)

	// Serve file
	http.ServeFile(w, r, filePath)
}
