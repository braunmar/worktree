package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"worktree/pkg/config"
	"worktree/pkg/registry"
	"worktree/pkg/ui"

	"github.com/spf13/cobra"
)

var (
	noFixtures bool
	presetName string
)

var startCmd = &cobra.Command{
	Use:   "start <feature-name>",
	Short: "Start all services for a feature worktree",
	Long: `Start all services for a feature worktree based on preset configuration.

Starts ALL projects defined in the preset sequentially. Works with detached Docker
services that return immediately.

Example:
  worktree start feature-user-auth                  # Start feature
  worktree start feature-reports --preset backend   # Use specific preset
  worktree start feature-api --no-fixtures          # Skip post-startup tasks`,
	Args: cobra.ExactArgs(1),
	Run:  runStart,
}

func init() {
	startCmd.Flags().BoolVar(&noFixtures, "no-fixtures", false, "skip post-startup tasks")
	startCmd.Flags().StringVar(&presetName, "preset", "", "preset to use (defaults to default_preset from config)")
}

func runStart(cmd *cobra.Command, args []string) {
	featureName := args[0]

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

	// Check if worktree directory exists
	if !cfg.WorktreeExists(featureName) {
		ui.Error(fmt.Sprintf("Feature directory not found: worktrees/%s", featureName))
		os.Exit(1)
	}

	// Get preset (from flag or use projects from registry)
	var projects []string
	if presetName != "" {
		preset, err := workCfg.GetPreset(presetName)
		checkError(err)
		projects = preset.Projects
	} else {
		projects = wt.Projects
	}

	if len(projects) == 0 {
		ui.Error("No projects found")
		os.Exit(1)
	}

	// Display header
	ui.Rocket(fmt.Sprintf("Starting Feature: %s", featureName))
	ui.Info(fmt.Sprintf("Branch: %s", wt.Branch))
	ui.NewLine()

	// Show port mapping from registry
	ui.Section("Port mapping:")
	displayServices := workCfg.GetDisplayableServices(wt.Ports)
	for name, url := range displayServices {
		ui.PrintStatusLine(name, url)
	}
	ui.NewLine()

	// Calculate instance from allocated APP_PORT in registry
	appPortCfg := workCfg.Ports["APP_PORT"]
	basePort := config.ExtractBasePort(appPortCfg.Port)
	instance := wt.Ports["APP_PORT"] - basePort

	// Export all environment variables (includes allocated ports + calculated values like INSTANCE, LOCALSTACK_EXT_*)
	baseEnvVars := workCfg.ExportEnvVars(instance)
	baseEnvVars["FEATURE_NAME"] = featureName

	// Override with allocated ports from registry
	for service, port := range wt.Ports {
		baseEnvVars[service] = fmt.Sprintf("%d", port)
	}

	featureDir := cfg.WorktreeFeaturePath(featureName)

	// Start ALL projects sequentially
	for i, projectName := range projects {
		project := workCfg.Projects[projectName]
		worktreePath := featureDir + "/" + project.Dir

		// Check if worktree exists
		if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
			ui.Error(fmt.Sprintf("Worktree for %s does not exist: %s", projectName, worktreePath))
			os.Exit(1)
		}

		// Start services
		ui.Loading(fmt.Sprintf("Starting %s...", projectName))
		ui.NewLine()

		// Build environment list with per-service COMPOSE_PROJECT_NAME
		envList := os.Environ()
		for key, value := range baseEnvVars {
			envList = append(envList, fmt.Sprintf("%s=%s", key, value))
		}
		// Add service-specific compose project name
		envList = append(envList, fmt.Sprintf("COMPOSE_PROJECT_NAME=%s", wt.GetComposeProject(projectName)))

		// Execute start command via shell
		shellCmd := exec.Command("sh", "-c", project.StartCommand)
		shellCmd.Dir = worktreePath
		shellCmd.Env = envList
		shellCmd.Stdout = os.Stdout
		shellCmd.Stderr = os.Stderr

		if err := shellCmd.Run(); err != nil {
			ui.Error(fmt.Sprintf("Failed to start %s: %v", projectName, err))
			os.Exit(1)
		}

		ui.NewLine()
		ui.Success(fmt.Sprintf("%s started!", projectName))
		ui.NewLine()

		// Run post command for first project (unless --no-fixtures)
		if i == 0 && !noFixtures && project.PostCommand != "" {
			ui.Loading("Running post-startup tasks...")
			ui.NewLine()

			// Execute via shell to support && chains
			postShellCmd := exec.Command("sh", "-c", project.PostCommand)
			postShellCmd.Dir = worktreePath
			postShellCmd.Env = envList
			postShellCmd.Stdout = os.Stdout
			postShellCmd.Stderr = os.Stderr

			if err := postShellCmd.Run(); err != nil {
				ui.Warning(fmt.Sprintf("Failed to run post-startup tasks: %v", err))
				ui.Info(fmt.Sprintf("You can run manually: %s", project.PostCommand))
			} else {
				ui.Success("Post-startup tasks completed!")
			}
			ui.NewLine()
		}
	}

	// Show final summary
	ui.Success("All services started!")
	ui.NewLine()
	displayServices = workCfg.GetDisplayableServices(wt.Ports)
	for name, url := range displayServices {
		ui.PrintStatusLine(name, url)
	}
	ui.NewLine()
}
