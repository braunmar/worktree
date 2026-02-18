package system_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── Test 1: shell steps ───────────────────────────────────────────────────────

// TestAgentRunEchoSteps validates that shell-type steps are executed and their
// stdout output is visible in the combined output of "worktree agent run".
func TestAgentRunEchoSteps(t *testing.T) {
	env := newTestEnv(t)

	env.writeConfig(minimalConfig(`  echo-test:
    name: "Echo Test Agent"
    description: "Simple echo test for system testing"
    schedule: "0 9 * * MON"
    context:
      preset: default
      instance: 1
      yolo: false
    steps:
      - name: "Say hello"
        type: shell
        command: "echo 'hello from system test'"
      - name: "Say world"
        type: shell
        command: "echo 'world from system test'"
    safety:
      git:
        push:
          enabled: false
`))

	out, err := env.run("agent", "run", "echo-test")
	t.Logf("output:\n%s", out)

	assertSuccess(t, out, err)
	assertContains(t, out, "Echo Test Agent")
	assertContains(t, out, "Say hello")
	assertContains(t, out, "hello from system test")
	assertContains(t, out, "Say world")
	assertContains(t, out, "world from system test")
	assertContains(t, out, "completed successfully")
}

// ── Test 2: skill step with mock claude ───────────────────────────────────────

// TestAgentRunSkillStep validates that skill-type steps invoke the "claude"
// binary (mocked here) and surface its output.
func TestAgentRunSkillStep(t *testing.T) {
	env := newTestEnv(t)

	// Place a fake claude binary on PATH that just prints a marker line.
	env.writeMockClaude()

	env.writeConfig(minimalConfig(`  skill-test:
    name: "Skill Step Agent"
    description: "Tests skill step invocation with mock claude"
    schedule: "0 10 * * MON"
    context:
      preset: default
      instance: 1
      yolo: false
    steps:
      - name: "Run backend skill"
        type: skill
        skill: "/backend 'test action'"
    safety:
      git:
        push:
          enabled: false
`))

	out, err := env.run("agent", "run", "skill-test")
	t.Logf("output:\n%s", out)

	assertSuccess(t, out, err)
	assertContains(t, out, "Run backend skill")
	assertContains(t, out, "mock-claude: invoked")
	assertContains(t, out, "completed successfully")
}

// ── Test 3: safety gates – optional failure is non-fatal ─────────────────────

// TestAgentRunWithSafetyGates verifies that:
//   - a required gate that passes is marked ✅
//   - an optional gate that fails is marked ⚠️ but does NOT abort the run
func TestAgentRunWithSafetyGates(t *testing.T) {
	env := newTestEnv(t)

	env.writeConfig(minimalConfig(`  gate-test:
    name: "Gate Test Agent"
    description: "Tests safety gate evaluation"
    schedule: "0 9 * * MON"
    context:
      preset: default
      instance: 1
      yolo: false
    steps:
      - name: "No-op step"
        type: shell
        command: "true"
    safety:
      gates:
        - name: "Always passes"
          command: "exit 0"
          required: true
        - name: "Always fails optional"
          command: "exit 1"
          required: false
      git:
        push:
          enabled: false
`))

	out, err := env.run("agent", "run", "gate-test")
	t.Logf("output:\n%s", out)

	assertSuccess(t, out, err)
	assertContains(t, out, "Always passes")
	assertContains(t, out, "✅ Passed")
	assertContains(t, out, "Always fails optional")
	assertContains(t, out, "Failed (optional")
	assertContains(t, out, "Safety Gates Summary:")
	assertNotContains(t, out, "Required safety gates failed")
}

// ── Test 4: required gate failure aborts the run ─────────────────────────────

// TestAgentRunRequiredGateFails verifies that a failing required gate causes
// a non-zero exit and prints the appropriate error message.
func TestAgentRunRequiredGateFails(t *testing.T) {
	env := newTestEnv(t)

	env.writeConfig(minimalConfig(`  required-fail-test:
    name: "Required Fail Agent"
    description: "Tests that required gate failure aborts run"
    schedule: "0 9 * * MON"
    context:
      preset: default
      instance: 1
      yolo: false
    steps:
      - name: "No-op step"
        type: shell
        command: "true"
    safety:
      gates:
        - name: "Required fail"
          command: "exit 1"
          required: true
      git:
        push:
          enabled: false
`))

	out, err := env.run("agent", "run", "required-fail-test")
	t.Logf("output:\n%s", out)

	assertFailure(t, err)
	assertContains(t, out, "❌ Failed (required)")
	assertContains(t, out, "Required safety gates failed")
}

