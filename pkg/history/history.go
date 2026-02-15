package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ExecutionRecord represents a single agent execution
type ExecutionRecord struct {
	ID            string    `json:"id"`
	AgentName     string    `json:"agent_name"`
	Worktree      string    `json:"worktree"`
	Status        string    `json:"status"` // "completed", "failed"
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time"`
	Duration      int64     `json:"duration_ms"`
	Error         string    `json:"error,omitempty"`
	StepsExecuted int       `json:"steps_executed,omitempty"`
	Commits       []string  `json:"commits,omitempty"`
	PRUrl         string    `json:"pr_url,omitempty"`
}

// History manages execution history
type History struct {
	Records []ExecutionRecord `json:"records"`
	mu      sync.RWMutex
	path    string
}

// AgentStats contains statistics for a specific agent
type AgentStats struct {
	TotalExecutions int
	SuccessCount    int
	FailureCount    int
	SuccessRate     float64
	AverageDuration time.Duration
}

// HistoryStats contains aggregate statistics
type HistoryStats struct {
	TotalExecutions int
	SuccessRate     float64
	AverageDuration time.Duration
	ByAgent         map[string]AgentStats
}

// Load loads history from worktrees/.history.json
func Load(worktreeDir string) (*History, error) {
	historyPath := filepath.Join(worktreeDir, ".history.json")

	h := &History{
		Records: []ExecutionRecord{},
		path:    historyPath,
	}

	// If file doesn't exist, return empty history
	if _, err := os.Stat(historyPath); os.IsNotExist(err) {
		return h, nil
	}

	// Read file
	data, err := os.ReadFile(historyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read history file: %w", err)
	}

	// Parse JSON
	if err := json.Unmarshal(data, h); err != nil {
		return nil, fmt.Errorf("failed to parse history file: %w", err)
	}

	return h, nil
}

// Save persists history atomically
func (h *History) Save() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.saveUnlocked()
}

// Record adds an execution record
func (h *History) Record(record ExecutionRecord) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.Records = append(h.Records, record)

	// Keep only last 1000 records to prevent unbounded growth
	if len(h.Records) > 1000 {
		h.Records = h.Records[len(h.Records)-1000:]
	}

	return h.saveUnlocked()
}

// Query filters records
func (h *History) Query(agentName string, status string, limit int) []ExecutionRecord {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var filtered []ExecutionRecord

	// Filter from newest to oldest
	for i := len(h.Records) - 1; i >= 0; i-- {
		record := h.Records[i]

		// Apply filters
		if agentName != "" && record.AgentName != agentName {
			continue
		}
		if status != "" && record.Status != status {
			continue
		}

		filtered = append(filtered, record)

		// Apply limit
		if limit > 0 && len(filtered) >= limit {
			break
		}
	}

	return filtered
}

// Stats returns aggregate statistics
func (h *History) Stats() HistoryStats {
	h.mu.RLock()
	defer h.mu.RUnlock()

	stats := HistoryStats{
		ByAgent: make(map[string]AgentStats),
	}

	if len(h.Records) == 0 {
		return stats
	}

	// Aggregate by agent
	agentRecords := make(map[string][]ExecutionRecord)
	for _, record := range h.Records {
		agentRecords[record.AgentName] = append(agentRecords[record.AgentName], record)
	}

	// Calculate overall stats
	var totalDuration int64
	successCount := 0

	for _, record := range h.Records {
		stats.TotalExecutions++
		totalDuration += record.Duration

		if record.Status == "completed" {
			successCount++
		}
	}

	if stats.TotalExecutions > 0 {
		stats.SuccessRate = float64(successCount) / float64(stats.TotalExecutions) * 100
		stats.AverageDuration = time.Duration(totalDuration/int64(stats.TotalExecutions)) * time.Millisecond
	}

	// Calculate per-agent stats
	for agentName, records := range agentRecords {
		agentStats := AgentStats{
			TotalExecutions: len(records),
		}

		var agentTotalDuration int64
		for _, record := range records {
			agentTotalDuration += record.Duration
			if record.Status == "completed" {
				agentStats.SuccessCount++
			} else {
				agentStats.FailureCount++
			}
		}

		if agentStats.TotalExecutions > 0 {
			agentStats.SuccessRate = float64(agentStats.SuccessCount) / float64(agentStats.TotalExecutions) * 100
			agentStats.AverageDuration = time.Duration(agentTotalDuration/int64(agentStats.TotalExecutions)) * time.Millisecond
		}

		stats.ByAgent[agentName] = agentStats
	}

	return stats
}

// saveUnlocked saves without locking (assumes caller has lock)
func (h *History) saveUnlocked() error {
	// Marshal to JSON
	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal history: %w", err)
	}

	// Atomic write
	tempPath := h.path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp history file: %w", err)
	}

	if err := os.Rename(tempPath, h.path); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename history file: %w", err)
	}

	return nil
}

// Clear removes all records
func (h *History) Clear() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.Records = []ExecutionRecord{}
	return h.saveUnlocked()
}
