package agent

import (
	"fmt"
	"time"

	"github.com/braunmar/worktree/pkg/config"
	"github.com/braunmar/worktree/pkg/queue"
)

// ProcessQueue runs the next pending task from the queue
func ProcessQueue(cfg *config.Config, workCfg *config.WorktreeConfig, q *queue.Queue) error {
	// Get next pending task
	task, err := q.Next()
	if err != nil {
		return fmt.Errorf("failed to get next task: %w", err)
	}

	if task == nil {
		return fmt.Errorf("no pending tasks in queue")
	}

	fmt.Printf("📋 Processing queued task\n")
	fmt.Printf("   ID: %s\n", task.ID)
	fmt.Printf("   Agent: %s\n", task.AgentName)
	fmt.Printf("   Worktree: %s\n", task.Worktree)
	fmt.Println()

	// Update status to running
	if err := q.UpdateStatus(task.ID, queue.StatusRunning, nil); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	// Get agent configuration
	agentTask, exists := workCfg.ScheduledAgents[task.AgentName]
	if !exists {
		updateErr := q.UpdateStatus(task.ID, queue.StatusFailed, fmt.Errorf("agent not found: %s", task.AgentName))
		if updateErr != nil {
			fmt.Printf("⚠️  Failed to update task status: %v\n", updateErr)
		}
		return fmt.Errorf("agent not found in configuration: %s", task.AgentName)
	}

	// Create executor
	executor := NewExecutor(cfg, workCfg, agentTask, task.AgentName)

	// Run task and track duration
	start := time.Now()
	execErr := executor.Run()
	duration := time.Since(start)

	// Update status based on result
	var finalStatus queue.TaskStatus
	if execErr != nil {
		finalStatus = queue.StatusFailed
		fmt.Printf("\n❌ Task failed after %s: %v\n", duration, execErr)
	} else {
		finalStatus = queue.StatusCompleted
		fmt.Printf("\n✅ Task completed successfully in %s\n", duration)
	}

	// Update queue with final status
	if err := q.UpdateStatus(task.ID, finalStatus, execErr); err != nil {
		return fmt.Errorf("failed to update final task status: %w", err)
	}

	return execErr
}

// ProcessQueueContinuous processes all pending tasks in sequence
func ProcessQueueContinuous(cfg *config.Config, workCfg *config.WorktreeConfig, q *queue.Queue) error {
	processedCount := 0
	failedCount := 0

	for {
		// Check if there are pending tasks
		pendingCount := q.Count(queue.StatusPending)
		if pendingCount == 0 {
			break
		}

		fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Printf("📊 Queue Status: %d pending, %d processed, %d failed\n", pendingCount, processedCount, failedCount)
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

		// Process next task
		err := ProcessQueue(cfg, workCfg, q)
		if err != nil {
			failedCount++
			fmt.Printf("⚠️  Continuing to next task after failure\n")
		} else {
			processedCount++
		}

		// Small delay between tasks
		time.Sleep(2 * time.Second)
	}

	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("🏁 Queue Processing Complete\n")
	fmt.Printf("   Total processed: %d\n", processedCount)
	fmt.Printf("   Failed: %d\n", failedCount)
	fmt.Printf("   Success rate: %.1f%%\n", float64(processedCount-failedCount)/float64(processedCount)*100)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	if failedCount > 0 {
		return fmt.Errorf("%d task(s) failed", failedCount)
	}

	return nil
}
