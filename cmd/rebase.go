package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"worktree/pkg/config"
	"worktree/pkg/git"
	"worktree/pkg/registry"
	"worktree/pkg/ui"

	"github.com/spf13/cobra"
)

var rebaseCmd = &cobra.Command{
	Use:   "rebase <feature-name>",
	Short: "Update main and rebase feature branch",
	Long: `Update main branch from origin and rebase the feature branch on top of it.

This command:
1. Updates main branch in backend and frontend repos (git pull origin main)
2. Rebases the feature worktree branches on top of updated main
3. Shows status and any conflicts that need resolution

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

	backendWorktree := cfg.WorktreeBackendPath(featureName)
	frontendWorktree := cfg.WorktreeFrontendPath(featureName)

	// Check for uncommitted changes
	backendChanges, _ := git.HasUncommittedChanges(backendWorktree)
	frontendChanges, _ := git.HasUncommittedChanges(frontendWorktree)

	if backendChanges || frontendChanges {
		ui.Error("Cannot rebase with uncommitted changes")
		if backendChanges {
			count, _ := git.GetUncommittedChangesCount(backendWorktree)
			ui.PrintStatusLine("Backend", fmt.Sprintf("%d uncommitted changes", count))
		}
		if frontendChanges {
			count, _ := git.GetUncommittedChangesCount(frontendWorktree)
			ui.PrintStatusLine("Frontend", fmt.Sprintf("%d uncommitted changes", count))
		}
		ui.NewLine()
		ui.Info("💡 Commit or stash your changes before rebasing")
		os.Exit(1)
	}

	// Get main branch names from config
	backendMainBranch := "main" // default
	frontendMainBranch := "main" // default
	if backendCfg, exists := workCfg.Projects["backend"]; exists && backendCfg.MainBranch != "" {
		backendMainBranch = backendCfg.MainBranch
	}
	if frontendCfg, exists := workCfg.Projects["frontend"]; exists && frontendCfg.MainBranch != "" {
		frontendMainBranch = frontendCfg.MainBranch
	}

	// Step 1: Update main branch in backend
	ui.Info(fmt.Sprintf("📥 Updating backend %s branch...", backendMainBranch))
	if err := updateMainBranch(cfg.BackendDir, backendMainBranch); err != nil {
		ui.Error(fmt.Sprintf("Failed to update backend %s: %v", backendMainBranch, err))
		os.Exit(1)
	}
	ui.CheckMark(fmt.Sprintf("Backend %s updated", backendMainBranch))

	// Step 2: Update main branch in frontend
	ui.Info(fmt.Sprintf("📥 Updating frontend %s branch...", frontendMainBranch))
	if err := updateMainBranch(cfg.FrontendDir, frontendMainBranch); err != nil {
		ui.Error(fmt.Sprintf("Failed to update frontend %s: %v", frontendMainBranch, err))
		os.Exit(1)
	}
	ui.CheckMark(fmt.Sprintf("Frontend %s updated", frontendMainBranch))
	ui.NewLine()

	// Step 3: Rebase backend worktree
	ui.Info("🔄 Rebasing backend branch...")
	if err := rebaseBranch(backendWorktree, wt.Branch, backendMainBranch); err != nil {
		ui.Error(fmt.Sprintf("Backend rebase failed: %v", err))
		ui.NewLine()
		ui.Info("💡 Resolve conflicts in:")
		ui.Info(fmt.Sprintf("   %s", backendWorktree))
		ui.Info("💡 Then run: git -C " + backendWorktree + " rebase --continue")
		os.Exit(1)
	}
	ui.CheckMark("Backend rebased successfully")

	// Step 4: Rebase frontend worktree
	ui.Info("🔄 Rebasing frontend branch...")
	if err := rebaseBranch(frontendWorktree, wt.Branch, frontendMainBranch); err != nil {
		ui.Error(fmt.Sprintf("Frontend rebase failed: %v", err))
		ui.NewLine()
		ui.Info("💡 Resolve conflicts in:")
		ui.Info(fmt.Sprintf("   %s", frontendWorktree))
		ui.Info("💡 Then run: git -C " + frontendWorktree + " rebase --continue")
		os.Exit(1)
	}
	ui.CheckMark("Frontend rebased successfully")

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
