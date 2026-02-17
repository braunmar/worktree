package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/braunmar/worktree/pkg/config"
)

// Executor manages the execution of a scheduled agent task
type Executor struct {
	cfg       *config.Config
	workCfg   *config.WorktreeConfig
	task      *config.AgentTask
	cleanup   bool
	agentName string
}

// NewExecutor creates a new agent executor
func NewExecutor(cfg *config.Config, workCfg *config.WorktreeConfig, task *config.AgentTask, agentName string) *Executor {
	return &Executor{
		cfg:       cfg,
		workCfg:   workCfg,
		task:      task,
		cleanup:   true,
		agentName: agentName,
	}
}

// Run executes the agent task
func (e *Executor) Run() error {
	fmt.Printf("🤖 Running agent task: %s\n", e.task.Name)
	fmt.Printf("   %s\n", e.task.Description)
	fmt.Println()

	// If GSD enabled, launch GSD workflow instead
	if e.task.GSD != nil && e.task.GSD.Enabled {
		return e.runGSDWorkflow()
	}

	// Phase 1: Execute steps
	if err := e.executeSteps(); err != nil {
		return fmt.Errorf("step execution failed: %w", err)
	}

	// Phase 2: Run safety gates
	if len(e.task.Safety.Gates) > 0 {
		fmt.Println()
		if err := e.runSafetyGates(); err != nil {
			// Rollback if enabled
			if e.task.Safety.Rollback.Enabled {
				fmt.Println()
				fmt.Printf("⚠️  Rolling back due to safety gate failures...\n")
				e.cleanupWorktree()
			}
			return fmt.Errorf("safety gates failed: %w", err)
		}
	}

	// Phase 3: Git operations
	if e.task.Safety.Git.Push.Enabled {
		fmt.Println()
		if err := e.commitAndPush(); err != nil {
			// Send failure notification
			e.sendNotifications(false, err)

			// Rollback if enabled
			if e.task.Safety.Rollback.Enabled {
				fmt.Println()
				fmt.Printf("⚠️  Rolling back due to git operation failure...\n")
				e.cleanupWorktree()
			}
			return fmt.Errorf("git operations failed: %w", err)
		}

		// Send success notification
		e.sendNotifications(true, nil)
	}

	fmt.Println()
	fmt.Printf("✅ Agent task '%s' completed successfully\n", e.task.Name)
	return nil
}

// executeSteps runs all configured steps
func (e *Executor) executeSteps() error {
	fmt.Println("📋 Executing steps...")
	fmt.Println()

	for i, step := range e.task.Steps {
		fmt.Printf("  [%d/%d] %s\n", i+1, len(e.task.Steps), step.Name)

		switch step.Type {
		case "shell":
			if err := e.executeShellStep(step); err != nil {
				return fmt.Errorf("step '%s' failed: %w", step.Name, err)
			}
		case "skill":
			if err := e.executeSkillStep(step); err != nil {
			return fmt.Errorf("step '%s' failed: %w", step.Name, err)
		}
		default:
			return fmt.Errorf("unknown step type: %s", step.Type)
		}

		fmt.Println()
	}

	return nil
}

// executeShellStep executes a shell command step
func (e *Executor) executeShellStep(step config.AgentStep) error {
	cmd := exec.Command("bash", "-c", step.Command)

	// Set working directory if specified
	if step.WorkingDir != "" {
		cmd.Dir = step.WorkingDir
	}

	// Connect stdout and stderr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command
	return cmd.Run()
}

// createWorktree creates a temporary agent worktree (placeholder for Phase 1)
func (e *Executor) createWorktree() error {
	fmt.Println("🔨 Creating agent worktree...")
	fmt.Printf("   Instance: %d\n", e.task.Context.Instance)
	fmt.Printf("   Preset: %s\n", e.task.Context.Preset)
	fmt.Printf("   YOLO mode: %v\n", e.task.Context.Yolo)
	fmt.Println()

	// TODO: Phase 1 - placeholder only
	// Actual worktree creation will be implemented later
	return nil
}

