package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/braunmar/worktree/pkg/agent"
	"github.com/braunmar/worktree/pkg/config"
	"github.com/braunmar/worktree/pkg/queue"
	"github.com/braunmar/worktree/pkg/ui"
)

var (
	queueContinuous bool
)

var agentQueueCmd = &cobra.Command{
	Use:   "queue",
	Short: "Manage agent task queue",
	Long:  `Manage the agent task queue for sequential execution of janitor tasks.`,
}

var queueAddCmd = &cobra.Command{
	Use:   "add <agent-name> <worktree>",
	Short: "Add task to queue",
	Long: `Add an agent task to the queue for execution.

The agent-name must be defined in .worktree.yml under scheduled_agents.
The worktree is the feature name (normalized branch name).

Example:
  worktree agent queue add npm-audit security-audit
  worktree agent queue add go-deps-update coverage-boost`,
	Args: cobra.ExactArgs(2),
	Run:  runQueueAdd,
}

var queueListCmd = &cobra.Command{
	Use:   "list",
	Short: "List queued tasks",
	Long: `List all tasks in the queue with their status.

Status values:
  - pending: Waiting to be executed
  - running: Currently executing
  - completed: Successfully finished
  - failed: Execution failed`,
	Run: runQueueList,
}

var queueStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start processing queue",
	Long: `Start processing the queue, running the next pending task.

By default, runs one task and exits. Use --continuous to process
all pending tasks sequentially.

Example:
  worktree agent queue start              # Run one task
  worktree agent queue start --continuous # Run all tasks`,
	Run: runQueueStart,
}

var queueRemoveCmd = &cobra.Command{
	Use:   "remove <task-id>",
	Short: "Remove task from queue",
	Long: `Remove a specific task from the queue by its ID.

Get the task ID from 'worktree agent queue list'.

Example:
  worktree agent queue remove abc123-def456-...`,
	Args: cobra.ExactArgs(1),
	Run:  runQueueRemove,
}

var queueClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear completed and failed tasks",
	Long: `Remove all completed and failed tasks from the queue,
keeping only pending and running tasks.

Example:
  worktree agent queue clear`,
	Run: runQueueClear,
}

func init() {
	// Add flags
	queueStartCmd.Flags().BoolVar(&queueContinuous, "continuous", false, "Process all pending tasks sequentially")

	// Register subcommands
	agentQueueCmd.AddCommand(queueAddCmd)
	agentQueueCmd.AddCommand(queueListCmd)
	agentQueueCmd.AddCommand(queueStartCmd)
	agentQueueCmd.AddCommand(queueRemoveCmd)
	agentQueueCmd.AddCommand(queueClearCmd)

	// Register queue command under agent
	agentCmd.AddCommand(agentQueueCmd)
}

func runQueueAdd(cmd *cobra.Command, args []string) {
	agentName := args[0]
	worktree := args[1]

	// Load config
	cfg, err := config.New()
	checkError(err)

	workCfg, err := config.LoadWorktreeConfig(cfg.ProjectRoot)
	checkError(err)

	// Verify agent exists
	if _, exists := workCfg.ScheduledAgents[agentName]; !exists {
		checkError(fmt.Errorf("agent not found in configuration: %s", agentName))
	}

	// Load queue
	q, err := queue.Load(cfg.WorktreeDir)
	checkError(err)

	// Add task
	task, err := q.Add(agentName, worktree)
	checkError(err)

	ui.Success(fmt.Sprintf("Task added to queue"))
	fmt.Printf("  ID: %s\n", task.ID)
	fmt.Printf("  Agent: %s\n", task.AgentName)
	fmt.Printf("  Worktree: %s\n", task.Worktree)
	fmt.Printf("  Status: %s\n", task.Status)
	fmt.Println()

	// Show queue position
	pendingCount := q.Count(queue.StatusPending)
	fmt.Printf("  Queue position: %d of %d pending tasks\n", pendingCount, pendingCount)
}

