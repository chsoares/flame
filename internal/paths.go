package internal

import (
	"os"
	"path/filepath"
)

const (
	currentAppDir = ".flame"
	legacyAppDir  = ".gummy"
)

func appHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return currentAppDir
	}
	current := filepath.Join(home, currentAppDir)
	legacy := filepath.Join(home, legacyAppDir)

	if _, err := os.Stat(current); err == nil {
		return current
	}
	if _, err := os.Stat(legacy); err == nil {
		_ = os.Rename(legacy, current)
		if _, err := os.Stat(current); err == nil {
			return current
		}
	}
	return current
}

func appDataPath(parts ...string) string {
	all := append([]string{appHomeDir()}, parts...)
	return filepath.Join(all...)
}

func AppDataPath(parts ...string) string {
	return appDataPath(parts...)
}
