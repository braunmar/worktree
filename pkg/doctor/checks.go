package doctor

import (
	"worktree/pkg/config"
	"worktree/pkg/docker"
	"worktree/pkg/registry"
)

// RunHealthCheck runs all diagnostic checks and returns a comprehensive report
func RunHealthCheck(cfg *config.Config, workCfg *config.WorktreeConfig, reg *registry.Registry, opts Options) *Report {
	report := &Report{}

	// 1. Check Docker health
	report.Docker = CheckDocker()

	// 2. Check consistency (registry vs directories vs containers)
	report.Consistency = CheckConsistency(cfg, reg, workCfg.ProjectName)

	// 3. Get worktrees to check
	worktrees := filterWorktrees(reg.List(), opts.FeatureFilter)

	// 4. Check git status for each worktree
	for _, wt := range worktrees {
		gitReport := CheckGitStatus(cfg, wt, !opts.NoFetch)
		report.GitStatus = append(report.GitStatus, gitReport)
	}

	// 5. Check staleness for each worktree
	for _, wt := range worktrees {
		stalenessReport := CheckStaleness(cfg, wt, workCfg.ProjectName)
		// Only include worktrees with some staleness
		if stalenessReport.Score > 0 {
			report.Staleness = append(report.Staleness, stalenessReport)
		}
	}

	// 6. Check port allocations
	report.Ports = CheckPorts(reg, workCfg)

	// 7. Build summary
	report.Summary = buildSummary(report, reg, workCfg.ProjectName)

	// 8. Auto-fix if requested
	if opts.AutoFix {
		applyFixes(cfg, reg, report)
	}

	return report
}

// buildSummary calculates overall health metrics
func buildSummary(report *Report, reg *registry.Registry, projectName string) Summary {
	summary := Summary{
		TotalWorktrees: len(reg.List()),
	}

	// Count running worktrees
	for _, wt := range reg.List() {
		if docker.IsFeatureRunning(projectName, wt.Normalized) {
			summary.RunningWorktrees++
		}
	}

	// Count stale worktrees (score >= 2)
	for _, s := range report.Staleness {
		if s.Score >= 2 {
			summary.StaleWorktrees++
		}
	}

	// Count errors and warnings
	// Errors: orphaned registry entries, missing directories, ports out of range, behind main
	summary.ErrorsCount += len(report.Consistency.OrphanedRegistryEntries)
	summary.ErrorsCount += len(report.Ports.OutOfRange)

	for _, gs := range report.GitStatus {
		if gs.Error != "" || !gs.BranchExists {
			summary.ErrorsCount++
		} else if gs.BehindMain > 0 {
			summary.ErrorsCount++
		}
	}

	// Warnings: orphaned directories/containers, uncommitted changes, high staleness
	summary.WarningsCount += len(report.Consistency.OrphanedDirectories)
	summary.WarningsCount += len(report.Consistency.OrphanedContainers)
	summary.WarningsCount += len(report.Ports.Conflicts)

	for _, gs := range report.GitStatus {
		if gs.UncommittedCount > 0 {
			summary.WarningsCount++
		}
	}

	for _, s := range report.Staleness {
		if s.Score >= 2 {
			summary.WarningsCount++
		}
	}

	// Determine overall health status
	if summary.ErrorsCount > 0 {
		summary.HealthStatus = "POOR"
	} else if summary.WarningsCount > 0 {
		summary.HealthStatus = "FAIR"
	} else {
		summary.HealthStatus = "GOOD"
	}

	return summary
}

// applyFixes attempts to fix safe issues automatically
func applyFixes(cfg *config.Config, reg *registry.Registry, report *Report) {
	// Fix: Remove orphaned registry entries
	for _, orphan := range report.Consistency.OrphanedRegistryEntries {
		reg.Remove(orphan)
	}

	// Save registry if any fixes were applied
	if len(report.Consistency.OrphanedRegistryEntries) > 0 {
		reg.Save()
	}
}
