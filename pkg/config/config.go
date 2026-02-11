package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	// WorktreeDir is the directory where worktrees are created
	WorktreeDir = "worktrees"
)

// Config holds the application configuration
type Config struct {
	ProjectRoot string
	WorktreeDir string
	BackendDir  string
	FrontendDir string
}

// New creates a new Config with default values
func New() (*Config, error) {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	// Find project root by looking for backend/ and frontend/ directories
	projectRoot := cwd
	for {
		backendPath := filepath.Join(projectRoot, "backend")
		frontendPath := filepath.Join(projectRoot, "frontend")

		// Check if both directories exist
		if _, err := os.Stat(backendPath); err == nil {
			if _, err := os.Stat(frontendPath); err == nil {
				// Found the project root
				break
			}
		}

		// Move up one directory
		parent := filepath.Dir(projectRoot)
		if parent == projectRoot {
			// Reached filesystem root without finding project
			return nil, fmt.Errorf("could not find project root (looking for backend/ and frontend/ directories)")
		}
		projectRoot = parent
	}

	return &Config{
		ProjectRoot: projectRoot,
		WorktreeDir: filepath.Join(projectRoot, WorktreeDir),
		BackendDir:  filepath.Join(projectRoot, "backend"),
		FrontendDir: filepath.Join(projectRoot, "frontend"),
	}, nil
}

// WorktreeFeaturePath returns the path to the worktree directory for a feature
func (c *Config) WorktreeFeaturePath(featureName string) string {
	return filepath.Join(c.WorktreeDir, featureName)
}

// WorktreeBackendPath returns the path to the backend worktree for a feature
func (c *Config) WorktreeBackendPath(featureName string) string {
	return filepath.Join(c.WorktreeDir, featureName, "backend")
}

// WorktreeFrontendPath returns the path to the frontend worktree for a feature
func (c *Config) WorktreeFrontendPath(featureName string) string {
	return filepath.Join(c.WorktreeDir, featureName, "frontend")
}

// WorktreeExists checks if a worktree exists for a feature
func (c *Config) WorktreeExists(featureName string) bool {
	path := c.WorktreeFeaturePath(featureName)
	stat, err := os.Stat(path)
	return err == nil && stat.IsDir()
}
