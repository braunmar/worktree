package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/braunmar/worktree/pkg/config"
	"github.com/braunmar/worktree/pkg/docker"
	"github.com/braunmar/worktree/pkg/git"
	"github.com/braunmar/worktree/pkg/registry"
	"github.com/braunmar/worktree/pkg/ui"

	"github.com/spf13/cobra"
)

var (
	preset       string
	noFixturesNF bool
	dryRun       bool
	yoloModeNF   bool
)

var newFeatureCmd = &cobra.Command{
	Use:   "new-feature <branch> [preset]",
	Short: "Create and start a complete feature development environment",
	Long: `Create worktrees, start services, and navigate Claude to the working directory.

This is a one-command setup for feature development that:
1. Reads configuration from .worktree.yml
2. Normalizes branch name to feature directory (e.g., feature/user-auth -> feature-user-auth)
3. Dynamically allocates ports from available ranges
4. Creates git worktrees for all projects in the preset
5. Starts services (backend, frontend, etc.)
6. Runs post-startup commands (if configured)
7. Navigates Claude to the backend worktree

Examples:
  worktree new-feature feature/user-auth              # Use default preset
  worktree new-feature feature/reports fullstack      # Use fullstack preset
  worktree new-feature feature/api backend            # Backend only
  worktree new-feature feature/ui --no-fixtures       # Skip fixtures
  worktree new-feature feature/coverage --yolo        # Enable YOLO mode`,
	Args: cobra.RangeArgs(1, 2),
	Run:  runNewFeature,
}

func init() {
	newFeatureCmd.Flags().BoolVar(&noFixturesNF, "no-fixtures", false, "skip running fixtures")
	newFeatureCmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview changes without creating anything")
	newFeatureCmd.Flags().BoolVar(&yoloModeNF, "yolo", false, "enable YOLO mode (Claude works autonomously)")
}

