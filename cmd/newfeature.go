package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"worktree/pkg/config"
	"worktree/pkg/docker"
	"worktree/pkg/git"
	"worktree/pkg/registry"
	"worktree/pkg/ui"

	"github.com/spf13/cobra"
)

var (
	preset       string
	noFixturesNF bool
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
  worktree new-feature feature/ui --no-fixtures       # Skip fixtures`,
	Args: cobra.RangeArgs(1, 2),
	Run:  runNewFeature,
}

func init() {
	newFeatureCmd.Flags().BoolVar(&noFixturesNF, "no-fixtures", false, "skip running fixtures")
}

func runNewFeature(cmd *cobra.Command, args []string) {
	branch := args[0]
	presetName := ""
	if len(args) > 1 {
		presetName = args[1]
	}

	// Normalize branch name to feature name
	featureName := registry.NormalizeBranchName(branch)

	// Get configuration
	cfg, err := config.New()
	checkError(err)

	// Load worktree configuration
	workCfg, err := config.LoadWorktreeConfig(cfg.ProjectRoot)
	checkError(err)

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

	// Allocate ports for all services
	ui.Section("Allocating ports...")
	services := workCfg.GetPortServiceNames()
	ports, err := reg.AllocatePorts(services)
	checkError(err)

	// Display allocated ports
	ui.CheckMark("Ports allocated")
	ui.NewLine()

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

	// Add to registry
	wt := &registry.Worktree{
		Branch:         branch,
		Normalized:     featureName,
		Created:        time.Now(),
		Projects:       presetCfg.Projects,
		Ports:          ports,
		ComposeProject: fmt.Sprintf("%s-%s", workCfg.ProjectName, featureName),
	}
	if err := reg.Add(wt); err != nil {
		checkError(err)
	}
	if err := reg.Save(); err != nil {
		checkError(err)
	}
	ui.CheckMark("Registry updated")
	ui.NewLine()

	// Export environment variables
	envVars := map[string]string{
		"FEATURE_NAME":         featureName,
		"COMPOSE_PROJECT_NAME": wt.ComposeProject,
	}
	for service, port := range ports {
		envVars[service] = fmt.Sprintf("%d", port)
	}

	envList := os.Environ()
	for key, value := range envVars {
		envList = append(envList, fmt.Sprintf("%s=%s", key, value))
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

			if project.PostCommand == "" {
				continue
			}

			ui.Loading(fmt.Sprintf("Running %s post-command...", projectName))

			worktreePath := featureDir + "/" + project.Dir
			postCmd := project.PostCommand

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
