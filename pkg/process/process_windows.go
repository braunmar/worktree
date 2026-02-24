//go:build windows

package process

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"
)

// StartBackground runs a command as a background process on Windows.
// Note: process group isolation (Setpgid) is not available on Windows;
// child processes may outlive the parent.
func StartBackground(name, command, dir string, env []string, pidFile string) error {
	cmd := exec.Command("cmd", "/C", command)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start %s: %w", name, err)
	}

	pid := cmd.Process.Pid
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644); err != nil {
		_ = cmd.Process.Kill()
		return fmt.Errorf("failed to write PID file for %s: %w", name, err)
	}

	return nil
}

// StopProcess kills the process recorded in pidFile and removes the file.
func StopProcess(pidFile string) error {
	pid, err := readPID(pidFile)
	if err != nil {
		return err
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		_ = os.Remove(pidFile)
		return nil
	}

	if err := proc.Kill(); err != nil {
		_ = os.Remove(pidFile)
		return fmt.Errorf("failed to kill process %d: %w", pid, err)
	}

	time.Sleep(200 * time.Millisecond)
	_ = os.Remove(pidFile)
	return nil
}

// IsRunning reports whether the process recorded in pidFile is still alive.
func IsRunning(pidFile string) bool {
	pid, err := readPID(pidFile)
	if err != nil {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Windows FindProcess always succeeds; use tasklist to verify.
	out, err := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/NH").Output()
	if err != nil {
		_ = proc
		return false
	}
	return len(out) > 0 && string(out) != "INFO: No tasks are running which match the specified criteria.\r\n"
}
