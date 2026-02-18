package history

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// makeRecord creates a test ExecutionRecord with sensible defaults
func makeRecord(agent, status string, durationMs int64) ExecutionRecord {
	start := time.Now()
	end := start.Add(time.Duration(durationMs) * time.Millisecond)
	return ExecutionRecord{
		ID:        "test-" + agent,
		AgentName: agent,
		Status:    status,
		StartTime: start,
		EndTime:   end,
		Duration:  durationMs,
	}
}

func TestLoadEmpty(t *testing.T) {
	dir := t.TempDir()
	h, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(h.Records) != 0 {
		t.Errorf("expected 0 records, got %d", len(h.Records))
	}
}

func TestLoadAndSave(t *testing.T) {
	dir := t.TempDir()
	h, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	h.path = filepath.Join(dir, ".history.json")

	record := makeRecord("npm-audit", "completed", 5000)
	h.Records = append(h.Records, record)

	if err := h.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Reload from disk
	h2, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() after save error = %v", err)
	}
	if len(h2.Records) != 1 {
		t.Fatalf("expected 1 record after reload, got %d", len(h2.Records))
	}
	if h2.Records[0].AgentName != "npm-audit" {
		t.Errorf("AgentName = %q, want %q", h2.Records[0].AgentName, "npm-audit")
	}
}

func TestRecord(t *testing.T) {
	dir := t.TempDir()
	h, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	h.path = filepath.Join(dir, ".history.json")

	t.Run("adds record and persists", func(t *testing.T) {
		rec := makeRecord("go-deps", "completed", 3000)
		if err := h.Record(rec); err != nil {
			t.Fatalf("Record() error = %v", err)
		}
		if len(h.Records) != 1 {
			t.Errorf("expected 1 record, got %d", len(h.Records))
		}
	})

	t.Run("enforces 1000 record limit", func(t *testing.T) {
		// Fill to over 1000
		for i := 0; i < 1005; i++ {
			h.Records = append(h.Records, makeRecord("agent", "completed", 100))
		}
		// One more Record() call should truncate
		if err := h.Record(makeRecord("overflow", "completed", 100)); err != nil {
			t.Fatalf("Record() error = %v", err)
		}
		if len(h.Records) != 1000 {
			t.Errorf("expected 1000 records after limit enforcement, got %d", len(h.Records))
		}
	})
}

func TestQuery(t *testing.T) {
	h := &History{
		Records: []ExecutionRecord{
			makeRecord("npm-audit", "completed", 1000),
			makeRecord("go-deps", "failed", 500),
			makeRecord("npm-audit", "failed", 200),
			makeRecord("go-version", "completed", 3000),
			makeRecord("npm-audit", "completed", 800),
		},
	}

	t.Run("no filters returns all newest-first", func(t *testing.T) {
		results := h.Query("", "", 0)
		if len(results) != 5 {
			t.Errorf("expected 5 results, got %d", len(results))
		}
		// Should be newest-first (last record first)
		if results[0].AgentName != "npm-audit" || results[0].Duration != 800 {
			t.Errorf("first result should be newest npm-audit(800ms), got %s(%dms)", results[0].AgentName, results[0].Duration)
		}
	})

	t.Run("filter by agent name", func(t *testing.T) {
		results := h.Query("npm-audit", "", 0)
		if len(results) != 3 {
			t.Errorf("expected 3 npm-audit records, got %d", len(results))
		}
		for _, r := range results {
			if r.AgentName != "npm-audit" {
				t.Errorf("unexpected agent %q in filtered results", r.AgentName)
			}
		}
	})

	t.Run("filter by status", func(t *testing.T) {
		results := h.Query("", "failed", 0)
		if len(results) != 2 {
			t.Errorf("expected 2 failed records, got %d", len(results))
		}
		for _, r := range results {
			if r.Status != "failed" {
				t.Errorf("unexpected status %q in filtered results", r.Status)
			}
		}
	})

	t.Run("filter by agent and status", func(t *testing.T) {
		results := h.Query("npm-audit", "completed", 0)
		if len(results) != 2 {
			t.Errorf("expected 2 npm-audit completed records, got %d", len(results))
		}
	})

	t.Run("limit applied", func(t *testing.T) {
		results := h.Query("", "", 2)
		if len(results) != 2 {
			t.Errorf("expected 2 results with limit=2, got %d", len(results))
		}
	})

	t.Run("limit 1 returns newest", func(t *testing.T) {
		results := h.Query("npm-audit", "", 1)
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		// Newest npm-audit has Duration=800
		if results[0].Duration != 800 {
			t.Errorf("expected newest record (800ms), got %dms", results[0].Duration)
		}
	})

	t.Run("empty history returns nil", func(t *testing.T) {
		empty := &History{Records: []ExecutionRecord{}}
		results := empty.Query("", "", 0)
		if len(results) != 0 {
			t.Errorf("expected empty results, got %d", len(results))
		}
	})
}