func runNewFeature(cmd *cobra.Command, args []string) {
	branch := args[0]
	presetName := ""
	if len(args) > 1 {
		presetName = args[1]
	}

	verbose, _ := cmd.Flags().GetBool("verbose")

	// Normalize branch name to feature name
	featureName := registry.NormalizeBranchName(branch)
	if verbose {
		ui.Info(fmt.Sprintf("Normalized branch '%s' to feature name '%s'", branch, featureName))
	}

	// Get configuration
	cfg, err := config.New()
	checkError(err)
	if verbose {
		ui.Info(fmt.Sprintf("Loaded configuration from: %s", cfg.ProjectRoot))
	}

	// Load worktree configuration
	workCfg, err := config.LoadWorktreeConfig(cfg.ProjectRoot)
	checkError(err)
	if verbose {
		ui.Info(fmt.Sprintf("Loaded worktree configuration with %d projects", len(workCfg.Projects)))
	}

	// Get preset
	presetCfg, err := workCfg.GetPreset(presetName)
	checkError(err)

	// Display header
	ui.Rocket(fmt.Sprintf("Setting up feature environment: %s", branch))
	ui.Info(fmt.Sprintf("Feature: %s", featureName))
	ui.Info(fmt.Sprintf("Preset: %s - %s", presetName, presetCfg.Description))
	ui.NewLine()

	// Check if worktree already exists
	if cfg.WorktreeExists(featureName) {
		ui.Error(fmt.Sprintf("Worktree '%s' already exists", featureName))
		fmt.Println("\nRemove it first with:")
		ui.PrintCommand(fmt.Sprintf("  worktree remove %s", featureName))
		os.Exit(1)
	}

	// Load registry
	reg, err := registry.Load(cfg.WorktreeDir, workCfg)
	checkError(err)
	if verbose {
		ui.Info(fmt.Sprintf("Loaded registry from: %s", cfg.WorktreeDir))
		ui.Info(fmt.Sprintf("Found %d existing worktrees", len(reg.Worktrees)))
	}

	// Allocate ports for all services
	ui.Section("Allocating ports...")
	services := workCfg.GetPortServiceNames()
	if verbose {
		ui.Info(fmt.Sprintf("Services requiring ports: %v", services))
	}
	ports, err := reg.AllocatePorts(services)
	checkError(err)
	if verbose {
		ui.Info(fmt.Sprintf("Allocated ports: %v", ports))
	}

	// Calculate INSTANCE from the first allocated ranged port
	instancePortName, err := workCfg.GetInstancePortName()
	checkError(err)
	instancePortCfg := workCfg.EnvVariables[instancePortName]
	basePort, err := config.ExtractBasePort(instancePortCfg.Port)
	checkError(err)
	instance := ports[instancePortName] - basePort

	// Display allocated ports
	ui.CheckMark("Ports allocated")
	ui.Info(fmt.Sprintf("Instance: %d", instance))
	ui.NewLine()

	// If dry-run, display preview and exit
	if dryRun {
		displayDryRunPreview(featureName, instance, ports, workCfg, presetCfg, cfg)
		os.Exit(0)
	}

	// Create feature directory
	featureDir := cfg.WorktreeFeaturePath(featureName)
	if err := os.MkdirAll(featureDir, 0755); err != nil {
		checkError(fmt.Errorf("failed to create feature directory: %w", err))
	}

	// Create worktrees for each project in preset
	ui.Section("Creating worktrees...")
	for _, projectName := range presetCfg.Projects {
		project := workCfg.Projects[projectName]
		ui.Loading(fmt.Sprintf("Creating %s worktree...", projectName))

		projectDir := cfg.ProjectRoot + "/" + project.Dir
		worktreePath := featureDir + "/" + project.Dir

		if verbose {
			ui.Info(fmt.Sprintf("Git worktree command: git worktree add %s %s", worktreePath, branch))
		}

		if err := git.CreateWorktree(projectDir, worktreePath, branch); err != nil {
			checkError(fmt.Errorf("failed to create %s worktree: %w", projectName, err))
		}
		ui.CheckMark(fmt.Sprintf("Created %s worktree", projectName))
	}
	ui.NewLine()

	// Create symlinks from configuration
	if len(workCfg.Symlinks) > 0 {
		ui.Section("Creating symlinks...")
		// Calculate relative path from worktree to project root
		// worktrees/feature-name -> ../..
		relPathToRoot := config.CalculateRelativePath(2) // 2 levels: worktrees/feature-name

		for _, link := range workCfg.Symlinks {
			sourcePath := relPathToRoot + "/" + link.Source
			targetPath := featureDir + "/" + link.Target

			// Check if target already exists
			if info, err := os.Lstat(targetPath); err == nil {
				// If it's a symlink, remove it
				if info.Mode()&os.ModeSymlink != 0 {
					if err := os.Remove(targetPath); err != nil {
						ui.Warning(fmt.Sprintf("Failed to remove existing symlink %s: %v", link.Target, err))
						continue
					}
				} else {
					// If it's a directory or file, back it up and remove
					backupPath := targetPath + ".backup." + time.Now().Format("20060102-150405")
					if err := os.Rename(targetPath, backupPath); err != nil {
						ui.Warning(fmt.Sprintf("Failed to backup existing %s: %v", link.Target, err))
						continue
					} else {
						ui.Info(fmt.Sprintf("Backed up existing %s", link.Target))
					}
				}
			}

			// Create symlink
			if err := os.Symlink(sourcePath, targetPath); err != nil {
				ui.Warning(fmt.Sprintf("Failed to create symlink %s: %v", link.Target, err))
			} else {
				ui.CheckMark(fmt.Sprintf("Linked %s -> %s", link.Target, link.Source))
			}
		}
		ui.NewLine()
	}

	// Copy files from configuration
	if len(workCfg.Copies) > 0 {
		ui.Section("Copying files...")
		for _, copy := range workCfg.Copies {
			sourcePath := cfg.ProjectRoot + "/" + copy.Source
			targetPath := featureDir + "/" + copy.Target

			// Check if source exists
			sourceInfo, err := os.Stat(sourcePath)
			if os.IsNotExist(err) {
				ui.Warning(fmt.Sprintf("Source not found: %s", copy.Source))
				continue
			}

			// Handle directories vs files
			if sourceInfo.IsDir() {
				// Copy directory recursively using cp command
				cpCmd := exec.Command("cp", "-R", sourcePath, targetPath)
				if err := cpCmd.Run(); err != nil {
					ui.Warning(fmt.Sprintf("Failed to copy directory %s: %v", copy.Source, err))
				} else {
					ui.CheckMark(fmt.Sprintf("Copied directory %s -> %s", copy.Source, copy.Target))
				}
			} else {
				// Copy file
				sourceData, err := os.ReadFile(sourcePath)
				if err != nil {
					ui.Warning(fmt.Sprintf("Failed to read %s: %v", copy.Source, err))
					continue
				}

				if err := os.WriteFile(targetPath, sourceData, 0644); err != nil {
					ui.Warning(fmt.Sprintf("Failed to copy %s: %v", copy.Target, err))
				} else {
					ui.CheckMark(fmt.Sprintf("Copied %s -> %s", copy.Source, copy.Target))
				}
			}
		}
		ui.NewLine()
	}

	// Generate compose project names for each service
	template := workCfg.GetComposeProjectTemplate()
	composeProjects := make(map[string]string)
	for _, projectName := range presetCfg.Projects {
		composeProjects[projectName] = workCfg.ReplaceComposeProjectPlaceholders(template, featureName, projectName)
	}

	// Add to registry
	wt := &registry.Worktree{
		Branch:          branch,
		Normalized:      featureName,
		Created:         time.Now(),
		Projects:        presetCfg.Projects,
		Ports:           ports,
		ComposeProjects: composeProjects,
		YoloMode:        yoloModeNF,
	}
	if err := reg.Add(wt); err != nil {
		checkError(err)
	}
	if err := reg.Save(); err != nil {
		checkError(err)
	}
	ui.CheckMark("Registry updated")

	// Write .worktree-instance marker file
	if err := config.WriteInstanceMarker(featureDir, featureName, instance, cfg.ProjectRoot, presetCfg.Projects, ports, yoloModeNF); err != nil {
		ui.Warning(fmt.Sprintf("Failed to write instance marker: %v", err))
	} else {
		ui.CheckMark("Instance marker created")
	}
	ui.NewLine()

	// Export all environment variables (includes allocated ports + calculated values like INSTANCE, LOCALSTACK_EXT_*)
	baseEnvVars := workCfg.ExportEnvVars(instance)
	baseEnvVars["FEATURE_NAME"] = featureName

	// Override with actually allocated ports (in case of conflicts)
	for service, port := range ports {
		baseEnvVars[service] = fmt.Sprintf("%d", port)
	}

	// Recompute value-template vars (e.g., GOOGLE_OAUTH_REDIRECT_URI) now that actual
	// allocated ports are in baseEnvVars. Without this, they resolve against base port
	// expressions (always 3000, 8080, etc.) instead of the real allocated ports.
	workCfg.ResolveValueVars(instance, baseEnvVars)

	// Persist all resolved env vars to registry for visibility and debugging
	wt.ComputedVars = workCfg.GetComputedVars(baseEnvVars)
	if err := reg.Save(); err != nil {
		ui.Warning(fmt.Sprintf("Failed to update registry computed vars: %v", err))
	}

	// Generate configured files for each project (e.g., .env.development.local)
	for _, projectName := range presetCfg.Projects {
		if err := workCfg.GenerateFiles(projectName, featureDir, baseEnvVars); err != nil {
			ui.Warning(fmt.Sprintf("Failed to generate files for %s: %v", projectName, err))
		}
	}

	// Start services for each project
	ui.Section("Starting services...")
	for _, projectName := range presetCfg.Projects {
		project := workCfg.Projects[projectName]

		if project.StartCommand == "" {
			ui.Info(fmt.Sprintf("No start command for %s, skipping...", projectName))
			continue
		}

		ui.Loading(fmt.Sprintf("Starting '%s' services...", projectName))

		worktreePath := featureDir + "/" + project.Dir

		// Build environment list with per-service COMPOSE_PROJECT_NAME
		envList := os.Environ()
		for key, value := range baseEnvVars {
			envList = append(envList, fmt.Sprintf("%s=%s", key, value))
		}
		// Add service-specific compose project name
		composeProject := wt.GetComposeProject(projectName)
		envList = append(envList, fmt.Sprintf("COMPOSE_PROJECT_NAME=%s", composeProject))

		if verbose {
			ui.Info(fmt.Sprintf("Starting %s with COMPOSE_PROJECT_NAME=%s", projectName, composeProject))
			ui.Info(fmt.Sprintf("Start command: %s", project.StartCommand))
		}

		// Replace placeholders in start command
		startCmd := project.StartCommand
		// Note: ReplaceInstancePlaceholder will be updated separately, for now use feature name
		// This might need adjustment based on what the command expects

		// Execute start command
		makeCmd := exec.Command("sh", "-c", startCmd)
		makeCmd.Dir = worktreePath
		makeCmd.Env = envList
		makeCmd.Stdout = os.Stdout
		makeCmd.Stderr = os.Stderr

		if err := makeCmd.Run(); err != nil {
			ui.Warning(fmt.Sprintf("Failed to start %s: %v", projectName, err))
		} else {
			// Verify containers are actually running (wait for startup)
			time.Sleep(3 * time.Second)

			containerStatus, err := docker.GetFeatureContainerStatus(workCfg.ProjectName, featureName)
			if err != nil {
				ui.Warning(fmt.Sprintf("Could not verify %s container status: %v", projectName, err))
			} else {
				// Check if any containers exited
				hasFailures := false
				for service, status := range containerStatus {
					if strings.Contains(strings.ToLower(status), "exited") {
						ui.Warning(fmt.Sprintf("%s service '%s' exited: %s", projectName, service, status))
						hasFailures = true
					}
				}

				if !hasFailures && len(containerStatus) > 0 {
					ui.CheckMark(fmt.Sprintf("Started %s", projectName))
				} else if len(containerStatus) == 0 {
					ui.Warning(fmt.Sprintf("No containers found for %s", projectName))
				}
			}
		}
	}
	ui.NewLine()

	// Run post-commands (fixtures, seed data, etc.)
	if workCfg.AutoFixtures && !noFixturesNF {
		ui.Section("Running post-startup commands...")
		for _, projectName := range presetCfg.Projects {
			project := workCfg.Projects[projectName]

			if project.StartPostCommand == "" {
				continue
			}

			ui.Loading(fmt.Sprintf("Running %s post-command...", projectName))

			worktreePath := featureDir + "/" + project.Dir
			postCmd := project.StartPostCommand

			// Build environment list with per-service COMPOSE_PROJECT_NAME
			envList := os.Environ()
			for key, value := range baseEnvVars {
				envList = append(envList, fmt.Sprintf("%s=%s", key, value))
			}
			// Add service-specific compose project name
			envList = append(envList, fmt.Sprintf("COMPOSE_PROJECT_NAME=%s", wt.GetComposeProject(projectName)))

			cmd := exec.Command("sh", "-c", postCmd)
			cmd.Dir = worktreePath
			cmd.Env = envList
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				ui.Warning(fmt.Sprintf("Failed to run post-command: %v", err))
			} else {
				ui.CheckMark(fmt.Sprintf("Post-command completed for %s", projectName))
			}
		}
		ui.NewLine()
	}

	// Get Claude working directory (from preset projects, not all projects)
	claudeProject := getClaudeWorkingProject(workCfg, presetCfg.Projects)
	claudePath := fmt.Sprintf("worktrees/%s/%s", featureName, workCfg.Projects[claudeProject].Dir)

	// Success message
	ui.Success("Feature environment ready!")
	ui.NewLine()

	// Navigate Claude
	ui.PrintHeader("Claude is ready to work:")
	ui.PrintStatusLine("  Working directory", claudePath)
	ui.PrintStatusLine("  Feature name", featureName)

	// Show YOLO mode status
	if wt.YoloMode {
		ui.PrintStatusLine("  YOLO Mode", "🚀 Enabled (autonomous mode)")
	}
	ui.NewLine()

	// Show access URLs dynamically from config
	displayServices := workCfg.GetDisplayableServices(ports)
	if len(displayServices) > 0 {
		for name, url := range displayServices {
			ui.PrintStatusLine("  "+name, url)
		}
		ui.NewLine()
	}

	// Change to working directory
	if err := os.Chdir(claudePath); err != nil {
		ui.Warning(fmt.Sprintf("Failed to change directory: %v", err))
		ui.Info(fmt.Sprintf("Please manually cd to: %s", claudePath))
	} else {
		ui.Success(fmt.Sprintf("Navigated to %s", claudePath))
		// Give a moment for services to stabilize
		time.Sleep(2 * time.Second)
	}

	ui.NewLine()
}

