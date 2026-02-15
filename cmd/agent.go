package cmd

import (
	"github.com/spf13/cobra"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage scheduled agent tasks",
	Long: `Scheduled agent task management.

Agent tasks are automated maintenance operations that run on a schedule
(via cron, launchd, or manually). They create temporary isolated worktrees,
execute tasks, run safety checks, and create PRs for review.

Examples:
  worktree agent run npm-audit        # Run npm audit task
  worktree agent list                  # List all agent tasks
  worktree agent validate npm-audit    # Validate task definition
  worktree agent status npm-audit      # Show last run time`,
}

func init() {
	// Subcommands are registered in their respective files:
	// - agentRunCmd (agent_run.go)
	// - agentListCmd (agent_list.go)
	// - agentValidateCmd (agent_validate.go)
	// - agentScheduleCmd (agent_schedule.go)
}
