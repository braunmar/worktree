package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/braunmar/worktree/pkg/config"
	"github.com/braunmar/worktree/pkg/registry"
	"github.com/braunmar/worktree/pkg/ui"

	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs <feature-name> [project-name]",
	Short: "Show logs for a feature",
	Long: `Show logs for a specific feature worktree.

This command:
1. Validates the feature exists
2. Executes 'make app-logs' in the specified project directory (defaults to first project)
3. Follows logs in real-time (Ctrl+C to exit)

Examples:
  worktree logs feature-user-auth              # Use first project
  worktree logs feature-user-auth backend      # Specify project
  worktree logs feature-reports frontend`,
	Args: cobra.RangeArgs(1, 2),
	Run:  runLogs,
}

func runLogs(cmd *cobra.Command, args []string) {
	featureName := args[0]
	projectName := ""
	if len(args) > 1 {
		projectName = args[1]
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

	// Check if worktree exists
	if !cfg.WorktreeExists(featureName) {
		ui.Error(fmt.Sprintf("Feature directory not found: worktrees/%s", featureName))
		os.Exit(1)
	}

	// Determine which project to use
	if projectName == "" {
		// Default to first project in the worktree
		if len(wt.Projects) == 0 {
			ui.Error("No projects found in worktree")
			os.Exit(1)
		}
		projectName = wt.Projects[0]
	}

	// Validate project exists in configuration
	project, exists := workCfg.Projects[projectName]
	if !exists {
		ui.Error(fmt.Sprintf("Project '%s' not found in configuration", projectName))
		fmt.Println("\nAvailable projects:")
		for _, p := range wt.Projects {
			fmt.Printf("  - %s\n", p)
		}
		os.Exit(1)
	}

	// Display header
	ui.Info(fmt.Sprintf("Showing logs for Feature: %s (Project: %s) - Ctrl+C to exit...", featureName, projectName))
	ui.Info(fmt.Sprintf("Branch: %s", wt.Branch))
	ui.NewLine()

	// Get project directory
	featureDir := cfg.WorktreeFeaturePath(featureName)
	projectDir := featureDir + "/" + project.Dir

	// Export environment variables for compose project
	envVars := map[string]string{
		"COMPOSE_PROJECT_NAME": wt.GetComposeProject(projectName),
	}
	for service, port := range wt.Ports {
		envVars[service] = fmt.Sprintf("%d", port)
	}

	envList := os.Environ()
	for key, value := range envVars {
		envList = append(envList, fmt.Sprintf("%s=%s", key, value))
	}

	// Execute make app-logs in project directory
	makeCmd := exec.Command("make", "app-logs")
	makeCmd.Dir = projectDir
	makeCmd.Env = envList
	makeCmd.Stdout = os.Stdout
	makeCmd.Stderr = os.Stderr
	makeCmd.Stdin = os.Stdin

	if err := makeCmd.Run(); err != nil {
		// Don't treat Ctrl+C as an error
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
			ui.NewLine()
			return
		}
		ui.Error(fmt.Sprintf("Failed to show logs: %v", err))
		os.Exit(1)
	}
}
