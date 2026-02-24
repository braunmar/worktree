package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	// WorktreeDir is the directory where worktrees are created
	WorktreeDir = "worktrees"

	// ConfigFileName is the name of the worktree configuration file
	ConfigFileName = ".worktree.yml"
)

// Config holds the application configuration
type Config struct {
	ProjectRoot string
	WorktreeDir string
}

// New creates a new Config by walking up from the current directory to find .worktree.yml
func New() (*Config, error) {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	// Find project root by looking for .worktree.yml
	projectRoot := cwd
	for {
		if _, err := os.Stat(filepath.Join(projectRoot, ConfigFileName)); err == nil {
			// Found the project root
			break
		}

		// Move up one directory
		parent := filepath.Dir(projectRoot)
		if parent == projectRoot {
			// Reached filesystem root without finding config
			return nil, fmt.Errorf("could not find project root (no %s found in current directory or any parent)", ConfigFileName)
		}
		projectRoot = parent
	}

	return &Config{
		ProjectRoot: projectRoot,
		WorktreeDir: filepath.Join(projectRoot, WorktreeDir),
	}, nil
}

// WorktreeFeaturePath returns the path to the worktree directory for a feature
func (c *Config) WorktreeFeaturePath(featureName string) string {
	return filepath.Join(c.WorktreeDir, featureName)
}

// WorktreeExists checks if a worktree exists for a feature
func (c *Config) WorktreeExists(featureName string) bool {
	path := c.WorktreeFeaturePath(featureName)
	stat, err := os.Stat(path)
	return err == nil && stat.IsDir()
}
