package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/braunmar/worktree/pkg/config"
	"github.com/braunmar/worktree/pkg/ui"

	"github.com/spf13/cobra"
)

var agentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured agent tasks",
	Long: `Display all scheduled agent tasks configured in .worktree.yml.

Shows task name, description, schedule (cron expression), and context.

Examples:
  worktree agent list`,
	Run: runAgentList,
}

func runAgentList(cmd *cobra.Command, args []string) {
	// Load configuration
	cfg, err := config.New()
	checkError(err)

	workCfg, err := config.LoadWorktreeConfig(cfg.ProjectRoot)
	checkError(err)

	// Check if scheduled_agents section exists
	if workCfg.ScheduledAgents == nil || len(workCfg.ScheduledAgents) == 0 {
		ui.Warning("No scheduled agents configured in .worktree.yml")
		fmt.Println()
		fmt.Println("Add a scheduled_agents section to .worktree.yml. See documentation for examples.")
		return
	}

	// Sort agent names
	names := make([]string, 0, len(workCfg.ScheduledAgents))
	for name := range workCfg.ScheduledAgents {
		names = append(names, name)
	}
	sort.Strings(names)

	// Display header
	ui.Section(fmt.Sprintf("Configured Agent Tasks (%d)", len(names)))
	fmt.Println()

	// Display each agent
	for i, name := range names {
		task := workCfg.ScheduledAgents[name]

		fmt.Printf("  %s\n", ui.Bold(task.Name))
		fmt.Printf("    Key: %s\n", name)
		fmt.Printf("    Description: %s\n", task.Description)
		fmt.Printf("    Schedule: %s (%s)\n", task.Schedule, parseCronSchedule(task.Schedule))
		fmt.Printf("    Preset: %s | Branch: %s | Instance: %d | YOLO: %v\n",
			task.Context.Preset, task.Context.Branch, task.Context.Instance, task.Context.Yolo)
		fmt.Printf("    Steps: %d | Safety Gates: %d\n", len(task.Steps), len(task.Safety.Gates))

		if i < len(names)-1 {
			fmt.Println()
		}
	}

	fmt.Println()
	ui.Info("Run 'worktree agent run <task-name>' to execute a task")
	ui.Info("Run 'worktree agent validate <task-name>' to validate configuration")
}

// parseCronSchedule returns a human-readable description of a cron schedule
func parseCronSchedule(cron string) string {
	parts := strings.Fields(cron)
	if len(parts) != 5 {
		return "invalid cron expression"
	}

	minute, hour, dayOfMonth, month, dayOfWeek := parts[0], parts[1], parts[2], parts[3], parts[4]

	// Common patterns
	if cron == "* * * * *" {
		return "every minute"
	}
	if cron == "0 * * * *" {
		return "every hour"
	}
	if cron == "0 0 * * *" {
		return "daily at midnight"
	}
	if cron == "0 9 * * MON" {
		return "every Monday at 9:00 AM"
	}
	if cron == "0 0 1 * *" {
		return "first day of month at midnight"
	}

	// Generic description
	desc := []string{}

	if minute == "0" {
		desc = append(desc, "at the hour")
	} else if minute == "*" {
		desc = append(desc, "every minute")
	} else {
		desc = append(desc, fmt.Sprintf("at minute %s", minute))
	}

	if hour != "*" {
		desc = append(desc, fmt.Sprintf("hour %s", hour))
	}

	if dayOfMonth != "*" {
		desc = append(desc, fmt.Sprintf("day %s", dayOfMonth))
	}

	if month != "*" {
		desc = append(desc, fmt.Sprintf("month %s", month))
	}

	if dayOfWeek != "*" {
		desc = append(desc, fmt.Sprintf("weekday %s", dayOfWeek))
	}

	return strings.Join(desc, ", ")
}

func init() {
	agentCmd.AddCommand(agentListCmd)
}
