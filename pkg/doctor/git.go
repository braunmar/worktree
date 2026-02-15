package doctor

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"worktree/pkg/config"
	"worktree/pkg/git"
	"worktree/pkg/registry"
)

// CheckGitStatus checks git status for a worktree
func CheckGitStatus(cfg *config.Config, wt *registry.Worktree, fetch bool) GitStatusReport {
	report := GitStatusReport{
		Feature:  wt.Normalized,
		Branch:   wt.Branch,
		YoloMode: wt.YoloMode,
	}

	backendPath := cfg.WorktreeBackendPath(wt.Normalized)

	// Check if directory exists
	if !cfg.WorktreeExists(wt.Normalized) {
		report.Error = "Directory not found"
		return report
	}

	// Get current branch
	branch, err := git.GetWorktreeBranch(backendPath)
	if err != nil {
		report.Error = fmt.Sprintf("Failed to get branch: %v", err)
		return report
	}
	report.BranchExists = true
	report.Branch = branch

	// Check uncommitted changes
	count, err := git.GetUncommittedChangesCount(backendPath)
	if err == nil {
		report.UncommittedCount = count
	}

	// Fetch if requested
	if fetch {
		fetchCmd := exec.Command("git", "-C", backendPath, "fetch", "origin")
		fetchCmd.Run() // Ignore errors (might be offline)
	}

	// Check how far behind origin/main
	behindCmd := exec.Command("git", "-C", backendPath, "rev-list", "--count", fmt.Sprintf("%s..origin/main", branch))
	var behindOut bytes.Buffer
	behindCmd.Stdout = &behindOut
	if err := behindCmd.Run(); err == nil {
		if behind, err := strconv.Atoi(strings.TrimSpace(behindOut.String())); err == nil {
			report.BehindMain = behind
		}
	}

	// Check how far ahead of origin
	aheadCmd := exec.Command("git", "-C", backendPath, "rev-list", "--count", fmt.Sprintf("origin/%s..%s", branch, branch))
	var aheadOut bytes.Buffer
	aheadCmd.Stdout = &aheadOut
	if err := aheadCmd.Run(); err == nil {
		if ahead, err := strconv.Atoi(strings.TrimSpace(aheadOut.String())); err == nil {
			report.AheadOrigin = ahead
		}
	}

	return report
}
