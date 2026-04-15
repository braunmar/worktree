package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/braunmar/worktree/pkg/config"
	"github.com/braunmar/worktree/pkg/registry"
	"github.com/braunmar/worktree/pkg/ui"

	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push <feature-name>",
	Short: "Push feature branches to remote",
	Long: `Push the feature branch to origin for all projects in the worktree.

This command:
1. Pushes the feature branch to origin in all project worktrees
2. Reports success or failure per project

The feature name is automatically normalized, so you can use either:
- The normalized feature name: feature-user-auth
- The original branch name: feature/user-auth

Examples:
  worktree push feature-user-auth
  worktree push feature/user-auth`,
	Args: cobra.ExactArgs(1),
	Run:  runPush,
}

func runPush(cmd *cobra.Command, args []string) {
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

	ui.PrintHeader(fmt.Sprintf("Pushing Feature: %s", featureName))
	ui.NewLine()
	ui.PrintStatusLine("Branch", wt.Branch)
	ui.NewLine()

	projects := wt.Projects
	if len(projects) == 0 {
		ui.Error("No projects found in worktree")
		os.Exit(1)
	}

	featureDir := cfg.WorktreeFeaturePath(featureName)

	ui.Section("Pushing branches...")
	allOk := true
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

		ui.Info(fmt.Sprintf("📤 Pushing %s...", projectName))
		pushCmd := exec.Command("git", "push", "origin", wt.Branch)
		pushCmd.Dir = worktreePath
		pushCmd.Stdout = os.Stdout
		pushCmd.Stderr = os.Stderr
		if err := pushCmd.Run(); err != nil {
			ui.CrossMark(fmt.Sprintf("%s push failed", projectName))
			allOk = false
			ui.NewLine()
			ui.Info("💡 If the remote is ahead, try: worktree pull " + featureName)
			ui.Info("💡 Or rebase first: worktree rebase " + featureName)
			continue
		}
		ui.CheckMark(fmt.Sprintf("%s pushed", projectName))
	}

	ui.NewLine()
	if allOk {
		ui.Success("✨ Push completed successfully!")
		ui.NewLine()
		ui.Info("Next steps:")
		ui.Info("  • Open a pull request from branch: " + wt.Branch)
	} else {
		ui.Error("Push completed with errors (see above)")
		os.Exit(1)
	}
	ui.NewLine()
}
