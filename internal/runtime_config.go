package internal

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
)

// InitRuntimeConfig initializes the runtime config from file (or defaults)
func InitRuntimeConfig(listenerIP string) (*RuntimeConfig, error) {
	// Load config from file (or defaults if doesn't exist)
	config, err := LoadConfig()
	if err != nil {
		// File doesn't exist, use defaults
		config = DefaultConfig()
	}

	// Create runtime config from loaded config
	rc := &RuntimeConfig{
		BinbagEnabled: config.Binbag.Enabled,
		BinbagPath:    os.ExpandEnv(config.Binbag.Path), // Expand ~/
		HTTPPort:      config.Binbag.HTTPPort,
		PivotEnabled:  config.Pivot.Enabled,
		PivotHost:     config.Pivot.Host,
		PivotPort:     config.Pivot.Port,
		ListenerIP:    listenerIP,
	}

	// If binbag is enabled in config, start HTTP server
	if rc.BinbagEnabled {
		if err := rc.EnableBinbag(rc.BinbagPath); err != nil {
			// Failed to start server, disable binbag
			rc.BinbagEnabled = false
			return rc, fmt.Errorf("failed to enable binbag: %w", err)
		}
	}

	return rc, nil
}

// RuntimeConfig holds mutable configuration that can be changed during runtime
// Thread-safe with RWMutex
type RuntimeConfig struct {
	mu sync.RWMutex

	// Binbag
	BinbagEnabled bool
	BinbagPath    string
	HTTPPort      int
	FileServer    *FileServer

	// Pivot
	PivotEnabled bool
	PivotHost    string
	PivotPort    int

	// Listener IP (for HTTP URLs)
	ListenerIP string
}

// Global runtime config instance
var GlobalRuntimeConfig *RuntimeConfig

// NewRuntimeConfig creates a new runtime config from loaded config
func NewRuntimeConfig(config *Config, listenerIP string) *RuntimeConfig {
	rc := &RuntimeConfig{
		BinbagEnabled: config.Binbag.Enabled,
		BinbagPath:    config.Binbag.Path,
		HTTPPort:      config.Binbag.HTTPPort,
		PivotEnabled:  config.Pivot.Enabled,
		PivotHost:     config.Pivot.Host,
		PivotPort:     config.Pivot.Port,
		ListenerIP:    listenerIP,
	}

	return rc
}

// autoPersist saves config to file silently (logs errors but doesn't bubble up)
func (rc *RuntimeConfig) autoPersist() {
	if err := rc.SaveToFile(); err != nil {
		log.Printf("auto-persist config failed: %v", err)
	}
}

// EnableBinbag enables binbag and starts HTTP server
func (rc *RuntimeConfig) EnableBinbag(path string) error {
	// Validate path exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("binbag path does not exist: %s", path)
	}

	rc.mu.Lock()

	// Stop existing server if running
	if rc.FileServer != nil {
		rc.FileServer.Stop()
	}

	// Create and start new server
	rc.BinbagPath = path
	rc.BinbagEnabled = true
	rc.FileServer = NewFileServer(path, rc.HTTPPort)

	if err := rc.FileServer.Start(); err != nil {
		rc.BinbagEnabled = false
		rc.FileServer = nil
		rc.mu.Unlock()
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	rc.mu.Unlock()
	rc.autoPersist()
	return nil
}

// DisableBinbag disables binbag and stops HTTP server
func (rc *RuntimeConfig) DisableBinbag() error {
	rc.mu.Lock()

	// Stop server if running
	if rc.FileServer != nil {
		rc.FileServer.Stop()
		rc.FileServer = nil
	}

	rc.BinbagEnabled = false
	rc.mu.Unlock()

	rc.autoPersist()
	return nil
}

