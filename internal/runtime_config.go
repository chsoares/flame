package internal

import (
	"fmt"
	"sync"
)

// RuntimeConfig holds mutable configuration that can be changed during runtime
// Thread-safe with RWMutex
type RuntimeConfig struct {
	mu sync.RWMutex

	// Execution mode
	ExecutionMode string // "stealth" or "speed"

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
		ExecutionMode: "stealth", // Default to stealth
		BinbagEnabled: config.Binbag.Enabled,
		BinbagPath:    config.Binbag.Path,
		HTTPPort:      config.Binbag.HTTPPort,
		PivotEnabled:  config.Pivot.Enabled,
		PivotHost:     config.Pivot.Host,
		PivotPort:     config.Pivot.Port,
		ListenerIP:    listenerIP,
	}

	// Set execution mode from config
	if config.Execution.DefaultMode == "speed" {
		rc.ExecutionMode = "speed"
	}

	return rc
}

// GetMode returns current execution mode (thread-safe)
func (rc *RuntimeConfig) GetMode() string {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.ExecutionMode
}

// SetExecutionMode changes execution mode at runtime
func (rc *RuntimeConfig) SetExecutionMode(mode string) error {
	// TODO: Validate mode ("stealth" or "speed")
	// TODO: Lock, set, unlock
	// TODO: Print confirmation message
	return nil
}

// EnableBinbag enables binbag and starts HTTP server
func (rc *RuntimeConfig) EnableBinbag(path string) error {
	// TODO: Validate path exists
	// TODO: Start FileServer
	// TODO: Lock, set enabled=true, unlock
	// TODO: Count files in binbag
	// TODO: Print confirmation with file count
	return nil
}

// DisableBinbag disables binbag and stops HTTP server
func (rc *RuntimeConfig) DisableBinbag() error {
	// TODO: Stop FileServer if running
	// TODO: Lock, set enabled=false, unlock
	// TODO: Print confirmation
	return nil
}

// SetPivot configures pivot point for HTTP downloads
func (rc *RuntimeConfig) SetPivot(host string, port int) error {
	// TODO: Validate host:port
	// TODO: Lock, set pivot config, unlock
	// TODO: Print confirmation
	return nil
}

// DisablePivot disables pivot configuration
func (rc *RuntimeConfig) DisablePivot() error {
	// TODO: Lock, set enabled=false, unlock
	// TODO: Print confirmation
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

// SaveToFile persists current runtime config to ~/.gummy/config.toml
func (rc *RuntimeConfig) SaveToFile() error {
	// TODO: Read current config
	// TODO: Update with runtime values
	// TODO: Write back to TOML
	// TODO: Print confirmation
	return nil
}
