package agent

import (
	"fmt"
	"os/exec"
	"strings"
)

// CreateAgentSession creates a tmux session for an agent task
func CreateAgentSession(sessionName, worktreeDir, command string) error {
	// Create new tmux session
	cmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName, "-c", worktreeDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create tmux session: %w", err)
	}

	// Send command to session
	cmd = exec.Command("tmux", "send-keys", "-t", sessionName, command, "C-m")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to send command to session: %w", err)
	}

	return nil
}

// ListAgentSessions returns all running tmux sessions
func ListAgentSessions() ([]string, error) {
	cmd := exec.Command("tmux", "list-sessions", "-F", "#{session_name}")
	output, err := cmd.Output()
	if err != nil {
		// tmux returns error if no sessions exist
		if strings.Contains(err.Error(), "no server running") ||
		   strings.Contains(err.Error(), "no sessions") {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	sessions := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(sessions) == 1 && sessions[0] == "" {
		return []string{}, nil
	}

	return sessions, nil
}

// AttachSession attaches to a tmux session
func AttachSession(sessionName string) error {
	cmd := exec.Command("tmux", "attach-session", "-t", sessionName)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	return cmd.Run()
}

// KillSession kills a tmux session
func KillSession(sessionName string) error {
	cmd := exec.Command("tmux", "kill-session", "-t", sessionName)
	return cmd.Run()
}

// SessionExists checks if a tmux session exists
func SessionExists(sessionName string) bool {
	cmd := exec.Command("tmux", "has-session", "-t", sessionName)
	return cmd.Run() == nil
}
