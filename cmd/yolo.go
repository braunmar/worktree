package cmd

import (
	"fmt"
	"os"

	"github.com/braunmar/worktree/pkg/config"
	"github.com/braunmar/worktree/pkg/registry"
	"github.com/braunmar/worktree/pkg/ui"

	"github.com/spf13/cobra"
)

var (
	yoloDisable bool
)

var yoloCmd = &cobra.Command{
	Use:   "yolo <feature-name>",
	Short: "Enable or disable YOLO mode for a feature worktree",
	Long: `Toggle YOLO mode for a feature worktree.

YOLO mode signals that Claude can work autonomously on this worktree
when the solution is clear (e.g., improving test coverage, fixing obvious bugs).

When enabled, Claude will be more proactive and make decisions without
asking for confirmation on straightforward tasks.

Examples:
  worktree yolo feature-user-auth       # Enable YOLO mode
  worktree yolo feature-user-auth --disable  # Disable YOLO mode`,
	Args: cobra.ExactArgs(1),
	Run:  runYolo,
}

func init() {
	yoloCmd.Flags().BoolVar(&yoloDisable, "disable", false, "disable YOLO mode")
}

func runYolo(cmd *cobra.Command, args []string) {
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

	// Toggle YOLO mode
	newState := !yoloDisable
	wt.YoloMode = newState

	// Save registry
	if err := reg.Save(); err != nil {
		checkError(fmt.Errorf("failed to save registry: %w", err))
	}

	// Update instance marker file
	featureDir := cfg.WorktreeFeaturePath(featureName)
	if err := config.UpdateInstanceYoloMode(featureDir, newState); err != nil {
		ui.Warning(fmt.Sprintf("Failed to update instance marker: %v", err))
	}

	// Display result
	ui.NewLine()
	if newState {
		ui.Success(fmt.Sprintf("🚀 YOLO mode ENABLED for '%s'", featureName))
		ui.Info("Claude will work autonomously when the solution is clear")
		ui.Info("Example: 'improve test coverage', 'fix obvious bugs'")
	} else {
		ui.Success(fmt.Sprintf("🛡️  YOLO mode DISABLED for '%s'", featureName))
		ui.Info("Claude will ask for confirmation on all changes")
	}
	ui.NewLine()
}
