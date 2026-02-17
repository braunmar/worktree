package doctor

import (
	"os"
	"github.com/braunmar/worktree/pkg/config"
	"github.com/braunmar/worktree/pkg/docker"
	"github.com/braunmar/worktree/pkg/registry"
)

// CheckConsistency checks for mismatches between registry, directories, and containers
func CheckConsistency(cfg *config.Config, reg *registry.Registry, projectName string) ConsistencyReport {
	report := ConsistencyReport{}

	// Check registry entries have directories
	for _, wt := range reg.List() {
		if !cfg.WorktreeExists(wt.Normalized) {
			report.OrphanedRegistryEntries = append(report.OrphanedRegistryEntries, wt.Normalized)
		}
	}

	// Check directories have registry entries
	worktreeDir := cfg.WorktreeDir
	entries, err := os.ReadDir(worktreeDir)
	if err == nil {
		for _, entry := range entries {
			// Skip hidden files and the registry file itself
			if entry.IsDir() && entry.Name()[0] != '.' {
				if _, exists := reg.Get(entry.Name()); !exists {
					report.OrphanedDirectories = append(report.OrphanedDirectories, entry.Name())
				}
			}
		}
	}

	// Check for orphaned containers (only if Docker is running)
	runningFeatures, err := docker.GetRunningFeatures(projectName)
	if err == nil {
		for _, feature := range runningFeatures {
			if _, exists := reg.Get(feature); !exists {
				report.OrphanedContainers = append(report.OrphanedContainers, feature)
			}
		}
	}

	return report
}
