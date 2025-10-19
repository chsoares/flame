package internal

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds all gummy configuration
type Config struct {
	Binbag struct {
		Enabled  bool   `toml:"enabled"`
		Path     string `toml:"path"`
		HTTPPort int    `toml:"http_port"`
	} `toml:"binbag"`

	Execution struct {
		DefaultMode string `toml:"default_mode"` // "stealth" or "speed"
	} `toml:"execution"`

	Pivot struct {
		Enabled bool   `toml:"enabled"`
		Host    string `toml:"host"`
		Port    int    `toml:"port"`
	} `toml:"pivot"`
}

// Global config instance
var GlobalConfig *Config

// DefaultConfig returns config with sensible defaults
func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()

	return &Config{
		Binbag: struct {
			Enabled  bool   `toml:"enabled"`
			Path     string `toml:"path"`
			HTTPPort int    `toml:"http_port"`
		}{
			Enabled:  false, // Disabled by default (backward compatible)
			Path:     filepath.Join(home, "Lab", "binbag"),
			HTTPPort: 8080,
		},
		Execution: struct {
			DefaultMode string `toml:"default_mode"`
		}{
			DefaultMode: "stealth", // Stealth by default
		},
		Pivot: struct {
			Enabled bool   `toml:"enabled"`
			Host    string `toml:"host"`
			Port    int    `toml:"port"`
		}{
			Enabled: false,
			Host:    "",
			Port:    0,
		},
	}
}

// LoadConfig loads config from ~/.gummy/config.toml or returns defaults
func LoadConfig() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return DefaultConfig(), nil
	}

	configPath := filepath.Join(home, ".gummy", "config.toml")

	// If config doesn't exist, return defaults
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}

	// Load config from file
	config := DefaultConfig()
	if _, err := toml.DecodeFile(configPath, config); err != nil {
		return nil, err
	}

	return config, nil
}

// SaveDefaultConfig creates a default config file if it doesn't exist
func SaveDefaultConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(home, ".gummy")
	configPath := filepath.Join(configDir, "config.toml")

	// Don't overwrite existing config
	if _, err := os.Stat(configPath); err == nil {
		return nil // Already exists
	}

	// Create directory
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	// Create default config file with comments
	configContent := `# Gummy Configuration File
# This file is optional - gummy works fine without it!

[binbag]
# Enable HTTP file server for fast transfers (100x faster than b64 chunks)
enabled = false
# Path to your binbag directory (collection of CTF tools)
path = "~/Lab/binbag"
# HTTP server port for file serving
http_port = 8080

[execution]
# Default execution mode on startup: "stealth" or "speed"
# - stealth: in-memory execution (slow, no disk artifacts)
# - speed: disk+cleanup execution (fast, shredded after)
# Can be changed at runtime with: set execution <mode>
default_mode = "stealth"

[pivot]
# Optional: Use pivot point for HTTP downloads in internal networks
# When enabled, HTTP URLs use this address instead of direct connection
enabled = false
host = ""
port = 0
`

	return os.WriteFile(configPath, []byte(configContent), 0644)
}
