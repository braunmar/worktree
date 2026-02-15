# Quick Start: Scheduled Agents

## Overview

Worktree Manager provides **two ways** to run scheduled agents:

1. **Built-in Go Scheduler (Recommended)** - System-independent daemon
2. **OS Schedulers** - Native cron/launchd integration

Both approaches use the same agent configurations in `.worktree.yml`.

---

## Option 1: Built-in Go Scheduler (Recommended) 🚀

**Benefits**:
- ✅ System-independent (works on macOS, Linux, Windows)
- ✅ Single process manages all agents
- ✅ No external dependencies
- ✅ Built-in logging and error handling
- ✅ Prevents overlapping runs
- ✅ Easy installation as system service

### Quick Start

```bash
# 1. List configured agents
worktree agent list

# 2. Validate configurations
worktree agent validate npm-audit
worktree agent validate go-deps-update

# 3. Start the scheduler daemon
worktree agent daemon

# Output:
# » Starting Agent Scheduler Daemon
#   Project: /path/to/skillsetup
#   Agents: 5 configured
#   Logs: ~/logs/worktree-scheduler.log
#
#   • NPM Security Audit & Fix (0 9 * * MON)
#   • Go Dependencies Update (0 10 * * MON)
#   • Go Version Upgrade (0 11 1 * *)
#   • Dead Code Cleanup (0 14 1 * *)
#
# Press Ctrl+C to stop the daemon
```

### Install as System Service

**macOS (launchd)**:
```bash
# Install and start service
worktree agent install-service

# Service runs automatically on boot
# Logs: ~/logs/worktree-scheduler.log

# Manage service
launchctl stop com.skillsetup.worktree.scheduler
launchctl start com.skillsetup.worktree.scheduler
launchctl unload ~/Library/LaunchAgents/com.skillsetup.worktree.scheduler.plist

# View logs
tail -f ~/logs/worktree-scheduler.log

# Uninstall
worktree agent uninstall-service
```

**Linux (systemd)**:
```bash
# Install and start service
worktree agent install-service

# Service runs automatically on boot
# Logs: ~/logs/worktree-scheduler.log

# Manage service
systemctl --user stop worktree-scheduler
systemctl --user start worktree-scheduler
systemctl --user restart worktree-scheduler
systemctl --user status worktree-scheduler

# View logs
journalctl --user -u worktree-scheduler -f
# or
tail -f ~/logs/worktree-scheduler.log

# Uninstall
worktree agent uninstall-service
```

**Windows**:
```powershell
# Manual daemon start
worktree agent daemon

# Or use Task Scheduler to run on boot:
# 1. Open Task Scheduler
# 2. Create Basic Task
# 3. Trigger: At startup
# 4. Action: Start a program
# 5. Program: C:\path\to\worktree.exe
# 6. Arguments: agent daemon
```

### Running in Background (Manual)

**macOS/Linux**:
```bash
# Using nohup
nohup worktree agent daemon > /dev/null 2>&1 &

# Get process ID
ps aux | grep "worktree agent daemon"

# Stop
kill <PID>
```

---

## Option 2: OS Schedulers (cron/launchd)

**Benefits**:
- ✅ Native OS integration
- ✅ Fine-grained control per task
- ✅ Separate logs per agent

**Drawbacks**:
- ❌ OS-specific configuration
- ❌ Separate setup for each agent
- ❌ Manual overlap prevention

### Quick Start

```bash
# Generate scheduling configuration for one agent
worktree agent schedule npm-audit

# Generate for all agents
worktree agent schedule --all
```

**macOS (launchd)**:
```bash
# Schedule npm-audit
worktree agent schedule npm-audit

# Output:
# » Generating launchd configuration (macOS)
#   ✅ Created ~/Library/LaunchAgents/com.skillsetup.worktree.npm-audit.plist
#   To enable: launchctl load ~/Library/LaunchAgents/com.skillsetup.worktree.npm-audit.plist

# Enable
launchctl load ~/Library/LaunchAgents/com.skillsetup.worktree.npm-audit.plist

# Start immediately
launchctl start com.skillsetup.worktree.npm-audit

# View logs
tail -f ~/logs/worktree-agent-npm-audit.log
```

**Linux (cron)**:
```bash
# Generate crontab entries
worktree agent schedule --all

# Output shows crontab entries to add

# Add to crontab
crontab -e
# Paste the generated entries

# View logs
tail -f ~/logs/worktree-agent-npm-audit.log
```

---

## Configured Agents

