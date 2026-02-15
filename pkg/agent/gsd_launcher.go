package agent

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"worktree/pkg/config"
)

// GSDWorkflow represents a GSD workflow configuration
type GSDWorkflow struct {
	Milestone   string
	Phase       string
	AutoExecute bool
	YoloMode    bool
}

// LaunchGSDWorkflow starts a GSD workflow with task content
func LaunchGSDWorkflow(cfg *config.Config, workflow GSDWorkflow) error {
	fmt.Printf("🔄 Launching GSD Workflow\n")
	fmt.Printf("   Milestone: %s\n", workflow.Milestone)
	if workflow.AutoExecute {
		fmt.Printf("   Auto-execute: enabled\n")
	}
	if workflow.YoloMode {
		fmt.Printf("   YOLO mode: enabled\n")
	}
	fmt.Println()

	// Build Claude command
	args := []string{}

	// Add YOLO mode flag if enabled
	if workflow.YoloMode {
		args = append(args, "--dangerously-skip-permissions")
	}

	// Build GSD command sequence
	var commands []string

	// Start with new milestone
	commands = append(commands, fmt.Sprintf("/gsd:new-milestone \"%s\"", workflow.Milestone))

	// Add plan-phase command with task content
	if workflow.Phase != "" {
		// Escape double quotes in task content
		escapedPhase := strings.ReplaceAll(workflow.Phase, "\"", "\\\"")
		commands = append(commands, fmt.Sprintf("/gsd:plan-phase \"%s\"", escapedPhase))
	}

	// Add execute-phase if auto-execute enabled
	if workflow.AutoExecute {
		commands = append(commands, "/gsd:execute-phase")
	}

	// Join commands with newlines
	commandSequence := strings.Join(commands, "\n")

	// Add -c flag to execute command sequence
	args = append(args, "-c", commandSequence)

	// Create command
	cmd := exec.Command("claude", args...)
	cmd.Dir = cfg.ProjectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Set environment for YOLO mode
	env := os.Environ()
	if workflow.YoloMode {
		env = append(env, "CLAUDE_DANGEROUSLY_SKIP_PERMISSIONS=1")
	}
	cmd.Env = env

	fmt.Printf("   Executing: claude %s\n\n", strings.Join(args, " "))

	// Run the command
	return cmd.Run()
}
