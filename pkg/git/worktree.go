package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// WorktreeInfo holds information about a worktree
type WorktreeInfo struct {
	Path   string
	Branch string
	Clean  bool
}

// CreateWorktree creates a new git worktree
func CreateWorktree(repoPath, worktreePath, branch string) error {
	// Convert to absolute paths
	absRepoPath, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for repo: %w", err)
	}

	absWorktreePath, err := filepath.Abs(worktreePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for worktree: %w", err)
	}

	// Check if branch already exists
	checkCmd := exec.Command("git", "-C", absRepoPath, "rev-parse", "--verify", branch)
	branchExists := checkCmd.Run() == nil

	var cmd *exec.Cmd
	if branchExists {
		// Check out existing branch
		cmd = exec.Command("git", "-C", absRepoPath, "worktree", "add", absWorktreePath, branch)
	} else {
		// Create new branch
		cmd = exec.Command("git", "-C", absRepoPath, "worktree", "add", "-b", branch, absWorktreePath)
	}

	var stderr bytes.Buffer
	var stdout bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		// Combine stdout and stderr for better error context
		errMsg := stderr.String()
		if outMsg := stdout.String(); outMsg != "" {
			errMsg = outMsg + "\n" + errMsg
		}
		return fmt.Errorf("git command failed: %s", strings.TrimSpace(errMsg))
	}

	// Validate that the worktree was actually created
	if _, err := os.Stat(absWorktreePath); os.IsNotExist(err) {
		return fmt.Errorf("worktree directory was not created at %s", absWorktreePath)
	}

	return nil
}

// RemoveWorktree removes a git worktree
func RemoveWorktree(repoPath, worktreePath string) error {
	// Convert to absolute paths
	absRepoPath, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for repo: %w", err)
	}

	absWorktreePath, err := filepath.Abs(worktreePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for worktree: %w", err)
	}

	// Remove the worktree
	cmd := exec.Command("git", "-C", absRepoPath, "worktree", "remove", absWorktreePath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove worktree: %s", stderr.String())
	}

	return nil
}

// PruneWorktrees prunes stale worktree metadata
func PruneWorktrees(repoPath string) error {
	absRepoPath, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for repo: %w", err)
	}

	cmd := exec.Command("git", "-C", absRepoPath, "worktree", "prune")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to prune worktrees: %w", err)
	}

	return nil
}

// ListWorktrees lists all worktrees for a repository
func ListWorktrees(repoPath string) ([]WorktreeInfo, error) {
	absRepoPath, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for repo: %w", err)
	}

	cmd := exec.Command("git", "-C", absRepoPath, "worktree", "list", "--porcelain")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	// Parse the output
	worktrees := []WorktreeInfo{}
	lines := strings.Split(stdout.String(), "\n")

	var currentWorktree WorktreeInfo
	for _, line := range lines {
		if strings.HasPrefix(line, "worktree ") {
			if currentWorktree.Path != "" {
				worktrees = append(worktrees, currentWorktree)
			}
			currentWorktree = WorktreeInfo{
				Path:  strings.TrimPrefix(line, "worktree "),
				Clean: true,
			}
		} else if strings.HasPrefix(line, "branch ") {
			currentWorktree.Branch = strings.TrimPrefix(line, "branch refs/heads/")
		}
	}

	if currentWorktree.Path != "" {
		worktrees = append(worktrees, currentWorktree)
	}

	return worktrees, nil
}

// GetWorktreeBranch returns the branch name for a worktree
func GetWorktreeBranch(worktreePath string) (string, error) {
	absWorktreePath, err := filepath.Abs(worktreePath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for worktree: %w", err)
	}

	cmd := exec.Command("git", "-C", absWorktreePath, "rev-parse", "--abbrev-ref", "HEAD")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get branch: %w", err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

// HasUncommittedChanges checks if a worktree has uncommitted changes
func HasUncommittedChanges(worktreePath string) (bool, error) {
	absWorktreePath, err := filepath.Abs(worktreePath)
	if err != nil {
		return false, fmt.Errorf("failed to get absolute path for worktree: %w", err)
	}

	cmd := exec.Command("git", "-C", absWorktreePath, "status", "--porcelain")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("failed to check git status: %w", err)
	}

	return strings.TrimSpace(stdout.String()) != "", nil
}

// GetUncommittedChangesCount returns the number of uncommitted changes
func GetUncommittedChangesCount(worktreePath string) (int, error) {
	absWorktreePath, err := filepath.Abs(worktreePath)
	if err != nil {
		return 0, fmt.Errorf("failed to get absolute path for worktree: %w", err)
	}

	cmd := exec.Command("git", "-C", absWorktreePath, "status", "--porcelain")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("failed to check git status: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return 0, nil
	}

	return len(lines), nil
}
