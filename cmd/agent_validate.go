package cmd

import (
	"fmt"

	"worktree/pkg/config"
	"worktree/pkg/ui"

	"github.com/spf13/cobra"
)

var agentValidateCmd = &cobra.Command{
	Use:   "validate <task-name>",
	Short: "Validate agent task configuration",
	Long: `Validate a scheduled agent task configuration.

Checks:
- Task exists in .worktree.yml
- Preset exists and is valid
- Branch is specified
- Steps are configured correctly
- Safety gates are configured
- Git configuration is valid

Examples:
  worktree agent validate npm-audit
  worktree agent validate go-version-upgrade`,
	Args: cobra.ExactArgs(1),
	Run:  runAgentValidate,
}

func runAgentValidate(cmd *cobra.Command, args []string) {
	taskName := args[0]

	// Load configuration
	cfg, err := config.New()
	checkError(err)

	workCfg, err := config.LoadWorktreeConfig(cfg.ProjectRoot)
	checkError(err)

	// Check if scheduled_agents section exists
	if workCfg.ScheduledAgents == nil {
		checkError(fmt.Errorf("no scheduled_agents defined in .worktree.yml"))
	}

	// Get agent task definition
	task, exists := workCfg.ScheduledAgents[taskName]
	if !exists {
		checkError(fmt.Errorf("agent task '%s' not found in .worktree.yml", taskName))
	}

	ui.Section(fmt.Sprintf("Validating Agent Task: %s", task.Name))
	fmt.Println()

	errors := 0

	// Validate task name
	if task.Name == "" {
		ui.Error("✗ Task name is empty")
		errors++
	} else {
		ui.CheckMark(fmt.Sprintf("Task name: %s", task.Name))
	}

	// Validate description
	if task.Description == "" {
		ui.Warning("⚠ Task description is empty (recommended to add)")
	} else {
		ui.CheckMark(fmt.Sprintf("Description: %s", task.Description))
	}

	// Validate schedule (cron expression)
	if task.Schedule == "" {
		ui.Error("✗ Schedule is empty (cron expression required)")
		errors++
	} else {
		ui.CheckMark(fmt.Sprintf("Schedule: %s", task.Schedule))
	}

	// Validate context
	if task.Context.Preset == "" {
		ui.Error("✗ Preset is empty")
		errors++
	} else {
		// Check if preset exists
		if _, exists := workCfg.Presets[task.Context.Preset]; !exists {
			ui.Error(fmt.Sprintf("✗ Preset '%s' not found in .worktree.yml", task.Context.Preset))
			errors++
		} else {
			ui.CheckMark(fmt.Sprintf("Preset: %s", task.Context.Preset))
		}
	}

	if task.Context.Branch == "" {
		ui.Error("✗ Branch is empty")
		errors++
	} else {
		ui.CheckMark(fmt.Sprintf("Branch: %s", task.Context.Branch))
	}

	if task.Context.Instance < 0 || task.Context.Instance > 100 {
		ui.Warning(fmt.Sprintf("⚠ Instance %d may be out of typical range (0-100)", task.Context.Instance))
	} else {
		ui.CheckMark(fmt.Sprintf("Instance: %d", task.Context.Instance))
	}

	ui.CheckMark(fmt.Sprintf("YOLO mode: %v", task.Context.Yolo))

	// Validate steps
	if len(task.Steps) == 0 {
		ui.Error("✗ No steps configured")
		errors++
	} else {
		ui.CheckMark(fmt.Sprintf("Steps: %d configured", len(task.Steps)))

		for i, step := range task.Steps {
			if step.Name == "" {
				ui.Error(fmt.Sprintf("  ✗ Step %d: name is empty", i+1))
				errors++
			}

			if step.Type != "shell" && step.Type != "skill" {
				ui.Error(fmt.Sprintf("  ✗ Step %d (%s): invalid type '%s' (must be 'shell' or 'skill')", i+1, step.Name, step.Type))
				errors++
			} else if step.Type == "shell" {
				if step.Command == "" {
					ui.Error(fmt.Sprintf("  ✗ Step %d (%s): command is empty", i+1, step.Name))
					errors++
				} else {
					ui.CheckMark(fmt.Sprintf("  Step %d: %s (shell)", i+1, step.Name))
				}
			} else if step.Type == "skill" {
				if step.Skill == "" {
					ui.Error(fmt.Sprintf("  ✗ Step %d (%s): skill is empty", i+1, step.Name))
					errors++
				} else {
					ui.CheckMark(fmt.Sprintf("  Step %d: %s (skill: %s)", i+1, step.Name, step.Skill))
				}
			}
		}
	}

	// Validate safety gates
	if len(task.Safety.Gates) == 0 {
		ui.Warning("⚠ No safety gates configured (recommended to add)")
	} else {
		ui.CheckMark(fmt.Sprintf("Safety gates: %d configured", len(task.Safety.Gates)))

		for i, gate := range task.Safety.Gates {
			if gate.Name == "" {
				ui.Error(fmt.Sprintf("  ✗ Gate %d: name is empty", i+1))
				errors++
			}
			if gate.Command == "" {
				ui.Error(fmt.Sprintf("  ✗ Gate %d (%s): command is empty", i+1, gate.Name))
				errors++
			} else {
				required := ""
				if gate.Required {
					required = " (required)"
				}
				ui.CheckMark(fmt.Sprintf("  Gate %d: %s%s", i+1, gate.Name, required))
			}
		}
	}

	// Validate git configuration
	if task.Safety.Git.Branch == "" {
		ui.Error("✗ Git branch is empty")
		errors++
	} else {
		ui.CheckMark(fmt.Sprintf("Git branch: %s", task.Safety.Git.Branch))
	}

	if task.Safety.Git.CommitMessage == "" {
		ui.Error("✗ Git commit message is empty")
		errors++
	} else {
		ui.CheckMark("Git commit message configured")
	}

	// Validate push configuration
	if task.Safety.Git.Push.Enabled {
		ui.CheckMark("Push enabled")

		if task.Safety.Git.Push.CreatePR {
			ui.CheckMark("PR creation enabled")

			if task.Safety.Git.Push.PRTitle == "" {
				ui.Warning("⚠ PR title is empty (recommended to add)")
			} else {
				ui.CheckMark("PR title configured")
			}

			if task.Safety.Git.Push.PRBody == "" {
				ui.Warning("⚠ PR body is empty (recommended to add)")
			} else {
				ui.CheckMark("PR body configured")
			}

			if task.Safety.Git.Push.AutoMerge {
				ui.Warning("⚠ Auto-merge is enabled (use with caution)")
			}
		}
	} else {
		ui.Info("Push disabled")
	}

	// Validate rollback configuration
	if task.Safety.Rollback.Enabled {
		ui.CheckMark(fmt.Sprintf("Rollback enabled (strategy: %s)", task.Safety.Rollback.Strategy))
	} else {
		ui.Warning("⚠ Rollback disabled (recommended to enable)")
	}

	// Summary
	fmt.Println()
	if errors == 0 {
		ui.Success(fmt.Sprintf("✅ Agent task '%s' is valid", taskName))
	} else {
		ui.Error(fmt.Sprintf("❌ Agent task '%s' has %d error(s)", taskName, errors))
		fmt.Println()
		fmt.Println("Fix the errors above before running this agent task.")
		checkError(fmt.Errorf("validation failed"))
	}
}

func init() {
	agentCmd.AddCommand(agentValidateCmd)
}
