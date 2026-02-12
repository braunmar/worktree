package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"worktree/pkg/config"
	"worktree/pkg/ui"

	"github.com/spf13/cobra"
)

var stopAllCmd = &cobra.Command{
	Use:   "stop-all",
	Short: "Stop all running instances",
	Long: `Stop all running instances.

This command:
1. Executes 'make down-all' in the main project directory
   (uses first configured project or project with claude_working_dir: true)
2. Shows status

Example:
  worktree stop-all`,
	Args: cobra.NoArgs,
	Run:  runStopAll,
}

func runStopAll(cmd *cobra.Command, args []string) {
	// Get configuration
	cfg, err := config.New()
	checkError(err)

	// Load worktree configuration
	workCfg, err := config.LoadWorktreeConfig(cfg.ProjectRoot)
	checkError(err)

	// Determine which project to use for the stop-all command
	// Prefer the claude working directory project, or use first project
	projectName := workCfg.GetClaudeWorkingProject()
	if projectName == "" {
		// Fallback to first project if no claude working dir configured
		for name := range workCfg.Projects {
			projectName = name
			break
		}
	}

	if projectName == "" {
		ui.Error("No projects configured")
		os.Exit(1)
	}

	project := workCfg.Projects[projectName]
	projectDir := cfg.ProjectRoot + "/" + project.Dir

	// Display header
	ui.Warning(fmt.Sprintf("Stopping all instances (using %s)...", projectName))
	ui.NewLine()

	// Execute make down-all in project directory
	makeCmd := exec.Command("make", "down-all")
	makeCmd.Dir = projectDir
	makeCmd.Stdout = os.Stdout
	makeCmd.Stderr = os.Stderr

	if err := makeCmd.Run(); err != nil {
		ui.Error(fmt.Sprintf("Failed to stop all instances: %v", err))
		os.Exit(1)
	}

	ui.NewLine()
	ui.Success("All instances stopped")
	ui.NewLine()
}
