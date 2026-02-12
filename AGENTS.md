# AGENTS.md

This file provides guidance to AI coding agent when working with code in this repository.

## Project Overview

Worktree Manager is a Go CLI tool for managing git worktrees in multi-instance development environments. It creates coordinated worktrees across multiple projects (backend, frontend), allocates ports dynamically, manages Docker instances, and integrates with Claude Code for autonomous development workflows.

## Core Architecture

### Registry System

**Location**: `pkg/registry/registry.go`

The registry is the source of truth for all worktree instances and port allocations:

- **File**: `worktrees/.registry.json`
- **Structure**: Maps normalized feature names to worktree metadata (branch, ports, projects, compose names)
- **Port Allocation**: Thread-safe allocation from configured ranges
- **Persistence**: Atomic writes with temp file + rename

**Key Functions**:
- `Load()` - Loads registry from disk, builds port ranges from config
- `AllocatePorts()` - Finds available ports across all configured services
- `FindAvailablePort()` - Checks registry + live port availability (binds to verify)
- `NormalizeBranchName()` - Converts `feature/user-auth` → `feature-user-auth`

### Configuration System

**Location**: `pkg/config/workconfig.go`, `pkg/config/agent.go`

Configuration is loaded from `.worktree.yml` in the project root:

**Key Structures**:
- `WorktreeConfig` - Main configuration (projects, presets, ports, symlinks, copies, scheduled agents)
- `ProjectConfig` - Per-project settings (dir, main_branch, start_command, post_command)
- `PresetConfig` - Groups of projects (e.g., "fullstack" = backend + frontend)
- `PortConfig` - Port/service configuration (name, URL template, port expression, env var)
- `AgentTask` - Scheduled agent task definitions (NEW: see Scheduled Agents below)

**Port Expression Syntax**:
Port values support dynamic calculation using `{instance}` placeholder:

- **Simple offset**: `"3000 + {instance}"` → 3000, 3001, 3002, ...
- **Multiplication**: `"4510 + {instance} * 50"` → 4510, 4560, 4610, ...
- **Static value**: `"8080"` → always 8080

**Expression Parsing** (`parseExpression()`, `CalculatePort()`):
1. Replace `{instance}` with actual instance number
2. Parse expression: base, offset, multiplier
3. Calculate: `base + (offset * multiplier)` or `base + offset` or just `base`
4. Validate result is in range 1-65535

**Instance Calculation**:
Instance number is derived from allocated APP_PORT: `instance = APP_PORT - basePort`

### Instance Detection System

**Location**: `pkg/config/instance.go`

The instance detection system allows commands to auto-detect which worktree instance they're running in, enabling context-aware operation from any directory within a worktree.

**How It Works**:
1. When creating a worktree, a `.worktree-instance` marker file is written to the feature root directory
2. Commands can call `DetectInstance()` to walk up from the current directory and find the marker
3. If found, commands automatically know which feature/instance they're operating on

**Key Functions**:
- `DetectInstance()` - Walks up from CWD to find `.worktree-instance`, returns InstanceContext
- `WriteInstanceMarker()` - Creates `.worktree-instance` file (called by `new-feature` command)
- `UpdateInstanceYoloMode()` - Updates yolo_mode field (called by `yolo` command)
- `RemoveInstanceMarker()` - Deletes `.worktree-instance` file (called by `remove` command)

**Commands Supporting Auto-Detection**:
All these commands accept an optional feature name argument. If omitted, they auto-detect:
- `worktree status` - Shows status for current instance
- `worktree ports` - Shows ports for current instance
- `worktree start` - Starts services for current instance
- `worktree stop` - Stops services for current instance

**Example Usage**:
```bash
# From project root - explicit feature name required
worktree status feature-outlook

# From feature root - auto-detects
cd worktrees/feature-outlook
worktree status                  # ✨ Auto-detected: feature-outlook

# From project subdirectory - auto-detects
cd worktrees/feature-outlook/backend
worktree ports                   # ✨ Auto-detected: feature-outlook
```

### Command Structure

**Location**: `cmd/*.go`

All commands follow Cobra patterns:

