package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/braunmar/worktree/pkg/config"
	"github.com/braunmar/worktree/pkg/git"
	"github.com/braunmar/worktree/pkg/registry"
	"github.com/braunmar/worktree/pkg/ui"

	"github.com/spf13/cobra"
)

var rebaseCmd = &cobra.Command{
	Use:   "rebase <feature-name>",
	Short: "Update main and rebase feature branch",
	Long: `Update main branch from origin and rebase the feature branch on top of it.

This command:
1. Checks for uncommitted changes in all project worktrees
2. Updates main branch in all project repositories (git pull origin main)
3. Rebases the feature worktree branches on top of updated main
4. Shows status and any conflicts that need resolution

Works with any projects defined in your .worktree.yml configuration.

The feature name is automatically normalized, so you can use either:
- The normalized feature name: feature-user-auth
- The original branch name: feature/user-auth

Examples:
  worktree rebase feature-user-auth
  worktree rebase feature/user-auth`,
	Args: cobra.ExactArgs(1),
	Run:  runRebase,
}

func runRebase(cmd *cobra.Command, args []string) {
	input := args[0]

	// Normalize the input
	featureName := registry.NormalizeBranchName(input)

	// Get configuration
	cfg, err := config.New()
	checkError(err)

	// Load worktree configuration
	workCfg, err := config.LoadWorktreeConfig(cfg.ProjectRoot)
	checkError(err)

	// Load registry
	reg, err := registry.Load(cfg.WorktreeDir, workCfg)
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
		ui.Error(fmt.Sprintf("Feature directory not found: worktrees/%s", featureName))
		os.Exit(1)
	}

	// Display header
	ui.PrintHeader(fmt.Sprintf("Rebasing Feature: %s", featureName))
	ui.NewLine()
	ui.PrintStatusLine("Branch", wt.Branch)
	ui.NewLine()

	// Get list of projects from the worktree
	projects := wt.Projects
	if len(projects) == 0 {
		ui.Error("No projects found in worktree")
		os.Exit(1)
	}

	featureDir := cfg.WorktreeFeaturePath(featureName)

	// Check for uncommitted changes in all projects
	hasUncommittedChanges := false
	for _, projectName := range projects {
		project, exists := workCfg.Projects[projectName]
		if !exists {
			ui.Warning(fmt.Sprintf("Project '%s' not found in configuration, skipping", projectName))
			continue
		}

		worktreePath := featureDir + "/" + project.Dir

		// Check if worktree exists
		if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
			ui.Warning(fmt.Sprintf("Worktree for %s does not exist: %s", projectName, worktreePath))
			continue
		}

		changes, _ := git.HasUncommittedChanges(worktreePath)
		if changes {
			hasUncommittedChanges = true
			count, _ := git.GetUncommittedChangesCount(worktreePath)
			ui.PrintStatusLine(projectName, fmt.Sprintf("%d uncommitted changes", count))
		}
	}

	if hasUncommittedChanges {
		ui.NewLine()
		ui.Error("Cannot rebase with uncommitted changes")
		ui.NewLine()
		ui.Info("💡 Commit or stash your changes before rebasing")
		os.Exit(1)
	}

	// Step 1: Update main branch in all project repositories
	ui.Section("Updating main branches...")
	for _, projectName := range projects {
		project, exists := workCfg.Projects[projectName]
		if !exists {
			continue
		}

		// Get main branch name from config (default to "main")
		mainBranch := "main"
		if project.MainBranch != "" {
			mainBranch = project.MainBranch
		}

		projectDir := cfg.ProjectRoot + "/" + project.Dir

		ui.Info(fmt.Sprintf("📥 Updating %s %s branch...", projectName, mainBranch))
		if err := updateMainBranch(projectDir, mainBranch); err != nil {
			ui.Error(fmt.Sprintf("Failed to update %s %s: %v", projectName, mainBranch, err))
			os.Exit(1)
		}
		ui.CheckMark(fmt.Sprintf("%s %s updated", projectName, mainBranch))
	}
	ui.NewLine()

	// Step 2: Rebase all project worktrees
	ui.Section("Rebasing worktrees...")
	for _, projectName := range projects {
		project, exists := workCfg.Projects[projectName]
		if !exists {
			continue
		}

		// Get main branch name from config (default to "main")
		mainBranch := "main"
		if project.MainBranch != "" {
			mainBranch = project.MainBranch
		}

		worktreePath := featureDir + "/" + project.Dir

		// Check if worktree exists
		if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
			ui.Warning(fmt.Sprintf("Worktree for %s does not exist, skipping", projectName))
			continue
		}

		ui.Info(fmt.Sprintf("🔄 Rebasing %s branch...", projectName))
		if err := rebaseBranch(worktreePath, wt.Branch, mainBranch); err != nil {
			ui.Error(fmt.Sprintf("%s rebase failed: %v", projectName, err))
			ui.NewLine()
			ui.Info("💡 Resolve conflicts in:")
			ui.Info(fmt.Sprintf("   %s", worktreePath))
			ui.Info("💡 Then run: git -C " + worktreePath + " rebase --continue")
			os.Exit(1)
		}
		ui.CheckMark(fmt.Sprintf("%s rebased successfully", projectName))
	}

	ui.NewLine()
	ui.Success("✨ Rebase completed successfully!")
	ui.NewLine()
	ui.Info("Next steps:")
	ui.Info(fmt.Sprintf("  • Test your changes: worktree start %s", featureName))
	ui.Info(fmt.Sprintf("  • Push to remote: git push --force-with-lease (if branch was previously pushed)"))
	ui.NewLine()
}

