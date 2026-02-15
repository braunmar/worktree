package cmd

import (
	"fmt"

	"worktree/pkg/ui"

	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart <feature-name>",
	Short: "Restart services for a feature worktree",
	Long: `Restart services for a specific feature worktree.

This command:
1. Stops all running Docker containers for the feature
2. Starts all services again with fresh state

This is useful when:
- Configuration has changed
- You need to pick up new environment variables
- Containers are in a bad state

Example:
  worktree restart feature-user-auth
  worktree restart feature-reports`,
	Args: cobra.ExactArgs(1),
	Run:  runRestart,
}

func runRestart(cmd *cobra.Command, args []string) {
	featureName := args[0]

	ui.Warning(fmt.Sprintf("Restarting Feature: %s", featureName))
	ui.NewLine()

	// Stop the worktree (reuse the stop command logic)
	ui.Info("Stopping services...")
	runStop(cmd, args)

	ui.NewLine()

	// Start the worktree (reuse the start command logic)
	ui.Info("Starting services...")
	runStart(cmd, args)

	ui.NewLine()
	ui.Success(fmt.Sprintf("Feature '%s' restarted", featureName))
	ui.NewLine()
}

func init() {
	rootCmd.AddCommand(restartCmd)
}