// runSafetyGates executes all configured safety gates
func (e *Executor) runSafetyGates() error {
	fmt.Println("🛡️  Running safety gates...")
	fmt.Println()

	var failedGates []string
	var warnings []string

	for i, gate := range e.task.Safety.Gates {
		requiredLabel := ""
		if gate.Required {
			requiredLabel = " (required)"
		} else {
			requiredLabel = " (optional)"
		}

		fmt.Printf("  [%d/%d] %s%s\n", i+1, len(e.task.Safety.Gates), gate.Name, requiredLabel)
		fmt.Printf("        Command: %s\n", gate.Command)

		// Execute the gate command
		cmd := exec.Command("bash", "-c", gate.Command)
		cmd.Dir = e.cfg.ProjectRoot

		// Capture output
		output, err := cmd.CombinedOutput()

		if err != nil {
			// Gate failed
			if gate.Required {
				fmt.Printf("        ❌ Failed (required)\n")
				failedGates = append(failedGates, gate.Name)
			} else {
				fmt.Printf("        ⚠️  Failed (optional - continuing)\n")
				warnings = append(warnings, gate.Name)
			}

			// Show error output if not too long
			if len(output) > 0 && len(output) < 500 {
				fmt.Printf("        Output: %s\n", string(output))
			}
		} else {
			// Gate passed
			fmt.Printf("        ✅ Passed\n")
		}

		fmt.Println()
	}

	// Summary
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Safety Gates Summary:")
	fmt.Printf("  Total: %d\n", len(e.task.Safety.Gates))
	fmt.Printf("  Passed: %d\n", len(e.task.Safety.Gates)-len(failedGates)-len(warnings))
	fmt.Printf("  Failed (required): %d\n", len(failedGates))
	fmt.Printf("  Failed (optional): %d\n", len(warnings))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// If any required gates failed, return error
	if len(failedGates) > 0 {
		fmt.Println()
		fmt.Printf("❌ Required safety gates failed:\n")
		for _, gate := range failedGates {
			fmt.Printf("   - %s\n", gate)
		}
		return fmt.Errorf("%d required safety gate(s) failed", len(failedGates))
	}

	// Show warnings for optional gates
	if len(warnings) > 0 {
		fmt.Println()
		fmt.Printf("⚠️  Optional safety gates failed (continuing anyway):\n")
		for _, gate := range warnings {
			fmt.Printf("   - %s\n", gate)
		}
	}

	return nil
}