// ── Test 5: worktree creation ─────────────────────────────────────────────────

// TestWorktreeNewFeature exercises "worktree new-feature" end-to-end:
//   - git repos are initialised for each project
//   - the command creates a worktree directory
//   - a .worktree-instance marker file is written
//   - the registry JSON is updated
func TestWorktreeNewFeature(t *testing.T) {
	env := newTestEnv(t)

	// backend/ and frontend/ must be git repos with at least one commit so that
	// "git worktree add" can create a linked working tree from them.
	env.gitInitProject("backend")
	env.gitInitProject("frontend")

	env.writeConfig(worktreeConfig())

	out, err := env.run("new-feature", "feature/test-workflow")
	t.Logf("output:\n%s", out)

	assertSuccess(t, out, err)

	featureDir := filepath.Join(env.root, "worktrees", "feature-test-workflow")

	// Worktree directories should exist.
	for _, project := range []string{"backend", "frontend"} {
		p := filepath.Join(featureDir, project)
		if _, statErr := os.Stat(p); os.IsNotExist(statErr) {
			t.Errorf("expected worktree dir %s to exist", p)
		}
	}

	// .worktree-instance marker file should be written.
	markerPath := filepath.Join(featureDir, ".worktree-instance")
	markerData, readErr := os.ReadFile(markerPath)
	if readErr != nil {
		t.Fatalf("expected .worktree-instance to exist: %v", readErr)
	}
	if !strings.Contains(string(markerData), "feature-test-workflow") {
		t.Errorf(".worktree-instance does not reference the feature name:\n%s", markerData)
	}

	// Registry should be written and contain the feature.
	registryPath := filepath.Join(env.root, "worktrees", ".registry.json")
	regData, readErr := os.ReadFile(registryPath)
	if readErr != nil {
		t.Fatalf("expected .registry.json to exist: %v", readErr)
	}
	if !strings.Contains(string(regData), "feature-test-workflow") {
		t.Errorf(".registry.json does not contain feature entry:\n%s", regData)
	}
}

// ── Test 6: agent run from inside a worktree directory ───────────────────────

// TestAgentRunFromWorktreeDir validates the navigation scenario:
// after running new-feature the user navigates into the worktrees area
// and runs an agent — config.New() must walk up and find the project root.
//
// Note: we run from the worktrees/ directory (not from the feature dir
// itself) because config.New() uses backend/ + frontend/ presence to
// locate the project root.  The feature worktree dir contains both as git
// worktrees and would therefore be misidentified as the project root.
// Running one level up (worktrees/) avoids that false match while still
// testing the walkup behaviour.
func TestAgentRunFromWorktreeDir(t *testing.T) {
	env := newTestEnv(t)

	env.gitInitProject("backend")
	env.gitInitProject("frontend")

	// Combine the worktree config with an echo agent so new-feature and
	// agent run can both work from this single .worktree.yml.
	combined := worktreeConfig() + `
scheduled_agents:
  echo-test:
    name: "Echo Test Agent"
    description: "Simple echo test for system testing"
    schedule: "0 9 * * MON"
    context:
      preset: default
      instance: 1
      yolo: false
    steps:
      - name: "Say hello"
        type: shell
        command: "echo 'hello from worktree dir'"
    safety:
      git:
        push:
          enabled: false
`
	env.writeConfig(combined)

	// Step 1: create the worktree.
	out, err := env.run("new-feature", "feature/nav-test")
	t.Logf("new-feature output:\n%s", out)
	assertSuccess(t, out, err)

	// Step 2: run the agent from the worktrees/ directory.
	// config.New() walks up: worktrees/ → <root>/ (has backend/ + frontend/) ✓
	worktreesDir := filepath.Join(env.root, "worktrees")
	if _, statErr := os.Stat(worktreesDir); os.IsNotExist(statErr) {
		t.Fatalf("worktrees dir %s does not exist", worktreesDir)
	}

	out, err = env.runFrom(worktreesDir, "agent", "run", "echo-test")
	t.Logf("agent run output:\n%s", out)

	assertSuccess(t, out, err)
	assertContains(t, out, "hello from worktree dir")
	assertContains(t, out, "completed successfully")
}
