package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/braunmar/worktree/pkg/config"
	"github.com/braunmar/worktree/pkg/docker"
	"github.com/braunmar/worktree/pkg/process"
	"github.com/braunmar/worktree/pkg/registry"
	"github.com/braunmar/worktree/pkg/ui"

	"github.com/spf13/cobra"
)

// buildStopEnvList builds the environment variable list needed for stop hooks.
// Falls back to instance=0 if no ranged port is configured.
func buildStopEnvList(workCfg *config.WorktreeConfig, wt *registry.Worktree, featureName string) []string {
	instance := 0
	if instancePortName, err := workCfg.GetInstancePortName(); err == nil {
		if instancePortCfg, ok := workCfg.EnvVariables[instancePortName]; ok && instancePortCfg.Port != "" {
			if basePort, err := config.ExtractBasePort(instancePortCfg.Port); err == nil {
				if allocatedPort, ok := wt.Ports[instancePortName]; ok {
					instance = allocatedPort - basePort
				}
			}
		}
	}

	baseEnvVars := workCfg.ExportEnvVars(instance)
	baseEnvVars["FEATURE_NAME"] = featureName
	for service, port := range wt.Ports {
		baseEnvVars[service] = fmt.Sprintf("%d", port)
	}

	envList := os.Environ()
	for key, value := range baseEnvVars {
		envList = append(envList, fmt.Sprintf("%s=%s", key, value))
	}
	return envList
}

var stopCmd = &cobra.Command{
	Use:   "stop [feature-name]",
	Short: "Stop services for a feature worktree",
	Long: `Stop services for a specific feature worktree.

This command:
1. Validates the feature exists
2. Stops all running Docker containers for the feature
3. Shows status

If no feature name is provided and you're in a worktree directory,
the feature will be auto-detected from .worktree-instance.

Examples:
  worktree stop feature-user-auth    # Explicit feature name
  worktree stop                      # Auto-detect from current directory`,
	Args: cobra.MaximumNArgs(1),
	Run:  runStop,
}

func runStop(cmd *cobra.Command, args []string) {
	var featureName string
	autoDetected := false

	// Auto-detect feature name if not provided
	if len(args) == 0 {
		instance, err := config.DetectInstance()
		if err != nil {
			ui.Error("Not in a worktree directory and no feature name provided")
			ui.Info("Usage: worktree stop <feature-name>")
			ui.Info("   or: cd to a worktree directory and run: worktree stop")
			os.Exit(1)
		}
		featureName = instance.Feature
		autoDetected = true
	} else {
		featureName = args[0]
	}

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
		ui.Error(fmt.Sprintf("Feature worktree '%s' not found", featureName))
		fmt.Println("\nAvailable features:")
		for _, w := range reg.List() {
			fmt.Printf("  - %s\n", w.Normalized)
		}
		os.Exit(1)
	}

	// Display header
	ui.Warning(fmt.Sprintf("Stopping Feature: %s", featureName))
	if autoDetected {
		ui.Info("✨ Auto-detected from current directory")
	}
	ui.Info(fmt.Sprintf("Branch: %s", wt.Branch))
	ui.NewLine()

	// Get worktree path
	featurePath := cfg.WorktreeFeaturePath(featureName)

	// Build env for hooks
	envList := buildStopEnvList(workCfg, wt, featureName)

	// Stop each project according to its executor
	ui.Loading("Stopping services...")
	for _, projectName := range wt.Projects {
		project, exists := workCfg.Projects[projectName]
		if !exists {
			continue
		}

		worktreePath := featurePath + "/" + project.Dir
		composeProject := wt.GetComposeProject(projectName)
		if composeProject == "" {
			composeProject = fmt.Sprintf("%s-%s-%s", workCfg.ProjectName, featureName, projectName)
		}
		projectEnv := append(envList, fmt.Sprintf("COMPOSE_PROJECT_NAME=%s", composeProject))

		runHookCommand(fmt.Sprintf("%s: stop_pre_command", projectName), project.StopPreCommand, worktreePath, projectEnv)

		switch project.GetExecutor() {
		case "process":
			pidFile := filepath.Join(featurePath, projectName+".pid")
			if !process.IsRunning(pidFile) {
				ui.Info(fmt.Sprintf("%s is not running", projectName))
			} else if err := process.StopProcess(pidFile); err != nil {
				ui.Warning(fmt.Sprintf("Failed to stop %s: %v", projectName, err))
			}
		default: // "docker"
			if !docker.IsFeatureRunning(workCfg.ProjectName, featureName) {
				ui.Info(fmt.Sprintf("%s is not running", projectName))
			} else {
				projectInfo := map[string]string{project.Dir: composeProject}
				if err := docker.StopFeature(workCfg.ProjectName, featureName, featurePath, projectInfo); err != nil {
					ui.Warning(fmt.Sprintf("Failed to stop %s: %v", projectName, err))
				}
			}
		}

		runHookCommand(fmt.Sprintf("%s: stop_post_command", projectName), project.StopPostCommand, worktreePath, projectEnv)
	}

	ui.NewLine()
	ui.Success(fmt.Sprintf("Feature '%s' stopped", featureName))
	ui.NewLine()
}
