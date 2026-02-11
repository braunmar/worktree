package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"worktree/pkg/config"
	"worktree/pkg/docker"
	"worktree/pkg/git"
	"worktree/pkg/registry"
	"worktree/pkg/ui"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all worktrees",
	Long: `List all existing worktrees with their status.

Shows:
- Feature name
- Branch name
- Backend status (clean/modified)
- Frontend status (clean/modified)
- Running status
- Port mapping

Example:
  worktree list`,
	Args: cobra.NoArgs,
	Run:  runList,
}

func runList(cmd *cobra.Command, args []string) {
	// Get configuration
	cfg, err := config.New()
	checkError(err)

	// Display header
	fmt.Printf("%s Worktree Features:\n\n", "📋")

	// Check if worktrees directory exists
	if _, err := os.Stat(cfg.WorktreeDir); os.IsNotExist(err) {
		ui.Info("No worktrees found")
		fmt.Println("\nCreate one with:")
		ui.PrintCommand("  worktree new-feature <branch> [preset]")
		return
	}

	// Load registry
	reg, err := registry.Load(cfg.WorktreeDir)
	if err != nil {
		checkError(fmt.Errorf("failed to load registry: %w", err))
	}

	worktrees := reg.List()

	if len(worktrees) == 0 {
		ui.Info("No worktrees found")
		fmt.Println("\nCreate one with:")
		ui.PrintCommand("  worktree new-feature <branch> [preset]")
		return
	}

	// Sort by creation time (most recent first)
	sort.Slice(worktrees, func(i, j int) bool {
		return worktrees[i].Created.After(worktrees[j].Created)
	})

	for _, wt := range worktrees {
		featureName := wt.Normalized

		// Check if worktree directory still exists
		if !cfg.WorktreeExists(featureName) {
			fmt.Printf("⚠️  %s: directory not found (orphaned registry entry)\n\n", featureName)
			continue
		}

		// Get worktree information
		backendWorktree := cfg.WorktreeBackendPath(featureName)
		frontendWorktree := cfg.WorktreeFrontendPath(featureName)

		backendBranch := wt.Branch
		if branchName, err := git.GetWorktreeBranch(backendWorktree); err == nil {
			backendBranch = branchName
		}

		backendChanges, _ := git.HasUncommittedChanges(backendWorktree)
		backendCount, _ := git.GetUncommittedChangesCount(backendWorktree)

		frontendChanges, _ := git.HasUncommittedChanges(frontendWorktree)
		frontendCount, _ := git.GetUncommittedChangesCount(frontendWorktree)

		// Check if feature is running
		running := docker.IsFeatureRunning(featureName)

		// Display feature information
		fmt.Printf("Feature: %s\n", featureName)
		fmt.Printf("  Path:     %s\n", filepath.Join("worktrees", featureName))
		fmt.Printf("  Branch:   %s\n", backendBranch)
		fmt.Printf("  Created:  %s\n", wt.Created.Format("2006-01-02 15:04"))

		// Backend status
		if backendChanges {
			fmt.Printf("  Backend:  ⚠️  modified (%d uncommitted changes)\n", backendCount)
		} else {
			fmt.Printf("  Backend:  ✅ clean\n")
		}

		// Frontend status
		if frontendChanges {
			fmt.Printf("  Frontend: ⚠️  modified (%d uncommitted changes)\n", frontendCount)
		} else {
			fmt.Printf("  Frontend: ✅ clean\n")
		}

		// Running status
		if running {
			fmt.Printf("  Status:   🟢 Running\n")
		} else {
			fmt.Printf("  Status:   ⚪ Stopped\n")
		}

		// Ports from registry
		if len(wt.Ports) > 0 {
			fmt.Print("  Ports:    ")
			portStrs := []string{}

			// Show main ports first
			mainPorts := []string{"FE_PORT", "BE_PORT", "POSTGRES_PORT"}
			for _, key := range mainPorts {
				if port, ok := wt.Ports[key]; ok {
					portStrs = append(portStrs, fmt.Sprintf("%s=%d", key, port))
				}
			}

			// Show other ports
			for key, port := range wt.Ports {
				isMain := false
				for _, mainKey := range mainPorts {
					if key == mainKey {
						isMain = true
						break
					}
				}
				if !isMain {
					portStrs = append(portStrs, fmt.Sprintf("%s=%d", key, port))
				}
			}

			if len(portStrs) > 0 {
				fmt.Printf("%s\n", portStrs[0])
				for i := 1; i < len(portStrs) && i < 3; i++ {
					fmt.Printf("            %s\n", portStrs[i])
				}
				if len(portStrs) > 3 {
					fmt.Printf("            ... (%d more)\n", len(portStrs)-3)
				}
			}
		}
		fmt.Println()
	}

	// Show summary
	fmt.Printf("Total: %d worktree(s)\n", len(worktrees))
}
