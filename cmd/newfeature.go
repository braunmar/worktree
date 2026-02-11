package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"worktree/pkg/config"
	"worktree/pkg/git"
	"worktree/pkg/registry"
	"worktree/pkg/ui"

	"github.com/spf13/cobra"
)

var (
	preset         string
	noFixturesNF   bool
	noMigrationsNF bool
)

var newFeatureCmd = &cobra.Command{
	Use:   "new-feature <branch> [preset]",
	Short: "Create and start a complete feature development environment",
	Long: `Create worktrees, start services, run migrations, and navigate Claude to the working directory.

This is a one-command setup for feature development that:
1. Reads configuration from .worktree.yml
2. Normalizes branch name to feature directory (e.g., feature/user-auth -> feature-user-auth)
3. Dynamically allocates ports from available ranges
4. Creates git worktrees for all projects in the preset
5. Starts services (backend, frontend, etc.)
6. Runs migrations (if configured)
7. Runs fixtures (if configured)
8. Navigates Claude to the backend worktree

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
	newFeatureCmd.Flags().BoolVar(&noMigrationsNF, "no-migrations", false, "skip running migrations")
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
	reg, err := registry.Load(cfg.WorktreeDir)
	checkError(err)

	// Allocate ports for all services
	ui.Section("Allocating ports...")
	services := []string{"FE_PORT", "BE_PORT", "POSTGRES_PORT", "MAILPIT_SMTP_PORT", "MAILPIT_UI_PORT", "LOCALSTACK_PORT"}
	ports, err := reg.AllocatePorts(services)
	checkError(err)

	// Display allocated ports
	ui.CheckMark("Ports allocated:")
	ui.PrintStatusLine("  Frontend", fmt.Sprintf("http://localhost:%d", ports["FE_PORT"]))
	ui.PrintStatusLine("  Backend", fmt.Sprintf("http://localhost:%d", ports["BE_PORT"]))
	ui.PrintStatusLine("  PostgreSQL", fmt.Sprintf("localhost:%d", ports["POSTGRES_PORT"]))
	ui.PrintStatusLine("  Mailpit UI", fmt.Sprintf("http://localhost:%d", ports["MAILPIT_UI_PORT"]))
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

	// Create single .claude symlink at worktree root
	ui.Section("Setting up Claude configuration...")
	claudeSymlink := featureDir + "/.claude"

	// Calculate relative path from worktree to root .claude
	// worktrees/feature-name/.claude -> ../../.claude
	relPath := "../../.claude"

	// Check if .claude already exists
	if info, err := os.Lstat(claudeSymlink); err == nil {
		// If it's a symlink, remove it
		if info.Mode()&os.ModeSymlink != 0 {
			if err := os.Remove(claudeSymlink); err != nil {
				ui.Warning(fmt.Sprintf("Failed to remove existing symlink: %v", err))
			}
		} else {
			// If it's a directory or file, back it up and remove
			backupPath := claudeSymlink + ".backup." + time.Now().Format("20060102-150405")
			if err := os.Rename(claudeSymlink, backupPath); err != nil {
				ui.Warning(fmt.Sprintf("Failed to backup existing .claude: %v", err))
			} else {
				ui.Info(fmt.Sprintf("Backed up existing .claude to %s", backupPath))
			}
		}
	}

	// Create symlink
	if err := os.Symlink(relPath, claudeSymlink); err != nil {
		ui.Warning(fmt.Sprintf("Failed to create .claude symlink: %v", err))
	} else {
		ui.CheckMark("Linked .claude to root")
	}
	ui.NewLine()

	// Add to registry
	wt := &registry.Worktree{
		Branch:         branch,
		Normalized:     featureName,
		Created:        time.Now(),
		Projects:       presetCfg.Projects,
		Ports:          ports,
		ComposeProject: fmt.Sprintf("skillsetup-%s", featureName),
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
			ui.CheckMark(fmt.Sprintf("Started %s", projectName))
		}
	}
	ui.NewLine()

	// Run migrations for backend (if configured)
	if workCfg.AutoMigrations && !noMigrationsNF {
		ui.Section("Running migrations...")
		for _, projectName := range presetCfg.Projects {
			project := workCfg.Projects[projectName]

			if project.MigrationCommand == "" {
				continue
			}

			ui.Loading(fmt.Sprintf("Running %s migrations...", projectName))

			worktreePath := featureDir + "/" + project.Dir
			migrationCmd := project.MigrationCommand

			cmd := exec.Command("sh", "-c", migrationCmd)
			cmd.Dir = worktreePath
			cmd.Env = envList
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				ui.Warning(fmt.Sprintf("Failed to run migrations: %v", err))
			} else {
				ui.CheckMark(fmt.Sprintf("Migrations completed for %s", projectName))
			}
		}
		ui.NewLine()
	}

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

	// Get Claude working directory
	claudeProject := workCfg.GetClaudeWorkingProject()
	claudePath := fmt.Sprintf("worktrees/%s/%s", featureName, workCfg.Projects[claudeProject].Dir)

	// Success message
	ui.Success("Feature environment ready!")
	ui.NewLine()

	// Show access URLs
	ui.PrintHeader("Access services:")
	ui.PrintStatusLine("Frontend", fmt.Sprintf("http://localhost:%d", ports["FE_PORT"]))
	ui.PrintStatusLine("Backend", fmt.Sprintf("http://localhost:%d", ports["BE_PORT"]))
	ui.PrintStatusLine("Mailpit", fmt.Sprintf("http://localhost:%d", ports["MAILPIT_UI_PORT"]))
	ui.NewLine()

	// Navigate Claude
	ui.PrintHeader("Claude is ready to work:")
	ui.PrintStatusLine("Working directory", claudePath)
	ui.PrintStatusLine("Feature name", featureName)
	ui.NewLine()

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
