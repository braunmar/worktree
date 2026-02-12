package queue

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

type TaskStatus string

const (
	StatusPending   TaskStatus = "pending"
	StatusRunning   TaskStatus = "running"
	StatusCompleted TaskStatus = "completed"
	StatusFailed    TaskStatus = "failed"
)

// QueuedTask represents a task in the queue
type QueuedTask struct {
	ID          string     `json:"id"`                     // UUID
	AgentName   string     `json:"agent_name"`             // From .worktree.yml
	Worktree    string     `json:"worktree"`               // Feature name
	Status      TaskStatus `json:"status"`                 // Current status
	CreatedAt   time.Time  `json:"created_at"`             // When added to queue
	StartedAt   *time.Time `json:"started_at,omitempty"`   // When execution started
	CompletedAt *time.Time `json:"completed_at,omitempty"` // When execution completed
	Error       string     `json:"error,omitempty"`        // Error message if failed
	Duration    int64      `json:"duration_ms,omitempty"`  // Duration in milliseconds
}

// Queue manages the task queue
type Queue struct {
	Tasks []QueuedTask `json:"tasks"`
	mu    sync.RWMutex
	path  string
}

// Load loads queue from worktrees/.queue.json
func Load(worktreeDir string) (*Queue, error) {
	queuePath := filepath.Join(worktreeDir, ".queue.json")

	q := &Queue{
		Tasks: []QueuedTask{},
		path:  queuePath,
	}

	// If file doesn't exist, return empty queue
	if _, err := os.Stat(queuePath); os.IsNotExist(err) {
		return q, nil
	}

	// Read file
	data, err := os.ReadFile(queuePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read queue file: %w", err)
	}

	// Parse JSON
	if err := json.Unmarshal(data, q); err != nil {
		return nil, fmt.Errorf("failed to parse queue file: %w", err)
	}

	return q, nil
}

// Save persists queue atomically
func (q *Queue) Save() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Marshal to JSON
	data, err := json.MarshalIndent(q, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal queue: %w", err)
	}

	// Atomic write: write to temp file, then rename
	tempPath := q.path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp queue file: %w", err)
	}

	if err := os.Rename(tempPath, q.path); err != nil {
		os.Remove(tempPath) // Clean up temp file
		return fmt.Errorf("failed to rename queue file: %w", err)
	}

	return nil
}

// Add adds a task to the queue
func (q *Queue) Add(agentName, worktree string) (*QueuedTask, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	task := &QueuedTask{
		ID:        uuid.New().String(),
		AgentName: agentName,
		Worktree:  worktree,
		Status:    StatusPending,
		CreatedAt: time.Now(),
	}

	q.Tasks = append(q.Tasks, *task)

	// Save immediately
	if err := q.saveUnlocked(); err != nil {
		return nil, err
	}

	return task, nil
}

// Next returns the next pending task (first in queue)
func (q *Queue) Next() (*QueuedTask, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	for i := range q.Tasks {
		if q.Tasks[i].Status == StatusPending {
			return &q.Tasks[i], nil
		}
	}

	return nil, nil // No pending tasks
}

// UpdateStatus updates the status of a task
func (q *Queue) UpdateStatus(taskID string, status TaskStatus, taskErr error) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	for i := range q.Tasks {
		if q.Tasks[i].ID == taskID {
			q.Tasks[i].Status = status

			now := time.Now()

			switch status {
			case StatusRunning:
				q.Tasks[i].StartedAt = &now
			case StatusCompleted, StatusFailed:
				q.Tasks[i].CompletedAt = &now
				if q.Tasks[i].StartedAt != nil {
					q.Tasks[i].Duration = now.Sub(*q.Tasks[i].StartedAt).Milliseconds()
				}
				if taskErr != nil {
					q.Tasks[i].Error = taskErr.Error()
				}
			}

			// Save immediately
			return q.saveUnlocked()
		}
	}

	return fmt.Errorf("task not found: %s", taskID)
}

// List returns all tasks, optionally filtered by status
func (q *Queue) List(status TaskStatus) []QueuedTask {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if status == "" {
		// Return all tasks
		tasks := make([]QueuedTask, len(q.Tasks))
		copy(tasks, q.Tasks)
		return tasks
	}

	// Filter by status
	var filtered []QueuedTask
	for _, task := range q.Tasks {
		if task.Status == status {
			filtered = append(filtered, task)
		}
	}
	return filtered
}

// Remove removes a task from the queue
func (q *Queue) Remove(taskID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	for i := range q.Tasks {
		if q.Tasks[i].ID == taskID {
			// Remove task
			q.Tasks = append(q.Tasks[:i], q.Tasks[i+1:]...)
			return q.saveUnlocked()
		}
	}

	return fmt.Errorf("task not found: %s", taskID)
}

// Clear removes all completed and failed tasks
func (q *Queue) Clear() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Keep only pending and running tasks
	var activeTasks []QueuedTask
	for _, task := range q.Tasks {
		if task.Status == StatusPending || task.Status == StatusRunning {
			activeTasks = append(activeTasks, task)
		}
	}

	q.Tasks = activeTasks
	return q.saveUnlocked()
}

// saveUnlocked saves without locking (assumes caller has lock)
func (q *Queue) saveUnlocked() error {
	// Marshal to JSON
	data, err := json.MarshalIndent(q, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal queue: %w", err)
	}

	// Atomic write
	tempPath := q.path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp queue file: %w", err)
	}

	if err := os.Rename(tempPath, q.path); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename queue file: %w", err)
	}

	return nil
}

// Count returns count of tasks by status
func (q *Queue) Count(status TaskStatus) int {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if status == "" {
		return len(q.Tasks)
	}

	count := 0
	for _, task := range q.Tasks {
		if task.Status == status {
			count++
		}
	}
	return count
}
