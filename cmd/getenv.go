package cmd

import (
	"fmt"
	"os"

	"github.com/braunmar/worktree/pkg/config"
	"github.com/braunmar/worktree/pkg/registry"
	"github.com/braunmar/worktree/pkg/ui"

	"github.com/spf13/cobra"
)

var getEnvCmd = &cobra.Command{
	Use:   "get-env [feature] <VAR_NAME>",
	Short: "Get an environment variable value for a worktree feature",
	Long: `Get an environment variable value for a worktree feature.

With explicit feature name (reads from registry, works from anywhere):
  worktree get-env feature-x FE_PORT

Auto-detected from within a feature directory (reads from .worktree-env):
  cd worktrees/feature-x && worktree get-env FE_PORT

Output is the raw value only, suitable for scripting:
  PORT=$(worktree get-env feature-x FE_PORT)`,
	Args: cobra.RangeArgs(1, 2),
	Run:  runGetEnv,
}

func runGetEnv(cmd *cobra.Command, args []string) {
	if len(args) == 2 {
		// Two-arg mode: read from registry
		featureName := registry.NormalizeBranchName(args[0])
		varName := args[1]

		cfg, err := config.New()
		checkError(err)

		workCfg, err := config.LoadWorktreeConfig(cfg.ProjectRoot)
		checkError(err)

		reg, err := registry.Load(cfg.WorktreeDir, workCfg)
		checkError(err)

		wt, exists := reg.Get(featureName)
		if !exists {
			ui.Error(fmt.Sprintf("Feature '%s' not found in registry", featureName))
			os.Exit(1)
		}

		val, ok := wt.ComputedVars[varName]
		if !ok {
			ui.Error(fmt.Sprintf("Variable '%s' not found for feature '%s'", varName, featureName))
			os.Exit(1)
		}

		fmt.Println(val)

	} else {
		// One-arg mode: auto-detect feature from current directory, read from .worktree-env
		varName := args[0]

		instance, err := config.DetectInstance()
		if err != nil {
			ui.Error("Not in a worktree directory and no feature name provided")
			ui.Info("Usage: worktree get-env <feature> <VAR_NAME>")
			ui.Info("   or: cd to a worktree directory and run: worktree get-env <VAR_NAME>")
			os.Exit(1)
		}

		vars, err := config.ReadEnvFile(instance.WorktreeRoot)
		if err != nil {
			ui.Error(fmt.Sprintf("Failed to read .worktree-env: %v", err))
			os.Exit(1)
		}

		val, ok := vars[varName]
		if !ok {
			ui.Error(fmt.Sprintf("Variable '%s' not found in .worktree-env", varName))
			os.Exit(1)
		}

		fmt.Println(val)
	}
}