func TestStats(t *testing.T) {
	t.Run("empty history returns zero stats", func(t *testing.T) {
		h := &History{Records: []ExecutionRecord{}}
		stats := h.Stats()
		if stats.TotalExecutions != 0 {
			t.Errorf("TotalExecutions = %d, want 0", stats.TotalExecutions)
		}
		if stats.SuccessRate != 0 {
			t.Errorf("SuccessRate = %f, want 0", stats.SuccessRate)
		}
		if len(stats.ByAgent) != 0 {
			t.Errorf("ByAgent should be empty, got %d entries", len(stats.ByAgent))
		}
	})

	t.Run("all completed gives 100% success rate", func(t *testing.T) {
		h := &History{
			Records: []ExecutionRecord{
				makeRecord("agent-a", "completed", 1000),
				makeRecord("agent-a", "completed", 2000),
			},
		}
		stats := h.Stats()
		if stats.TotalExecutions != 2 {
			t.Errorf("TotalExecutions = %d, want 2", stats.TotalExecutions)
		}
		if stats.SuccessRate != 100.0 {
			t.Errorf("SuccessRate = %f, want 100.0", stats.SuccessRate)
		}
	})

	t.Run("all failed gives 0% success rate", func(t *testing.T) {
		h := &History{
			Records: []ExecutionRecord{
				makeRecord("agent-a", "failed", 500),
				makeRecord("agent-a", "failed", 500),
			},
		}
		stats := h.Stats()
		if stats.SuccessRate != 0.0 {
			t.Errorf("SuccessRate = %f, want 0.0", stats.SuccessRate)
		}
	})

	t.Run("mixed results with multiple agents", func(t *testing.T) {
		h := &History{
			Records: []ExecutionRecord{
				makeRecord("npm-audit", "completed", 1000),
				makeRecord("npm-audit", "failed", 500),
				makeRecord("go-deps", "completed", 2000),
				makeRecord("go-deps", "completed", 3000),
			},
		}
		stats := h.Stats()

		if stats.TotalExecutions != 4 {
			t.Errorf("TotalExecutions = %d, want 4", stats.TotalExecutions)
		}
		// 3 out of 4 completed = 75%
		if stats.SuccessRate != 75.0 {
			t.Errorf("SuccessRate = %f, want 75.0", stats.SuccessRate)
		}

		// Per-agent stats
		npmStats, ok := stats.ByAgent["npm-audit"]
		if !ok {
			t.Fatal("expected npm-audit in ByAgent")
		}
		if npmStats.TotalExecutions != 2 {
			t.Errorf("npm-audit TotalExecutions = %d, want 2", npmStats.TotalExecutions)
		}
		if npmStats.SuccessCount != 1 {
			t.Errorf("npm-audit SuccessCount = %d, want 1", npmStats.SuccessCount)
		}
		if npmStats.FailureCount != 1 {
			t.Errorf("npm-audit FailureCount = %d, want 1", npmStats.FailureCount)
		}
		if npmStats.SuccessRate != 50.0 {
			t.Errorf("npm-audit SuccessRate = %f, want 50.0", npmStats.SuccessRate)
		}

		goStats, ok := stats.ByAgent["go-deps"]
		if !ok {
			t.Fatal("expected go-deps in ByAgent")
		}
		if goStats.SuccessRate != 100.0 {
			t.Errorf("go-deps SuccessRate = %f, want 100.0", goStats.SuccessRate)
		}
	})

	t.Run("average duration is calculated correctly", func(t *testing.T) {
		h := &History{
			Records: []ExecutionRecord{
				makeRecord("agent", "completed", 1000),
				makeRecord("agent", "completed", 3000),
			},
		}
		stats := h.Stats()
		// (1000 + 3000) / 2 = 2000ms
		expectedAvg := 2000 * time.Millisecond
		if stats.AverageDuration != expectedAvg {
			t.Errorf("AverageDuration = %v, want %v", stats.AverageDuration, expectedAvg)
		}
	})
}

func TestClear(t *testing.T) {
	dir := t.TempDir()
	h := &History{
		Records: []ExecutionRecord{
			makeRecord("agent-a", "completed", 1000),
			makeRecord("agent-b", "failed", 500),
		},
		path: filepath.Join(dir, ".history.json"),
	}

	if err := h.Clear(); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	if len(h.Records) != 0 {
		t.Errorf("expected 0 records after Clear(), got %d", len(h.Records))
	}

	// Verify persisted
	h2, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(h2.Records) != 0 {
		t.Errorf("expected 0 records after reload, got %d", len(h2.Records))
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	historyPath := filepath.Join(dir, ".history.json")
	if err := os.WriteFile(historyPath, []byte("not-valid-json{{{"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(dir)
	if err == nil {
		t.Error("expected error for invalid JSON history file")
	}
}
