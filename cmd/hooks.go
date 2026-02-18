package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/braunmar/worktree/pkg/ui"
)

// runHookCommand executes a lifecycle hook command (pre/post for start, stop, restart).
// Hook failures are non-fatal: they print a warning but do not exit.
// Returns true if the command succeeded or was skipped (empty).
func runHookCommand(label, command, workDir string, envList []string) bool {
	if command == "" {
		return true
	}

	ui.Loading(fmt.Sprintf("Running %s...", label))
	ui.NewLine()

	hookCmd := exec.Command("sh", "-c", command)
	hookCmd.Dir = workDir
	hookCmd.Env = envList
	hookCmd.Stdout = os.Stdout
	hookCmd.Stderr = os.Stderr

	if err := hookCmd.Run(); err != nil {
		ui.Warning(fmt.Sprintf("%s failed: %v", label, err))
		ui.Info(fmt.Sprintf("You can run manually: %s", command))
		return false
	}

	ui.Success(fmt.Sprintf("%s completed", label))
	ui.NewLine()
	return true
}
