package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"worktree/pkg/config"
	"worktree/pkg/docker"
	"worktree/pkg/git"
	"worktree/pkg/registry"
	"worktree/pkg/ui"

	"github.com/spf13/cobra"
)

var (
	forceRemove bool
)

var removeCmd = &cobra.Command{
	Use:   "remove <feature-name>",
	Short: "Remove a feature worktree",
	Long: `Remove git worktrees for a feature and update the registry.

This command performs safety checks:
- Warns if feature is still running
- Warns if there are uncommitted changes
- Prompts for confirmation (unless --force is used)
- Removes from registry

Example:
  worktree remove feature-user-auth
  worktree remove feature-reports --force`,
	Args: cobra.ExactArgs(1),
	Run:  runRemove,
}

func init() {
	removeCmd.Flags().BoolVarP(&forceRemove, "force", "f", false, "skip confirmation prompts")
}

func runRemove(cmd *cobra.Command, args []string) {
	featureName := args[0]

	// Get configuration
	cfg, err := config.New()
	checkError(err)

	// Load registry
	reg, err := registry.Load(cfg.WorktreeDir)
	checkError(err)

	// Get worktree from registry
	wt, exists := reg.Get(featureName)
	if !exists {
		ui.Error(fmt.Sprintf("Feature worktree '%s' not found in registry", featureName))
		fmt.Println("\nAvailable features:")
		for _, w := range reg.List() {
			fmt.Printf("  - %s\n", w.Normalized)
		}
		os.Exit(1)
	}

	// Check if worktree directory exists
	if !cfg.WorktreeExists(featureName) {
		ui.Warning(fmt.Sprintf("Feature directory not found: worktrees/%s", featureName))
		ui.Info("Removing from registry only...")

		if err := reg.Remove(featureName); err != nil {
			ui.Error(fmt.Sprintf("Failed to remove from registry: %v", err))
			os.Exit(1)
		}
		if err := reg.Save(); err != nil {
			ui.Error(fmt.Sprintf("Failed to save registry: %v", err))
			os.Exit(1)
		}

		ui.Success("Removed from registry")
		os.Exit(0)
	}

	// Get worktree paths
	backendWorktree := cfg.WorktreeBackendPath(featureName)
	frontendWorktree := cfg.WorktreeFrontendPath(featureName)

	backendBranch, _ := git.GetWorktreeBranch(backendWorktree)
	frontendBranch, _ := git.GetWorktreeBranch(frontendWorktree)

	// Display header
	ui.Warning(fmt.Sprintf("Removing Feature: %s", featureName))
	ui.NewLine()
	ui.PrintStatusLine("Branch", wt.Branch)
	ui.PrintStatusLine("Backend", fmt.Sprintf("worktrees/%s/backend (branch: %s)", featureName, backendBranch))
	ui.PrintStatusLine("Frontend", fmt.Sprintf("worktrees/%s/frontend (branch: %s)", featureName, frontendBranch))
	ui.NewLine()

	// Check if feature is running
	if docker.IsFeatureRunning(featureName) && !forceRemove {
		ui.Warning(fmt.Sprintf("Feature '%s' is currently running. Stop it first? [y/N]: ", featureName))
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response == "y" || response == "yes" {
			ui.Loading(fmt.Sprintf("Stopping feature %s...", featureName))
			featurePath := cfg.WorktreeFeaturePath(featureName)
			if err := docker.StopFeature(featureName, featurePath); err != nil {
				ui.Error(fmt.Sprintf("Failed to stop feature: %v", err))
				os.Exit(1)
			}
			ui.Success("Feature stopped")
			ui.NewLine()
		}
	}

	// Check for uncommitted changes
	backendChanges, _ := git.HasUncommittedChanges(backendWorktree)
	frontendChanges, _ := git.HasUncommittedChanges(frontendWorktree)

	if (backendChanges || frontendChanges) && !forceRemove {
		ui.Warning("Uncommitted changes detected:")
		if backendChanges {
			count, _ := git.GetUncommittedChangesCount(backendWorktree)
			ui.PrintStatusLine("Backend", fmt.Sprintf("%d uncommitted changes", count))
		}
		if frontendChanges {
			count, _ := git.GetUncommittedChangesCount(frontendWorktree)
			ui.PrintStatusLine("Frontend", fmt.Sprintf("%d uncommitted changes", count))
		}
		ui.NewLine()
	}

	// Confirm removal
	if !forceRemove {
		fmt.Print("Are you sure you want to remove this worktree? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response != "y" && response != "yes" {
			ui.Info("Removal cancelled")
			os.Exit(0)
		}
	}

	// Remove worktrees
	ui.NewLine()
	ui.Loading("Removing worktrees...")

	// Remove backend worktree
	if err := git.RemoveWorktree(cfg.BackendDir, backendWorktree); err != nil {
		ui.Error(fmt.Sprintf("Failed to remove backend worktree: %v", err))
	} else {
		ui.CheckMark("Removed backend worktree")
	}

	// Remove frontend worktree
	if err := git.RemoveWorktree(cfg.FrontendDir, frontendWorktree); err != nil {
		ui.Error(fmt.Sprintf("Failed to remove frontend worktree: %v", err))
	} else {
		ui.CheckMark("Removed frontend worktree")
	}

	// Prune worktree metadata
	git.PruneWorktrees(cfg.BackendDir)
	git.PruneWorktrees(cfg.FrontendDir)

	// Remove feature directory
	featureDir := cfg.WorktreeFeaturePath(featureName)
	if err := os.RemoveAll(featureDir); err != nil {
		ui.Warning(fmt.Sprintf("Failed to remove feature directory: %v", err))
	} else {
		ui.CheckMark("Removed feature directory")
	}

	// Remove from registry
	if err := reg.Remove(featureName); err != nil {
		ui.Warning(fmt.Sprintf("Failed to remove from registry: %v", err))
	} else {
		ui.CheckMark("Removed from registry")
	}

	// Save registry
	if err := reg.Save(); err != nil {
		ui.Warning(fmt.Sprintf("Failed to save registry: %v", err))
	}

	ui.Success("Cleanup complete")
	ui.NewLine()
}
