package cmd

import (
	"context"
	"fmt"

	"github.com/braunmar/worktree/pkg/agent"
	"github.com/braunmar/worktree/pkg/config"
	"github.com/braunmar/worktree/pkg/ui"

	"github.com/spf13/cobra"
)

var daemonForeground bool

var agentDaemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run the agent scheduler daemon",
	Long: `Start the agent scheduler daemon to run scheduled tasks automatically.

The daemon reads all scheduled_agents from .worktree.yml and runs them
according to their cron schedules. The daemon runs in the foreground by
default, but can be registered as a system service.

Examples:
  worktree agent daemon                 # Run in foreground
  worktree agent daemon --foreground    # Same as above

Logs are written to ~/logs/worktree-scheduler.log

To run in background:
  nohup worktree agent daemon > /dev/null 2>&1 &

To register as system service:
  worktree agent install-service        # Install systemd/launchd service
  worktree agent uninstall-service      # Remove system service`,
	Run: runAgentDaemon,
}

func runAgentDaemon(cmd *cobra.Command, args []string) {
	// Load configuration
	cfg, err := config.New()
	checkError(err)

	workCfg, err := config.LoadWorktreeConfig(cfg.ProjectRoot)
	checkError(err)

	// Check if scheduled_agents section exists
	if workCfg.ScheduledAgents == nil || len(workCfg.ScheduledAgents) == 0 {
		checkError(fmt.Errorf("no scheduled_agents defined in .worktree.yml\n\nAdd a scheduled_agents section to .worktree.yml. See documentation for examples."))
	}

	// Create scheduler
	scheduler, err := agent.NewScheduler(cfg, workCfg)
	checkError(err)

	// Show startup message
	ui.Section("Starting Agent Scheduler Daemon")
	fmt.Printf("  Project: %s\n", cfg.ProjectRoot)
	fmt.Printf("  Agents: %d configured\n", len(workCfg.ScheduledAgents))
	fmt.Printf("  Logs: ~/logs/worktree-scheduler.log\n")
	fmt.Println()

	// List scheduled agents
	for taskName, task := range workCfg.ScheduledAgents {
		fmt.Printf("  • %s (%s)\n", task.Name, task.Schedule)
		fmt.Printf("    Key: %s\n", taskName)
	}

	fmt.Println()
	ui.Info("Press Ctrl+C to stop the daemon")
	fmt.Println()

	// Start scheduler
	ctx := context.Background()
	if err := scheduler.Start(ctx); err != nil {
		checkError(fmt.Errorf("scheduler failed: %w", err))
	}
}

func init() {
	agentDaemonCmd.Flags().BoolVar(&daemonForeground, "foreground", true, "Run in foreground (default)")
	agentCmd.AddCommand(agentDaemonCmd)
}
