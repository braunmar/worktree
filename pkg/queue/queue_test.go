package queue

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// newTestQueue creates a Queue backed by a temp directory
func newTestQueue(t *testing.T) *Queue {
	t.Helper()
	dir := t.TempDir()
	q, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	q.path = filepath.Join(dir, ".queue.json")
	return q
}

func TestLoadEmpty(t *testing.T) {
	dir := t.TempDir()
	q, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(q.Tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(q.Tasks))
	}
}

func TestLoadAndSave(t *testing.T) {
	dir := t.TempDir()
	q, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	q.path = filepath.Join(dir, ".queue.json")
	q.Tasks = append(q.Tasks, QueuedTask{
		ID:        "test-id",
		AgentName: "npm-audit",
		Worktree:  "feature-x",
		Status:    StatusPending,
		CreatedAt: time.Now(),
	})

	if err := q.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	q2, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() after save error = %v", err)
	}
	if len(q2.Tasks) != 1 {
		t.Fatalf("expected 1 task after reload, got %d", len(q2.Tasks))
	}
	if q2.Tasks[0].AgentName != "npm-audit" {
		t.Errorf("AgentName = %q, want %q", q2.Tasks[0].AgentName, "npm-audit")
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".queue.json"), []byte("{bad json"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(dir)
	if err == nil {
		t.Error("expected error for invalid JSON queue file")
	}
}

func TestAdd(t *testing.T) {
	q := newTestQueue(t)

	task, err := q.Add("npm-audit", "feature-x")
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if task.ID == "" {
		t.Error("expected non-empty task ID")
	}
	if task.AgentName != "npm-audit" {
		t.Errorf("AgentName = %q, want %q", task.AgentName, "npm-audit")
	}
	if task.Worktree != "feature-x" {
		t.Errorf("Worktree = %q, want %q", task.Worktree, "feature-x")
	}
	if task.Status != StatusPending {
		t.Errorf("Status = %q, want %q", task.Status, StatusPending)
	}
	if task.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
	if len(q.Tasks) != 1 {
		t.Errorf("expected 1 task in queue, got %d", len(q.Tasks))
	}
}

func TestNext(t *testing.T) {
	t.Run("returns nil when no pending tasks", func(t *testing.T) {
		q := &Queue{Tasks: []QueuedTask{}}
		task, err := q.Next()
		if err != nil {
			t.Fatalf("Next() error = %v", err)
		}
		if task != nil {
			t.Errorf("expected nil, got %+v", task)
		}
	})

	t.Run("returns nil when only running/completed tasks", func(t *testing.T) {
		q := &Queue{Tasks: []QueuedTask{
			{ID: "a", Status: StatusRunning},
			{ID: "b", Status: StatusCompleted},
			{ID: "c", Status: StatusFailed},
		}}
		task, err := q.Next()
		if err != nil {
			t.Fatalf("Next() error = %v", err)
		}
		if task != nil {
			t.Errorf("expected nil, got %+v", task)
		}
	})

	t.Run("returns first pending task", func(t *testing.T) {
		q := &Queue{Tasks: []QueuedTask{
			{ID: "running", Status: StatusRunning},
			{ID: "first-pending", AgentName: "agent-1", Status: StatusPending},
			{ID: "second-pending", AgentName: "agent-2", Status: StatusPending},
		}}
		task, err := q.Next()
		if err != nil {
			t.Fatalf("Next() error = %v", err)
		}
		if task == nil {
			t.Fatal("expected a task, got nil")
		}
		if task.ID != "first-pending" {
			t.Errorf("expected first-pending, got %q", task.ID)
		}
	})
}

func TestUpdateStatus(t *testing.T) {
	t.Run("transitions to running sets StartedAt", func(t *testing.T) {
		q := newTestQueue(t)
		task, _ := q.Add("agent", "worktree")

		if err := q.UpdateStatus(task.ID, StatusRunning, nil); err != nil {
			t.Fatalf("UpdateStatus() error = %v", err)
		}

		next, _ := q.Next() // No more pending
		if next != nil {
			t.Error("expected no pending tasks after status update to running")
		}

		// Verify StartedAt was set
		list := q.List(StatusRunning)
		if len(list) != 1 {
			t.Fatalf("expected 1 running task, got %d", len(list))
		}
		if list[0].StartedAt == nil {
			t.Error("StartedAt should be set when transitioning to running")
		}
	})

	t.Run("transitions to completed sets CompletedAt and Duration", func(t *testing.T) {
		q := newTestQueue(t)
		task, _ := q.Add("agent", "worktree")
		q.UpdateStatus(task.ID, StatusRunning, nil)
		time.Sleep(5 * time.Millisecond) // ensure measurable duration
		q.UpdateStatus(task.ID, StatusCompleted, nil)

		list := q.List(StatusCompleted)
		if len(list) != 1 {
			t.Fatalf("expected 1 completed task, got %d", len(list))
		}
		if list[0].CompletedAt == nil {
			t.Error("CompletedAt should be set when completed")
		}
		if list[0].Duration <= 0 {
			t.Errorf("Duration should be > 0 after running, got %d ms", list[0].Duration)
		}
	})

	t.Run("transitions to failed with error message", func(t *testing.T) {
		q := newTestQueue(t)
		task, _ := q.Add("agent", "worktree")
		q.UpdateStatus(task.ID, StatusRunning, nil)
		testErr := errors.New("something went wrong")
		q.UpdateStatus(task.ID, StatusFailed, testErr)

		list := q.List(StatusFailed)
		if len(list) != 1 {
			t.Fatalf("expected 1 failed task, got %d", len(list))
		}
		if list[0].Error != "something went wrong" {
			t.Errorf("Error = %q, want %q", list[0].Error, "something went wrong")
		}
	})

	t.Run("returns error for unknown task ID", func(t *testing.T) {
		q := newTestQueue(t)
		err := q.UpdateStatus("nonexistent-id", StatusRunning, nil)
		if err == nil {
			t.Error("expected error for unknown task ID")
		}
	})
}

