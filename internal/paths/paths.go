// Package paths provides shared path resolution for pivot's data directory.
package paths

import (
	"os"
	"path/filepath"
)

// Home returns the pivot home directory.
// If PIVOT_HOME is set, that is used; otherwise the user's home directory.
func Home() (string, error) {
	if dir := os.Getenv("PIVOT_HOME"); dir != "" {
		return dir, nil
	}
	return os.UserHomeDir()
}

// ConfigFile returns the path to pivot's config file.
func ConfigFile() (string, error) {
	home, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".pivot", "config.yaml"), nil
}

// StateFile returns the path to pivot's SQLite state database.
func StateFile() (string, error) {
	home, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".pivot", "state.db"), nil
}
