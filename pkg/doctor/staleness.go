package doctor

import (
	"bytes"
	"github.com/braunmar/worktree/pkg/config"
	"github.com/braunmar/worktree/pkg/docker"
	"github.com/braunmar/worktree/pkg/registry"
	"os"
	"os/exec"
	"strings"
	"time"
)

// CheckStaleness checks if a worktree is stale based on multiple criteria
func CheckStaleness(cfg *config.Config, wt *registry.Worktree, projectName string) StalenessReport {
	report := StalenessReport{
		Feature: wt.Normalized,
		Branch:  wt.Branch,
	}

	// Check if directory exists
	if !cfg.WorktreeExists(wt.Normalized) {
		return report // Can't check staleness if directory doesn't exist
	}

	// Check last modified time
	backendPath := cfg.WorktreeBackendPath(wt.Normalized)
	if info, err := os.Stat(backendPath); err == nil {
		report.LastModified = info.ModTime()
		report.DaysSinceModified = int(time.Since(info.ModTime()).Hours() / 24)
		if report.DaysSinceModified >= 7 {
			report.Score++
		}
	}

	// Check if branch merged to main
	mergeCheckCmd := exec.Command("git", "-C", backendPath, "branch", "--merged", "origin/main", "--format=%(refname:short)")
	var mergeOut bytes.Buffer
	mergeCheckCmd.Stdout = &mergeOut
	if err := mergeCheckCmd.Run(); err == nil {
		branches := strings.Split(strings.TrimSpace(mergeOut.String()), "\n")
		for _, b := range branches {
			// Check if this branch or its corresponding branch name matches
			if strings.Contains(b, wt.Branch) || b == wt.Branch {
				report.BranchMerged = true
				report.Score++

				// Get merge date (when was the last commit)
				mergeDate := exec.Command("git", "-C", backendPath, "log", "-1", "--format=%ar", wt.Branch)
				var dateOut bytes.Buffer
				mergeDate.Stdout = &dateOut
				if err := mergeDate.Run(); err == nil {
					report.MergedDate = strings.TrimSpace(dateOut.String())
				}
				break
			}
		}
	}

	// Check if containers running
	if !docker.IsFeatureRunning(projectName, wt.Normalized) {
		report.NoContainers = true
		report.Score++
	}

	return report
}
