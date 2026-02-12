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
	Use:   "remove <feature-name-or-branch>",
	Short: "Remove a feature worktree",
	Long: `Remove git worktrees for a feature and update the registry.

The feature name/branch is automatically normalized, so you can use either:
- The normalized feature name: feature-user-auth
- The original branch name: feature/user-auth

This command performs safety checks:
- Warns if feature is still running
- Warns if there are uncommitted changes
- Prompts for confirmation (unless --force is used)
- Removes from registry

Examples:
  worktree remove feature-user-auth           # Using normalized name
  worktree remove feature/user-auth           # Using branch name
  worktree remove feature/reports --force`,
	Args: cobra.ExactArgs(1),
	Run:  runRemove,
}

func init() {
	removeCmd.Flags().BoolVarP(&forceRemove, "force", "f", false, "skip confirmation prompts")
}

func runRemove(cmd *cobra.Command, args []string) {
	input := args[0]
	verbose, _ := cmd.Flags().GetBool("verbose")

	// Normalize the input to match behavior of new-feature command
	// This allows users to use either the normalized name or the original branch name
	featureName := registry.NormalizeBranchName(input)
	if verbose {
		ui.Info(fmt.Sprintf("Normalized input '%s' to feature name '%s'", input, featureName))
	}

	// Get configuration
	cfg, err := config.New()
	checkError(err)
	if verbose {
		ui.Info(fmt.Sprintf("Loaded configuration from: %s", cfg.ProjectRoot))
	}

	// Load worktree configuration
	workCfg, err := config.LoadWorktreeConfig(cfg.ProjectRoot)
	checkError(err)
	if verbose {
		ui.Info(fmt.Sprintf("Loaded worktree configuration with %d projects", len(workCfg.Projects)))
	}

	// Load registry
	reg, err := registry.Load(cfg.WorktreeDir, workCfg)
	checkError(err)
	if verbose {
		ui.Info(fmt.Sprintf("Loaded registry with %d worktrees", len(reg.Worktrees)))
	}

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

	// Display header
	ui.Warning(fmt.Sprintf("Removing Feature: %s", featureName))
	ui.NewLine()
	ui.PrintStatusLine("Branch", wt.Branch)

	// Get list of projects from the worktree
	projects := wt.Projects
	if len(projects) == 0 {
		ui.Error("No projects found in worktree")
		os.Exit(1)
	}

	featureDir := cfg.WorktreeFeaturePath(featureName)

	// Display each project's worktree information
	for _, projectName := range projects {
		project, exists := workCfg.Projects[projectName]
		if !exists {
			ui.Warning(fmt.Sprintf("Project '%s' not found in configuration", projectName))
			continue
		}

		worktreePath := featureDir + "/" + project.Dir
		branch, _ := git.GetWorktreeBranch(worktreePath)
		ui.PrintStatusLine(projectName, fmt.Sprintf("worktrees/%s/%s (branch: %s)", featureName, project.Dir, branch))
	}

	// Display compose project names (supports both old and new format)
	if len(wt.ComposeProjects) > 0 {
		for project, composeName := range wt.ComposeProjects {
			ui.PrintStatusLine(fmt.Sprintf("Compose (%s)", project), composeName)
		}
	} else if wt.ComposeProject != "" {
		ui.PrintStatusLine("Compose Project", wt.ComposeProject)
	}
	ui.NewLine()

	// Always stop services before removing (prevents stale containers)
	ui.Info("Stopping services (if running)...")
	featurePath := cfg.WorktreeFeaturePath(featureName)
	if err := docker.StopFeature(workCfg.ProjectName, featureName, featurePath); err != nil {
		ui.Warning(fmt.Sprintf("Failed to stop services: %v", err))
		ui.Info("Continuing with removal...")
	} else {
		ui.CheckMark("Services stopped")
	}
	ui.NewLine()

	// Check for uncommitted changes in all projects
	hasUncommittedChanges := false
	uncommittedProjects := []string{}

	for _, projectName := range projects {
		project, exists := workCfg.Projects[projectName]
		if !exists {
			continue
		}

		worktreePath := featureDir + "/" + project.Dir

		// Check if worktree exists
		if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
			continue
		}

		changes, _ := git.HasUncommittedChanges(worktreePath)
		if changes {
			hasUncommittedChanges = true
			count, _ := git.GetUncommittedChangesCount(worktreePath)
			uncommittedProjects = append(uncommittedProjects, fmt.Sprintf("%s: %d uncommitted changes", projectName, count))
		}
	}

	if hasUncommittedChanges && !forceRemove {
		ui.Warning("Uncommitted changes detected:")
		for _, msg := range uncommittedProjects {
			ui.PrintStatusLine("", msg)
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

	// Remove worktrees for all projects
	ui.NewLine()
	ui.Loading("Removing worktrees...")

	for _, projectName := range projects {
		project, exists := workCfg.Projects[projectName]
		if !exists {
			continue
		}

		projectDir := cfg.ProjectRoot + "/" + project.Dir
		worktreePath := featureDir + "/" + project.Dir

		// Check if worktree exists before trying to remove
		if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
			ui.Warning(fmt.Sprintf("Worktree for %s does not exist, skipping", projectName))
			continue
		}

		if err := git.RemoveWorktree(projectDir, worktreePath); err != nil {
			ui.Error(fmt.Sprintf("Failed to remove %s worktree: %v", projectName, err))
		} else {
			ui.CheckMark(fmt.Sprintf("Removed %s worktree", projectName))
		}
	}

	// Prune worktree metadata for all projects
	for _, projectName := range projects {
		project, exists := workCfg.Projects[projectName]
		if !exists {
			continue
		}

		projectDir := cfg.ProjectRoot + "/" + project.Dir
		git.PruneWorktrees(projectDir)
	}

	// Remove feature directory
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
