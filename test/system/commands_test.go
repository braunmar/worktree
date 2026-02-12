package system_test

// commands_test.go tests every CLI command that can run without external
// services (docker, running processes, network).  Tests are organised into
// three logical groups:
//
//   1. Config-only commands  – only need a valid .worktree.yml
//   2. Registry commands     – need a worktree created first
//   3. History / Queue       – need the worktrees/ dir and a valid config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── helpers shared across this file ──────────────────────────────────────────

// validAgentYAML is a fully-valid scheduled_agents block accepted by validate.
const validAgentYAML = `  valid-task:
    name: "Valid Task"
    description: "A correctly configured agent task"
    schedule: "0 9 * * MON"
    context:
      preset: default
      branch: "main"
      instance: 1
      yolo: false
    steps:
      - name: "Do something"
        type: shell
        command: "echo 'working'"
    safety:
      git:
        branch: "automated/valid-task"
        commit_message: "chore: valid task run"
        push:
          enabled: false
`

// invalidAgentYAML is missing branch, steps, and git config – fails validate.
const invalidAgentYAML = `  broken-task:
    name: "Broken Task"
    description: "Missing required fields"
    schedule: "0 9 * * MON"
    context:
      preset: default
      branch: ""
      instance: 1
      yolo: false
    safety:
      git:
        branch: ""
        commit_message: ""
        push:
          enabled: false
`

// ── Group 1: Config-only commands ────────────────────────────────────────────

// TestAgentList verifies "worktree agent list" when agents are configured.
func TestAgentList(t *testing.T) {
	env := newTestEnv(t)
	env.writeConfig(minimalConfig(validAgentYAML))

	out, err := env.run("agent", "list")
	t.Logf("output:\n%s", out)

	assertSuccess(t, out, err)
	assertContains(t, out, "Configured Agent Tasks")
	assertContains(t, out, "Valid Task")
	assertContains(t, out, "valid-task")
	assertContains(t, out, "every Monday at 9:00 AM")
}

// TestAgentListNoAgents verifies "worktree agent list" when no agents are configured.
func TestAgentListNoAgents(t *testing.T) {
	env := newTestEnv(t)
	// Config without scheduled_agents section.
	env.writeConfig(`project_name: "testproject"
projects:
  backend:
    dir: "backend"
    main_branch: "main"
  frontend:
    dir: "frontend"
    main_branch: "main"
presets:
  default:
    projects: ["backend", "frontend"]
default_preset: default
`)

	out, err := env.run("agent", "list")
	t.Logf("output:\n%s", out)

	assertSuccess(t, out, err)
	assertContains(t, out, "No scheduled agents configured")
}

// TestAgentValidateValid verifies a fully-configured task passes validation.
func TestAgentValidateValid(t *testing.T) {
	env := newTestEnv(t)
	env.writeConfig(minimalConfig(validAgentYAML))

	out, err := env.run("agent", "validate", "valid-task")
	t.Logf("output:\n%s", out)

	assertSuccess(t, out, err)
	assertContains(t, out, "Validating Agent Task: Valid Task")
	assertContains(t, out, "valid-task' is valid")
}

// TestAgentValidateInvalid verifies that a broken task causes non-zero exit
// and reports specific validation errors.
func TestAgentValidateInvalid(t *testing.T) {
	env := newTestEnv(t)
	env.writeConfig(minimalConfig(invalidAgentYAML))

	out, err := env.run("agent", "validate", "broken-task")
	t.Logf("output:\n%s", out)

	assertFailure(t, err)
	assertContains(t, out, "✗ Branch is empty")
	assertContains(t, out, "✗ No steps configured")
	assertContains(t, out, "broken-task' has")
	assertContains(t, out, "error(s)")
}

// TestAgentValidateUnknownTask verifies that requesting an unknown task returns
// a non-zero exit with a helpful message.
func TestAgentValidateUnknownTask(t *testing.T) {
	env := newTestEnv(t)
	env.writeConfig(minimalConfig(validAgentYAML))

	out, err := env.run("agent", "validate", "does-not-exist")
	t.Logf("output:\n%s", out)

	assertFailure(t, err)
	assertContains(t, out, "does-not-exist")
}

// ── Group 2: Registry commands ────────────────────────────────────────────────

// TestListEmpty verifies "worktree list" with no worktrees registered.
func TestListEmpty(t *testing.T) {
	env := newTestEnv(t)
	env.writeConfig(minimalConfig(""))

	// The worktrees/ dir already exists (created by newTestEnv) but is empty.
	out, err := env.run("list")
	t.Logf("output:\n%s", out)

	assertSuccess(t, out, err)
	assertContains(t, out, "No worktrees found")
}

