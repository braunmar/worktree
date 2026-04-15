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

var pullCmd = &cobra.Command{
	Use:   "pull <feature-name>",
	Short: "Pull latest changes into feature branches",
	Long: `Pull the latest remote changes into the feature branch for all projects.

This command:
1. Checks for uncommitted changes in all project worktrees
2. Runs git pull origin <branch> in each project worktree

The feature name is automatically normalized, so you can use either:
- The normalized feature name: feature-user-auth
- The original branch name: feature/user-auth

Examples:
  worktree pull feature-user-auth
  worktree pull feature/user-auth`,
	Args: cobra.ExactArgs(1),
	Run:  runPull,
}

func runPull(cmd *cobra.Command, args []string) {
	featureName := registry.NormalizeBranchName(args[0])

	cfg, err := config.New()
	checkError(err)

	workCfg, err := config.LoadWorktreeConfig(cfg.ProjectRoot)
	checkError(err)

	reg, err := registry.Load(cfg.WorktreeDir, workCfg)
	checkError(err)

	wt, exists := reg.Get(featureName)
	if !exists {
		ui.Error(fmt.Sprintf("Feature worktree '%s' not found in registry", featureName))
		fmt.Println("\nAvailable features:")
		for _, w := range reg.List() {
			fmt.Printf("  - %s\n", w.Normalized)
		}
		os.Exit(1)
	}

	if !cfg.WorktreeExists(featureName) {
		ui.Error(fmt.Sprintf("Feature directory not found: worktrees/%s", featureName))
		os.Exit(1)
	}

	ui.PrintHeader(fmt.Sprintf("Pulling Feature: %s", featureName))
	ui.NewLine()
	ui.PrintStatusLine("Branch", wt.Branch)
	ui.NewLine()

	projects := wt.Projects
	if len(projects) == 0 {
		ui.Error("No projects found in worktree")
		os.Exit(1)
	}

	featureDir := cfg.WorktreeFeaturePath(featureName)

	// Pre-check: uncommitted changes
	hasUncommittedChanges := false
	for _, projectName := range projects {
		project, exists := workCfg.Projects[projectName]
		if !exists {
			continue
		}

		worktreePath := featureDir + "/" + project.Dir

		if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
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
		ui.Error("Cannot pull with uncommitted changes")
		ui.NewLine()
		ui.Info("💡 Commit or stash your changes before pulling")
		os.Exit(1)
	}

	ui.Section("Pulling branches...")
	for _, projectName := range projects {
		project, exists := workCfg.Projects[projectName]
		if !exists {
			ui.Warning(fmt.Sprintf("Project '%s' not found in configuration, skipping", projectName))
			continue
		}

		worktreePath := featureDir + "/" + project.Dir

		if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
			ui.Warning(fmt.Sprintf("Worktree for %s does not exist, skipping", projectName))
			continue
		}

		ui.Info(fmt.Sprintf("📥 Pulling %s...", projectName))
		pullExec := exec.Command("git", "pull", "origin", wt.Branch)
		pullExec.Dir = worktreePath
		pullExec.Stdout = os.Stdout
		pullExec.Stderr = os.Stderr
		if err := pullExec.Run(); err != nil {
			ui.Error(fmt.Sprintf("%s pull failed: conflict or remote error", projectName))
			ui.NewLine()
			ui.Info("💡 Resolve conflicts in:")
			ui.Info(fmt.Sprintf("   %s", worktreePath))
			ui.Info("💡 Then run: git -C " + worktreePath + " pull --continue")
			os.Exit(1)
		}
		ui.CheckMark(fmt.Sprintf("%s updated", projectName))
	}

	ui.NewLine()
	ui.Success("✨ Pull completed successfully!")
	ui.NewLine()
}