// displayDryRunPreview shows what would be created without actually creating it
func displayDryRunPreview(featureName string, instance int, ports map[string]int, workCfg *config.WorktreeConfig, presetCfg *config.PresetConfig, cfg *config.Config) {
	ui.Section("🔍 Dry Run - Preview Mode")

	// Port allocation preview
	fmt.Println("Port Allocation:")
	for service, port := range ports {
		ui.CheckMark(fmt.Sprintf("%s: %d", service, port))
	}
	ui.NewLine()

	// Instance and environment variables
	fmt.Printf("Instance: %d\n", instance)
	baseEnvVars := workCfg.ExportEnvVars(instance)
	for key, value := range baseEnvVars {
		ui.CheckMark(fmt.Sprintf("%s=%s", key, value))
	}
	ui.NewLine()

	// Worktrees to be created
	fmt.Println("Worktrees to create:")
	featureDir := cfg.WorktreeFeaturePath(featureName)
	for _, projectName := range presetCfg.Projects {
		project := workCfg.Projects[projectName]
		worktreePath := featureDir + "/" + project.Dir
		ui.CheckMark(worktreePath)
	}
	ui.NewLine()

	// Symlinks
	if len(workCfg.Symlinks) > 0 {
		fmt.Println("Symlinks to create:")
		for _, link := range workCfg.Symlinks {
			ui.CheckMark(fmt.Sprintf("%s -> %s", link.Target, link.Source))
		}
		ui.NewLine()
	}

	// Copies
	if len(workCfg.Copies) > 0 {
		fmt.Println("Files to copy:")
		for _, copy := range workCfg.Copies {
			ui.CheckMark(fmt.Sprintf("%s -> %s", copy.Source, copy.Target))
		}
		ui.NewLine()
	}

	// Services to start
	fmt.Println("Services to start:")
	for _, projectName := range presetCfg.Projects {
		project := workCfg.Projects[projectName]
		if project.StartCommand != "" {
			ui.CheckMark(fmt.Sprintf("%s: %s", projectName, project.StartCommand))
		}
	}
	ui.NewLine()

	// Post commands
	if workCfg.AutoFixtures && !noFixturesNF {
		fmt.Println("Post-startup commands:")
		for _, projectName := range presetCfg.Projects {
			project := workCfg.Projects[projectName]
			if project.StartPostCommand != "" {
				ui.CheckMark(fmt.Sprintf("%s: %s", projectName, project.StartPostCommand))
			}
		}
		ui.NewLine()
	}

	ui.Info("This is a dry run - no changes were made")
	fmt.Println("💡 Run without --dry-run to create the feature")
}