// TestNewFeatureDryRun verifies "worktree new-feature --dry-run" prints a
// preview without creating any directories.
func TestNewFeatureDryRun(t *testing.T) {
	env := newTestEnv(t)
	env.gitInitProject("backend")
	env.gitInitProject("frontend")
	env.writeConfig(worktreeConfig())

	out, err := env.run("new-feature", "--dry-run", "feature/dry-run-test")
	t.Logf("output:\n%s", out)

	assertSuccess(t, out, err)
	assertContains(t, out, "Dry Run")
	assertContains(t, out, "feature-dry-run-test")
	// No actual directory should have been created.
	featureDir := filepath.Join(env.root, "worktrees", "feature-dry-run-test")
	if _, statErr := os.Stat(featureDir); !os.IsNotExist(statErr) {
		t.Error("dry-run should not create the worktree directory")
	}
}

// TestWorktreeLifecycle creates a worktree once and exercises list, ports,
// yolo, and remove in sequence.  This avoids repeating the expensive
// new-feature setup in each test.
func TestWorktreeLifecycle(t *testing.T) {
	env := newTestEnv(t)
	env.gitInitProject("backend")
	env.gitInitProject("frontend")

	combined := worktreeConfig() + `
scheduled_agents:
  echo-task:
    name: "Echo Task"
    description: "Quick echo task"
    schedule: "0 9 * * MON"
    context:
      preset: default
      instance: 1
      yolo: false
    steps:
      - name: "Echo"
        type: shell
        command: "echo hi"
    safety:
      git:
        push:
          enabled: false
`
	env.writeConfig(combined)

	// Create worktree.
	out, err := env.run("new-feature", "feature/lifecycle-test")
	assertSuccess(t, out, err)

	t.Run("status shows feature info", func(t *testing.T) {
		out, err := env.run("status", "feature-lifecycle-test")
		t.Logf("output:\n%s", out)
		assertSuccess(t, out, err)
		assertContains(t, out, "Status for Feature: feature-lifecycle-test")
		assertContains(t, out, "Branch")
		assertContains(t, out, "YOLO Mode")
		assertContains(t, out, "✅ Exists")
		assertContains(t, out, "⚪ Not running")
	})

	t.Run("list shows feature", func(t *testing.T) {
		out, err := env.run("list")
		t.Logf("output:\n%s", out)
		assertSuccess(t, out, err)
		assertContains(t, out, "feature-lifecycle-test")
		assertContains(t, out, "Total: 1 worktree(s)")
	})

	t.Run("ports shows header", func(t *testing.T) {
		out, err := env.run("ports", "feature-lifecycle-test")
		t.Logf("output:\n%s", out)
		// Command exits 0 and prints the feature name header.
		// No URL-based services are shown because the test config omits url fields;
		// that is fine — we're testing the command routes, not display formatting.
		assertSuccess(t, out, err)
		assertContains(t, out, "Ports for Feature: feature-lifecycle-test")
	})

	t.Run("yolo enable", func(t *testing.T) {
		out, err := env.run("yolo", "feature-lifecycle-test")
		t.Logf("output:\n%s", out)
		assertSuccess(t, out, err)
		assertContains(t, out, "YOLO mode ENABLED")

		// Verify registry was updated: yolo_mode:true appears in the JSON.
		// The registry may use compact or indented JSON, so we just check the key is present
		// (omitempty means the key is absent when false, present when true).
		regData, _ := os.ReadFile(filepath.Join(env.root, "worktrees", ".registry.json"))
		if !strings.Contains(string(regData), "yolo_mode") {
			t.Errorf("registry does not contain yolo_mode field:\n%s", regData)
		}
	})

	t.Run("yolo disable", func(t *testing.T) {
		out, err := env.run("yolo", "feature-lifecycle-test", "--disable")
		t.Logf("output:\n%s", out)
		assertSuccess(t, out, err)
		assertContains(t, out, "YOLO mode DISABLED")
	})

	t.Run("remove force", func(t *testing.T) {
		out, err := env.run("remove", "feature-lifecycle-test", "--force")
		t.Logf("output:\n%s", out)
		assertSuccess(t, out, err)
		assertContains(t, out, "Cleanup complete")

		// Feature dir should be gone.
		featureDir := filepath.Join(env.root, "worktrees", "feature-lifecycle-test")
		if _, statErr := os.Stat(featureDir); !os.IsNotExist(statErr) {
			t.Error("expected feature directory to be removed")
		}

		// Registry entry should be gone.
		regData, _ := os.ReadFile(filepath.Join(env.root, "worktrees", ".registry.json"))
		if strings.Contains(string(regData), "feature-lifecycle-test") {
			t.Errorf("registry still contains removed feature:\n%s", regData)
		}
	})

	t.Run("list empty after remove", func(t *testing.T) {
		out, err := env.run("list")
		t.Logf("output:\n%s", out)
		assertSuccess(t, out, err)
		assertContains(t, out, "No worktrees found")
	})
}

