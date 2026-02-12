package cmd

import (
	"fmt"
	"os"
	"worktree/pkg/config"
	"worktree/pkg/doctor"
	"worktree/pkg/registry"

	"github.com/spf13/cobra"
)

var (
	featureFilter string
	noFetch       bool
	autoFix       bool
	jsonOutput    bool
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check worktree environment health",
	Long: `Diagnose and report issues with worktree setup:

- Docker availability and status
- Orphaned containers, directories, or registry entries
- Git status and branch tracking
- Stale worktrees (old, merged, or unused)
- Port allocation conflicts

The doctor command helps maintain a healthy worktree environment and
identifies issues before they cause problems.

Examples:
  worktree doctor                      # Check all worktrees
  worktree doctor --feature user-auth  # Check specific feature
  worktree doctor --no-fetch           # Skip git fetch (faster)
  worktree doctor --fix                # Auto-fix safe issues
  worktree doctor --json               # JSON output for scripting`,
	Args: cobra.NoArgs,
	Run:  runDoctor,
}

func init() {
	doctorCmd.Flags().StringVar(&featureFilter, "feature", "", "check specific feature only")
	doctorCmd.Flags().BoolVar(&noFetch, "no-fetch", false, "skip git fetch before comparing")
	doctorCmd.Flags().BoolVar(&autoFix, "fix", false, "auto-fix safe issues (orphaned registry entries)")
	doctorCmd.Flags().BoolVar(&jsonOutput, "json", false, "output results as JSON")
}

func runDoctor(cmd *cobra.Command, args []string) {
	// Load config
	cfg, err := config.New()
	checkError(err)

	workCfg, err := config.LoadWorktreeConfig(cfg.ProjectRoot)
	checkError(err)

	// Load registry
	reg, err := registry.Load(cfg.WorktreeDir, workCfg)
	checkError(err)

	// Run health checks
	report := doctor.RunHealthCheck(cfg, workCfg, reg, doctor.Options{
		FeatureFilter: featureFilter,
		NoFetch:       noFetch,
		AutoFix:       autoFix,
	})

	// Output report
	if jsonOutput {
		fmt.Println(report.ToJSON())
	} else {
		report.Print()
	}

	// Exit with appropriate code
	os.Exit(report.ExitCode())
}
