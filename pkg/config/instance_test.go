package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAndLoadInstanceMarker(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	featureDir := filepath.Join(tmpDir, "feature-test")
	if err := os.MkdirAll(featureDir, 0755); err != nil {
		t.Fatalf("Failed to create feature directory: %v", err)
	}

	// Test data
	feature := "feature-test"
	instance := 1
	projectRoot := "/Users/test/skillsetup"
	projects := []string{"backend", "frontend"}
	ports := map[string]int{
		"APP_PORT": 8081,
		"FE_PORT":  3001,
		"PG_PORT":  5433,
	}
	yoloMode := true

	// Write instance marker
	err := WriteInstanceMarker(featureDir, feature, instance, projectRoot, projects, ports, yoloMode)
	if err != nil {
		t.Fatalf("WriteInstanceMarker failed: %v", err)
	}

	// Load instance marker
	markerPath := filepath.Join(featureDir, instanceMarkerFile)
	ctx, err := loadInstanceMarker(markerPath)
	if err != nil {
		t.Fatalf("loadInstanceMarker failed: %v", err)
	}

	// Verify values
	if ctx.Feature != feature {
		t.Errorf("Expected feature %s, got %s", feature, ctx.Feature)
	}
	if ctx.Instance != instance {
		t.Errorf("Expected instance %d, got %d", instance, ctx.Instance)
	}
	if ctx.ProjectRoot != projectRoot {
		t.Errorf("Expected project root %s, got %s", projectRoot, ctx.ProjectRoot)
	}
	if ctx.WorktreeRoot != featureDir {
		t.Errorf("Expected worktree root %s, got %s", featureDir, ctx.WorktreeRoot)
	}
	if len(ctx.Projects) != len(projects) {
		t.Errorf("Expected %d projects, got %d", len(projects), len(ctx.Projects))
	}
	if len(ctx.Ports) != len(ports) {
		t.Errorf("Expected %d ports, got %d", len(ports), len(ctx.Ports))
	}
	if ctx.YoloMode != yoloMode {
		t.Errorf("Expected yolo mode %v, got %v", yoloMode, ctx.YoloMode)
	}

	// Verify ports
	for key, expectedPort := range ports {
		if actualPort, exists := ctx.Ports[key]; !exists {
			t.Errorf("Port %s not found in context", key)
		} else if actualPort != expectedPort {
			t.Errorf("Expected port %s to be %d, got %d", key, expectedPort, actualPort)
		}
	}

	// Verify projects
	for i, expectedProject := range projects {
		if ctx.Projects[i] != expectedProject {
			t.Errorf("Expected project %s at index %d, got %s", expectedProject, i, ctx.Projects[i])
		}
	}
}

func TestDetectInstanceFromDir(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	featureDir := filepath.Join(tmpDir, "worktrees", "feature-test")
	backendDir := filepath.Join(featureDir, "backend")
	if err := os.MkdirAll(backendDir, 0755); err != nil {
		t.Fatalf("Failed to create directory structure: %v", err)
	}

	// Write instance marker in feature root
	feature := "feature-test"
	instance := 2
	projectRoot := "/Users/test/skillsetup"
	projects := []string{"backend", "frontend"}
	ports := map[string]int{"APP_PORT": 8082}
	yoloMode := false

	err := WriteInstanceMarker(featureDir, feature, instance, projectRoot, projects, ports, yoloMode)
	if err != nil {
		t.Fatalf("WriteInstanceMarker failed: %v", err)
	}

	// Test detection from feature root
	ctx, err := detectInstanceFromDir(featureDir)
	if err != nil {
		t.Fatalf("detectInstanceFromDir failed from feature root: %v", err)
	}
	if ctx.Feature != feature {
		t.Errorf("Expected feature %s, got %s", feature, ctx.Feature)
	}

	// Test detection from subdirectory (backend)
	ctx, err = detectInstanceFromDir(backendDir)
	if err != nil {
		t.Fatalf("detectInstanceFromDir failed from subdirectory: %v", err)
	}
	if ctx.Feature != feature {
		t.Errorf("Expected feature %s from subdirectory, got %s", feature, ctx.Feature)
	}

	// Test detection from directory without marker
	noMarkerDir := filepath.Join(tmpDir, "no-marker")
	if err := os.MkdirAll(noMarkerDir, 0755); err != nil {
		t.Fatalf("Failed to create no-marker directory: %v", err)
	}
	_, err = detectInstanceFromDir(noMarkerDir)
	if err == nil {
		t.Error("Expected error when no marker found, got nil")
	}
}

