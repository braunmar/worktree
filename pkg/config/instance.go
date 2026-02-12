package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const instanceMarkerFile = ".worktree-instance"

// InstanceContext represents the worktree instance metadata
type InstanceContext struct {
	Feature      string         `json:"feature"`
	Instance     int            `json:"instance"`
	ProjectRoot  string         `json:"project_root"`
	WorktreeRoot string         `json:"worktree_root"`
	Projects     []string       `json:"projects"`
	Ports        map[string]int `json:"ports"`
	YoloMode     bool           `json:"yolo_mode"`
	CreatedAt    string         `json:"created_at"`
}

// DetectInstance walks up from the current working directory to find .worktree-instance
// Returns the instance context if found, or an error if not in a worktree
func DetectInstance() (*InstanceContext, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	return detectInstanceFromDir(dir)
}

// detectInstanceFromDir walks up from the given directory to find .worktree-instance
func detectInstanceFromDir(startDir string) (*InstanceContext, error) {
	dir := startDir

	for {
		markerPath := filepath.Join(dir, instanceMarkerFile)

		// Check if .worktree-instance exists
		if info, err := os.Stat(markerPath); err == nil && !info.IsDir() {
			return loadInstanceMarker(markerPath)
		}

		// Move to parent directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root without finding marker
			return nil, errors.New("not in a worktree (no .worktree-instance found)")
		}
		dir = parent
	}
}

// loadInstanceMarker reads and parses a .worktree-instance file
func loadInstanceMarker(path string) (*InstanceContext, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read instance marker: %w", err)
	}

	var ctx InstanceContext
	if err := json.Unmarshal(data, &ctx); err != nil {
		return nil, fmt.Errorf("failed to parse instance marker: %w", err)
	}

	return &ctx, nil
}

// WriteInstanceMarker creates a .worktree-instance file in the feature root directory
func WriteInstanceMarker(featureDir string, feature string, instance int, projectRoot string, projects []string, ports map[string]int, yoloMode bool) error {
	markerPath := filepath.Join(featureDir, instanceMarkerFile)

	ctx := InstanceContext{
		Feature:      feature,
		Instance:     instance,
		ProjectRoot:  projectRoot,
		WorktreeRoot: featureDir,
		Projects:     projects,
		Ports:        ports,
		YoloMode:     yoloMode,
		CreatedAt:    time.Now().Format(time.RFC3339),
	}

	data, err := json.MarshalIndent(ctx, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal instance context: %w", err)
	}

	if err := os.WriteFile(markerPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write instance marker: %w", err)
	}

	return nil
}

// RemoveInstanceMarker deletes the .worktree-instance file from the feature directory
func RemoveInstanceMarker(featureDir string) error {
	markerPath := filepath.Join(featureDir, instanceMarkerFile)

	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		return nil // Already removed or never existed
	}

	if err := os.Remove(markerPath); err != nil {
		return fmt.Errorf("failed to remove instance marker: %w", err)
	}

	return nil
}

// UpdateInstanceYoloMode updates the yolo_mode field in the .worktree-instance file
func UpdateInstanceYoloMode(featureDir string, yoloMode bool) error {
	markerPath := filepath.Join(featureDir, instanceMarkerFile)

	ctx, err := loadInstanceMarker(markerPath)
	if err != nil {
		return err
	}

	ctx.YoloMode = yoloMode

	data, err := json.MarshalIndent(ctx, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal instance context: %w", err)
	}

	if err := os.WriteFile(markerPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write instance marker: %w", err)
	}

	return nil
}
