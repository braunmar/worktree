package cmd

import (
	"fmt"
	"os"

	"worktree/pkg/config"
	"worktree/pkg/registry"
	"worktree/pkg/ui"

	"github.com/spf13/cobra"
)

var portsCmd = &cobra.Command{
	Use:   "ports <feature-name>",
	Short: "Show port mapping for a feature",
	Long: `Show port mapping for a specific feature worktree.

Displays:
- Frontend port
- Backend port
- PostgreSQL port
- Mailpit UI port
- Mailpit SMTP port
- LocalStack port

Example:
  worktree ports feature-user-auth
  worktree ports feature-reports`,
	Args: cobra.ExactArgs(1),
	Run:  runPorts,
}

func runPorts(cmd *cobra.Command, args []string) {
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

	// Display header
	ui.PrintHeader(fmt.Sprintf("Ports for Feature: %s", featureName))
	ui.NewLine()

	// Show ports from registry dynamically
	displayServices := workCfg.GetDisplayableServices(wt.Ports)
	for name, url := range displayServices {
		ui.PrintStatusLine(name, url)
	}

	// Show additional ports that have port numbers but no URL
	if port, exists := wt.Ports["MAILPIT_SMTP_PORT"]; exists {
		ui.PrintStatusLine("Mailpit SMTP", fmt.Sprintf("%s:%d", workCfg.Hostname, port))
	}
	ui.NewLine()
}
