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
	Long: `Stop all running instances (0-5).

This command:
1. Executes 'make down-all' in the backend directory
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

	// Display header
	ui.Warning("Stopping all instances...")
	ui.NewLine()

	// Execute make down-all in backend directory
	makeCmd := exec.Command("make", "down-all")
	makeCmd.Dir = cfg.BackendDir
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