// commitAndPush performs git operations
func (e *Executor) commitAndPush() error {
	fmt.Println("📝 Git Operations...")
	fmt.Println()

	// Replace {date} placeholder in branch name and messages
	dateStr := time.Now().Format("2006-01-02")
	branch := strings.ReplaceAll(e.task.Safety.Git.Branch, "{date}", dateStr)
	prTitle := strings.ReplaceAll(e.task.Safety.Git.Push.PRTitle, "{date}", dateStr)
	prBody := strings.ReplaceAll(e.task.Safety.Git.Push.PRBody, "{date}", dateStr)

	// Check if there are changes to commit
	fmt.Printf("  Checking for changes...\n")
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusCmd.Dir = e.cfg.ProjectRoot
	output, err := statusCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	}

	if len(output) == 0 {
		fmt.Printf("  ℹ️  No changes to commit\n")
		return nil
	}

	fmt.Printf("  ✅ Changes detected\n")
	fmt.Println()

	// Create and checkout branch
	fmt.Printf("  Creating branch: %s\n", branch)
	checkoutCmd := exec.Command("git", "checkout", "-b", branch)
	checkoutCmd.Dir = e.cfg.ProjectRoot
	if branchOutput, err := checkoutCmd.CombinedOutput(); err != nil {
		// Branch might already exist, try to checkout
		checkoutCmd = exec.Command("git", "checkout", branch)
		checkoutCmd.Dir = e.cfg.ProjectRoot
		if checkoutOutput, err := checkoutCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to checkout branch: %w\nOutput: %s", err, string(checkoutOutput))
		}
		_ = branchOutput // Ignore unused
	}
	fmt.Printf("  ✅ Branch created/checked out\n")
	fmt.Println()

	// Stage all changes
	fmt.Printf("  Staging changes...\n")
	addCmd := exec.Command("git", "add", ".")
	addCmd.Dir = e.cfg.ProjectRoot
	if output, err := addCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stage changes: %w\nOutput: %s", err, string(output))
	}
	fmt.Printf("  ✅ Changes staged\n")
	fmt.Println()

	// Commit
	fmt.Printf("  Creating commit...\n")
	commitMsg := e.task.Safety.Git.CommitMessage
	commitCmd := exec.Command("git", "commit", "-m", commitMsg)
	commitCmd.Dir = e.cfg.ProjectRoot
	if output, err := commitCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to commit: %w\nOutput: %s", err, string(output))
	}
	fmt.Printf("  ✅ Commit created\n")
	fmt.Println()

	// Push to remote
	fmt.Printf("  Pushing to remote...\n")
	pushCmd := exec.Command("git", "push", "-u", "origin", branch)
	pushCmd.Dir = e.cfg.ProjectRoot
	if output, err := pushCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to push: %w\nOutput: %s", err, string(output))
	}
	fmt.Printf("  ✅ Pushed to origin/%s\n", branch)
	fmt.Println()

	// Create PR if requested
	if e.task.Safety.Git.Push.CreatePR {
		fmt.Printf("  Creating pull request...\n")

		prCmd := exec.Command("gh", "pr", "create",
			"--title", prTitle,
			"--body", prBody,
			"--head", branch)
		prCmd.Dir = e.cfg.ProjectRoot

		output, err := prCmd.CombinedOutput()
		if err != nil {
			// Check if gh is installed
			if strings.Contains(err.Error(), "executable file not found") {
				fmt.Printf("  ⚠️  GitHub CLI (gh) not installed - skipping PR creation\n")
				fmt.Printf("      Install: brew install gh (macOS) or see https://cli.github.com\n")
			} else {
				return fmt.Errorf("failed to create PR: %w\nOutput: %s", err, string(output))
			}
		} else {
			prURL := strings.TrimSpace(string(output))
			fmt.Printf("  ✅ Pull request created: %s\n", prURL)
		}
	}

	fmt.Println()
	fmt.Printf("✅ Git operations completed successfully\n")
	return nil
}

// cleanupWorktree removes the agent worktree
func (e *Executor) cleanupWorktree() {
	fmt.Println("🧹 Cleaning up...")

	// Reset to main branch
	checkoutCmd := exec.Command("git", "checkout", e.task.Context.Branch)
	checkoutCmd.Dir = e.cfg.ProjectRoot
	if err := checkoutCmd.Run(); err != nil {
		fmt.Printf("  ⚠️  Failed to checkout %s: %v\n", e.task.Context.Branch, err)
	}

	// Discard all changes
	resetCmd := exec.Command("git", "reset", "--hard", "HEAD")
	resetCmd.Dir = e.cfg.ProjectRoot
	if err := resetCmd.Run(); err != nil {
		fmt.Printf("  ⚠️  Failed to reset: %v\n", err)
	}

	// Clean untracked files
	cleanCmd := exec.Command("git", "clean", "-fd")
	cleanCmd.Dir = e.cfg.ProjectRoot
	if err := cleanCmd.Run(); err != nil {
		fmt.Printf("  ⚠️  Failed to clean: %v\n", err)
	}

	fmt.Printf("  ✅ Cleanup completed\n")
}

// sendNotifications sends configured notifications
func (e *Executor) sendNotifications(success bool, err error) {
	var notifications []config.Notification

	if success {
		notifications = e.task.Notifications.OnSuccess
	} else {
		notifications = e.task.Notifications.OnFailure
	}

	if len(notifications) == 0 {
		return
	}

	fmt.Println()
	fmt.Println("📢 Sending notifications...")
	fmt.Println()

	for _, notification := range notifications {
		switch notification.Type {
		case "slack":
			e.sendSlackNotification(notification, success, err)
		case "gitlab_issue":
			fmt.Printf("  ⚠️  GitLab issue notifications not yet implemented\n")
		case "email":
			fmt.Printf("  ⚠️  Email notifications not yet implemented\n")
		default:
			fmt.Printf("  ⚠️  Unknown notification type: %s\n", notification.Type)
		}
	}
}

