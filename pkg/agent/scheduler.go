package agent

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/braunmar/worktree/pkg/config"

	"github.com/robfig/cron/v3"
)

// Scheduler manages scheduled agent tasks
type Scheduler struct {
	cfg      *config.Config
	workCfg  *config.WorktreeConfig
	cron     *cron.Cron
	logFile  *os.File
	mu       sync.Mutex
	running  map[string]bool // Track running tasks to prevent overlaps
	stopChan chan struct{}
}

// NewScheduler creates a new agent scheduler
func NewScheduler(cfg *config.Config, workCfg *config.WorktreeConfig) (*Scheduler, error) {
	// Set up logging
	logDir := filepath.Join(os.Getenv("HOME"), "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	logPath := filepath.Join(logDir, "worktree-scheduler.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// Create cron scheduler (standard 5-field cron format: minute hour day month weekday)
	cronScheduler := cron.New()

	return &Scheduler{
		cfg:      cfg,
		workCfg:  workCfg,
		cron:     cronScheduler,
		logFile:  logFile,
		running:  make(map[string]bool),
		stopChan: make(chan struct{}),
	}, nil
}

// Start starts the scheduler daemon
func (s *Scheduler) Start(ctx context.Context) error {
	// Redirect logs to file
	log.SetOutput(s.logFile)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("Worktree Agent Scheduler Starting")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Load and schedule all agent tasks
	if s.workCfg.ScheduledAgents == nil || len(s.workCfg.ScheduledAgents) == 0 {
		return fmt.Errorf("no scheduled agents configured in .worktree.yml")
	}

	log.Printf("Loading %d agent tasks...\n", len(s.workCfg.ScheduledAgents))

	// Add each agent to the scheduler
	for taskName, task := range s.workCfg.ScheduledAgents {
		if err := s.addTask(taskName, task); err != nil {
			log.Printf("ERROR: Failed to schedule task '%s': %v\n", taskName, err)
			continue
		}
		log.Printf("✓ Scheduled: %s (%s) - cron: %s\n", task.Name, taskName, task.Schedule)
	}

	// Start the cron scheduler
	s.cron.Start()
	log.Println("Scheduler started successfully")
	log.Printf("Scheduled tasks: %d\n", len(s.cron.Entries()))
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Print next run times
	for _, entry := range s.cron.Entries() {
		log.Printf("Next run: %s\n", entry.Next.Format("2006-01-02 15:04:05"))
	}

	log.Println()
	log.Println("Waiting for scheduled tasks... (Press Ctrl+C to stop)")
	log.Println()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		log.Printf("Received signal: %v\n", sig)
		return s.Stop()
	case <-ctx.Done():
		log.Println("Context cancelled")
		return s.Stop()
	case <-s.stopChan:
		log.Println("Stop requested")
		return nil
	}
}

// Stop stops the scheduler daemon
func (s *Scheduler) Stop() error {
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("Stopping scheduler...")

	// Stop accepting new jobs
	ctx := s.cron.Stop()

	// Wait for running jobs to complete (with timeout)
	select {
	case <-ctx.Done():
		log.Println("All jobs completed")
	case <-time.After(30 * time.Second):
		log.Println("Timeout waiting for jobs to complete")
	}

	// Close log file
	if s.logFile != nil {
		s.logFile.Sync()
		s.logFile.Close()
	}

	log.Println("Scheduler stopped")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	close(s.stopChan)
	return nil
}

// addTask adds a task to the scheduler
func (s *Scheduler) addTask(taskName string, task *config.AgentTask) error {
	// Validate cron expression
	if task.Schedule == "" {
		return fmt.Errorf("schedule is empty")
	}

	// Create a job wrapper that prevents overlapping runs
	job := func() {
		s.mu.Lock()
		if s.running[taskName] {
			log.Printf("⚠️  Skipping '%s' - previous run still in progress\n", taskName)
			s.mu.Unlock()
			return
		}
		s.running[taskName] = true
		s.mu.Unlock()

		// Run the task
		s.runTask(taskName, task)

		s.mu.Lock()
		s.running[taskName] = false
		s.mu.Unlock()
	}

	// Add to cron scheduler
	_, err := s.cron.AddFunc(task.Schedule, job)
	if err != nil {
		return fmt.Errorf("invalid cron expression '%s': %w", task.Schedule, err)
	}

	return nil
}

// runTask executes an agent task
func (s *Scheduler) runTask(taskName string, task *config.AgentTask) {
	startTime := time.Now()

	log.Println()
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("🤖 Running: %s (%s)\n", task.Name, taskName)
	log.Printf("   Started: %s\n", startTime.Format("2006-01-02 15:04:05"))
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Create executor
	executor := NewExecutor(s.cfg, s.workCfg, task, taskName)

	// Run the task
	err := executor.Run()

	// Calculate duration
	duration := time.Since(startTime)

	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	if err != nil {
		log.Printf("❌ Failed: %s (%s)\n", task.Name, taskName)
		log.Printf("   Error: %v\n", err)
		log.Printf("   Duration: %s\n", duration.Round(time.Second))
	} else {
		log.Printf("✅ Completed: %s (%s)\n", task.Name, taskName)
		log.Printf("   Duration: %s\n", duration.Round(time.Second))
	}
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println()

	// TODO: Send notifications based on err != nil
}

// GetNextRuns returns the next scheduled run times for all tasks
func (s *Scheduler) GetNextRuns() map[string]time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()

	nextRuns := make(map[string]time.Time)
	entries := s.cron.Entries()

	i := 0
	for taskName := range s.workCfg.ScheduledAgents {
		if i < len(entries) {
			nextRuns[taskName] = entries[i].Next
			i++
		}
	}

	return nextRuns
}

// IsRunning checks if a task is currently running
func (s *Scheduler) IsRunning(taskName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running[taskName]
}
