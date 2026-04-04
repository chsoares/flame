package internal

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds all flame configuration
type Config struct {
	Binbag struct {
		Enabled  bool   `toml:"enabled"`
		Path     string `toml:"path"`
		HTTPPort int    `toml:"http_port"`
	} `toml:"binbag"`

	Pivot struct {
		Enabled bool   `toml:"enabled"`
		Host    string `toml:"host"`
	} `toml:"pivot"`
}

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
		Pivot: struct {
			Enabled bool   `toml:"enabled"`
			Host    string `toml:"host"`
		}{
			Enabled: false,
			Host:    "",
		},
	}
}

// LoadConfig loads config from ~/.flame/config.toml or returns defaults
func LoadConfig() (*Config, error) {
	if _, err := os.UserHomeDir(); err != nil {
		return DefaultConfig(), nil
	}

	configPath := appDataPath("config.toml")

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
	if _, err := os.UserHomeDir(); err != nil {
		return err
	}

	configDir := appDataPath()
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
	configContent := `# Flame Configuration File
# This file is optional - flame works fine without it!

[binbag]
# Enable HTTP file server for fast transfers (100x faster than b64 chunks)
enabled = false
# Path to your binbag directory (collection of CTF tools)
path = "~/Lab/binbag"
# HTTP server port for file serving
http_port = 8080

[pivot]
# Optional: Route all URLs/payloads through a pivot IP (e.g., ligolo)
# Only the IP is replaced — ports are preserved from original services
enabled = false
host = ""
`

	return os.WriteFile(configPath, []byte(configContent), 0644)
}