// updateMainBranch pulls latest changes from origin/main
func updateMainBranch(repoDir string, mainBranch string) error {
	// Fetch latest from origin
	fetchCmd := exec.Command("git", "fetch", "origin", mainBranch)
	fetchCmd.Dir = repoDir
	if err := fetchCmd.Run(); err != nil {
		return fmt.Errorf("git fetch failed: %w", err)
	}

	// Get current branch
	currentBranchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	currentBranchCmd.Dir = repoDir
	currentBranchOutput, err := currentBranchCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}
	currentBranch := string(currentBranchOutput)
	currentBranch = currentBranch[:len(currentBranch)-1] // Remove newline

	// If not on main, checkout main
	if currentBranch != mainBranch {
		checkoutCmd := exec.Command("git", "checkout", mainBranch)
		checkoutCmd.Dir = repoDir
		if err := checkoutCmd.Run(); err != nil {
			return fmt.Errorf("git checkout %s failed: %w", mainBranch, err)
		}
	}

	// Pull latest changes
	pullCmd := exec.Command("git", "pull", "origin", mainBranch)
	pullCmd.Dir = repoDir
	if err := pullCmd.Run(); err != nil {
		return fmt.Errorf("git pull failed: %w", err)
	}

	// Checkout back to original branch if needed
	if currentBranch != mainBranch {
		checkoutBackCmd := exec.Command("git", "checkout", currentBranch)
		checkoutBackCmd.Dir = repoDir
		if err := checkoutBackCmd.Run(); err != nil {
			// Don't fail here, just warn
			fmt.Printf("Warning: Could not checkout back to %s\n", currentBranch)
		}
	}

	return nil
}

// rebaseBranch rebases the current branch on top of main
func rebaseBranch(worktreePath string, branchName string, mainBranch string) error {
	// Ensure we're on the right branch
	checkoutCmd := exec.Command("git", "checkout", branchName)
	checkoutCmd.Dir = worktreePath
	if err := checkoutCmd.Run(); err != nil {
		return fmt.Errorf("git checkout failed: %w", err)
	}

	// Rebase on main
	rebaseCmd := exec.Command("git", "rebase", mainBranch)
	rebaseCmd.Dir = worktreePath
	rebaseCmd.Stdout = os.Stdout
	rebaseCmd.Stderr = os.Stderr
	if err := rebaseCmd.Run(); err != nil {
		return fmt.Errorf("git rebase failed (conflicts or other issues)")
	}

	return nil
}