- `cmd/root.go` - Root command setup, global flags, subcommand registration
- `cmd/newfeature.go` - Main workflow: create worktrees, allocate ports, start services
- `cmd/start.go`, `cmd/stop.go`, `cmd/remove.go` - Lifecycle commands
- `cmd/list.go`, `cmd/status.go`, `cmd/ports.go` - Status commands
- `cmd/doctor.go` - Health checks and diagnostics
- `cmd/agent.go`, `cmd/agent_run.go` - Scheduled agent tasks (NEW)
- `cmd/yolo.go` - Toggle autonomous Claude mode

**Command Pattern**:
```go
var myCmd = &cobra.Command{
    Use:   "mycommand <args>",
    Short: "Brief description",
    Long:  `Detailed description with examples`,
    Args:  cobra.ExactArgs(1), // or RangeArgs, NoArgs, etc.
    Run:   runMyCommand,
}

func init() {
    myCmd.Flags().BoolVar(&myFlag, "my-flag", false, "flag description")
    rootCmd.AddCommand(myCmd) // Add in root.go init()
}
```

### Package Organization

**`pkg/config/`**
- `config.go` - Project root discovery, paths
- `workconfig.go` - `.worktree.yml` parsing, port calculations, env vars
- `instance.go` - Instance detection, `.worktree-instance` marker file management (NEW)
- `instance_test.go` - Instance detection tests (NEW)
- `agent.go` - Scheduled agent task configuration (NEW)

**`pkg/registry/`**
- `registry.go` - Worktree tracking, port allocation
- `registry_test.go` - Registry tests

**`pkg/git/`**
- `worktree.go` - Git worktree operations (create, remove, list)

**`pkg/docker/`**
- `instance.go` - Docker container status checks

**`pkg/ui/`**
- `output.go` - Colored terminal output (sections, checkmarks, loading)
- `errors.go` - Error formatting

**`pkg/doctor/`**
- `checks.go` - Health check orchestration
- `docker.go`, `git.go`, `ports.go`, `staleness.go`, `consistency.go` - Specific checks
- `types.go`, `report.go` - Check results and reporting

## Scheduled Agents

**Documentation**: See [AGENTS.md](AGENTS.md) for complete guide
**Status**: Phase 1 implemented ✅
**Files**: `pkg/config/agent.go`, `pkg/agent/executor.go`, `cmd/agent*.go`

Scheduled agents are automated maintenance tasks that run on a schedule (cron/launchd) to keep your codebase healthy.

**Current Implementation (Phase 1)**:
- ✅ Shell command execution
- ✅ Agent list/validate/run commands
- ✅ Scheduling setup (launchd/cron)
- ✅ Configuration validation
- ✅ Error handling and logging

**Production Agents Configured**:
1. **npm-audit** - NPM security audit & fix (Mondays 9 AM)
2. **go-deps-update** - Go dependency updates (Mondays 10 AM)
3. **go-version-upgrade** - Go version upgrades (1st of month 11 AM)
4. **dead-code-cleanup** - Dead code removal (1st of month 2 PM)

**Commands**:
```bash
worktree agent list                    # List all agents
worktree agent validate npm-audit      # Validate configuration
worktree agent run npm-audit          # Run manually
worktree agent schedule npm-audit     # Set up cron/launchd
worktree agent schedule --all         # Schedule all agents
```

**Configuration** (in `.worktree.yml`):
```yaml
scheduled_agents:
  npm-audit:
    name: "NPM Security Audit & Fix"
    description: "Check and fix npm vulnerabilities in frontend"
    schedule: "0 9 * * MON"  # Cron expression

    context:
      preset: frontend
      branch: main
      instance: 91
      yolo: true

    steps:
      - name: "Run npm audit fix"
        type: shell
        command: "npm audit fix --audit-level=moderate"
        working_dir: "frontend"

    safety:
      gates:
        - name: "Lint check"
          command: "cd frontend && npm run lint"
          required: true
      git:
        branch: "automated/npm-audit-{date}"
        commit_message: "chore: npm audit fix"
        push:
          enabled: true
          create_pr: true
          pr_title: "Security: NPM Audit Fixes ({date})"
      rollback:
        enabled: true
        strategy: "cleanup-worktree"

    notifications:
      on_failure:
        - type: gitlab_issue
          project: "frontend"
          title: "NPM Audit: Failed ({date})"
          labels: ["security", "automated", "failed"]
```

