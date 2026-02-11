package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"worktree/pkg/config"
	"worktree/pkg/docker"
	"worktree/pkg/registry"
	"worktree/pkg/ui"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status <feature-name>",
	Short: "Show detailed status for a feature",
	Long: `Show detailed status for a specific feature worktree.

This command shows:
- Running status
- Port mapping
- Container health
- Worktree location

Example:
  worktree status feature-user-auth
  worktree status feature-reports`,
	Args: cobra.ExactArgs(1),
	Run:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) {
	featureName := args[0]

	// Get configuration
	cfg, err := config.New()
	checkError(err)

	// Load registry
	reg, err := registry.Load(cfg.WorktreeDir)
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
	ui.PrintHeader(fmt.Sprintf("Status for Feature: %s", featureName))
	ui.NewLine()

	// Show basic info
	ui.PrintStatusLine("Branch", wt.Branch)
	ui.PrintStatusLine("Created", wt.Created.Format("2006-01-02 15:04"))
	ui.NewLine()

	// Check worktree status
	if cfg.WorktreeExists(featureName) {
		ui.PrintStatusLine("Worktree", "✅ Exists")
		ui.Info(fmt.Sprintf("  Backend:  %s", cfg.WorktreeBackendPath(featureName)))
		ui.Info(fmt.Sprintf("  Frontend: %s", cfg.WorktreeFrontendPath(featureName)))
		ui.NewLine()
	} else {
		ui.PrintStatusLine("Worktree", "⚠️  Directory not found")
		ui.NewLine()
		return
	}

	// Check if feature is running
	running := docker.IsFeatureRunning(featureName)

	if running {
		ui.PrintStatusLine("Status", "🟢 Running")
		ui.NewLine()

		// Show port mapping from registry
		ui.PrintHeader("Port Mapping")
		ui.PrintStatusLine("Frontend", fmt.Sprintf("http://localhost:%d", wt.Ports["FE_PORT"]))
		ui.PrintStatusLine("Backend", fmt.Sprintf("http://localhost:%d", wt.Ports["BE_PORT"]))
		ui.PrintStatusLine("PostgreSQL", fmt.Sprintf("localhost:%d", wt.Ports["POSTGRES_PORT"]))
		ui.PrintStatusLine("Mailpit", fmt.Sprintf("http://localhost:%d", wt.Ports["MAILPIT_UI_PORT"]))
		ui.NewLine()

		// Show container health
		ui.PrintHeader("Container Health")
		dockerCmd := exec.Command(
			"docker", "ps",
			"--filter", fmt.Sprintf("name=skillsetup-%s-", featureName),
			"--format", "table {{.Names}}\t{{.Status}}",
		)

		output, err := dockerCmd.CombinedOutput()
		if err == nil {
			fmt.Println(string(output))
		}
	} else {
		ui.PrintStatusLine("Status", "⚪ Not running")
		ui.NewLine()
		ui.Info(fmt.Sprintf("Start with: worktree start %s", featureName))
		ui.NewLine()
	}
}