// sendSlackNotification sends a Slack webhook notification
func (e *Executor) sendSlackNotification(notification config.Notification, success bool, notifErr error) {
	// Get webhook URL from notification or environment
	webhookURL := notification.Recipients[0] // Webhook URL stored in recipients[0]
	if webhookURL == "" {
		fmt.Printf("  ⚠️  Slack webhook URL not configured\n")
		return
	}

	// Build message
	var message string
	var color string

	if success {
		color = "good" // Green
		message = fmt.Sprintf("✅ *%s* completed successfully", e.task.Name)
		if notification.Body != "" {
			message = notification.Body
		}
	} else {
		color = "danger" // Red
		message = fmt.Sprintf("❌ *%s* failed", e.task.Name)
		if notification.Body != "" {
			message = notification.Body
		}
		if notifErr != nil {
			message += fmt.Sprintf("\n\nError: `%v`", notifErr)
		}
	}

	// Replace placeholders
	dateStr := time.Now().Format("2006-01-02")
	message = strings.ReplaceAll(message, "{date}", dateStr)
	message = strings.ReplaceAll(message, "{task}", e.agentName)

	// Create Slack payload
	payload := map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"color":      color,
				"text":       message,
				"footer":     "Worktree Agent Scheduler",
				"footer_icon": "⚙️",
				"ts":         time.Now().Unix(),
			},
		},
	}

	// Add channel if specified
	if notification.Title != "" { // Using Title as channel name
		payload["channel"] = notification.Title
	}

	// Marshal to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("  ❌ Failed to create Slack payload: %v\n", err)
		return
	}

	// Send HTTP POST request
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		fmt.Printf("  ❌ Failed to send Slack notification: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("  ❌ Slack returned error: %s\n", resp.Status)
		return
	}

	fmt.Printf("  ✅ Slack notification sent\n")
}

// updateRegistry updates the last run time in the registry (placeholder)
func (e *Executor) updateRegistry() {
	// TODO: Update registry with last run time
	_ = time.Now() // Placeholder to avoid unused import warning
}

// runGSDWorkflow executes the agent task using GSD framework
func (e *Executor) runGSDWorkflow() error {
	var taskContent string
	var err error

	// Read .task.md if configured
	if e.task.GSD.ReadTaskFile {
		taskContent, err = ReadTaskFile(e.cfg.ProjectRoot)
		if err != nil {
			return fmt.Errorf("failed to read task file: %w", err)
		}

		if taskContent == "" {
			return fmt.Errorf(".task.md not found, but read_task_file is enabled")
		}

		fmt.Printf("📄 Task file loaded: .task.md\n")
		fmt.Printf("   Length: %d characters\n", len(taskContent))
		fmt.Println()
	}

	// Build GSD workflow
	workflow := GSDWorkflow{
		Milestone:   e.task.GSD.Milestone,
		Phase:       taskContent,
		AutoExecute: e.task.GSD.AutoExecute,
		YoloMode:    e.task.Context.Yolo,
	}

	// Launch GSD workflow
	err = LaunchGSDWorkflow(e.cfg, workflow)
	if err != nil {
		// Send failure notification
		e.sendNotifications(false, err)

		// Rollback if enabled
		if e.task.Safety.Rollback.Enabled {
			fmt.Println()
			fmt.Printf("⚠️  Rolling back due to GSD workflow failure...\n")
			e.cleanupWorktree()
		}

		return fmt.Errorf("GSD workflow failed: %w", err)
	}

	// Send success notification
	if len(e.task.Notifications.OnSuccess) > 0 {
		e.sendNotifications(true, nil)
	}

	fmt.Println()
	fmt.Printf("✅ GSD workflow completed successfully\n")
	return nil
}