**Implementation Phases**:
- ✅ Phase 1: Shell steps (current)
- 🔄 Phase 2: Safety gates (in progress)
- 📋 Phase 3: Git operations (worktree creation, commits, PRs)
- 📋 Phase 4: Claude Code skills (`/backend`, `/frontend`)
- 📋 Phase 5: Notifications (GitLab, email, Slack)

## Common Development Tasks

### Build and Install

```bash
# Build binary
make build                  # → ./worktree

# Install (choose one)
make install                # → $GOBIN/worktree (~/go/bin/worktree)
make install-user           # → ~/.local/bin/worktree
make install-global         # → /usr/local/bin/worktree (requires sudo)

# Uninstall
make uninstall              # Removes from all locations
```

### Testing

```bash
# Run all tests
make test

# Run specific package tests
go test ./pkg/registry -v
go test ./pkg/config -v

# Run specific test
go test ./pkg/registry -run TestNormalizeBranchName -v
```

### Code Quality

```bash
# Format code
make fmt

# Vet code
make vet

# Tidy dependencies
make tidy
```

## Adding New Commands

1. **Create command file**: `cmd/mycommand.go`
2. **Define command**:
   ```go
   var myCmd = &cobra.Command{
       Use:   "mycommand <arg>",
       Short: "Brief description",
       Args:  cobra.ExactArgs(1),
       Run:   runMyCommand,
   }

   func init() {
       myCmd.Flags().StringVar(&myFlag, "my-flag", "", "description")
   }
   ```
3. **Register in `cmd/root.go`**:
   ```go
   func init() {
       rootCmd.AddCommand(myCmd)  // Add to init()
   }
   ```
4. **Rebuild**: `make build`

## Adding New Packages

1. Create `pkg/mypackage/myfile.go`
2. Export functions with capital letters (Go convention)
3. Import in commands: `"worktree/pkg/mypackage"`

## Key Patterns and Conventions

### Branch Normalization

**Always normalize user-provided branch names**:
```go
featureName := registry.NormalizeBranchName(branch)
// "feature/user-auth" → "feature-user-auth"
// "FEAT/User_Auth.v2" → "feat-user-auth-v2"
```

### Registry Loading

**Always load registry with WorktreeConfig for port ranges**:
```go
workCfg, err := config.LoadWorktreeConfig(cfg.ProjectRoot)
reg, err := registry.Load(cfg.WorktreeDir, workCfg)
// Port ranges are built from workCfg.Ports
```

### Port Allocation

**Allocate ports for all services before creating worktrees**:
```go
services := workCfg.GetPortServiceNames()
ports, err := reg.AllocatePorts(services)
// ports = map[string]int{"APP_PORT": 8080, "FE_PORT": 3000, ...}
```

### Environment Variables

**Export all env vars (ports + calculated values)**:
```go
envVars := workCfg.ExportEnvVars(instance)
// First pass: export port values
// Second pass: export string templates that depend on ports
```

### Compose Project Names

**Generate per-service compose project names**:
```go
template := workCfg.GetComposeProjectTemplate()
// template = "{project}-{feature}-{service}"
composeProject := workCfg.ReplaceComposeProjectPlaceholders(template, featureName, serviceName)
// "feature-user-auth-backend"
```

### Instance Detection

**Auto-detect feature name from current directory** (NEW):
```go
// In command Run functions - support optional feature name arg
var featureName string
autoDetected := false

if len(args) == 0 {
    // No feature name provided - try auto-detection
    instance, err := config.DetectInstance()
    if err != nil {
        ui.Error("Not in a worktree directory and no feature name provided")
        ui.Info("Usage: worktree command <feature-name>")
        ui.Info("   or: cd to a worktree directory and run: worktree command")
        os.Exit(1)
    }
    featureName = instance.Feature
    autoDetected = true
} else {
    // Explicit feature name provided
    featureName = args[0]
}

// Show auto-detection in UI
if autoDetected {
    ui.Info("✨ Auto-detected from current directory")
}
```

