package doctor

import (
	"time"
	"worktree/pkg/registry"
)

// Options for running health checks
type Options struct {
	FeatureFilter string
	NoFetch       bool
	AutoFix       bool
}

// Report contains all diagnostic results
type Report struct {
	Docker      DockerHealth
	Consistency ConsistencyReport
	GitStatus   []GitStatusReport
	Staleness   []StalenessReport
	Ports       PortReport
	Summary     Summary
}

// DockerHealth contains Docker availability status
type DockerHealth struct {
	Installed        bool
	Running          bool
	ComposeAvailable bool
	Version          string
	Error            string
}

// ConsistencyReport contains registry/directory/container consistency issues
type ConsistencyReport struct {
	OrphanedRegistryEntries []string // In registry but no directory
	OrphanedDirectories     []string // Directory exists but not in registry
	OrphanedContainers      []string // Containers running but not in registry
	InvalidWorktrees        []string // Directory exists but not valid git worktree
}

// GitStatusReport contains git status for a single worktree
type GitStatusReport struct {
	Feature          string
	Branch           string
	UncommittedCount int
	BehindMain       int
	AheadOrigin      int
	BranchExists     bool
	YoloMode         bool
	Error            string
}

// StalenessReport contains staleness metrics for a worktree
type StalenessReport struct {
	Feature           string
	Branch            string
	LastModified      time.Time
	DaysSinceModified int
	BranchMerged      bool
	MergedDate        string
	NoContainers      bool
	Score             int // 0-3 based on criteria met
}

// PortReport contains port allocation status
type PortReport struct {
	Conflicts      []PortConflict
	OutOfRange     []PortOutOfRange
	TotalAllocated int
	TotalAvailable int
	PortRanges     map[string]PortRangeInfo
}

// PortConflict represents a port that's allocated but in use
type PortConflict struct {
	Service string
	Port    int
	Feature string
}

// PortOutOfRange represents a port allocation outside configured ranges
type PortOutOfRange struct {
	Service string
	Port    int
	Feature string
	Range   [2]int
}

// PortRangeInfo contains information about a port range
type PortRangeInfo struct {
	Min       int
	Max       int
	Allocated int
	Available int
}

// Summary contains overall health metrics
type Summary struct {
	TotalWorktrees   int
	RunningWorktrees int
	WarningsCount    int
	ErrorsCount      int
	StaleWorktrees   int
	HealthStatus     string // GOOD, FAIR, POOR
}

// Helper function to filter worktrees by feature name
func filterWorktrees(worktrees []*registry.Worktree, filter string) []*registry.Worktree {
	if filter == "" {
		return worktrees
	}

	filtered := []*registry.Worktree{}
	for _, wt := range worktrees {
		if wt.Normalized == filter {
			filtered = append(filtered, wt)
			break
		}
	}
	return filtered
}
