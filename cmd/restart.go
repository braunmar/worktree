package cmd

import (
	"fmt"
	"os"

	"github.com/braunmar/worktree/pkg/config"
	"github.com/braunmar/worktree/pkg/registry"
	"github.com/braunmar/worktree/pkg/ui"

	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart [feature-name]",
	Short: "Restart services for a feature worktree",
	Long: `Restart services for a specific feature worktree.

This command:
1. Runs restart_pre_command for each project (if configured)
2. Stops all running services (including stop_pre/post hooks)
3. Starts all services again (including start_pre/post hooks)
4. Runs restart_post_command for each project (if configured)

This is useful when:
- Configuration has changed
- You need to pick up new environment variables
- Containers are in a bad state

Examples:
  worktree restart feature-user-auth
  worktree restart                    # Auto-detect from current directory`,
	Args: cobra.MaximumNArgs(1),
	Run:  runRestart,
}

func runRestart(cmd *cobra.Command, args []string) {
	var featureName string

	// Auto-detect feature name if not provided
	if len(args) == 0 {
		instance, err := config.DetectInstance()
		if err != nil {
			ui.Error("Not in a worktree directory and no feature name provided")
			ui.Info("Usage: worktree restart <feature-name>")
			ui.Info("   or: cd to a worktree directory and run: worktree restart")
			os.Exit(1)
		}
		featureName = instance.Feature
		ui.Info("✨ Auto-detected from current directory")
	} else {
		featureName = args[0]
	}

	ui.Warning(fmt.Sprintf("Restarting Feature: %s", featureName))
	ui.NewLine()

	// Load config to run restart_pre/post hooks
	cfg, err := config.New()
	checkError(err)
	workCfg, err := config.LoadWorktreeConfig(cfg.ProjectRoot)
	checkError(err)
	reg, err := registry.Load(cfg.WorktreeDir, workCfg)
	checkError(err)

	wt, exists := reg.Get(featureName)
	if !exists {
		ui.Error(fmt.Sprintf("Feature worktree '%s' not found", featureName))
		os.Exit(1)
	}

	featureDir := cfg.WorktreeFeaturePath(featureName)
	envList := buildStopEnvList(workCfg, wt, featureName)

	// Restart pre-hooks (per project, before stop)
	for _, projectName := range wt.Projects {
		if project, ok := workCfg.Projects[projectName]; ok {
			worktreePath := featureDir + "/" + project.Dir
			composeProject := wt.GetComposeProject(projectName)
			projectEnv := append(envList, fmt.Sprintf("COMPOSE_PROJECT_NAME=%s", composeProject))
			runHookCommand(fmt.Sprintf("%s: restart_pre_command", projectName), project.RestartPreCommand, worktreePath, projectEnv)
		}
	}

	// Stop (includes stop_pre/post hooks)
	ui.Info("Stopping services...")
	runStop(cmd, []string{featureName})

	ui.NewLine()

	// Start (includes start_pre/post hooks)
	ui.Info("Starting services...")
	runStart(cmd, []string{featureName})

	ui.NewLine()

	// Restart post-hooks (per project, after start)
	for _, projectName := range wt.Projects {
		if project, ok := workCfg.Projects[projectName]; ok {
			worktreePath := featureDir + "/" + project.Dir
			composeProject := wt.GetComposeProject(projectName)
			projectEnv := append(envList, fmt.Sprintf("COMPOSE_PROJECT_NAME=%s", composeProject))
			runHookCommand(fmt.Sprintf("%s: restart_post_command", projectName), project.RestartPostCommand, worktreePath, projectEnv)
		}
	}

	ui.Success(fmt.Sprintf("Feature '%s' restarted", featureName))
	ui.NewLine()
}

func init() {
	rootCmd.AddCommand(restartCmd)
}
