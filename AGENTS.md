# Scheduled Agents Guide

Scheduled agents are automated maintenance tasks that run on a schedule (cron/launchd) to keep your codebase healthy and up-to-date.

## Quick Start

```bash
# List all configured agents
worktree agent list

# Validate an agent configuration
worktree agent validate npm-audit

# Run an agent task manually
worktree agent run npm-audit

# Set up scheduling (macOS launchd or Linux cron)
worktree agent schedule npm-audit
worktree agent schedule --all
```

## Configured Production Agents

### 1. NPM Security Audit (`npm-audit`)

**Schedule**: Every Monday at 9:00 AM
**Purpose**: Check and fix npm vulnerabilities in frontend
**Instance**: 91

**What it does**:
- Runs `npm audit fix --audit-level=moderate`
- Updates `package-lock.json`
- Runs lint, build, and unit tests
- Creates PR if all quality gates pass

**Quality Gates**:
- ✅ Lint check (required)
- ✅ Build check (required)
- ⚠️ Unit tests (optional)

**Output**: Creates PR to `automated/npm-audit-{date}` branch

---

### 2. Go Dependencies Update (`go-deps-update`)

**Schedule**: Every Monday at 10:00 AM
**Purpose**: Update Go dependencies in backend
**Instance**: 92

**What it does**:
- Runs `go get -u ./...`
- Runs `go mod tidy`
- Verifies dependencies with `go mod verify`
- Runs vet, unit tests, and integration tests
- Creates PR if all quality gates pass

**Quality Gates**:
- ✅ Go vet (required)
- ✅ Unit tests (required)
- ✅ Integration tests (required)

**Output**: Creates PR to `automated/go-deps-update-{date}` branch

---

### 3. Go Version Upgrade (`go-version-upgrade`)

**Schedule**: First day of month at 11:00 AM
**Purpose**: Upgrade Go version across all projects and infrastructure
**Instance**: 93

**What it does**:
- Checks latest Go version from golang.org
- Updates `backend/go.mod`
- Updates all Dockerfiles
- Updates docker-compose files
- Updates GitHub Actions workflows
- Updates tools (worktree-manager, etc.)
- Runs build and test checks
- Creates PR if all quality gates pass

**Quality Gates**:
- ✅ Backend build (required)
- ✅ Backend tests (required)
- ✅ Tools build (required)

**Output**: Creates PR to `automated/go-version-upgrade-{date}` branch

⚠️ **Important**: This affects all Go code. Review carefully before merging.

---

### 4. Dead Code Cleanup (`dead-code-cleanup`)

**Schedule**: First day of month at 2:00 PM
**Purpose**: Identify and remove dead code in backend
**Instance**: 94

**What it does** (Phase 1 - shell commands only):
- Currently: Placeholder for dead code analysis
- **Phase 4 (upcoming)**: Will use Claude Code `/backend` skill for intelligent dead code removal

**Quality Gates**:
- ✅ Go build (required)
- ✅ Go vet (required)
- ✅ Unit tests (required)
- ✅ Integration tests (required)

**Output**: Creates PR to `automated/dead-code-cleanup-{date}` branch

---

## Configuration

Agents are configured in `.worktree.yml` under the `scheduled_agents` section:

```yaml
scheduled_agents:
  npm-audit:
    name: "NPM Security Audit & Fix"
    description: "Check and fix npm vulnerabilities in frontend"
    schedule: "0 9 * * MON"  # Cron expression

    context:
      preset: frontend          # Which projects to work on
      branch: main             # Base branch
      instance: 91             # Instance number for port allocation
      yolo: true               # Enable autonomous Claude mode

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
        commit_message: |
          chore: npm audit fix

          Automated security fixes from npm audit.
        push:
          enabled: true
          create_pr: true
          pr_title: "Security: NPM Audit Fixes ({date})"
          auto_merge: false

      rollback:
        enabled: true
        strategy: "cleanup-worktree"

    notifications:
      on_failure:
        - type: gitlab_issue
          project: "skillsetup/frontend"
          title: "NPM Audit: Failed ({date})"
          labels: ["security", "automated", "failed"]
```

## Scheduling

### macOS (launchd)

Generate launchd plist files:

```bash
worktree agent schedule npm-audit
# Creates ~/Library/LaunchAgents/com.skillsetup.worktree.npm-audit.plist

# Enable the agent
launchctl load ~/Library/LaunchAgents/com.skillsetup.worktree.npm-audit.plist

# Start immediately
launchctl start com.skillsetup.worktree.npm-audit

# Check status
launchctl list | grep worktree
```

