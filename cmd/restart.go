package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/braunmar/worktree/pkg/config"
	"github.com/braunmar/worktree/pkg/docker"
	"github.com/braunmar/worktree/pkg/process"
	"github.com/braunmar/worktree/pkg/registry"
	"github.com/braunmar/worktree/pkg/ui"

	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart [feature-name]",
	Short: "Restart services for a feature worktree",
	Long: `Restart services for a specific feature worktree.

This command runs ONLY restart hooks (not start/stop hooks):
1. Runs restart_pre_command for each project (if configured)
2. Stops services (NO stop_pre/post_command)
3. Starts services (NO start_pre/post_command)
4. Runs restart_post_command for each project (if configured)

Use restart_pre/post_command for restart-specific operations:
- restart_pre_command: backup state, drain connections, etc.
- restart_post_command: verify health, warm caches, etc.

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

	// Load config
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

	// Build environment variables
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

	// Phase 1: restart_pre_command for each project
	for _, projectName := range wt.Projects {
		if project, ok := workCfg.Projects[projectName]; ok {
			worktreePath := featureDir + "/" + project.Dir
			composeProject := wt.GetComposeProject(projectName)
			projectEnv := append(envList, fmt.Sprintf("COMPOSE_PROJECT_NAME=%s", composeProject))
			runHookCommand(fmt.Sprintf("%s: restart_pre_command", projectName), project.RestartPreCommand, worktreePath, projectEnv)
		}
	}

	// Phase 2: Stop services (NO stop_pre/post hooks)
	ui.Loading("Stopping services...")
	for _, projectName := range wt.Projects {
		project, exists := workCfg.Projects[projectName]
		if !exists {
			continue
		}

		composeProject := wt.GetComposeProject(projectName)
		if composeProject == "" {
			composeProject = fmt.Sprintf("%s-%s-%s", workCfg.ProjectName, featureName, projectName)
		}

		switch project.GetExecutor() {
		case "process":
			pidFile := filepath.Join(featureDir, projectName+".pid")
			if process.IsRunning(pidFile) {
				if err := process.StopProcess(pidFile); err != nil {
					ui.Warning(fmt.Sprintf("Failed to stop %s: %v", projectName, err))
				}
			}
		default: // "docker"
			if docker.IsFeatureRunning(workCfg.ProjectName, featureName) {
				projectInfo := map[string]string{project.Dir: composeProject}
				if err := docker.StopFeature(workCfg.ProjectName, featureName, featureDir, projectInfo); err != nil {
					ui.Warning(fmt.Sprintf("Failed to stop %s: %v", projectName, err))
				}
			}
		}
	}
	ui.NewLine()

	// Phase 3: Start services (NO start_pre/post hooks)
	ui.Loading("Starting services...")
	for _, projectName := range wt.Projects {
		project := workCfg.Projects[projectName]
		worktreePath := featureDir + "/" + project.Dir

		// Build environment with COMPOSE_PROJECT_NAME
		projectEnvList := os.Environ()
		for key, value := range baseEnvVars {
			projectEnvList = append(projectEnvList, fmt.Sprintf("%s=%s", key, value))
		}
		composeProject := wt.GetComposeProject(projectName)
		projectEnvList = append(projectEnvList, fmt.Sprintf("COMPOSE_PROJECT_NAME=%s", composeProject))

		// Start service
		var startErr error
		switch project.GetExecutor() {
		case "process":
			pidFile := filepath.Join(featureDir, projectName+".pid")
			startErr = process.StartBackground(projectName, project.StartCommand, worktreePath, projectEnvList, pidFile)
		default: // "docker"
			shellCmd := exec.Command("sh", "-c", project.StartCommand)
			shellCmd.Dir = worktreePath
			shellCmd.Env = projectEnvList
			shellCmd.Stdout = os.Stdout
			shellCmd.Stderr = os.Stderr
			startErr = shellCmd.Run()
		}
		if startErr != nil {
			ui.Error(fmt.Sprintf("Failed to start %s: %v", projectName, startErr))
			os.Exit(1)
		}
	}
	ui.NewLine()

	// Phase 4: restart_post_command for each project
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