// ── Group 3: History commands ─────────────────────────────────────────────────

// TestHistoryEmpty checks all three history subcommands against an empty store.
func TestHistoryEmpty(t *testing.T) {
	env := newTestEnv(t)
	env.writeConfig(minimalConfig(""))

	t.Run("list is empty", func(t *testing.T) {
		out, err := env.run("agent", "history", "list")
		t.Logf("output:\n%s", out)
		assertSuccess(t, out, err)
		assertContains(t, out, "No execution history found")
	})

	t.Run("stats is empty", func(t *testing.T) {
		out, err := env.run("agent", "history", "stats")
		t.Logf("output:\n%s", out)
		assertSuccess(t, out, err)
		assertContains(t, out, "No execution history found")
	})

	t.Run("clear on empty", func(t *testing.T) {
		out, err := env.run("agent", "history", "clear")
		t.Logf("output:\n%s", out)
		assertSuccess(t, out, err)
		assertContains(t, out, "History is already empty")
	})
}

// ── Group 4: Queue commands ───────────────────────────────────────────────────

// TestQueueLifecycle exercises add → list → remove → clear in sequence.
func TestQueueLifecycle(t *testing.T) {
	env := newTestEnv(t)
	env.writeConfig(minimalConfig(validAgentYAML))

	t.Run("list empty queue", func(t *testing.T) {
		out, err := env.run("agent", "queue", "list")
		t.Logf("output:\n%s", out)
		assertSuccess(t, out, err)
		assertContains(t, out, "Queue is empty")
	})

	// Add a task to the queue.
	var taskID string

	t.Run("add task", func(t *testing.T) {
		out, err := env.run("agent", "queue", "add", "valid-task", "feature-test")
		t.Logf("output:\n%s", out)
		assertSuccess(t, out, err)
		assertContains(t, out, "valid-task")
		assertContains(t, out, "feature-test")
		assertContains(t, out, "pending")

		// Extract task ID from queue file so we can remove it later.
		queuePath := filepath.Join(env.root, "worktrees", ".queue.json")
		data, readErr := os.ReadFile(queuePath)
		if readErr != nil {
			t.Fatalf("failed to read queue file: %v", readErr)
		}
		var q struct {
			Tasks []struct {
				ID string `json:"id"`
			} `json:"tasks"`
		}
		if jsonErr := json.Unmarshal(data, &q); jsonErr != nil {
			t.Fatalf("failed to parse queue file: %v", jsonErr)
		}
		if len(q.Tasks) == 0 {
			t.Fatal("expected at least one task in queue file")
		}
		taskID = q.Tasks[0].ID
	})

	t.Run("list shows pending task", func(t *testing.T) {
		out, err := env.run("agent", "queue", "list")
		t.Logf("output:\n%s", out)
		assertSuccess(t, out, err)
		assertContains(t, out, "valid-task")
		assertContains(t, out, "pending")
	})

	t.Run("remove task", func(t *testing.T) {
		if taskID == "" {
			t.Skip("taskID not set (add task failed)")
		}
		out, err := env.run("agent", "queue", "remove", taskID)
		t.Logf("output:\n%s", out)
		assertSuccess(t, out, err)
		assertContains(t, out, taskID[:8]) // partial ID match
	})

	t.Run("clear pending-only queue", func(t *testing.T) {
		// Add two more pending tasks.
		_, _ = env.run("agent", "queue", "add", "valid-task", "worktree-a")
		_, _ = env.run("agent", "queue", "add", "valid-task", "worktree-b")

		// clear only removes completed/failed tasks; pending tasks remain.
		// Result: "Cleared 0 tasks from queue" with Remaining > 0.
		out, err := env.run("agent", "queue", "clear")
		t.Logf("output:\n%s", out)
		assertSuccess(t, out, err)
		assertContains(t, out, "Cleared")
		assertContains(t, out, "Remaining:")
	})
}

// ── Error paths for registry commands ────────────────────────────────────────

