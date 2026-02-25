package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/braunmar/worktree/pkg/config"
	"github.com/braunmar/worktree/pkg/docker"
	"github.com/braunmar/worktree/pkg/git"
	"github.com/braunmar/worktree/pkg/registry"
	"github.com/braunmar/worktree/pkg/ui"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all worktrees",
	Long: `List all existing worktrees with their status.

Shows:
- Feature name
- Branch name
- Project status for each configured project (clean/modified)
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

	// Load worktree configuration
	workCfg, err := config.LoadWorktreeConfig(cfg.ProjectRoot)
	checkError(err)

	// Load registry
	reg, err := registry.Load(cfg.WorktreeDir, workCfg)
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

		featureDir := cfg.WorktreeFeaturePath(featureName)

		// Get branch name from first project
		displayBranch := wt.Branch
		if len(wt.Projects) > 0 {
			firstProject := workCfg.Projects[wt.Projects[0]]
			firstWorktreePath := featureDir + "/" + firstProject.Dir
			if branchName, err := git.GetWorktreeBranch(firstWorktreePath); err == nil {
				displayBranch = branchName
			}
		}

		// Check if feature is running
		running := docker.IsFeatureRunning(workCfg.ProjectName, featureName)

		// Display feature information
		fmt.Printf("Feature: %s\n", featureName)
		fmt.Printf("  Path:     %s\n", filepath.Join("worktrees", featureName))
		fmt.Printf("  Branch:   %s\n", displayBranch)
		fmt.Printf("  Created:  %s\n", wt.Created.Format("2006-01-02 15:04"))

		// Show status for each project
		for _, projectName := range wt.Projects {
			project, exists := workCfg.Projects[projectName]
			if !exists {
				continue
			}

			worktreePath := featureDir + "/" + project.Dir

			// Check if worktree exists
			if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
				fmt.Printf("  %s: ⚠️  worktree not found\n", projectName)
				continue
			}

			changes, _ := git.HasUncommittedChanges(worktreePath)
			count, _ := git.GetUncommittedChangesCount(worktreePath)

			if changes {
				fmt.Printf("  %s: ⚠️  modified (%d uncommitted changes)\n", projectName, count)
			} else {
				fmt.Printf("  %s: ✅ clean\n", projectName)
			}
		}

		// Running status
		if running {
			fmt.Printf("  Status:   🟢 Running\n")
		} else {
			fmt.Printf("  Status:   ⚪ Stopped\n")
		}

		// Show allocated port numbers sorted alphabetically
		if len(wt.Ports) > 0 {
			portKeys := make([]string, 0, len(wt.Ports))
			for k := range wt.Ports {
				portKeys = append(portKeys, k)
			}
			sort.Strings(portKeys)

			fmt.Printf("  Ports:    %s=%d\n", portKeys[0], wt.Ports[portKeys[0]])
			for i := 1; i < len(portKeys); i++ {
				fmt.Printf("            %s=%d\n", portKeys[i], wt.Ports[portKeys[i]])
			}
		}
		fmt.Println()
	}

	// Show summary
	fmt.Printf("Total: %d worktree(s)\n", len(worktrees))
}