**Writing instance markers** (in `new-feature` command):
```go
// After registry.Save(), write .worktree-instance marker
err := config.WriteInstanceMarker(
    featureDir,      // worktrees/feature-name/
    featureName,     // "feature-name"
    instance,        // 1, 2, 3, ...
    cfg.ProjectRoot, // "/projects"
    projects,        // ["backend", "frontend"]
    ports,           // map[string]int{"APP_PORT": 8081, ...}
    yoloMode,        // true/false
)
```

**Updating YOLO mode** (in `yolo` command):
```go
// After updating registry, update instance marker
featureDir := cfg.WorktreeFeaturePath(featureName)
err := config.UpdateInstanceYoloMode(featureDir, newYoloState)
```

### Error Handling

**Use `checkError()` for fatal errors**:
```go
cfg, err := config.New()
checkError(err)  // Prints error and exits with status 1
```

**Use `ui.Warning()` for non-fatal errors**:
```go
if err := someOperation(); err != nil {
    ui.Warning(fmt.Sprintf("Operation failed: %v", err))
}
```

### UI Output

**Use consistent UI patterns**:
```go
ui.Section("Creating worktrees...")
ui.Loading("Creating backend worktree...")
ui.CheckMark("Created backend worktree")
ui.Success("Feature environment ready!")
ui.Error("Failed to create worktree")
ui.Warning("Skipping fixtures")
ui.Info("Using default preset")
```

## Configuration File (.worktree.yml)

The `.worktree.yml` file is located in the project root (not in this directory). It defines:

- **projects**: Map of project names to ProjectConfig (dir, main_branch, start_command, post_command)
- **presets**: Named groups of projects (e.g., "fullstack", "backend", "frontend")
- **default_preset**: Which preset to use if none specified
- **ports**: Port/service definitions with expressions, ranges, and env var names
- **symlinks**: Files to symlink into worktrees (e.g., shared configs)
- **copies**: Files to copy into worktrees
- **generated_files**: Templates for auto-generated files per project
- **scheduled_agents**: Automated maintenance tasks (NEW)

**Port Configuration Pattern**:
```yaml
ports:
  APP_PORT:
    name: "Backend API"
    url: "http://{host}:{port}"
    port: "8080 + {instance}"
    env: "APP_PORT"
    range: [8080, 8180]  # Explicit range for allocation
```

## Testing Patterns

**Registry Tests** (`pkg/registry/registry_test.go`):
- Test normalization edge cases
- Test port allocation conflicts
- Test concurrent operations (registry uses mutex)

**Config Tests** (`pkg/config/workconfig_test.go`):
- Test port expression parsing
- Test validation logic
- Test edge cases in calculations

**When adding new features**:
1. Add tests for core logic in `pkg/`
2. Test edge cases (empty inputs, conflicts, invalid data)
3. Test concurrent operations if using shared state
4. Run `make test` before committing

## Integration with Claude Code

### YOLO Mode

YOLO mode (`worktree.YoloMode` in registry) signals that Claude can work autonomously:
- **Enabled**: Claude makes decisions without confirmation on clear tasks
- **Disabled**: Claude asks for confirmation on all changes

**Commands**:
- `worktree new-feature feature/x --yolo` - Enable at creation
- `worktree yolo feature-x` - Enable for existing worktree
- `worktree yolo feature-x --disable` - Disable

### Claude Working Directory

One project in the preset can be marked as `claude_working_dir: true`. This is where Claude navigates after `worktree new-feature` completes.

**Selection logic** (see `cmd/newfeature_helper.go`):
1. Find project with `claude_working_dir: true` in the preset
2. Default to first project in preset if none specified

## Important Notes

- **Port allocation is atomic**: Uses registry mutex + file locking
- **Branch names are normalized**: Always use `registry.NormalizeBranchName()`
- **Instance number**: Derived from APP_PORT allocation, not user-specified
- **Port expressions are validated**: At config load time, not runtime
- **Registry is source of truth**: Not Docker, not Git worktree list
- **Compose project names**: Per-service to avoid conflicts
- **Instance detection**: Commands auto-detect feature from `.worktree-instance` marker file
- **Scheduled agents**: Still in development, expect changes