// TestRegistryCommandsUnknownFeature verifies that every command that accepts
// a feature name exits non-zero and prints a useful error when the feature
// does not exist in the registry.
func TestRegistryCommandsUnknownFeature(t *testing.T) {
	env := newTestEnv(t)
	env.writeConfig(minimalConfig(""))

	cases := []struct {
		name    string
		args    []string
		errFrag string // substring expected in error output
	}{
		{
			name:    "ports unknown feature",
			args:    []string{"ports", "ghost-feature"},
			errFrag: "ghost-feature",
		},
		{
			name:    "yolo unknown feature",
			args:    []string{"yolo", "ghost-feature"},
			errFrag: "ghost-feature",
		},
		{
			name:    "status unknown feature",
			args:    []string{"status", "ghost-feature"},
			errFrag: "ghost-feature",
		},
		{
			name:    "remove unknown feature",
			args:    []string{"remove", "ghost-feature", "--force"},
			errFrag: "ghost-feature",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := env.run(tc.args...)
			t.Logf("output:\n%s", out)
			assertFailure(t, err)
			assertContains(t, out, tc.errFrag)
		})
	}
}

// ── History with real records ─────────────────────────────────────────────────

// writeHistoryFixture writes a .history.json file directly into the
// worktrees/ directory so we can test list/stats/clear with real data
// without running actual agent tasks.
func writeHistoryFixture(t *testing.T, worktreesDir string) {
	t.Helper()

	fixture := `{
  "records": [
    {
      "id": "aaa111bb-cccc-dddd-eeee-ffffffffffff",
      "agent_name": "echo-test",
      "worktree": "feature-history-a",
      "status": "completed",
      "start_time": "2026-02-18T09:00:00Z",
      "end_time":   "2026-02-18T09:05:00Z",
      "duration_ms": 300000,
      "steps_executed": 2
    },
    {
      "id": "bbb222cc-dddd-eeee-ffff-000000000000",
      "agent_name": "echo-test",
      "worktree": "feature-history-b",
      "status": "failed",
      "start_time": "2026-02-18T10:00:00Z",
      "end_time":   "2026-02-18T10:02:00Z",
      "duration_ms": 120000,
      "error": "required gate failed"
    }
  ]
}`
	path := filepath.Join(worktreesDir, ".history.json")
	if err := os.WriteFile(path, []byte(fixture), 0644); err != nil {
		t.Fatalf("write history fixture: %v", err)
	}
}

// TestHistoryWithRecords tests history list, stats, and clear against a
// pre-populated history fixture.
func TestHistoryWithRecords(t *testing.T) {
	env := newTestEnv(t)
	env.writeConfig(minimalConfig(""))
	writeHistoryFixture(t, filepath.Join(env.root, "worktrees"))

	t.Run("list shows records", func(t *testing.T) {
		out, err := env.run("agent", "history", "list")
		t.Logf("output:\n%s", out)
		assertSuccess(t, out, err)
		assertContains(t, out, "Execution History")
		assertContains(t, out, "echo-test")
		assertContains(t, out, "completed")
		assertContains(t, out, "failed")
		assertContains(t, out, "required gate failed")
	})

	t.Run("list filter by agent", func(t *testing.T) {
		out, err := env.run("agent", "history", "list", "--agent", "echo-test")
		t.Logf("output:\n%s", out)
		assertSuccess(t, out, err)
		assertContains(t, out, "echo-test")
	})

	t.Run("list filter by status completed", func(t *testing.T) {
		out, err := env.run("agent", "history", "list", "--status", "completed")
		t.Logf("output:\n%s", out)
		assertSuccess(t, out, err)
		assertContains(t, out, "completed")
		assertNotContains(t, out, "failed")
	})

	t.Run("stats shows aggregate", func(t *testing.T) {
		out, err := env.run("agent", "history", "stats")
		t.Logf("output:\n%s", out)
		assertSuccess(t, out, err)
		assertContains(t, out, "Execution Statistics")
		assertContains(t, out, "Total executions: 2")
		assertContains(t, out, "50.0%") // 1 of 2 succeeded
		assertContains(t, out, "echo-test")
	})

	t.Run("clear removes records", func(t *testing.T) {
		out, err := env.run("agent", "history", "clear")
		t.Logf("output:\n%s", out)
		assertSuccess(t, out, err)
		assertContains(t, out, "Cleared 2 execution record(s)")

		// Subsequent list should now be empty.
		out2, err2 := env.run("agent", "history", "list")
		assertSuccess(t, out2, err2)
		assertContains(t, out2, "No execution history found")
	})
}