### 1. npm-audit
- **Schedule**: Every Monday at 9:00 AM (`0 9 * * MON`)
- **Action**: Run `npm audit fix` on frontend
- **Gates**: Lint, build, unit tests
- **Output**: PR to `automated/npm-audit-{date}`

### 2. go-deps-update
- **Schedule**: Every Monday at 10:00 AM (`0 10 * * MON`)
- **Action**: Update Go dependencies
- **Gates**: Go vet, unit tests, integration tests
- **Output**: PR to `automated/go-deps-update-{date}`

### 3. go-version-upgrade
- **Schedule**: 1st of month at 11:00 AM (`0 11 1 * *`)
- **Action**: Upgrade Go version everywhere
- **Gates**: Backend build/tests, tools build
- **Output**: PR to `automated/go-version-upgrade-{date}`

### 4. dead-code-cleanup
- **Schedule**: 1st of month at 2:00 PM (`0 14 1 * *`)
- **Action**: Remove dead code (backend)
- **Gates**: Build, vet, unit tests, integration tests
- **Output**: PR to `automated/dead-code-cleanup-{date}`

---

## Comparison: Daemon vs OS Schedulers

| Feature | Go Scheduler Daemon | OS Schedulers |
|---------|-------------------|---------------|
| System Independence | ✅ Works everywhere | ❌ OS-specific |
| Setup Complexity | ✅ One command | ❌ Per-agent setup |
| Process Management | ✅ Single process | ❌ Multiple processes |
| Overlap Prevention | ✅ Built-in | ❌ Manual |
| Logging | ✅ Unified log | ❌ Separate logs |
| Service Installation | ✅ One command | ❌ Manual per-agent |
| Fine-grained Control | ❌ All or nothing | ✅ Per-agent control |
| Resource Usage | ⚠️ Always running | ✅ On-demand only |

---

## Recommendation

**Use the Go Scheduler Daemon** unless you need:
- Fine-grained per-agent control
- Minimal resource usage (agents run rarely)
- Native OS integration requirements

---

## Manual Testing

Run any agent manually for testing:

```bash
# Run immediately (without waiting for schedule)
worktree agent run npm-audit

# Validate before running
worktree agent validate npm-audit
worktree agent run npm-audit
```

---

## Monitoring

### Go Scheduler Daemon

```bash
# View scheduler logs
tail -f ~/logs/worktree-scheduler.log

# Sample output:
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# 🤖 Running: NPM Security Audit & Fix (npm-audit)
#    Started: 2026-02-15 09:00:00
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# 📋 Executing steps...
#   [1/2] Run npm audit fix
#   ...
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# ✅ Completed: NPM Security Audit & Fix (npm-audit)
#    Duration: 2m 15s
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

### OS Schedulers

```bash
# View specific agent logs
tail -f ~/logs/worktree-agent-npm-audit.log
tail -f ~/logs/worktree-agent-npm-audit.err

# macOS: Check launchd status
launchctl list | grep worktree

# Linux: Check cron jobs
crontab -l | grep worktree
```

---

## Troubleshooting

### Daemon won't start

```bash
# Check configuration
worktree agent validate <task-name>

# Check for errors
tail ~/logs/worktree-scheduler.err

# Test agent manually
worktree agent run <task-name>
```

### Agent not running on schedule

```bash
# Go Scheduler: Check daemon is running
ps aux | grep "worktree agent daemon"

# macOS: Check launchd service
launchctl list | grep worktree

# Linux: Check systemd service
systemctl --user status worktree-scheduler
```

### Multiple agents running simultaneously

**Go Scheduler**: Built-in overlap prevention (no action needed)

**OS Schedulers**: Stagger schedules in `.worktree.yml`:
```yaml
npm-audit:
  schedule: "0 9 * * MON"   # 9 AM
go-deps-update:
  schedule: "0 10 * * MON"  # 10 AM (1 hour later)
```

---

## Next Steps

1. ✅ **Validate all agents**: `worktree agent validate <task>`
2. ✅ **Test manually**: `worktree agent run <task>`
3. ✅ **Install daemon service**: `worktree agent install-service`
4. ✅ **Monitor logs**: `tail -f ~/logs/worktree-scheduler.log`
5. ⏳ **Wait for first scheduled run**
6. ✅ **Review PRs**: Check automated PRs in your repositories

---

## See Also

- **[AGENTS.md](AGENTS.md)** - Complete agent documentation
- **[CLAUDE.md](CLAUDE.md)** - Developer guide
- **[IMPLEMENTATION_SUMMARY.md](IMPLEMENTATION_SUMMARY.md)** - Implementation details