// SetPivot configures pivot point for HTTP downloads
func (rc *RuntimeConfig) SetPivot(host string, port int) error {
	// Validate port
	if port <= 0 || port > 65535 {
		return fmt.Errorf("invalid port: %d (must be 1-65535)", port)
	}

	// Validate host (basic check - not empty)
	if host == "" {
		return fmt.Errorf("host cannot be empty")
	}

	rc.mu.Lock()
	rc.PivotEnabled = true
	rc.PivotHost = host
	rc.PivotPort = port
	rc.mu.Unlock()

	rc.autoPersist()
	return nil
}

// DisablePivot disables pivot configuration
func (rc *RuntimeConfig) DisablePivot() error {
	rc.mu.Lock()
	rc.PivotEnabled = false
	rc.mu.Unlock()

	rc.autoPersist()
	return nil
}

// GetHTTPURL returns HTTP URL for file, using pivot if configured
func (rc *RuntimeConfig) GetHTTPURL(filename string) string {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	var host string
	var port int

	if rc.PivotEnabled {
		host = rc.PivotHost
		port = rc.PivotPort
	} else {
		host = rc.ListenerIP
		port = rc.HTTPPort
	}

	return fmt.Sprintf("http://%s:%d/%s", host, port, filename)
}

// SetBinbagPort validates and updates the HTTP port, restarting the server if running
func (rc *RuntimeConfig) SetBinbagPort(port int) error {
	if port <= 0 || port > 65535 {
		return fmt.Errorf("invalid port: %d (must be 1-65535)", port)
	}

	rc.mu.Lock()
	rc.HTTPPort = port
	needRestart := rc.BinbagEnabled && rc.FileServer != nil
	path := rc.BinbagPath
	rc.mu.Unlock()

	// Restart server if it was running
	if needRestart {
		rc.mu.Lock()
		if rc.FileServer != nil {
			rc.FileServer.Stop()
			rc.FileServer = nil
		}
		rc.mu.Unlock()

		rc.mu.Lock()
		rc.FileServer = NewFileServer(path, port)
		if err := rc.FileServer.Start(); err != nil {
			rc.FileServer = nil
			rc.mu.Unlock()
			return fmt.Errorf("failed to restart HTTP server on port %d: %w", port, err)
		}
		rc.mu.Unlock()
	}

	rc.autoPersist()
	return nil
}

// SetBinbagPath validates and updates the binbag path, restarting the server if running
func (rc *RuntimeConfig) SetBinbagPath(path string) error {
	// Expand tilde
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		if path == "~" {
			path = home
		} else {
			path = filepath.Join(home, path[2:])
		}
	}

	// Validate path exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("path does not exist: %s", path)
	}

	rc.mu.Lock()
	rc.BinbagPath = path
	needRestart := rc.BinbagEnabled && rc.FileServer != nil
	port := rc.HTTPPort
	rc.mu.Unlock()

	// Restart server if it was running
	if needRestart {
		rc.mu.Lock()
		if rc.FileServer != nil {
			rc.FileServer.Stop()
			rc.FileServer = nil
		}
		rc.mu.Unlock()

		rc.mu.Lock()
		rc.FileServer = NewFileServer(path, port)
		if err := rc.FileServer.Start(); err != nil {
			rc.FileServer = nil
			rc.mu.Unlock()
			return fmt.Errorf("failed to restart HTTP server: %w", err)
		}
		rc.mu.Unlock()
	}

	rc.autoPersist()
	return nil
}

// SaveToFile persists current runtime config to ~/.gummy/config.toml
func (rc *RuntimeConfig) SaveToFile() error {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(home, ".gummy", "config.toml")

	// Load current config (or defaults if doesn't exist)
	config, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load current config: %w", err)
	}

	// Update with runtime values
	config.Binbag.Enabled = rc.BinbagEnabled
	config.Binbag.Path = rc.BinbagPath
	config.Binbag.HTTPPort = rc.HTTPPort
	config.Pivot.Enabled = rc.PivotEnabled
	config.Pivot.Host = rc.PivotHost
	config.Pivot.Port = rc.PivotPort

	// Write to file
	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	return nil
}
