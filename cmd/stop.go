package cmd

import (
	"fmt"
	"os"

	"worktree/pkg/config"
	"worktree/pkg/docker"
	"worktree/pkg/registry"
	"worktree/pkg/ui"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop <feature-name>",
	Short: "Stop services for a feature worktree",
	Long: `Stop services for a specific feature worktree.

This command:
1. Validates the feature exists
2. Stops all running Docker containers for the feature
3. Shows status

Example:
  worktree stop feature-user-auth
  worktree stop feature-reports`,
	Args: cobra.ExactArgs(1),
	Run:  runStop,
}

func runStop(cmd *cobra.Command, args []string) {
	featureName := args[0]

	// Get configuration
	cfg, err := config.New()
	checkError(err)

	// Load registry
	reg, err := registry.Load(cfg.WorktreeDir)
	checkError(err)

	// Get worktree from registry
	wt, exists := reg.Get(featureName)
	if !exists {
		ui.Error(fmt.Sprintf("Feature worktree '%s' not found", featureName))
		fmt.Println("\nAvailable features:")
		for _, w := range reg.List() {
			fmt.Printf("  - %s\n", w.Normalized)
		}
		os.Exit(1)
	}

	// Display header
	ui.Warning(fmt.Sprintf("Stopping Feature: %s", featureName))
	ui.Info(fmt.Sprintf("Branch: %s", wt.Branch))
	ui.NewLine()

	// Check if feature is running
	if !docker.IsFeatureRunning(featureName) {
		ui.Info(fmt.Sprintf("Feature '%s' is not running", featureName))
		os.Exit(0)
	}

	// Get worktree path
	featurePath := cfg.WorktreeFeaturePath(featureName)

	// Stop feature services
	ui.Loading("Stopping services...")
	if err := docker.StopFeature(featureName, featurePath); err != nil {
		ui.Error(fmt.Sprintf("Failed to stop services: %v", err))
		os.Exit(1)
	}

	ui.NewLine()
	ui.Success(fmt.Sprintf("Feature '%s' stopped", featureName))
	ui.NewLine()
}
