//go:build !windows

package process

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"
)

// StartBackground runs a shell command as a background process, saves its PID to pidFile,
// and returns immediately. The process is started in its own process group so it can
// be killed cleanly with StopProcess.
func StartBackground(name, command, dir string, env []string, pidFile string) error {
	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = dir
	cmd.Env = env
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Redirect stdout/stderr to the terminal so the user can see startup output
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start %s: %w", name, err)
	}

	pid := cmd.Process.Pid
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644); err != nil {
		// Best-effort kill if we can't save the PID
		_ = cmd.Process.Kill()
		return fmt.Errorf("failed to write PID file for %s: %w", name, err)
	}

	return nil
}

// StopProcess sends SIGTERM to the process group, waits up to 5 seconds,
// then sends SIGKILL if the process is still running. Removes the PID file.
func StopProcess(pidFile string) error {
	pid, err := readPID(pidFile)
	if err != nil {
		return err
	}

	// Send SIGTERM to the entire process group (negative PID)
	pgid := -pid
	if err := syscall.Kill(pgid, syscall.SIGTERM); err != nil && err != syscall.ESRCH {
		// Process may already be gone; treat ESRCH as success
		_ = os.Remove(pidFile)
		return fmt.Errorf("failed to send SIGTERM to %d: %w", pid, err)
	}

	// Wait up to 5 seconds for graceful shutdown
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !isAlive(pid) {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Force kill if still running
	if isAlive(pid) {
		_ = syscall.Kill(pgid, syscall.SIGKILL)
	}

	_ = os.Remove(pidFile)
	return nil
}

// IsRunning reports whether the process recorded in pidFile is still alive.
func IsRunning(pidFile string) bool {
	pid, err := readPID(pidFile)
	if err != nil {
		return false
	}
	return isAlive(pid)
}

// isAlive checks whether a process with the given PID is running by sending
// signal 0 (a no-op that still returns ESRCH if the process doesn't exist).
func isAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}
