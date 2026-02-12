package cmd

import (
	"fmt"

	"github.com/braunmar/worktree/pkg/agent"
	"github.com/braunmar/worktree/pkg/config"

	"github.com/spf13/cobra"
)

var agentRunCmd = &cobra.Command{
	Use:   "run <task-name>",
	Short: "Run a scheduled agent task",
	Long: `Execute a scheduled agent task in an isolated temporary worktree.

The task is loaded from .worktree.yml, a temporary worktree is created,
steps are executed, safety gates are run, and changes are committed/pushed
if all gates pass.

Examples:
  worktree agent run npm-audit         # Run npm audit task
  worktree agent run backend-deps      # Run backend dependency update
  worktree agent run go-version-update # Run Go version update`,
	Args: cobra.ExactArgs(1),
	Run:  runAgentTask,
}

func runAgentTask(cmd *cobra.Command, args []string) {
	taskName := args[0]

	// Load configuration
	cfg, err := config.New()
	checkError(err)

	workCfg, err := config.LoadWorktreeConfig(cfg.ProjectRoot)
	checkError(err)

	// Check if scheduled_agents section exists
	if workCfg.ScheduledAgents == nil {
		checkError(fmt.Errorf("no scheduled_agents defined in .worktree.yml\n\nAdd a scheduled_agents section to .worktree.yml. See documentation for examples."))
	}

	// Get agent task definition
	task, exists := workCfg.ScheduledAgents[taskName]
	if !exists {
		// List available tasks
		available := []string{}
		for name := range workCfg.ScheduledAgents {
			available = append(available, name)
		}

		if len(available) > 0 {
			checkError(fmt.Errorf("agent task '%s' not found in .worktree.yml\n\nAvailable tasks: %v", taskName, available))
		} else {
			checkError(fmt.Errorf("agent task '%s' not found in .worktree.yml\n\nNo agent tasks defined. Add them to the scheduled_agents section.", taskName))
		}
	}

	// Create and run agent executor
	executor := agent.NewExecutor(cfg, workCfg, task, taskName)
	err = executor.Run()
	if err != nil {
		checkError(fmt.Errorf("agent task failed: %w", err))
	}
}

func init() {
	agentCmd.AddCommand(agentRunCmd)
}
