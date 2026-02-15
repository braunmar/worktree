package agent

import (
	"fmt"
	"os"
	"os/exec"

	"worktree/pkg/config"
)

// executeSkillStep executes a Claude Code skill command
func (e *Executor) executeSkillStep(step config.AgentStep) error {
	// Skills are invoked via claude CLI with -c flag
	// Example: claude -c "/backend 'Run npm audit fix'"

	if step.Skill == "" {
		return fmt.Errorf("skill field is empty for skill step")
	}

	fmt.Printf("      Skill: %s\n", step.Skill)

	// Build command arguments
	args := []string{"-c", step.Skill}

	// Add dangerously-skip-permissions flag if YOLO mode enabled
	if e.task.Context.Yolo {
		args = append([]string{"--dangerously-skip-permissions"}, args...)
		fmt.Printf("      YOLO mode: enabled\n")
	}

	cmd := exec.Command("claude", args...)

	// Set working directory
	if step.WorkingDir != "" {
		cmd.Dir = step.WorkingDir
		fmt.Printf("      Working directory: %s\n", step.WorkingDir)
	} else {
		cmd.Dir = e.cfg.ProjectRoot
	}

	// Connect stdout and stderr for visibility
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin // Important for interactive skills

	// Set environment variables for YOLO mode
	env := os.Environ()
	if e.task.Context.Yolo {
		env = append(env, "CLAUDE_DANGEROUSLY_SKIP_PERMISSIONS=1")
	}
	cmd.Env = env

	fmt.Println()

	// Run the skill
	return cmd.Run()
}
