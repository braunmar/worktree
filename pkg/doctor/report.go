package doctor

import (
	"encoding/json"
	"fmt"
	"strings"
	"worktree/pkg/ui"
)

// Print outputs the report in human-readable format
func (r *Report) Print() {
	ui.PrintHeader("🏥 Worktree Doctor - Health Check Report")
	ui.NewLine()

	printSeparator()
	r.printDockerHealth()

	printSeparator()
	r.printConsistency()

	printSeparator()
	r.printGitStatus()

	printSeparator()
	r.printStaleness()

	printSeparator()
	r.printPorts()

	printSeparator()
	r.printSummary()

	ui.NewLine()
}

// ToJSON outputs the report in JSON format
func (r *Report) ToJSON() string {
	data, _ := json.MarshalIndent(r, "", "  ")
	return string(data)
}

// ExitCode returns the appropriate exit code based on report status
func (r *Report) ExitCode() int {
	if r.Summary.ErrorsCount > 0 {
		return 2
	}
	if r.Summary.WarningsCount > 0 {
		return 1
	}
	return 0
}

func printSeparator() {
	fmt.Println(strings.Repeat("━", 70))
	ui.NewLine()
}

func (r *Report) printDockerHealth() {
	ui.Section("🐳 DOCKER HEALTH")

	if !r.Docker.Installed {
		ui.Error("Docker not installed or not in PATH")
		if r.Docker.Error != "" {
			fmt.Printf("  %s\n", r.Docker.Error)
		}
		return
	}

	ui.Success(fmt.Sprintf("Docker installed (%s)", r.Docker.Version))

	if !r.Docker.Running {
		ui.Error("Docker daemon not running")
		ui.Info("💡 Start Docker Desktop and try again")
		return
	}

	ui.Success("Docker daemon running")

	if r.Docker.ComposeAvailable {
		ui.Success("Docker Compose available")
	} else {
		ui.Warning("Docker Compose not available")
	}
}

func (r *Report) printConsistency() {
	ui.Section("📁 WORKTREE CONSISTENCY")

	allGood := true

	// Orphaned registry entries
	if len(r.Consistency.OrphanedRegistryEntries) > 0 {
		allGood = false
		ui.Error(fmt.Sprintf("%d orphaned registry entries (no directory):",
			len(r.Consistency.OrphanedRegistryEntries)))
		for _, entry := range r.Consistency.OrphanedRegistryEntries {
			fmt.Printf("    - %s\n", entry)
		}
		ui.Info("💡 Fix: Run 'worktree doctor --fix' to clean up registry")
		ui.NewLine()
	}

	// Orphaned directories
	if len(r.Consistency.OrphanedDirectories) > 0 {
		allGood = false
		ui.Warning(fmt.Sprintf("%d orphaned directories (not in registry):",
			len(r.Consistency.OrphanedDirectories)))
		for _, dir := range r.Consistency.OrphanedDirectories {
			fmt.Printf("    - %s\n", dir)
		}
		ui.Info("💡 Add to registry with 'worktree new-feature' or remove manually")
		ui.NewLine()
	}

	// Orphaned containers
	if len(r.Consistency.OrphanedContainers) > 0 {
		allGood = false
		ui.Warning(fmt.Sprintf("%d orphaned containers (not in registry):",
			len(r.Consistency.OrphanedContainers)))
		for _, container := range r.Consistency.OrphanedContainers {
			fmt.Printf("    - %s\n", container)
		}
		ui.Info("💡 Fix: Run 'docker container prune' to clean up")
		ui.NewLine()
	}

	if allGood {
		ui.Success("All registry entries, directories, and containers are consistent")
	}
}

func (r *Report) printGitStatus() {
	ui.Section("🌿 GIT STATUS")

	if len(r.GitStatus) == 0 {
		ui.Info("No worktrees to check")
		return
	}

	for _, gs := range r.GitStatus {
		ui.NewLine()
		fmt.Printf("  %s (%s)\n", gs.Feature, gs.Branch)

		if gs.Error != "" {
			ui.Error(fmt.Sprintf("    %s", gs.Error))
			continue
		}

		// Uncommitted changes
		if gs.UncommittedCount > 0 {
			ui.Warning(fmt.Sprintf("    %d uncommitted changes", gs.UncommittedCount))
		} else {
			ui.Success("    Clean working tree")
		}

		// Behind main
		if gs.BehindMain > 0 {
			ui.Error(fmt.Sprintf("    %d commits behind origin/main", gs.BehindMain))
			ui.Info("    💡 Pull latest changes and rebase")
		} else {
			ui.Success("    Up to date with origin/main")
		}

		// Ahead of origin
		if gs.AheadOrigin > 0 {
			ui.Info(fmt.Sprintf("    %d commits ahead of origin (unpushed)", gs.AheadOrigin))
		}
	}
}