func runQueueList(cmd *cobra.Command, args []string) {
	// Load config
	cfg, err := config.New()
	checkError(err)

	// Load queue
	q, err := queue.Load(cfg.WorktreeDir)
	checkError(err)

	tasks := q.List("")

	if len(tasks) == 0 {
		ui.Info("Queue is empty")
		return
	}

	ui.Section("Task Queue")
	fmt.Println()

	// Group by status
	statusGroups := map[queue.TaskStatus][]queue.QueuedTask{
		queue.StatusPending:   {},
		queue.StatusRunning:   {},
		queue.StatusCompleted: {},
		queue.StatusFailed:    {},
	}

	for _, task := range tasks {
		statusGroups[task.Status] = append(statusGroups[task.Status], task)
	}

	// Display each group
	for _, status := range []queue.TaskStatus{queue.StatusRunning, queue.StatusPending, queue.StatusCompleted, queue.StatusFailed} {
		groupTasks := statusGroups[status]
		if len(groupTasks) == 0 {
			continue
		}

		// Status header with emoji
		var emoji string
		switch status {
		case queue.StatusPending:
			emoji = "⏸️"
		case queue.StatusRunning:
			emoji = "▶️"
		case queue.StatusCompleted:
			emoji = "✅"
		case queue.StatusFailed:
			emoji = "❌"
		}

		fmt.Printf("%s %s (%d)\n", emoji, status, len(groupTasks))
		fmt.Println()

		for _, task := range groupTasks {
			fmt.Printf("  ID: %s\n", task.ID[:8]+"...")
			fmt.Printf("  Agent: %s\n", task.AgentName)
			fmt.Printf("  Worktree: %s\n", task.Worktree)
			fmt.Printf("  Created: %s\n", task.CreatedAt.Format("2006-01-02 15:04:05"))

			if task.StartedAt != nil {
				fmt.Printf("  Started: %s\n", task.StartedAt.Format("2006-01-02 15:04:05"))
			}

			if task.CompletedAt != nil {
				fmt.Printf("  Completed: %s\n", task.CompletedAt.Format("2006-01-02 15:04:05"))
				fmt.Printf("  Duration: %s\n", time.Duration(task.Duration)*time.Millisecond)
			}

			if task.Error != "" {
				fmt.Printf("  Error: %s\n", task.Error)
			}

			fmt.Println()
		}
	}

	// Summary
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Total: %d tasks\n", len(tasks))
	fmt.Printf("  Pending: %d\n", len(statusGroups[queue.StatusPending]))
	fmt.Printf("  Running: %d\n", len(statusGroups[queue.StatusRunning]))
	fmt.Printf("  Completed: %d\n", len(statusGroups[queue.StatusCompleted]))
	fmt.Printf("  Failed: %d\n", len(statusGroups[queue.StatusFailed]))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

func runQueueStart(cmd *cobra.Command, args []string) {
	// Load config
	cfg, err := config.New()
	checkError(err)

	workCfg, err := config.LoadWorktreeConfig(cfg.ProjectRoot)
	checkError(err)

	// Load queue
	q, err := queue.Load(cfg.WorktreeDir)
	checkError(err)

	// Check for pending tasks
	pendingCount := q.Count(queue.StatusPending)
	if pendingCount == 0 {
		ui.Info("No pending tasks in queue")
		return
	}

	ui.Section(fmt.Sprintf("Processing Queue (%d pending)", pendingCount))
	fmt.Println()

	// Process queue
	if queueContinuous {
		err = agent.ProcessQueueContinuous(cfg, workCfg, q)
	} else {
		err = agent.ProcessQueue(cfg, workCfg, q)
	}

	if err != nil {
		fmt.Println()
		ui.Warning(fmt.Sprintf("Queue processing encountered errors: %v", err))
		fmt.Println()
		ui.Info("Use 'worktree agent queue list' to see task status")
	}
}

func runQueueRemove(cmd *cobra.Command, args []string) {
	taskID := args[0]

	// Load config
	cfg, err := config.New()
	checkError(err)

	// Load queue
	q, err := queue.Load(cfg.WorktreeDir)
	checkError(err)

	// Remove task
	err = q.Remove(taskID)
	checkError(err)

	ui.Success(fmt.Sprintf("Task removed from queue: %s", taskID))
}

func runQueueClear(cmd *cobra.Command, args []string) {
	// Load config
	cfg, err := config.New()
	checkError(err)

	// Load queue
	q, err := queue.Load(cfg.WorktreeDir)
	checkError(err)

	// Count before clear
	before := q.Count("")
	completedCount := q.Count(queue.StatusCompleted)
	failedCount := q.Count(queue.StatusFailed)

	// Clear
	err = q.Clear()
	checkError(err)

	// Count after clear
	after := q.Count("")
	removed := before - after

	ui.Success(fmt.Sprintf("Cleared %d tasks from queue", removed))
	fmt.Printf("  Completed: %d\n", completedCount)
	fmt.Printf("  Failed: %d\n", failedCount)
	fmt.Printf("  Remaining: %d\n", after)
}
