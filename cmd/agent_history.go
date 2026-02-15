package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"worktree/pkg/config"
	"worktree/pkg/history"
	"worktree/pkg/ui"
)

var (
	historyAgent  string
	historyStatus string
	historyLimit  int
)

var agentHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "View agent execution history",
	Long:  `View execution history and statistics for agent tasks.`,
}

var historyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent executions",
	Long: `List recent agent executions with details.

Filter by agent name or status. By default, shows last 20 executions.

Example:
  worktree agent history list
  worktree agent history list --agent npm-audit
  worktree agent history list --status failed --limit 10`,
	Run: runHistoryList,
}

var historyStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show execution statistics",
	Long: `Show aggregate statistics for agent executions.

Displays overall success rate, average duration, and per-agent statistics.

Example:
  worktree agent history stats`,
	Run: runHistoryStats,
}

var historyClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear execution history",
	Long: `Clear all execution history records.

Example:
  worktree agent history clear`,
	Run: runHistoryClear,
}

func init() {
	// Add flags
	historyListCmd.Flags().StringVar(&historyAgent, "agent", "", "Filter by agent name")
	historyListCmd.Flags().StringVar(&historyStatus, "status", "", "Filter by status (completed, failed)")
	historyListCmd.Flags().IntVar(&historyLimit, "limit", 20, "Limit number of results")

	// Register subcommands
	agentHistoryCmd.AddCommand(historyListCmd)
	agentHistoryCmd.AddCommand(historyStatsCmd)
	agentHistoryCmd.AddCommand(historyClearCmd)

	// Register history command under agent
	agentCmd.AddCommand(agentHistoryCmd)
}

func runHistoryList(cmd *cobra.Command, args []string) {
	// Load config
	cfg, err := config.New()
	checkError(err)

	// Load history
	h, err := history.Load(cfg.WorktreeDir)
	checkError(err)

	// Query history
	records := h.Query(historyAgent, historyStatus, historyLimit)

	if len(records) == 0 {
		ui.Info("No execution history found")
		return
	}

	ui.Section(fmt.Sprintf("Execution History (%d records)", len(records)))
	fmt.Println()

	for _, record := range records {
		// Status emoji
		var emoji string
		if record.Status == "completed" {
			emoji = "✅"
		} else {
			emoji = "❌"
		}

		fmt.Printf("%s %s\n", emoji, record.AgentName)
		fmt.Printf("   ID: %s\n", record.ID[:8]+"...")
		fmt.Printf("   Worktree: %s\n", record.Worktree)
		fmt.Printf("   Started: %s\n", record.StartTime.Format("2006-01-02 15:04:05"))
		fmt.Printf("   Duration: %s\n", time.Duration(record.Duration)*time.Millisecond)
		fmt.Printf("   Status: %s\n", record.Status)

		if record.Error != "" {
			fmt.Printf("   Error: %s\n", record.Error)
		}

		if len(record.Commits) > 0 {
			fmt.Printf("   Commits: %d\n", len(record.Commits))
		}

		if record.PRUrl != "" {
			fmt.Printf("   PR: %s\n", record.PRUrl)
		}

		fmt.Println()
	}
}

func runHistoryStats(cmd *cobra.Command, args []string) {
	// Load config
	cfg, err := config.New()
	checkError(err)

	// Load history
	h, err := history.Load(cfg.WorktreeDir)
	checkError(err)

	// Get statistics
	stats := h.Stats()

	if stats.TotalExecutions == 0 {
		ui.Info("No execution history found")
		return
	}

	ui.Section("Execution Statistics")
	fmt.Println()

	// Overall stats
	fmt.Println("📊 Overall")
	fmt.Printf("   Total executions: %d\n", stats.TotalExecutions)
	fmt.Printf("   Success rate: %.1f%%\n", stats.SuccessRate)
	fmt.Printf("   Average duration: %s\n", stats.AverageDuration)
	fmt.Println()

	// Per-agent stats
	if len(stats.ByAgent) > 0 {
		fmt.Println("🤖 By Agent")
		fmt.Println()

		for agentName, agentStats := range stats.ByAgent {
			fmt.Printf("   %s\n", agentName)
			fmt.Printf("      Executions: %d\n", agentStats.TotalExecutions)
			fmt.Printf("      Success: %d (%.1f%%)\n", agentStats.SuccessCount, agentStats.SuccessRate)
			fmt.Printf("      Failed: %d\n", agentStats.FailureCount)
			fmt.Printf("      Avg duration: %s\n", agentStats.AverageDuration)
			fmt.Println()
		}
	}
}

func runHistoryClear(cmd *cobra.Command, args []string) {
	// Load config
	cfg, err := config.New()
	checkError(err)

	// Load history
	h, err := history.Load(cfg.WorktreeDir)
	checkError(err)

	// Count before clear
	records := h.Query("", "", 0)
	count := len(records)

	if count == 0 {
		ui.Info("History is already empty")
		return
	}

	// Clear
	err = h.Clear()
	checkError(err)

	ui.Success(fmt.Sprintf("Cleared %d execution record(s)", count))
}
