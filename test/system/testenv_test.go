package system_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// testBinary holds the path to the built binary (populated in TestMain).
var testBinary string

// TestMain builds the CLI binary once before all tests in this package run.
func TestMain(m *testing.M) {
	moduleRoot, err := filepath.Abs("../../")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to resolve module root: %v\n", err)
		os.Exit(1)
	}

	binDir, err := os.MkdirTemp("", "worktree-bin-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp bin dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(binDir)

	testBinary = filepath.Join(binDir, "worktree")

	build := exec.Command("go", "build", "-o", testBinary, ".")
	build.Dir = moduleRoot
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build worktree binary: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// TestEnv holds the state for a single system test.
// Each test gets an isolated temporary project root. config.New() locates the
// project root by walking up from CWD until it finds a .worktree.yml file,
// which each test writes via writeConfig() before invoking the binary.
type TestEnv struct {
	t      *testing.T
	root   string // temp dir used as the project root
	binDir string // temp dir for mock binaries (prepended to PATH)
}

// newTestEnv creates a fresh, isolated test environment.
func newTestEnv(t *testing.T) *TestEnv {
	t.Helper()

	root, err := os.MkdirTemp("", "worktree-system-*")
	if err != nil {
		t.Fatalf("failed to create root temp dir: %v", err)
	}

	binDir, err := os.MkdirTemp("", "worktree-mocks-*")
	if err != nil {
		t.Fatalf("failed to create mock bin dir: %v", err)
	}

	t.Cleanup(func() {
		os.RemoveAll(root)
		os.RemoveAll(binDir)
	})

	e := &TestEnv{t: t, root: root, binDir: binDir}
	e.mkdir("worktrees")

	return e
}

// mkdir creates a directory relative to env.root.
func (e *TestEnv) mkdir(rel string) {
	e.t.Helper()
	if err := os.MkdirAll(filepath.Join(e.root, rel), 0755); err != nil {
		e.t.Fatalf("mkdir %s: %v", rel, err)
	}
}

// gitInitProject initialises a bare git repository in a project subdirectory
// (relative to env.root) with an empty initial commit on branch "main".
// This is required before git worktree add can be used.
func (e *TestEnv) gitInitProject(relDir string) {
	e.t.Helper()
	dir := filepath.Join(e.root, relDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		e.t.Fatalf("mkdir %s: %v", relDir, err)
	}
	e.gitRun(dir, "init", "-b", "main")
	e.gitRun(dir, "config", "user.email", "test@example.com")
	e.gitRun(dir, "config", "user.name", "System Test")
	e.gitRun(dir, "commit", "--allow-empty", "-m", "initial commit")
}

func (e *TestEnv) gitRun(dir string, args ...string) {
	e.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		e.t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
	}
}

// writeConfig writes content to .worktree.yml in env.root.
func (e *TestEnv) writeConfig(yaml string) {
	e.t.Helper()
	path := filepath.Join(e.root, ".worktree.yml")
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		e.t.Fatalf("write .worktree.yml: %v", err)
	}
}

// writeMockClaude writes a fake claude shell script to env.binDir.
// When invoked, the script prints "mock-claude: invoked" plus any extraLines
// and exits 0. PATH is injected by run/runFrom so this script is found first.
func (e *TestEnv) writeMockClaude(extraLines ...string) {
	e.t.Helper()
	var sb strings.Builder
	sb.WriteString("#!/bin/bash\n")
	sb.WriteString("echo 'mock-claude: invoked'\n")
	sb.WriteString("echo \"mock-claude: args=$*\"\n")
	for _, line := range extraLines {
		sb.WriteString(line)
		sb.WriteByte('\n')
	}
	sb.WriteString("exit 0\n")

	path := filepath.Join(e.binDir, "claude")
	if err := os.WriteFile(path, []byte(sb.String()), 0755); err != nil {
		e.t.Fatalf("write mock claude: %v", err)
	}
}

// run invokes the worktree binary from env.root with the given arguments.
// It returns combined stdout+stderr and the command error (nil on exit 0).
func (e *TestEnv) run(args ...string) (string, error) {
	return e.runFrom(e.root, args...)
}

// runFrom invokes the worktree binary from an arbitrary directory.
// env.binDir is prepended to PATH so mock binaries take precedence.
func (e *TestEnv) runFrom(dir string, args ...string) (string, error) {
	e.t.Helper()

	cmd := exec.Command(testBinary, args...)
	cmd.Dir = dir

	// Inject mock binary directory at the front of PATH.
	injectedPath := e.binDir + string(os.PathListSeparator) + os.Getenv("PATH")
	env := make([]string, 0, len(os.Environ())+1)
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "PATH=") {
			env = append(env, kv)
		}
	}
	env = append(env, "PATH="+injectedPath)
	cmd.Env = env

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()
	return buf.String(), err
}

// ── assertion helpers ─────────────────────────────────────────────────────────

func assertContains(t *testing.T, output, want string) {
	t.Helper()
	if !strings.Contains(output, want) {
		t.Errorf("output does not contain %q\n--- output ---\n%s", want, output)
	}
}

func assertNotContains(t *testing.T, output, unwanted string) {
	t.Helper()
	if strings.Contains(output, unwanted) {
		t.Errorf("output unexpectedly contains %q\n--- output ---\n%s", unwanted, output)
	}
}

func assertSuccess(t *testing.T, output string, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("expected exit 0, got error: %v\n--- output ---\n%s", err, output)
	}
}

func assertFailure(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected non-zero exit code, got exit 0")
	}
}

// ── shared YAML configs ───────────────────────────────────────────────────────

// minimalConfig returns a base .worktree.yml content that is valid (passes
// Validate()) and includes the supplied scheduled_agents block appended at
// the end.  Use it for agent-only tests that don't exercise new-feature.
func minimalConfig(agentsYAML string) string {
	return `project_name: "testproject"
hostname: localhost

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

scheduled_agents:
` + agentsYAML
}

// worktreeConfig returns a .worktree.yml suitable for new-feature tests.
// GetInstancePortName() picks the first alphabetical ranged port (APP_PORT < FE_PORT)
// for instance calculation.
func worktreeConfig() string {
	return `project_name: "testproject"
hostname: localhost

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
    description: "fullstack"

default_preset: default

env_variables:
  APP_PORT:
    name: "Backend API"
    port: "9090"
    env: "APP_PORT"
    range: [9090, 9190]
  FE_PORT:
    name: "Frontend"
    port: "9200"
    env: "FE_PORT"
    range: [9200, 9290]
`
}
