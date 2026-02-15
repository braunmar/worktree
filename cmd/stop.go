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
	Use:   "stop [feature-name]",
	Short: "Stop services for a feature worktree",
	Long: `Stop services for a specific feature worktree.

This command:
1. Validates the feature exists
2. Stops all running Docker containers for the feature
3. Shows status

If no feature name is provided and you're in a worktree directory,
the feature will be auto-detected from .worktree-instance.

Examples:
  worktree stop feature-user-auth    # Explicit feature name
  worktree stop                      # Auto-detect from current directory`,
	Args: cobra.MaximumNArgs(1),
	Run:  runStop,
}

func runStop(cmd *cobra.Command, args []string) {
	var featureName string
	autoDetected := false

	// Auto-detect feature name if not provided
	if len(args) == 0 {
		instance, err := config.DetectInstance()
		if err != nil {
			ui.Error("Not in a worktree directory and no feature name provided")
			ui.Info("Usage: worktree stop <feature-name>")
			ui.Info("   or: cd to a worktree directory and run: worktree stop")
			os.Exit(1)
		}
		featureName = instance.Feature
		autoDetected = true
	} else {
		featureName = args[0]
	}

	// Get configuration
	cfg, err := config.New()
	checkError(err)

	// Load worktree configuration
	workCfg, err := config.LoadWorktreeConfig(cfg.ProjectRoot)
	checkError(err)

	// Load registry
	reg, err := registry.Load(cfg.WorktreeDir, workCfg)
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
	if autoDetected {
		ui.Info("✨ Auto-detected from current directory")
	}
	ui.Info(fmt.Sprintf("Branch: %s", wt.Branch))
	ui.NewLine()

	// Check if feature is running
	if !docker.IsFeatureRunning(workCfg.ProjectName, featureName) {
		ui.Info(fmt.Sprintf("Feature '%s' is not running", featureName))
		os.Exit(0)
	}

	// Get worktree path
	featurePath := cfg.WorktreeFeaturePath(featureName)

	// Build map of project directory to compose project name
	projectInfo := make(map[string]string)
	for _, projectName := range wt.Projects {
		if projectCfg, exists := workCfg.Projects[projectName]; exists {
			// Get the compose project name for this project from registry
			composeName := wt.GetComposeProject(projectName)
			if composeName == "" {
				// Fallback to default naming if not in registry
				composeName = fmt.Sprintf("%s-%s-%s", workCfg.ProjectName, featureName, projectName)
			}
			projectInfo[projectCfg.Dir] = composeName
		}
	}

	// Stop feature services
	ui.Loading("Stopping services...")
	if err := docker.StopFeature(workCfg.ProjectName, featureName, featurePath, projectInfo); err != nil {
		ui.Error(fmt.Sprintf("Failed to stop services: %v", err))
		os.Exit(1)
	}

	ui.NewLine()
	ui.Success(fmt.Sprintf("Feature '%s' stopped", featureName))
	ui.NewLine()
}
