package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"worktree/pkg/config"
	"worktree/pkg/ui"
)

// BatchTasksFile represents the structure of the batch tasks YAML file
type BatchTasksFile struct {
	Tasks []BatchTask `yaml:"tasks"`
}

// BatchTask represents a single task in the batch file
type BatchTask struct {
	Name   string `yaml:"name"`   // Feature name for worktree
	Agent  string `yaml:"agent"`  // Agent name from .worktree.yml
	Preset string `yaml:"preset"` // Preset to use (backend, frontend, fullstack)
}

var batchCmd = &cobra.Command{
	Use:   "batch",
	Short: "Batch worktree operations",
	Long:  `Batch operations for creating multiple worktrees for night shift workflows.`,
}

var batchCreateCmd = &cobra.Command{
	Use:   "create <tasks-file>",
	Short: "Create multiple worktrees from task list",
	Long: `Create multiple worktrees from a YAML task file.

The task file should have the following format:

  tasks:
    - name: security-audit
      agent: npm-audit
      preset: backend
    - name: coverage-boost
      agent: go-deps-update
      preset: backend

This creates isolated worktrees for each task, ready for queuing.

Example:
  worktree batch create night-tasks.yml`,
	Args: cobra.ExactArgs(1),
	Run:  runBatchCreate,
}

func init() {
	// Register subcommands
	batchCmd.AddCommand(batchCreateCmd)

	// Register batch command at root level
	rootCmd.AddCommand(batchCmd)
}

func runBatchCreate(cmd *cobra.Command, args []string) {
	tasksFile := args[0]

	// Load config
	cfg, err := config.New()
	checkError(err)

	workCfg, err := config.LoadWorktreeConfig(cfg.ProjectRoot)
	checkError(err)

	// Read tasks file
	data, err := os.ReadFile(tasksFile)
	checkError(err)

	// Parse YAML
	var batchTasks BatchTasksFile
	err = yaml.Unmarshal(data, &batchTasks)
	checkError(err)

	if len(batchTasks.Tasks) == 0 {
		checkError(fmt.Errorf("no tasks found in %s", tasksFile))
	}

	// Validate tasks
	for i, task := range batchTasks.Tasks {
		if task.Name == "" {
			checkError(fmt.Errorf("task %d: name is required", i+1))
		}
		if task.Agent == "" {
			checkError(fmt.Errorf("task %d: agent is required", i+1))
		}
		if task.Preset == "" {
			checkError(fmt.Errorf("task %d: preset is required", i+1))
		}

		// Verify agent exists
		if _, exists := workCfg.ScheduledAgents[task.Agent]; !exists {
			checkError(fmt.Errorf("task %d: agent not found in configuration: %s", i+1, task.Agent))
		}

		// Verify preset exists
		if _, exists := workCfg.Presets[task.Preset]; !exists {
			checkError(fmt.Errorf("task %d: preset not found in configuration: %s", i+1, task.Preset))
		}
	}

	ui.Section(fmt.Sprintf("Creating %d worktrees", len(batchTasks.Tasks)))
	fmt.Println()

	successCount := 0
	failedCount := 0

	for i, task := range batchTasks.Tasks {
		fmt.Printf("[%d/%d] Creating worktree: %s\n", i+1, len(batchTasks.Tasks), task.Name)
		fmt.Printf("        Agent: %s\n", task.Agent)
		fmt.Printf("        Preset: %s\n", task.Preset)
		fmt.Println()

		// Call the newfeature command programmatically
		// Note: This is a simplified version - in practice, you might want to
		// extract the core logic from runNewFeature into a reusable function

		// For now, we'll use a shell command approach
		err := createWorktreeForTask(task, cfg, workCfg)
		if err != nil {
			ui.Warning(fmt.Sprintf("Failed to create worktree for %s: %v", task.Name, err))
			failedCount++
		} else {
			ui.CheckMark(fmt.Sprintf("Created worktree: %s", task.Name))
			successCount++
		}
		fmt.Println()
	}

	// Summary
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Batch Create Summary\n")
	fmt.Printf("  Total: %d\n", len(batchTasks.Tasks))
	fmt.Printf("  Success: %d\n", successCount)
	fmt.Printf("  Failed: %d\n", failedCount)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if failedCount > 0 {
		fmt.Println()
		ui.Warning("Some worktrees failed to create. Check errors above.")
	} else {
		fmt.Println()
		ui.Success("All worktrees created successfully!")
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Println("  1. Queue tasks: worktree agent queue add <agent> <worktree>")
		fmt.Println("  2. Start processing: worktree agent queue start --continuous")
		fmt.Println("  3. Or use tmux for night shift: tmux new-session -s night-shift")
	}
}

// createWorktreeForTask creates a worktree for a single batch task
func createWorktreeForTask(task BatchTask, cfg *config.Config, workCfg *config.WorktreeConfig) error {
	// This is a placeholder - in a real implementation, you would:
	// 1. Extract the core worktree creation logic from runNewFeature
	// 2. Call it here with the appropriate parameters
	// 3. Handle errors appropriately

	// For now, return a helpful message
	return fmt.Errorf("batch worktree creation requires refactoring newfeature command logic")
}