### Linux (cron)

Generate crontab entries:

```bash
worktree agent schedule --all
# Prints crontab entries to add

# Edit crontab
crontab -e
# Paste the generated entries
```

## Logs

Logs are written to:
- **macOS**: `~/logs/worktree-agent-<task-name>.log`
- **Linux**: `~/logs/worktree-agent-<task-name>.log`

View logs:
```bash
tail -f ~/logs/worktree-agent-npm-audit.log
tail -f ~/logs/worktree-agent-npm-audit.err
```

## Implementation Phases

### ✅ Phase 1: Shell Steps (Current)
- Basic executor framework
- Shell command execution
- Simple test agent
- Agent list/validate commands

### 🔄 Phase 2: Safety Gates (In Progress)
- Run quality gates before commit
- Block commit on failure
- Support required vs. optional gates

### 📋 Phase 3: Git Operations (Planned)
- Create temporary worktrees
- Commit changes
- Push to remote
- Create PRs via `gh` CLI

### 📋 Phase 4: Claude Code Skills (Planned)
- Execute Claude Code skills (e.g., `/backend`, `/frontend`)
- Autonomous dead code cleanup
- Intelligent dependency updates

### 📋 Phase 5: Notifications (Planned)
- GitLab issue creation
- Email notifications
- Slack notifications

## Customization

### Adding a New Agent

1. Edit `.worktree.yml` and add to `scheduled_agents`:

```yaml
scheduled_agents:
  my-agent:
    name: "My Custom Agent"
    description: "Does something useful"
    schedule: "0 12 * * *"  # Daily at noon

    context:
      preset: backend
      branch: main
      instance: 95
      yolo: true

    steps:
      - name: "My step"
        type: shell
        command: "echo 'Hello!'"

    safety:
      gates: []
      git:
        branch: "automated/my-agent-{date}"
        commit_message: "chore: automated task"
        push:
          enabled: false
      rollback:
        enabled: true
        strategy: "cleanup-worktree"
```

2. Validate:
```bash
worktree agent validate my-agent
```

3. Test manually:
```bash
worktree agent run my-agent
```

4. Set up scheduling:
```bash
worktree agent schedule my-agent
```

### Cron Expression Syntax

```
* * * * *
│ │ │ │ │
│ │ │ │ └─── Day of week (0-6, SUN-SAT)
│ │ │ └───── Month (1-12)
│ │ └─────── Day of month (1-31)
│ └───────── Hour (0-23)
└─────────── Minute (0-59)
```

**Examples**:
- `0 9 * * MON` - Every Monday at 9:00 AM
- `0 0 1 * *` - First day of month at midnight
- `*/15 * * * *` - Every 15 minutes
- `0 */6 * * *` - Every 6 hours

## Best Practices

1. **Use unique instance numbers** (90-99 recommended for agents)
2. **Always enable safety gates** for production agents
3. **Test manually** before scheduling (`worktree agent run`)
4. **Review PRs** before merging (don't use `auto_merge: true` for critical tasks)
5. **Enable rollback** to clean up failed worktrees
6. **Monitor logs** regularly
7. **Use YOLO mode** for autonomous Claude operations
8. **Schedule wisely** - avoid overlapping agent runs

## Troubleshooting

### Agent fails to run

1. Check validation:
   ```bash
   worktree agent validate <task-name>
   ```

2. Check logs:
   ```bash
   tail -f ~/logs/worktree-agent-<task-name>.err
   ```

3. Run manually for debugging:
   ```bash
   worktree agent run <task-name>
   ```

### Quality gates failing

1. Run the gate command manually in the project directory:
   ```bash
   cd backend && npm test
   ```

2. Check that the command path is correct in `.worktree.yml`

3. Verify the worktree has the correct dependencies installed

### PRs not being created

1. Check that `push.enabled: true` and `push.create_pr: true`
2. Verify `gh` CLI is installed and authenticated:
   ```bash
   gh auth status
   ```
3. Check logs for git/gh errors

## Future Enhancements

- [ ] Phase 2: Safety gates implementation
- [ ] Phase 3: Git operations (worktree creation, commits, PRs)
- [ ] Phase 4: Claude Code skill execution
- [ ] Phase 5: Notifications (GitLab, email, Slack)
- [ ] Web UI for agent status and logs
- [ ] Agent metrics and reporting
- [ ] Conditional execution (run only if changes detected)
- [ ] Parallel agent execution
- [ ] Agent dependencies (run A before B)
