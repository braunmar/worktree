package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"github.com/braunmar/worktree/pkg/config"
	"github.com/braunmar/worktree/pkg/registry"
	"github.com/braunmar/worktree/pkg/ui"

	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff <feature-name>",
	Short: "Show diff of feature branch against main",
	Long: `Show git diff between the feature branch and main for all projects in the worktree.

This command:
1. Runs git diff <main>...<branch> in each project worktree
2. Shows the combined diff across all projects

The feature name is automatically normalized, so you can use either:
- The normalized feature name: feature-user-auth
- The original branch name: feature/user-auth

Examples:
  worktree diff feature-user-auth
  worktree diff feature/user-auth`,
	Args: cobra.ExactArgs(1),
	Run:  runDiff,
}

func runDiff(cmd *cobra.Command, args []string) {
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

	ui.PrintHeader(fmt.Sprintf("Diff: %s", featureName))
	ui.NewLine()
	ui.PrintStatusLine("Branch", wt.Branch)
	ui.NewLine()

	projects := wt.Projects
	if len(projects) == 0 {
		ui.Error("No projects found in worktree")
		os.Exit(1)
	}

	featureDir := cfg.WorktreeFeaturePath(featureName)

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

		mainBranch := "main"
		if project.MainBranch != "" {
			mainBranch = project.MainBranch
		}

		ui.Section(fmt.Sprintf("%s", projectName))

		// Check if there's any diff first
		checkCmd := exec.Command("git", "diff", mainBranch+"..."+wt.Branch, "--name-only")
		checkCmd.Dir = worktreePath
		var checkOut bytes.Buffer
		checkCmd.Stdout = &checkOut
		checkCmd.Run()

		if checkOut.Len() == 0 {
			ui.Info("No changes relative to " + mainBranch)
			ui.NewLine()
			continue
		}

		diffExec := exec.Command("git", "diff", mainBranch+"..."+wt.Branch)
		diffExec.Dir = worktreePath
		diffExec.Stdout = os.Stdout
		diffExec.Stderr = os.Stderr
		diffExec.Run()
		ui.NewLine()
	}
}