func TestUpdateInstanceYoloMode(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	featureDir := filepath.Join(tmpDir, "feature-yolo")
	if err := os.MkdirAll(featureDir, 0755); err != nil {
		t.Fatalf("Failed to create feature directory: %v", err)
	}

	// Write instance marker with yolo disabled
	feature := "feature-yolo"
	instance := 3
	projectRoot := "/Users/test/skillsetup"
	projects := []string{"backend"}
	ports := map[string]int{"APP_PORT": 8083}
	yoloMode := false

	err := WriteInstanceMarker(featureDir, feature, instance, projectRoot, projects, ports, yoloMode)
	if err != nil {
		t.Fatalf("WriteInstanceMarker failed: %v", err)
	}

	// Load and verify initial state
	markerPath := filepath.Join(featureDir, instanceMarkerFile)
	ctx, err := loadInstanceMarker(markerPath)
	if err != nil {
		t.Fatalf("loadInstanceMarker failed: %v", err)
	}
	if ctx.YoloMode {
		t.Error("Expected yolo mode to be false initially")
	}

	// Update yolo mode to true
	err = UpdateInstanceYoloMode(featureDir, true)
	if err != nil {
		t.Fatalf("UpdateInstanceYoloMode failed: %v", err)
	}

	// Load and verify updated state
	ctx, err = loadInstanceMarker(markerPath)
	if err != nil {
		t.Fatalf("loadInstanceMarker failed after update: %v", err)
	}
	if !ctx.YoloMode {
		t.Error("Expected yolo mode to be true after update")
	}

	// Update yolo mode to false
	err = UpdateInstanceYoloMode(featureDir, false)
	if err != nil {
		t.Fatalf("UpdateInstanceYoloMode failed: %v", err)
	}

	// Load and verify final state
	ctx, err = loadInstanceMarker(markerPath)
	if err != nil {
		t.Fatalf("loadInstanceMarker failed after second update: %v", err)
	}
	if ctx.YoloMode {
		t.Error("Expected yolo mode to be false after second update")
	}
}

func TestRemoveInstanceMarker(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	featureDir := filepath.Join(tmpDir, "feature-remove")
	if err := os.MkdirAll(featureDir, 0755); err != nil {
		t.Fatalf("Failed to create feature directory: %v", err)
	}

	// Write instance marker
	feature := "feature-remove"
	instance := 4
	projectRoot := "/Users/test/skillsetup"
	projects := []string{"backend"}
	ports := map[string]int{"APP_PORT": 8084}
	yoloMode := false

	err := WriteInstanceMarker(featureDir, feature, instance, projectRoot, projects, ports, yoloMode)
	if err != nil {
		t.Fatalf("WriteInstanceMarker failed: %v", err)
	}

	// Verify marker exists
	markerPath := filepath.Join(featureDir, instanceMarkerFile)
	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		t.Fatal("Marker file should exist before removal")
	}

	// Remove marker
	err = RemoveInstanceMarker(featureDir)
	if err != nil {
		t.Fatalf("RemoveInstanceMarker failed: %v", err)
	}

	// Verify marker is removed
	if _, err := os.Stat(markerPath); !os.IsNotExist(err) {
		t.Error("Marker file should not exist after removal")
	}

	// Try removing again (should not error)
	err = RemoveInstanceMarker(featureDir)
	if err != nil {
		t.Fatalf("RemoveInstanceMarker should not error when file doesn't exist: %v", err)
	}
}
