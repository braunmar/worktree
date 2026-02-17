package registry

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/braunmar/worktree/pkg/config"
)

// testConfig creates a WorktreeConfig with standard port ranges for testing
func testConfig() *config.WorktreeConfig {
	feRange := [2]int{3000, 3100}
	beRange := [2]int{8080, 8180}
	pgRange := [2]int{5432, 5532}

	return &config.WorktreeConfig{
		Ports: map[string]config.PortConfig{
			"FE_PORT": {
				Range: &feRange,
			},
			"BE_PORT": {
				Range: &beRange,
			},
			"POSTGRES_PORT": {
				Range: &pgRange,
			},
		},
	}
}

func TestNormalizeBranchName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"feature/user-auth", "feature-user-auth"},
		{"bugfix/API-Error", "bugfix-api-error"},
		{"feature/new_asset_system", "feature-new-asset-system"},
		{"release/v2.0.0", "release-v2-0-0"},
		{"refs/heads/feature/test", "feature-test"},
		{"FEATURE/UPPER-CASE", "feature-upper-case"},
		{"feature//double-slash", "feature-double-slash"},
		{"--leading-trailing--", "leading-trailing"},
		{"feature@#$%special", "featurespecial"},
		{"main", "main"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizeBranchName(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeBranchName(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRegistryLoadAndSave(t *testing.T) {
	// Create temp directory for test
	tempDir, err := os.MkdirTemp("", "registry-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Load registry (should create empty one)
	reg, err := Load(tempDir, nil)
	if err != nil {
		t.Fatalf("Failed to load registry: %v", err)
	}

	// Verify empty registry
	if len(reg.Worktrees) != 0 {
		t.Errorf("Expected empty registry, got %d worktrees", len(reg.Worktrees))
	}

	// Add a worktree
	wt := &Worktree{
		Branch:         "feature/test",
		Normalized:     "feature-test",
		Created:        time.Now(),
		Projects:       []string{"backend", "frontend"},
		Ports:          map[string]int{"FE_PORT": 3001, "BE_PORT": 8081},
		ComposeProject: "myproject-feature-test",
	}

	if err := reg.Add(wt); err != nil {
		t.Fatalf("Failed to add worktree: %v", err)
	}

	// Save registry
	if err := reg.Save(); err != nil {
		t.Fatalf("Failed to save registry: %v", err)
	}

	// Verify file exists
	registryPath := filepath.Join(tempDir, registryFileName)
	if _, err := os.Stat(registryPath); os.IsNotExist(err) {
		t.Error("Registry file was not created")
	}

	// Load registry again
	reg2, err := Load(tempDir, nil)
	if err != nil {
		t.Fatalf("Failed to load registry second time: %v", err)
	}

	// Verify worktree exists
	if len(reg2.Worktrees) != 1 {
		t.Errorf("Expected 1 worktree, got %d", len(reg2.Worktrees))
	}

	wt2, exists := reg2.Get("feature-test")
	if !exists {
		t.Error("Worktree not found in loaded registry")
	}

	if wt2.Branch != "feature/test" {
		t.Errorf("Expected branch 'feature/test', got '%s'", wt2.Branch)
	}
}

func TestRegistryAddRemove(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "registry-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	reg, err := Load(tempDir, nil)
	if err != nil {
		t.Fatal(err)
	}

	wt := &Worktree{
		Branch:     "feature/test",
		Normalized: "feature-test",
		Created:    time.Now(),
	}

	// Add worktree
	if err := reg.Add(wt); err != nil {
		t.Fatalf("Failed to add worktree: %v", err)
	}

	// Try to add duplicate
	if err := reg.Add(wt); err == nil {
		t.Error("Expected error when adding duplicate worktree")
	}

	// Get worktree
	if _, exists := reg.Get("feature-test"); !exists {
		t.Error("Worktree should exist")
	}

	// Remove worktree
	if err := reg.Remove("feature-test"); err != nil {
		t.Fatalf("Failed to remove worktree: %v", err)
	}

	// Verify removed
	if _, exists := reg.Get("feature-test"); exists {
		t.Error("Worktree should not exist after removal")
	}

	// Try to remove non-existent
	if err := reg.Remove("feature-test"); err == nil {
		t.Error("Expected error when removing non-existent worktree")
	}
}

func TestPortAllocation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "registry-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	reg, err := Load(tempDir, testConfig())
	if err != nil {
		t.Fatal(err)
	}

	// Allocate ports for first worktree
	services := []string{"FE_PORT", "BE_PORT", "POSTGRES_PORT"}
	ports1, err := reg.AllocatePorts(services)
	if err != nil {
		t.Fatalf("Failed to allocate ports: %v", err)
	}

	// Verify ports are in valid ranges
	if ports1["FE_PORT"] < 3000 || ports1["FE_PORT"] > 3100 {
		t.Errorf("FE_PORT %d is outside expected range 3000-3100", ports1["FE_PORT"])
	}
	if ports1["BE_PORT"] < 8080 || ports1["BE_PORT"] > 8180 {
		t.Errorf("BE_PORT %d is outside expected range 8080-8180", ports1["BE_PORT"])
	}
	if ports1["POSTGRES_PORT"] < 5432 || ports1["POSTGRES_PORT"] > 5532 {
		t.Errorf("POSTGRES_PORT %d is outside expected range 5432-5532", ports1["POSTGRES_PORT"])
	}

	// Add first worktree to registry
	wt1 := &Worktree{
		Branch:     "feature/one",
		Normalized: "feature-one",
		Created:    time.Now(),
		Ports:      ports1,
	}
	reg.Add(wt1)

	// Allocate ports for second worktree
	ports2, err := reg.AllocatePorts(services)
	if err != nil {
		t.Fatalf("Failed to allocate ports for second worktree: %v", err)
	}

	// Verify different ports allocated
	if ports2["FE_PORT"] == ports1["FE_PORT"] {
		t.Error("Second worktree should get different FE_PORT")
	}
	if ports2["BE_PORT"] == ports1["BE_PORT"] {
		t.Error("Second worktree should get different BE_PORT")
	}
	if ports2["POSTGRES_PORT"] == ports1["POSTGRES_PORT"] {
		t.Error("Second worktree should get different POSTGRES_PORT")
	}
}

func TestFindAvailablePort(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "registry-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	reg, err := Load(tempDir, testConfig())
	if err != nil {
		t.Fatal(err)
	}

	// Find available port for FE_PORT
	port, err := reg.FindAvailablePort("FE_PORT")
	if err != nil {
		t.Fatalf("Failed to find available port: %v", err)
	}

	// Should be in valid range
	if port < 3000 || port > 3100 {
		t.Errorf("Port %d is outside expected range 3000-3100", port)
	}

	// Test invalid service
	_, err = reg.FindAvailablePort("INVALID_SERVICE")
	if err == nil {
		t.Error("Expected error for invalid service")
	}
}

func TestBuildPortRanges(t *testing.T) {
	// Test with nil config (should return empty)
	ranges := BuildPortRanges(nil)
	if len(ranges) != 0 {
		t.Errorf("Expected empty ranges with nil config, got %v", ranges)
	}

	// Test with configured explicit ranges
	explicitRange := [2]int{4000, 4100}
	workCfg := &config.WorktreeConfig{
		Ports: map[string]config.PortConfig{
			"FE_PORT": {
				Range: &explicitRange,
			},
		},
	}

	ranges = BuildPortRanges(workCfg)
	if ranges["FE_PORT"] != [2]int{4000, 4100} {
		t.Errorf("Expected FE_PORT range [4000, 4100], got %v", ranges["FE_PORT"])
	}
	// BE_PORT should not be in ranges since it wasn't configured
	if _, exists := ranges["BE_PORT"]; exists {
		t.Errorf("BE_PORT should not exist when not configured, got %v", ranges["BE_PORT"])
	}

	// Test with port expression (no explicit range)
	workCfg2 := &config.WorktreeConfig{
		Ports: map[string]config.PortConfig{
			"BE_PORT": {
				Port: "9000 + {instance}",
			},
		},
	}

	ranges2 := BuildPortRanges(workCfg2)
	// Should extract from expression: 9000 base + 100 range
	if ranges2["BE_PORT"] != [2]int{9000, 9100} {
		t.Errorf("Expected BE_PORT range [9000, 9100], got %v", ranges2["BE_PORT"])
	}
}

func TestConfiguredPortAllocation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "registry-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create config with custom range
	customRange := [2]int{5000, 5100}
	workCfg := &config.WorktreeConfig{
		Ports: map[string]config.PortConfig{
			"FE_PORT": {
				Range: &customRange,
			},
		},
	}

	// Load registry with config
	reg, err := Load(tempDir, workCfg)
	if err != nil {
		t.Fatal(err)
	}

	// Allocate FE_PORT
	port, err := reg.FindAvailablePort("FE_PORT")
	if err != nil {
		t.Fatalf("Failed to find available port: %v", err)
	}

	// Should be in configured range, not default range
	if port < 5000 || port > 5100 {
		t.Errorf("Port %d is outside configured range 5000-5100", port)
	}
	if port >= 3000 && port <= 3100 {
		t.Errorf("Port %d is in default range, should use configured range", port)
	}
}