func (r *Report) printStaleness() {
	ui.Section("💤 STALE WORKTREES")

	if len(r.Staleness) == 0 {
		ui.Success("No stale worktrees detected")
		return
	}

	for _, s := range r.Staleness {
		ui.NewLine()
		fmt.Printf("  %s (%s)\n", s.Feature, s.Feature)

		if s.DaysSinceModified >= 7 {
			ui.Warning(fmt.Sprintf("    🕐 Last modified: %d days ago", s.DaysSinceModified))
		}

		if s.BranchMerged {
			ui.Warning("    🔀 Branch merged to main")
			if s.MergedDate != "" {
				fmt.Printf("       Merged: %s\n", s.MergedDate)
			}
		}

		if s.NoContainers {
			ui.Info("    💤 No containers running")
		}

		// Staleness score
		scoreLabel := "LOW"
		scoreEmoji := "ℹ️"
		if s.Score >= 3 {
			scoreLabel = "HIGH"
			scoreEmoji = "❌"
		} else if s.Score >= 2 {
			scoreLabel = "MEDIUM"
			scoreEmoji = "⚠️"
		}

		fmt.Printf("    %s Staleness score: %s (%d/3 criteria)\n", scoreEmoji, scoreLabel, s.Score)

		if s.Score >= 2 {
			ui.Info(fmt.Sprintf("    💡 Consider removing: worktree remove %s", s.Feature))
		}
	}
}

func (r *Report) printPorts() {
	ui.Section("🔌 PORT ALLOCATIONS")

	// Port conflicts
	if len(r.Ports.Conflicts) > 0 {
		ui.Warning(fmt.Sprintf("%d port conflicts detected:", len(r.Ports.Conflicts)))
		for _, c := range r.Ports.Conflicts {
			fmt.Printf("    %s: port %d (%s)\n", c.Service, c.Port, c.Feature)
		}
		ui.NewLine()
	}

	// Out of range ports
	if len(r.Ports.OutOfRange) > 0 {
		ui.Error(fmt.Sprintf("%d ports outside configured ranges:", len(r.Ports.OutOfRange)))
		for _, o := range r.Ports.OutOfRange {
			fmt.Printf("    %s: port %d (range: %d-%d) - %s\n",
				o.Service, o.Port, o.Range[0], o.Range[1], o.Feature)
		}
		ui.NewLine()
	}

	if len(r.Ports.Conflicts) == 0 && len(r.Ports.OutOfRange) == 0 {
		ui.Success("All allocated ports within configured ranges")
	}

	// Show available ports
	if len(r.Ports.PortRanges) > 0 {
		ui.NewLine()
		ui.Info("Available ports:")
		for service, info := range r.Ports.PortRanges {
			fmt.Printf("    %-18s %d-%d (%d allocated, %d available)\n",
				service+":", info.Min, info.Max, info.Allocated, info.Available)
		}
	}
}

func (r *Report) printSummary() {
	ui.Section("📊 SUMMARY")

	ui.PrintStatusLine("Total worktrees", fmt.Sprintf("%d", r.Summary.TotalWorktrees))
	ui.PrintStatusLine("Running worktrees", fmt.Sprintf("%d", r.Summary.RunningWorktrees))
	ui.PrintStatusLine("Stale worktrees", fmt.Sprintf("%d", r.Summary.StaleWorktrees))
	ui.NewLine()

	// Health status
	statusEmoji := "✅"
	if r.Summary.HealthStatus == "POOR" {
		statusEmoji = "❌"
	} else if r.Summary.HealthStatus == "FAIR" {
		statusEmoji = "⚠️"
	}

	issueCount := r.Summary.ErrorsCount + r.Summary.WarningsCount
	fmt.Printf("Overall health: %s %s", r.Summary.HealthStatus, statusEmoji)
	if issueCount > 0 {
		fmt.Printf(" (%d errors, %d warnings)", r.Summary.ErrorsCount, r.Summary.WarningsCount)
	}
	fmt.Println()

	// Suggestions
	if r.Summary.ErrorsCount > 0 || r.Summary.WarningsCount > 0 {
		ui.NewLine()
		ui.Info("💡 Run 'worktree doctor --fix' to auto-fix safe issues")
	}
}