func TestList(t *testing.T) {
	q := &Queue{Tasks: []QueuedTask{
		{ID: "p1", Status: StatusPending},
		{ID: "r1", Status: StatusRunning},
		{ID: "c1", Status: StatusCompleted},
		{ID: "f1", Status: StatusFailed},
		{ID: "p2", Status: StatusPending},
	}}

	t.Run("list all (empty status)", func(t *testing.T) {
		all := q.List("")
		if len(all) != 5 {
			t.Errorf("expected 5 tasks, got %d", len(all))
		}
	})

	t.Run("list pending", func(t *testing.T) {
		pending := q.List(StatusPending)
		if len(pending) != 2 {
			t.Errorf("expected 2 pending, got %d", len(pending))
		}
	})

	t.Run("list running", func(t *testing.T) {
		running := q.List(StatusRunning)
		if len(running) != 1 {
			t.Errorf("expected 1 running, got %d", len(running))
		}
	})

	t.Run("list completed", func(t *testing.T) {
		completed := q.List(StatusCompleted)
		if len(completed) != 1 {
			t.Errorf("expected 1 completed, got %d", len(completed))
		}
	})

	t.Run("list failed", func(t *testing.T) {
		failed := q.List(StatusFailed)
		if len(failed) != 1 {
			t.Errorf("expected 1 failed, got %d", len(failed))
		}
	})

	t.Run("list returns copy not reference", func(t *testing.T) {
		all := q.List("")
		if len(all) != 5 {
			t.Fatalf("expected 5 tasks")
		}
		// Modifying the copy should not affect the original
		all[0].Status = StatusFailed
		original := q.List(StatusPending)
		if len(original) != 2 {
			t.Error("modifying List() result should not affect original queue")
		}
	})
}

func TestRemove(t *testing.T) {
	t.Run("removes existing task", func(t *testing.T) {
		q := newTestQueue(t)
		task, _ := q.Add("agent", "worktree")

		if err := q.Remove(task.ID); err != nil {
			t.Fatalf("Remove() error = %v", err)
		}
		if len(q.Tasks) != 0 {
			t.Errorf("expected 0 tasks after remove, got %d", len(q.Tasks))
		}
	})

	t.Run("returns error for unknown task", func(t *testing.T) {
		q := newTestQueue(t)
		err := q.Remove("nonexistent")
		if err == nil {
			t.Error("expected error when removing nonexistent task")
		}
	})
}

func TestClear(t *testing.T) {
	q := newTestQueue(t)

	// Add tasks in various states
	task1, _ := q.Add("agent-a", "worktree")
	task2, _ := q.Add("agent-b", "worktree")
	task3, _ := q.Add("agent-c", "worktree")

	q.UpdateStatus(task1.ID, StatusRunning, nil)
	q.UpdateStatus(task2.ID, StatusCompleted, nil)
	q.UpdateStatus(task3.ID, StatusFailed, nil)

	// Add another pending
	q.Add("agent-d", "worktree")

	if err := q.Clear(); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	// Should have: task1 (running) + agent-d (pending) = 2
	if len(q.Tasks) != 2 {
		t.Errorf("expected 2 active tasks after Clear(), got %d", len(q.Tasks))
	}
	for _, task := range q.Tasks {
		if task.Status != StatusPending && task.Status != StatusRunning {
			t.Errorf("unexpected status %q after Clear()", task.Status)
		}
	}
}

func TestCount(t *testing.T) {
	q := &Queue{Tasks: []QueuedTask{
		{ID: "p1", Status: StatusPending},
		{ID: "p2", Status: StatusPending},
		{ID: "r1", Status: StatusRunning},
		{ID: "c1", Status: StatusCompleted},
	}}

	if n := q.Count(""); n != 4 {
		t.Errorf("Count(\"\") = %d, want 4", n)
	}
	if n := q.Count(StatusPending); n != 2 {
		t.Errorf("Count(pending) = %d, want 2", n)
	}
	if n := q.Count(StatusRunning); n != 1 {
		t.Errorf("Count(running) = %d, want 1", n)
	}
	if n := q.Count(StatusCompleted); n != 1 {
		t.Errorf("Count(completed) = %d, want 1", n)
	}
	if n := q.Count(StatusFailed); n != 0 {
		t.Errorf("Count(failed) = %d, want 0", n)
	}
}
