# Worktree Manager Setup Guide for AI Agents

**Version:** 1.0.0
**Last Updated:** 2026-02-23
**Target Audience:** AI Coding Assistants (Claude Code, Cursor, Copilot, etc.)

---

## ⚡ AGENT ENTRY POINT — Start Here

> **If you are an AI agent who just loaded this file, follow this protocol immediately.**

### Step 1: Explore the project structure

Run these commands first — don't ask, just do it:

```bash
ls -la                              # What directories exist?
find . -maxdepth 2 -name ".git" -type d  # Which are git repos?
find . -maxdepth 2 -name "docker-compose.yml" -o -name "package.json" -o -name "go.mod" | head -20
```

### Step 2: Say this to the user

```
I found these potential projects in your workspace: [list from ls output]

To set up worktree-manager, I need to know a few things:
1. What should I name your project? (used for Docker container naming, e.g. "myapp")
2. Which directories are the projects you want to manage? (e.g., backend/, frontend/)
3. Do you work on multiple features simultaneously, or one at a time?
```

### Step 3: Follow the wizard

Use **Section 3 (Configuration Wizard)** as your guide:
- Section 3.1 → Basic info (project name, hostname, default preset)
- Section 3.2 → Discover and configure each project
- Section 3.3 → Design presets
- Section 3.4 → Configure ports (**critical — read carefully**)

### Step 4: Fetch the example config for reference

Before writing `.worktree.yml`, fetch the official example to use as a reference for correct syntax:

```bash
curl -s "https://raw.githubusercontent.com/braunmar/worktree/main/.worktree.example.yml"
```

Use it as your syntax reference. Section 6 also has ready-made copy-paste patterns for common setups.

### Step 5: Generate and write `.worktree.yml`

Once you have enough info, generate the config file based on what you learned from the user and the example.

### Step 6: Validate

Run `worktree new-feature test/validation` and verify it works. Remove it after.

> **Reference sections below as needed. The full workflow summary is in the Appendix.**

---

## Table of Contents

1. [Overview and Quick Context](#1-overview-and-quick-context)
2. [Prerequisites Check](#2-prerequisites-check)
3. [Configuration Wizard](#3-configuration-wizard)
4. [Advanced Configuration](#4-advanced-configuration)
5. [Claude Code Integration](#5-claude-code-integration)
6. [Common Patterns](#6-common-patterns)
7. [Troubleshooting](#7-troubleshooting)
8. [Quick Reference](#8-quick-reference)
9. [Validation Workflow](#9-validation-workflow)
10. [Additional Resources](#10-additional-resources)

---

## 1. Overview and Quick Context

### 1.1 What is Worktree Manager?

**Worktree Manager** is a CLI tool for managing multiple git worktrees with coordinated Docker environments and dynamic port allocation. It enables **multi-instance development**: running 3-5 parallel feature environments simultaneously, each with isolated ports, databases, and services.

### 1.2 The Problem

**Scenario:** Developer needs to work on 3 features simultaneously:
- `feature-a`: New authentication system
- `feature-b`: Payment integration
- `feature-c`: Admin dashboard

**Traditional approach problems:**
```
❌ Port conflicts (backend always on 8080, frontend on 3000)
❌ Database conflicts (shared PostgreSQL instance)
❌ Environment variable conflicts
❌ Git branch switching overhead
❌ Lost context when switching between features
```

### 1.3 The Solution

**Multi-instance development with worktree-manager:**

```
Instance 1 (feature-a):
  └─ worktrees/feature-a/
     ├─ backend/       (git worktree, ports: BE=8080, PG=5432)
     └─ frontend/      (git worktree, port: FE=3000)

Instance 2 (feature-b):
  └─ worktrees/feature-b/
     ├─ backend/       (git worktree, ports: BE=8081, PG=5433)
     └─ frontend/      (git worktree, port: FE=3001)

Instance 3 (feature-c):
  └─ worktrees/feature-c/
     ├─ backend/       (git worktree, ports: BE=8082, PG=5434)
     └─ frontend/      (git worktree, port: FE=3002)
```

**Benefits:**
- ✅ No port conflicts (auto-allocated from ranges)
- ✅ Isolated databases per instance
- ✅ Parallel development (all instances run simultaneously)
- ✅ No branch switching (each worktree = separate branch)
- ✅ Preserved context (each instance maintains state)

### 1.4 Visual Example

```
Developer Workspace:
┌─────────────────────────────────────────────────────┐
│ Project Root                                        │
│ ├── backend/          (main repo)                  │
│ ├── frontend/         (main repo)                  │
│ ├── .worktree.yml     (configuration)              │
│ └── worktrees/        (all instances)              │
│     ├── feature-a/    🟢 Running (8080, 3000, 5432)│
│     ├── feature-b/    🟢 Running (8081, 3001, 5433)│
│     └── feature-c/    ⚪ Stopped                    │
└─────────────────────────────────────────────────────┘

Commands:
$ worktree list           # See all instances
$ worktree start feature-a # Start instance 1
$ worktree status feature-b # Check instance 2 status
$ worktree ports feature-a  # See port allocations
```

### 1.5 When to Recommend This Tool

**✅ Recommend if project has:**

1. **Multiple git repositories** (monorepo or microservices)
   - Example: backend/, frontend/, worker/, api/

2. **Docker-based development OR process-based**
   - Docker: docker-compose up -d
   - Process: npm start, go run, python app.py

3. **Need for parallel feature development**
   - Developer works on 2+ features simultaneously
   - QA needs multiple test environments
   - Pair programming with different features

4. **Port conflicts in current setup**
   - "Port 8080 already in use"
   - Manual port management (APP_PORT=8081 npm start)
   - Services can't run simultaneously

5. **Frequent branch switching**
   - Context loss
   - Re-setup overhead (migrations, seed data)
   - Lost uncommitted changes

### 1.6 When NOT to Recommend

**❌ Don't recommend if:**

1. **Single repository with no parallel work needed**
   - Developer works on one feature at a time
   - No need for multiple environments

2. **Production environment**
   - This is dev-only tool
   - Not for staging, production, or CI/CD

3. **No git worktree knowledge in team**
   - Team unfamiliar with git worktrees
   - Would require training

4. **Simple scripts/tools without services**
   - No long-running processes
   - No port management needed

5. **Cloud-based development (GitHub Codespaces, etc.)**
   - Ephemeral environments
   - Port management handled by platform

### 1.7 AI Action Points

> **If you reached this section while setting up worktree-manager, go back to the ⚡ AGENT ENTRY POINT at the top.** This section is for context only — it describes when to recommend the tool, not how to set it up.

**This section applies when you are suggesting worktree-manager to a user who hasn't asked for it. If the user already asked you to set it up, skip to Section 2.**

**Before recommending, AI should:**

1. ✅ Confirm project matches "When to Recommend" criteria
2. ✅ Ask user: "Do you work on multiple features simultaneously?"
3. ✅ Ask user: "Do you currently have port conflicts?"
4. ✅ Explain benefits clearly
5. ✅ Set expectations (dev-only, requires git worktrees)
6. ✅ Use the user's **existing** variable names — there are no required names; use whatever the project already calls its ports
7. ✅ Only suggest changes to existing files when **strictly necessary** for multi-instance isolation — always explain *why* before suggesting any change to the user's files

**Example dialogue:**
```
AI: I found worktree-manager which can help with multi-instance development.
    This would let you run feature-a (ports 8080, 3000) and feature-b (ports 8081, 3001)
    simultaneously without conflicts.

    Do you currently work on multiple features at the same time?

User: Yes, usually 2-3 features in parallel.

AI: Perfect! This tool would be a good fit. Let me help you set it up.
```

---

## 2. Prerequisites Check

Before creating `.worktree.yml`, systematically verify all prerequisites are met.

### 2.1 Decision Tree: Prerequisites

```
┌─ Prerequisites Check ─────────────────────────────────────────────┐
│                                                                    │
│  1. Git repositories exist?                                       │
│     ├─[NO]─→ git init && git commit --allow-empty -m "init"      │
│     └─[YES]→ Continue                                             │
│                                                                    │
│  2. Main branch exists?                                           │
│     ├─[NO]─→ git branch main && git checkout main                │
│     └─[YES]→ Continue                                             │
│                                                                    │
│  3. At least 1 commit?                                            │
│     ├─[NO]─→ git commit --allow-empty -m "init"                  │
│     └─[YES]→ Continue                                             │
│                                                                    │
│  4. worktree-manager installed?                                   │
│     ├─[NO]─→ go install github.com/braunmar/worktree@latest      │
│     └─[YES]→ Continue                                             │
│                                                                    │
│  5. Go 1.22+ installed? (if needed)                               │
│     ├─[NO]─→ Guide user to install Go                            │
│     └─[YES]→ READY TO CONFIGURE ✓                                │
│                                                                    │
└────────────────────────────────────────────────────────────────────┘
```

### 2.2 Git Repositories

**Requirement:** All projects must be git repositories with ≥1 commit.

**AI Diagnostic Commands:**
```bash
# Check if directory is a git repo
cd backend && git status
cd frontend && git status

# Check if main branch exists
git branch --list main develop master

# Check commit history
git log --oneline -n 1
```

**AI Auto-Fix:**
```bash
# If not a git repo
cd backend
git init
git add .
git commit -m "initial commit"

# If no main branch
git branch main
git checkout main

# If no commits
git commit --allow-empty -m "init"
```

**AI Instructions:**
```
FOR EACH project directory (backend/, frontend/, etc.):
1. Run: cd <project> && git status
2. If "not a git repository" → Run auto-fix
3. If successful → Mark ✓
4. If fails → Ask user for help

EXAMPLE:
cd backend
git status  # ✗ not a git repository
→ git init && git commit --allow-empty -m "init"  # ✓ Fixed

cd frontend
git status  # ✓ On branch main
→ Already a git repo  # ✓ Good
```

### 2.3 Directory Structure

**Requirement:** Projects in subdirectories, identifiable project root.

**AI Diagnostic Commands:**
```bash
# List directories
ls -la

# Check for common project patterns
ls -la backend/ frontend/ services/
tree -L 2
```

**AI Instructions:**
```
STEP 1: Identify project root
- Look for parent directory containing all projects
- Usually has: backend/, frontend/, and/or services/
- May have: .git/, docker-compose.yml, package.json, go.mod

STEP 2: Verify structure
Expected:
  project-root/
  ├── backend/     (git repo)
  ├── frontend/    (git repo)
  ├── worker/      (git repo, optional)
  └── .worktree.yml (will be created)

IMPORTANT: Worktree-manager finds the project root by walking up from the
current directory until it finds a .worktree.yml file. Project directories
can have any names — backend/, frontend/, api/, worker/, or anything else.

STEP 3: If structure is wrong
ASK USER:
"I found these directories: [list]
Should I configure worktree-manager for all of these?
Or would you like a different structure?"
```

### 2.4 Service Start Methods

**Requirement:** Know how to start each service (determines executor type).

**AI Diagnostic Commands:**
```bash
# Check for docker-compose
ls -la backend/docker-compose.yml
ls -la frontend/docker-compose.yml

# Check for package managers
ls -la backend/package.json    # Node.js
ls -la backend/go.mod           # Go
ls -la backend/requirements.txt # Python
ls -la backend/Gemfile          # Ruby
```

**AI Decision Matrix:**

| File Detected | Likely Start Method | Executor |
|---------------|---------------------|----------|
| docker-compose.yml | `docker-compose up -d` | docker |
| package.json | `npm start` or `npm run dev` | process |
| go.mod | `go run main.go` | process |
| requirements.txt | `python app.py` | process |
| Makefile | `make run` | process |

**AI Instructions:**
```
FOR EACH project:
1. Detect project type (docker, node, go, python, ruby)
2. Suggest start command based on detection
3. Ask user to confirm or provide correct command
4. Determine executor type (docker vs process)

EXAMPLE:
Project: backend/
Detected: docker-compose.yml ✓
Suggested: docker-compose up -d (executor: docker)
ASK: "Is this the correct start command for backend?"
```

### 2.5 Installation

**Requirement:** worktree-manager CLI installed and in PATH.

**Installation Methods:**

#### Option 1: Go Install (Recommended)
```bash
# Requires Go 1.22+
go install github.com/braunmar/worktree@latest

# Installs to: $GOBIN/worktree (usually ~/go/bin/worktree)
```

#### Option 2: Download Binary
```bash
# Download from GitHub releases
# https://github.com/braunmar/worktree/releases
```

#### Option 3: Build from Source
```bash
git clone https://github.com/braunmar/worktree
cd worktree
make build
make install
```

**PATH Configuration:**

If `worktree` command not found, add to PATH:

```bash
# Option 1: Add $GOBIN to PATH (recommended)
echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc

# Option 2: Install to ~/.local/bin
make install-user

# Option 3: Install globally (requires sudo)
sudo make install-global
```

**AI Verification:**
```bash
# Check installation
worktree --version

# Check if in PATH
which worktree

# Expected output:
# ~/go/bin/worktree
# worktree version 0.1.0
```

**AI Instructions:**
```
STEP 1: Check if installed
Run: worktree --version

IF successful → DONE ✓
IF "command not found" → Continue to Step 2

STEP 2: Check Go installation
Run: go version

IF Go not installed → Ask user to install Go first
IF Go installed → Continue to Step 3

STEP 3: Install worktree
Run: go install github.com/braunmar/worktree@latest

STEP 4: Verify PATH
Run: worktree --version

IF still "command not found":
  ASK USER: "Add ~/go/bin to your PATH?
  Run: echo 'export PATH=\"$HOME/go/bin:$PATH\"' >> ~/.bashrc && source ~/.bashrc"
```

### 2.6 Docker (if using docker executor)

**Requirement:** Docker and docker-compose installed and running.

**AI Diagnostic Commands:**
```bash
# Check Docker installation
docker --version
docker-compose --version

# Check Docker daemon is running
docker ps

# Expected: List of containers (or empty list)
# Error: "Cannot connect to the Docker daemon"
```

**AI Instructions:**
```
IF any project uses executor: docker:
  STEP 1: Check Docker installed
  Run: docker --version

  IF "command not found":
    ASK USER: "Please install Docker Desktop or Docker Engine"
    Link: https://docs.docker.com/get-docker/

  STEP 2: Check Docker running
  Run: docker ps

  IF "Cannot connect to the Docker daemon":
    ASK USER: "Please start Docker Desktop or Docker daemon"

  IF successful → Docker is ready ✓
```

### 2.7 Prerequisites Checklist

**AI should verify before proceeding to configuration:**

```
Prerequisites Checklist:
□ Project directories identified (backend/, frontend/, etc.)
□ All projects are git repositories
□ All projects have main branch (main/develop/master)
□ All projects have ≥1 commit
□ worktree-manager installed (worktree --version works)
□ Go 1.22+ installed (if building from source)
□ Docker installed and running (if using docker executor)
□ Know start command for each service
□ Understand executor types (docker vs process)
```

---

## 3. Configuration Wizard

This is the **PRIMARY** section. Follow this step-by-step process to build `.worktree.yml` through AI-user dialogue.

> **Before starting the wizard:** Fetch the official example config for syntax reference — it has inline comments explaining every field:
> ```bash
> curl -s "https://raw.githubusercontent.com/braunmar/worktree/main/.worktree.example.yml"
> ```
> Read it once, then use it alongside this wizard when writing the final `.worktree.yml`.

### 3.1 Basic Information

**Goal:** Gather project name, hostname, and default preset.

**AI Instructions:**

```
ASK USER these questions IN ORDER:

Q1: "What should I name your project?"
    - Used for Docker container naming
    - Must be: alphanumeric + hyphens only
    - Cannot start/end with hyphen
    - Examples: "myproject", "acme-app", "dev-env"
    → Save as: project_name

Q2: "What hostname should services use?"
    - Default: localhost
    - For tunnels: "myapp.ngrok.io"
    - For remote: "192.168.1.100"
    → Save as: hostname

Q3: "What should be the default preset?"
    - Explain: "Preset = group of projects to start together"
    - Suggest based on projects: "fullstack" (backend+frontend)
    → Save as: default_preset (will create presets later)
```

**Example Dialogue:**

```
AI: What should I name your project? (Used for Docker container naming)
    Examples: "myproject", "acme-app"

User: acme-app

AI: ✓ Using project_name: "acme-app"
    Docker containers will be named: acme-app-feature-backend, acme-app-feature-frontend

AI: What hostname should services use? (Press Enter for localhost)

User: [Enter]

AI: ✓ Using hostname: localhost

AI: I'll configure presets next (groups of projects to start together).
    Would you like a "fullstack" preset for backend+frontend? (Yes)

User: Yes

AI: ✓ Will create "fullstack" preset with backend + frontend
```

**Generated YAML (so far):**

```yaml
project_name: "acme-app"
hostname: localhost
default_preset: fullstack
```

### 3.2 Project Discovery

**Goal:** Identify projects and configure each one (dir, branch, start command, executor).

#### Step 1: Discover Projects

**AI Instructions:**

```
STEP 1: List directories
Run: ls -la

STEP 2: Identify projects
Look for directories with:
- Git repositories (.git/ directory)
- Code (package.json, go.mod, requirements.txt, Gemfile)
- Docker files (docker-compose.yml, Dockerfile)

STEP 3: Present to user
"I found these potential projects:
  - backend/ (has docker-compose.yml, git repo)
  - frontend/ (has package.json, git repo)
  - worker/ (has go.mod, git repo)

Should I configure all of these, or just some?"

STEP 4: User selects projects
User might say:
- "All of them" → Configure all
- "Just backend and frontend" → Configure only those
- "Add api/ too" → Add additional project
```

#### Step 2: Configure Each Project

**For EACH project, ask these questions:**

**Q1: What's the main branch?**
```
AI: What's the main branch for {project}?
    Common: main, develop, master

User: main

AI: ✓ main_branch: main
```

**Q2: How do I start this service?**
```
AI: How do I start {project}?
    Examples:
    - docker-compose up -d
    - npm start
    - go run main.go
    - python app.py

User: docker-compose up -d

AI: ✓ start_command: "docker-compose up -d"
```

**Q3: What executor type?** (Use decision tree below)

**Q4: Should I run anything before start?** (Optional)
```
AI: Should I run anything BEFORE starting {project}? (Optional)
    Examples: make check-deps, npm install
    Press Enter to skip.

User: [Enter]

AI: ✓ No pre-start command
```

**Q5: Should I run anything after start?** (Optional)
```
AI: Should I run anything AFTER starting {project}? (Optional)
    Examples: make migrate && make seed, npm run generate-types
    Press Enter to skip.

User: make migrate && make seed

AI: ✓ start_post_command: "make migrate && make seed"
```

**Q6: Is this your main working directory?** (Claude Code only — optional)
```
AI: Will you primarily work in {project}/ when Claude Code starts a new worktree?
    (Press Enter to skip — Claude will just stay in the worktree root)

User: Yes → set claude_working_dir: true for that project
User: No / Enter → omit this field entirely (most users prefer this)
```

> **Note:** Most users do NOT set `claude_working_dir`. Omit this field unless the user explicitly wants Claude to navigate to a specific project directory after `worktree new-feature` runs. Default behavior keeps Claude in the worktree root.

#### Step 3: Executor Type Decision Tree

**CRITICAL:** Choose correct executor for each project.

```
FOR PROJECT: {project_name}
│
├─ Q: "Does this project use docker-compose?"
│  │
│  ├─[YES]─→ executor: docker
│  │         start_command: "docker-compose up -d"
│  │         EXPLANATION: docker executor runs command synchronously,
│  │                      expects it to return quickly (detached mode).
│  │                      Stop uses: docker-compose down
│  │         ✓ DONE
│  │
│  └─[NO]──→ Q: "Is this a long-running process?"
│            │   (npm start, go run, python app.py)
│            │
│            ├─[YES]─→ executor: process
│            │         start_command: "npm start" | "go run main.go" | ...
│            │         EXPLANATION: process executor runs command in background,
│            │                      saves PID to <feature>/<project>.pid.
│            │                      Stop sends: SIGTERM (then SIGKILL after 5s)
│            │         ✓ DONE
│            │
│            └─[NO]──→ Q: "Is this a build/setup command?"
│                      │   (make build, npm install)
│                      │
│                      ├─[YES]─→ NO executor needed
│                      │         start_command: "make build"
│                      │         EXPLANATION: Runs once, no start/stop management
│                      │         ✓ DONE
│                      │
│                      └─[NO]──→ ASK USER: "How should I start this service?"
│                                Then classify based on answer
```

**AI Decision Examples:**

| Start Command | Long-Running? | Executor | Reasoning |
|---------------|---------------|----------|-----------|
| `docker-compose up -d` | N/A | docker | Uses docker-compose |
| `npm start` | Yes | process | Starts dev server (foreground) |
| `npm run dev` | Yes | process | Starts dev server |
| `go run main.go` | Yes | process | Runs Go server (foreground) |
| `python app.py` | Yes | process | Runs Python server |
| `make run` | Yes | process | Usually runs server |
| `make build` | No | (none) | One-shot, builds artifacts |
| `npm install` | No | (none) | One-shot, installs deps |

#### Step 4: Generate Project YAML

**After configuring all projects, generate:**

```yaml
projects:
  backend:
    executor: docker
    dir: backend
    main_branch: main
    start_command: "docker-compose up -d"
    start_post_command: "make migrate && make seed"
    claude_working_dir: true

  frontend:
    executor: process
    dir: frontend
    main_branch: main
    start_command: "npm start"
    start_post_command: "npm run generate-types"

  worker:
    executor: process
    dir: worker
    main_branch: main
    start_command: "go run main.go"
```

### 3.3 Preset Design

**Goal:** Group projects for simultaneous setup.

**AI Instructions:**

```
PRESETS define which projects to start together.

STEP 1: Suggest common presets based on discovered projects

IF backend + frontend exist:
  Suggest: "fullstack" preset (backend + frontend)

IF backend + frontend + worker exist:
  Suggest: "fullstack" (backend + frontend)
           "all" (backend + frontend + worker)

IF only backend exists:
  Suggest: "backend" preset (backend only)

STEP 2: Present suggestions to user

"I can create these presets:
  - fullstack: backend + frontend
  - backend: backend only
  - frontend: frontend only
  - all: backend + frontend + worker

Should I create these, or do you want different groupings?"

STEP 3: Generate preset YAML based on user confirmation
```

**Example Dialogue:**

```
AI: I can create these presets for you:
      - fullstack: backend + frontend (starts both together)
      - backend: backend only (for backend-focused work)
      - frontend: frontend only (for frontend-focused work)
      - all: backend + frontend + worker (everything)

    Should I create these presets?

User: Yes, those look good. Make fullstack the default.

AI: ✓ Creating presets with "fullstack" as default
```

**Generated YAML:**

```yaml
presets:
  fullstack:
    projects: [backend, frontend]
    description: "Backend + Frontend"

  backend:
    projects: [backend]
    description: "Backend API only"

  frontend:
    projects: [frontend]
    description: "Frontend only"

  all:
    projects: [backend, frontend, worker]
    description: "All services"

default_preset: fullstack
```

### 3.4 Port Configuration

**⚠️ CRITICAL SECTION ⚠️**

This is the **most complex** part of configuration. AI must understand **4 port types** and correctly classify each service port.

> **ℹ️ Variable names are completely flexible**
>
> There are no required variable names. The tool automatically picks the first port with a `range` (alphabetically) to calculate instance numbers. Name your ports whatever your project already uses — `BE_PORT`, `APP_PORT`, `API_PORT`, `BACKEND_PORT` — they all work equally.

> **⚠️ NON-INVASIVE SETUP PRINCIPLE**
>
> Do NOT rename the user's existing environment variables. Do NOT restructure their docker-compose files unless strictly required for multi-instance isolation (and always explain why). Scan existing configs first to discover names already in use.

#### Overview: 4 Port Types

| Type | When to Use | Required Fields | Key Characteristic |
|------|-------------|-----------------|-------------------|
| **Type 1: Allocated** | Service needs unique port per instance | name, url, port, env, **range** | Most common, auto-allocated from range |
| **Type 2: Calculated** | Service needs block of ports | port (with `{instance}`), env | Rare, uses expression, NO range |
| **Type 3: String Template** | Env var references other ports | value (with `{PLACEHOLDER}`), env | For URLs, connection strings |
| **Type 4: Display-Only** | Show in UI but don't export | name, url, port, **env: null** | For endpoints using other ports |

#### Type 1: Allocated Ports (MOST COMMON)

**Decision Criteria:**

```
┌─ Does service bind to a network port? ────────────────────┐
│                                                            │
│  YES → Type 1: Allocated Port                             │
│                                                            │
│  Examples:                                                 │
│  • Web servers (frontend, backend)                        │
│  • Databases (postgres, mysql, redis, mongodb)            │
│  • Message queues (rabbitmq, kafka)                       │
│  • Cache servers (redis, memcached)                       │
│  • Any service that listens on a port                     │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

**Required Fields:**

- `name`: Human-readable display name
- `url`: URL template with `{host}` and `{port}` placeholders
- `port`: Base port (hint for allocation, not final value)
- `env`: Environment variable name (exported to service)
- **`range`**: [min, max] for allocation (**CRITICAL** - prevents conflicts)

**AI Instructions for Allocated Ports:**

```
STEP 1: Identify services that bind to ports

Auto-detect from:
- docker-compose.yml (ports: section)
- .env files (PORT variables)
- package.json (scripts with --port)
- Config files (config.yaml, settings.py, application.yml)

OR ask user:
"What ports do your services need?
 Examples: Backend API, Frontend, PostgreSQL, Redis"

STEP 2: For EACH port, determine:

Q: "What's the base port for {service}?"
   - Suggest common defaults:
     * Backend API: 8080
     * Frontend: 3000
     * PostgreSQL: 5432
     * Redis: 6379
     * MySQL: 3306

Q: "How many instances might you run simultaneously?"
   - 2-5 instances → range of 20-50
   - 5-10 instances → range of 50-100
   - 10+ instances → range of 100+

STEP 3: Calculate range

Formula: [base, base + (instances * safety_factor)]
Safety factor: 2x expected instances (headroom for growth)

Example:
  Base port: 8080
  Expected instances: 5
  Range: [8080, 8080 + (5 * 2)] = [8080, 8090]
  Better: [8080, 8130] (50 ports, allows 10 instances comfortably)

STEP 4: Validate

CHECK: Ranges don't overlap
CHECK: Ranges don't conflict with system ports (< 1024)
CHECK: Ranges have sufficient size (≥20 ports recommended)
```

**Examples:**

```yaml
env_variables:
  # All names below are examples — use whatever your project already calls these ports
  BE_PORT:
    name: "Backend API"
    url: "http://{host}:{port}"
    port: "8080"                    # Base port hint
    env: "BE_PORT"                  # Could be APP_PORT, API_PORT, BACKEND_PORT — your choice
    range: [8080, 8180]             # 100 ports → supports 10 instances

  FE_PORT:
    name: "Frontend"
    url: "http://{host}:{port}"
    port: "3000"
    env: "FE_PORT"
    range: [3000, 3100]             # 100 ports

  POSTGRES_PORT:
    name: "PostgreSQL"
    url: "postgresql://{host}:{port}/dbname"
    port: "5432"
    env: "POSTGRES_PORT"
    range: [5432, 5532]             # 100 ports

  REDIS_PORT:
    name: "Redis"
    url: "redis://{host}:{port}"
    port: "6379"
    env: "REDIS_PORT"
    range: [6379, 6479]             # 100 ports

  # Mailpit SMTP
  MAILPIT_SMTP_PORT:
    name: "Mailpit SMTP"
    url: "smtp://{host}:{port}"
    port: "1025"
    env: "MAILPIT_SMTP_PORT"
    range: [1025, 1125]             # 100 ports

  # Mailpit Web UI
  MAILPIT_UI_PORT:
    name: "Mailpit UI"
    url: "http://{host}:{port}"
    port: "8025"
    env: "MAILPIT_UI_PORT"
    range: [8025, 8125]             # 100 ports
```

**Common Services and Defaults:**

| Service | Base Port | Range Size | Example Range |
|---------|-----------|------------|---------------|
| Backend API | 8080 | 100 | [8080, 8180] |
| Frontend | 3000 | 100 | [3000, 3100] |
| PostgreSQL | 5432 | 100 | [5432, 5532] |
| MySQL | 3306 | 100 | [3306, 3406] |
| Redis | 6379 | 100 | [6379, 6479] |
| MongoDB | 27017 | 100 | [27017, 27117] |
| RabbitMQ | 5672 | 100 | [5672, 5772] |
| RabbitMQ UI | 15672 | 100 | [15672, 15772] |
| Elasticsearch | 9200 | 100 | [9200, 9300] |
| Kafka | 9092 | 100 | [9092, 9192] |

#### Type 2: Calculated Ports (RARE)

**Decision Criteria:**

```
┌─ Does service need a BLOCK of contiguous ports? ──────────┐
│                                                            │
│  YES → Type 2: Calculated Port                            │
│                                                            │
│  Examples:                                                 │
│  • LocalStack external ports (50 ports per instance)      │
│  • Kubernetes node ports (range per cluster)              │
│  • Port ranges for load balancers                         │
│  • Multi-port services (start + end range)                │
│                                                            │
│  NO → Use Type 1 (Allocated) instead                      │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

**Required Fields:**

- `port`: Expression with `{instance}` placeholder
- `env`: Environment variable name
- **NO `range` field** (not allocated from registry)

**Port Expression Syntax:**

```
Simple offset:
  "3000 + {instance}"
  → Instance 0: 3000
  → Instance 1: 3001
  → Instance 2: 3002

Multiplication (for blocks):
  "4510 + {instance} * 50"
  → Instance 0: 4510
  → Instance 1: 4560  (4510 + 1*50)
  → Instance 2: 4610  (4510 + 2*50)

Static (probably wrong!):
  "8080"
  → All instances: 8080
  → WARNING: This causes port conflicts! Use Type 1 instead.
```

**AI Instructions:**

```
ONLY use calculated ports if:
1. Service explicitly needs a BLOCK of ports (e.g., 50 ports)
2. Ports are contiguous (start to end range)
3. Cannot use single allocated port

EXAMPLE: LocalStack
- Needs ports 4510-4559 (50 ports) for instance 0
- Needs ports 4560-4609 (50 ports) for instance 1
- Solution: Calculated port blocks

MOST services should use Type 1 (Allocated) instead!
```

**Examples:**

```yaml
env_variables:
  # LocalStack external service ports (each instance needs 50 ports)
  LOCALSTACK_EXT_START:
    port: "4510 + {instance} * 50"
    env: "LOCALSTACK_EXT_START"
    # NO range - not allocated, just calculated

  LOCALSTACK_EXT_END:
    port: "4559 + {instance} * 50"
    env: "LOCALSTACK_EXT_END"
    # NO range

  # Custom port block example
  SERVICE_PORT_START:
    port: "10000 + {instance} * 100"
    env: "SERVICE_PORT_START"

  SERVICE_PORT_END:
    port: "10099 + {instance} * 100"
    env: "SERVICE_PORT_END"
```

#### Type 3: String Templates

**Decision Criteria:**

```
┌─ Does env var reference another port? ────────────────────┐
│                                                            │
│  YES → Type 3: String Template                            │
│                                                            │
│  Examples:                                                 │
│  • REACT_APP_API_BASE_URL="http://localhost:{BE_PORT}"    │
│  • DATABASE_URL="postgresql://localhost:{PG_PORT}/db"     │
│  • REDIS_URL="redis://localhost:{REDIS_PORT}"             │
│  • API_ENDPOINT="https://{host}:{BE_PORT}/api"            │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

**Required Fields:**

- `value`: String with `{PLACEHOLDER}` for port variable names
- `env`: Environment variable name

**Placeholder Rules:**

- `{PLACEHOLDER}` must reference a port variable defined elsewhere
- `{host}` is replaced with `hostname` value
- Common placeholders: `{BE_PORT}`, `{FE_PORT}`, `{POSTGRES_PORT}`, etc.

**AI Instructions:**

```
STEP 1: Identify env vars that reference ports
Common patterns:
- *_URL (DATABASE_URL, REDIS_URL, API_URL)
- *_ENDPOINT (API_ENDPOINT, WEBHOOK_ENDPOINT)
- *_BASE_URL (REACT_APP_API_BASE_URL)
- Connection strings

STEP 2: Extract placeholders
Example: "http://localhost:{BE_PORT}/api"
Placeholder: {BE_PORT}

STEP 3: Validate placeholders exist
CHECK: BE_PORT is defined as allocated or calculated port

STEP 4: Generate template
```

**Examples:**

```yaml
env_variables:
  # Frontend needs backend API URL
  REACT_APP_API_BASE_URL:
    value: "http://localhost:{BE_PORT}"
    env: "REACT_APP_API_BASE_URL"
    # Result: REACT_APP_API_BASE_URL=http://localhost:8080

  # Backend needs database URL
  DATABASE_URL:
    value: "postgresql://user:pass@localhost:{POSTGRES_PORT}/mydb"
    env: "DATABASE_URL"
    # Result: DATABASE_URL=postgresql://user:pass@localhost:5432/mydb

  # Backend needs Redis URL
  REDIS_URL:
    value: "redis://localhost:{REDIS_PORT}"
    env: "REDIS_URL"
    # Result: REDIS_URL=redis://localhost:6379

  # Complex example with multiple placeholders
  SERVICES_CONFIG:
    value: "api={BE_PORT},db={POSTGRES_PORT},cache={REDIS_PORT}"
    env: "SERVICES_CONFIG"
    # Result: SERVICES_CONFIG=api=8080,db=5432,cache=6379

  # External webhook with host
  WEBHOOK_URL:
    value: "https://{host}:{BE_PORT}/webhooks/callback"
    env: "WEBHOOK_URL"
    # Result: WEBHOOK_URL=https://localhost:8080/webhooks/callback
```

#### Type 4: Display-Only

**Decision Criteria:**

```
┌─ Should this show in `worktree ports` output? ────────────┐
│   but NOT be exported as env var?                         │
│                                                            │
│  YES → Type 4: Display-Only                               │
│                                                            │
│  Examples:                                                 │
│  • Swagger UI (uses backend port, not separate)           │
│  • GraphQL Playground (uses backend port)                 │
│  • Admin panels (uses main app port)                      │
│  • Health check endpoints (/health, /metrics)             │
│  • Documentation endpoints                                │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

**Required Fields:**

- `name`: Display name
- `url`: URL template with `{host}` and `{port}` placeholders
- `port`: Which port it uses (reference to allocated port's base value)
- **`env: null`** (**CRITICAL** - don't export as env var)

**AI Instructions:**

```
Use display-only when:
1. Endpoint is hosted on another service's port
2. User wants to see URL in `worktree ports` output
3. Don't need to export as environment variable

EXAMPLE: Swagger UI
- Hosted at: http://localhost:8080/swagger
- Uses backend port (8080)
- User wants to see "Swagger UI" in ports list
- Solution: Display-only entry

IMPORTANT: Set env: null (not env: "SWAGGER_URL")
```

**Examples:**

```yaml
env_variables:
  # Swagger uses backend port
  swagger:
    name: "Swagger UI"
    url: "http://{host}:{port}/swagger"
    port: "8080"              # Uses BE_PORT (base value)
    env: null                 # ← CRITICAL: Don't export

  # GraphQL Playground uses backend port
  graphql:
    name: "GraphQL Playground"
    url: "http://{host}:{port}/graphql"
    port: "8080"              # Uses BE_PORT
    env: null

  # Admin panel uses backend port
  admin:
    name: "Admin Panel"
    url: "http://{host}:{port}/admin"
    port: "8080"
    env: null

  # Health check endpoint
  health:
    name: "Health Check"
    url: "http://{host}:{port}/health"
    port: "8080"
    env: null

  # Metrics endpoint
  metrics:
    name: "Metrics"
    url: "http://{host}:{port}/metrics"
    port: "8080"
    env: null
```

#### Master Decision Tree: Port Configuration

```
FOR EACH SERVICE PORT:
│
├─ Q: "Does this service bind to a network port?"
│  │
│  ├─[YES]──→ Q: "Does it need ONE port per instance?"
│  │          │
│  │          ├─[YES]─→ Type 1: Allocated Port
│  │          │         Required: name, url, port, env, range
│  │          │         MOST COMMON CASE
│  │          │         ✓ DONE
│  │          │
│  │          └─[NO]──→ Q: "Does it need a BLOCK of ports?"
│  │                    │   (e.g., 50 contiguous ports per instance)
│  │                    │
│  │                    ├─[YES]─→ Type 2: Calculated Port
│  │                    │         Required: port (with {instance}), env
│  │                    │         NO range field
│  │                    │         RARE CASE
│  │                    │         ✓ DONE
│  │                    │
│  │                    └─[NO]──→ ERROR: Should use Type 1
│  │                              Single port = Allocated
│  │
│  └─[NO]───→ Q: "Does env var reference another port?"
│             │   (e.g., DATABASE_URL with {POSTGRES_PORT})
│             │
│             ├─[YES]─→ Type 3: String Template
│             │         Required: value (with {PLACEHOLDER}), env
│             │         NO port or range
│             │         ✓ DONE
│             │
│             └─[NO]──→ Q: "Should this show in UI but not export?"
│                       │   (e.g., Swagger UI using backend port)
│                       │
│                       ├─[YES]─→ Type 4: Display-Only
│                       │         Required: name, url, port, env: null
│                       │         IMPORTANT: env must be null
│                       │         ✓ DONE
│                       │
│                       └─[NO]──→ SKIP: Not a port configuration
```

#### AI Workflow: Complete Port Configuration

```
STEP 0: DISCOVER EXISTING VARIABLE NAMES (do this FIRST)

Before defining any port variables, scan the user's existing configs:
1. Read docker-compose.yml — what variable names are already used in ports: sections?
   e.g., "${APP_PORT}:8080" → use APP_PORT (keep their existing name)
   e.g., "${DATABASE_PORT}:5432" → use DATABASE_PORT, NOT POSTGRES_PORT
   e.g., "${REDIS_PORT}:6379" → use REDIS_PORT
2. Read .env and .env.example files — what PORT variables already exist?
3. Check package.json scripts for --port flags with variable names

RULE: Use names from the user's existing configs. Do NOT rename variables
      that already work. There are no required variable names.

STEP 1: AUTO-DETECT ports

Scan files:
- docker-compose.yml (ports: section)
- .env files (PORT variables, *_URL variables)
- package.json (scripts: {"dev": "vite --port 3000"})
- config files (config.yaml, settings.py)

Extract:
- Service names (backend, frontend, postgres, redis)
- Port numbers (8080, 3000, 5432, 6379)
- URL patterns (DATABASE_URL, REACT_APP_API_BASE_URL)
- IMPORTANT: Also extract the existing variable names used for each port

STEP 2: CLASSIFY each port using decision tree

FOR EACH detected port:
1. Run through decision tree
2. Determine type (1, 2, 3, or 4)
3. Collect required fields

STEP 3: ASK USER to confirm

Present findings:
"I found these ports:
  1. Backend API: 8080 → Type 1 (Allocated, range: [8080, 8180])
  2. Frontend: 3000 → Type 1 (Allocated, range: [3000, 3100])
  3. PostgreSQL: 5432 → Type 1 (Allocated, range: [5432, 5532])
  4. REACT_APP_API_BASE_URL → Type 3 (String template: http://localhost:{BE_PORT})
  5. Swagger: /swagger → Type 4 (Display-only, uses backend port)

Is this correct, or should I adjust?"

STEP 4: GENERATE YAML

Based on classifications, generate complete env_variables section

STEP 5: VALIDATE

CHECK:
□ All Type 1 ports have ranges
□ Ranges don't overlap
□ Ranges are sufficient size (≥20 ports)
□ Type 3 placeholders reference valid port variables
□ Type 4 entries have env: null
□ No duplicate env variable names
```

#### Example: Complete Port Configuration

**Scenario:** Fullstack app with backend, frontend, PostgreSQL, Redis, Swagger UI

```yaml
env_variables:
  # Type 1: Allocated Ports — all names are examples, use your project's existing names
  BE_PORT:
    name: "Backend API"
    url: "http://{host}:{port}"
    port: "8080"
    env: "BE_PORT"
    range: [8080, 8180]

  FE_PORT:
    name: "Frontend"
    url: "http://{host}:{port}"
    port: "3000"
    env: "FE_PORT"
    range: [3000, 3100]

  POSTGRES_PORT:
    name: "PostgreSQL"
    url: "postgresql://{host}:{port}/dbname"
    port: "5432"
    env: "POSTGRES_PORT"
    range: [5432, 5532]

  REDIS_PORT:
    name: "Redis"
    url: "redis://{host}:{port}"
    port: "6379"
    env: "REDIS_PORT"
    range: [6379, 6479]

  # Type 3: String Templates — placeholder names must match your env_variables keys above
  REACT_APP_API_BASE_URL:
    value: "http://localhost:{BE_PORT}"
    env: "REACT_APP_API_BASE_URL"

  DATABASE_URL:
    value: "postgresql://user:pass@localhost:{POSTGRES_PORT}/mydb"
    env: "DATABASE_URL"

  REDIS_URL:
    value: "redis://localhost:{REDIS_PORT}"
    env: "REDIS_URL"

  # Type 4: Display-Only
  swagger:
    name: "Swagger UI"
    url: "http://{host}:{port}/swagger"
    port: "8080"
    env: null

  graphql:
    name: "GraphQL Playground"
    url: "http://{host}:{port}/graphql"
    port: "8080"
    env: null
```

---

## 4. Advanced Configuration

### 4.1 Symlinks vs Copies

**Symlinks:** Share files across worktrees (same file in all instances)
**Copies:** Duplicate files per worktree (independent copies)

**Decision Matrix:**

| File Type | Recommendation | Reason |
|-----------|----------------|--------|
| `.claude/` | Symlink | Shared AI context across features |
| `node_modules/` | Symlink | Save disk space (same dependencies) |
| `.env` | Copy | Each instance needs different ports |
| `.env.example` | Symlink | Template file, doesn't change |
| `package.json` | (neither) | Part of git worktree |
| `docker-compose.override.yml` | Copy | Each instance needs different ports |

**Configuration:**

```yaml
# Symlinks - shared files (one copy, linked to all worktrees)
symlinks:
  - .claude

# Copies - duplicated files (separate copy per worktree)
copies:
  - .env.example

# Note: If file doesn't exist at source, operation is skipped (warning only)
```

**AI Instructions:**

```
ASK USER: "Do you have any shared configuration files?"
Examples:
- .claude directory (AI context)
- .env.example (template)
- .editorconfig (editor settings)

FOR EACH file:
Q: "Should {file} be shared (symlink) or independent (copy)?"
  - Symlink: Changes in one place affect all instances
  - Copy: Each instance has independent copy

RECOMMENDATION:
- AI context directories → Symlink
- Template files (.example) → Symlink
- Environment files (.env) → Copy (or generate per instance)
- Large dependencies (node_modules) → Symlink (careful!)
```

### 4.2 Generated Files

**Generated files** are templates with port placeholders, automatically created per worktree.

**Example: Generate .env file per instance**

```yaml
generated_files:
  backend:
    - file: .env
      template: |
        # Auto-generated .env for instance {instance}
        NODE_ENV=development
        PORT={BE_PORT}
        DATABASE_URL=postgresql://user:pass@localhost:{POSTGRES_PORT}/mydb
        REDIS_URL=redis://localhost:{REDIS_PORT}
        API_BASE_URL=http://localhost:{BE_PORT}

  frontend:
    - file: .env.local
      template: |
        # Auto-generated .env.local for instance {instance}
        VITE_API_BASE_URL=http://localhost:{BE_PORT}
        VITE_APP_PORT={FE_PORT}
```

**Placeholders Available:**

- `{instance}` - Instance number (0, 1, 2, ...)
- `{feature}` - Feature name (feature-name)
- `{BE_PORT}`, `{FE_PORT}`, etc. - Any port variable from env_variables
- Any env variable defined in env_variables section

**AI Instructions:**

```
ASK USER: "Do you need auto-generated .env files per instance?"

IF YES:
  Q: "Which projects need .env files?" (backend, frontend, worker)
  Q: "What variables should be in the .env file?"

Generate template with port placeholders.

EXAMPLE:
User: "Backend needs PORT, DATABASE_URL, REDIS_URL"
AI generates:
  PORT={BE_PORT}
  DATABASE_URL=postgresql://localhost:{POSTGRES_PORT}/db
  REDIS_URL=redis://localhost:{REDIS_PORT}
```

### 4.3 Lifecycle Hooks

**Hooks** are commands run at specific points in the lifecycle.

**Available Hooks:**

| Hook | When | Use Case |
|------|------|----------|
| `start_pre_command` | Before start_command | Check dependencies, validate env |
| `start_command` | Main start | Start service (docker-compose up, npm start) |
| `start_post_command` | After start_command | Migrations, seed data, generate types |
| `stop_pre_command` | Before stop | Drain connections, backup data |
| `stop_post_command` | After stop | Cleanup, verify stopped |
| `restart_pre_command` | Before restart (before stop) | Backup state before restart |
| `restart_post_command` | After restart (after start) | Verify health, warm up cache |

**Hook Behavior:**

- `start_command` failure = FATAL (stops workflow)
- All other hooks failure = WARNING (continues workflow)

**Examples:**

```yaml
projects:
  backend:
    start_pre_command: "make check-deps"           # Verify deps before start
    start_command: "docker-compose up -d"          # Start services
    start_post_command: "make migrate && make seed" # Run migrations + seed
    stop_pre_command: "make drain-connections"     # Graceful drain
    stop_post_command: "make verify-stopped"       # Verify clean stop
    restart_pre_command: "make backup-state"       # Backup before restart
    restart_post_command: "make verify-health"     # Health check after restart

  frontend:
    start_command: "npm start"
    start_post_command: "npm run generate-types"   # Generate types from API
```

**AI Instructions:**

```
FOR EACH project:
  Q: "Should I run anything BEFORE starting {project}?" (pre)
     Examples: npm install, make check-deps
     → start_pre_command

  Q: "Should I run anything AFTER starting {project}?" (post)
     Examples: make migrate, npm run generate-types
     → start_post_command

  Q: "Should I run anything BEFORE stopping {project}?" (pre-stop)
     Examples: make drain-connections, backup data
     → stop_pre_command

  Q: "Should I run anything AFTER stopping {project}?" (post-stop)
     Examples: cleanup, verify stopped
     → stop_post_command

Most projects only need start_post_command (migrations, seed).
```

---

## 5. Claude Code Integration

This section covers **Claude-specific** features for autonomous operation.

### 5.1 YOLO Mode

**YOLO Mode** signals Claude Code to work autonomously without confirmation on clear tasks.

**Enabling YOLO Mode:**

```bash
# Enable at worktree creation
worktree new-feature feature/my-feature --yolo

# Enable for existing worktree
worktree yolo feature-my-feature

# Disable YOLO mode
worktree yolo feature-my-feature --disable
```

**What YOLO Mode Does:**

- **Enabled:** Claude makes decisions without asking for confirmation
  - Example: "Should I run migrations?" → Runs automatically
  - Example: "Should I commit this?" → Commits automatically

- **Disabled (default):** Claude asks before risky operations
  - Example: "Should I run migrations?" → Asks user first
  - Example: "Should I commit this?" → Asks user first

**AI Instructions:**

```
WHEN to suggest YOLO mode:

User signals trust/autonomy:
- "Work autonomously"
- "Don't ask me for confirmation"
- "Just do it"
- "I trust you"

Clear task with low risk:
- "Implement user CRUD endpoints" (standard pattern)
- "Add validation to form" (clear scope)

WHEN to disable YOLO mode (or not suggest):

Default behavior:
- First time working in codebase
- No explicit trust signal

Risky operations:
- Destructive changes (delete endpoints, drop tables)
- Production deployments
- Large refactors

ASK USER:
"Should I enable YOLO mode? This allows me to work autonomously
without asking for confirmation on clear tasks. (Recommended for
experienced users working on standard features.)"
```

**Configuration Persistence:**

YOLO mode is stored in:
1. Registry: `worktrees/.registry.json` (YoloMode field)
2. Instance marker: `worktrees/feature-name/.worktree-instance` (yolo_mode field)

**Commands:**

```bash
# Check YOLO mode status
worktree status feature-my-feature
# Output: YOLO Mode: Enabled / Disabled

# Toggle YOLO mode
worktree yolo feature-my-feature         # Enable
worktree yolo feature-my-feature --disable # Disable
```

### 5.2 Claude Working Directory

**Claude Working Directory** is where Claude navigates after `worktree new-feature` completes.

**Configuration:**

```yaml
projects:
  backend:
    claude_working_dir: true    # ← Claude navigates here after setup
    dir: backend
    # ... other config

  frontend:
    dir: frontend
    # ... other config (no claude_working_dir)
```

**Selection Logic:**

```
Priority order:
1. User explicitly specifies project
2. Project has claude_working_dir: true
3. First project in preset (default)

Example:
Preset: fullstack [backend, frontend]
Config: backend has claude_working_dir: true
Result: Claude navigates to worktrees/feature-name/backend/
```

**AI Instructions:**

```
WHEN configuring projects:
  Q: "Will you primarily work in {project}/ when Claude Code starts?"

  IF user says YES:
    Set claude_working_dir: true for that project

  IF user says NO or unsure:
    Leave unset (defaults to first project in preset)

RECOMMENDATION:
- Fullstack preset → Set backend as working dir
- Backend preset → Set backend as working dir
- Frontend preset → Set frontend as working dir
```

**Example Scenario:**

```bash
# Create worktree with fullstack preset
worktree new-feature feature/auth

# After completion, Claude navigates to:
cd worktrees/feature-auth/backend/

# Because backend has claude_working_dir: true
```

### 5.3 Instance Auto-Detection

**Instance Auto-Detection** allows commands to automatically detect which worktree instance they're in.

**How It Works:**

1. When worktree created: `.worktree-instance` marker file written to feature root
2. Commands walk up from CWD to find marker
3. If found: Auto-detect feature name (no need to specify)

**Marker File Contents:**

```json
{
  "feature": "feature-auth",
  "instance": 1,
  "project_root": "/Users/me/project",
  "projects": ["backend", "frontend"],
  "ports": {
    "BE_PORT": 8081,
    "FE_PORT": 3001,
    "POSTGRES_PORT": 5433
  },
  "yolo_mode": true
}
```

**Commands Supporting Auto-Detection:**

All these commands accept **optional** feature name:

```bash
# From project root - feature name REQUIRED
worktree status feature-auth
worktree ports feature-auth
worktree start feature-auth
worktree stop feature-auth

# From worktree directory - feature name OPTIONAL (auto-detected)
cd worktrees/feature-auth/backend
worktree status                  # ✨ Auto-detected: feature-auth
worktree ports                   # ✨ Auto-detected: feature-auth
worktree start                   # ✨ Auto-detected: feature-auth
worktree stop                    # ✨ Auto-detected: feature-auth
```

**AI Instructions:**

```
WHEN running worktree commands:

CHECK: Are we in a worktree directory?
  Run: pwd
  IF path contains "worktrees/" → Likely in worktree

IF in worktree directory:
  TRY: Run command WITHOUT feature name
  Example: worktree status
  IF successful → Auto-detection worked ✓
  IF error "Not in worktree and no feature name" → Use explicit name

IF NOT in worktree directory:
  ALWAYS: Provide feature name explicitly
  Example: worktree status feature-auth

BENEFIT:
- Less typing when working inside worktree
- Commands "just work" from any subdirectory
```

**Example Session:**

```bash
# Start in project root
pwd
# /Users/me/project

# Need to specify feature
worktree status feature-auth

# Navigate into worktree
cd worktrees/feature-auth/backend

# Auto-detection works
worktree status
# ✨ Auto-detected from current directory: feature-auth
# Status: Running
# Ports: BE_PORT=8081, FE_PORT=3001, POSTGRES_PORT=5433

# Even from subdirectories
cd src/controllers
worktree status
# ✨ Auto-detected from current directory: feature-auth
```

---

## 6. Common Patterns

Copy-paste configurations for typical scenarios.

### 6.1 Pattern 1: Simple Fullstack (Node.js + Docker)

**Scenario:** Node.js backend (docker-compose) + React frontend (npm start)

```yaml
project_name: "myproject"
hostname: localhost
default_preset: fullstack

projects:
  backend:
    executor: docker
    dir: backend
    main_branch: main
    start_command: "docker-compose up -d"
    start_post_command: "make migrate && make seed"
    claude_working_dir: true

  frontend:
    executor: process
    dir: frontend
    main_branch: main
    start_command: "npm start"

presets:
  fullstack:
    projects: [backend, frontend]
    description: "Backend + Frontend"
  backend:
    projects: [backend]
    description: "Backend only"
  frontend:
    projects: [frontend]
    description: "Frontend only"

env_variables:
  # All names are examples — use whatever your project already calls these ports
  BE_PORT:
    name: "Backend API"
    url: "http://{host}:{port}"
    port: "8080"
    env: "BE_PORT"
    range: [8080, 8180]

  FE_PORT:
    name: "Frontend"
    url: "http://{host}:{port}"
    port: "3000"
    env: "FE_PORT"
    range: [3000, 3100]

  POSTGRES_PORT:
    name: "PostgreSQL"
    url: "postgresql://{host}:{port}/dbname"
    port: "5432"
    env: "POSTGRES_PORT"
    range: [5432, 5532]

  REACT_APP_API_BASE_URL:
    value: "http://localhost:{BE_PORT}"
    env: "REACT_APP_API_BASE_URL"

  DATABASE_URL:
    value: "postgresql://user:pass@localhost:{POSTGRES_PORT}/mydb"
    env: "DATABASE_URL"

  swagger:
    name: "Swagger UI"
    url: "http://{host}:{port}/swagger"
    port: "8080"
    env: null
```

### 6.2 Pattern 2: Go Microservices

**Scenario:** Multiple Go services with process executor

```yaml
project_name: "myservices"
hostname: localhost
default_preset: all

projects:
  api:
    executor: process
    dir: services/api
    main_branch: main
    start_command: "go run main.go"
    start_post_command: "make migrate"
    claude_working_dir: true

  worker:
    executor: process
    dir: services/worker
    main_branch: main
    start_command: "go run main.go"

  auth:
    executor: process
    dir: services/auth
    main_branch: main
    start_command: "go run main.go"

presets:
  all:
    projects: [api, worker, auth]
    description: "All services"
  api-only:
    projects: [api]
    description: "API service only"

env_variables:
  # All names are examples — use whatever your project already calls these ports
  API_PORT:
    name: "API Service"
    url: "http://{host}:{port}"
    port: "8080"
    env: "API_PORT"
    range: [8080, 8180]

  WORKER_PORT:
    name: "Worker Service"
    url: "http://{host}:{port}"
    port: "8081"
    env: "WORKER_PORT"
    range: [8081, 8181]

  AUTH_PORT:
    name: "Auth Service"
    url: "http://{host}:{port}"
    port: "8082"
    env: "AUTH_PORT"
    range: [8082, 8182]

  POSTGRES_PORT:
    name: "PostgreSQL"
    url: "postgresql://{host}:{port}/dbname"
    port: "5432"
    env: "POSTGRES_PORT"
    range: [5432, 5532]

  REDIS_PORT:
    name: "Redis"
    url: "redis://{host}:{port}"
    port: "6379"
    env: "REDIS_PORT"
    range: [6379, 6479]

  DATABASE_URL:
    value: "postgresql://user:pass@localhost:{POSTGRES_PORT}/mydb"
    env: "DATABASE_URL"

  REDIS_URL:
    value: "redis://localhost:{REDIS_PORT}"
    env: "REDIS_URL"
```

### 6.3 Pattern 3: Python Monorepo

**Scenario:** Django backend + React frontend

```yaml
project_name: "myapp"
hostname: localhost
default_preset: fullstack

projects:
  backend:
    executor: process
    dir: backend
    main_branch: main
    start_pre_command: "source venv/bin/activate"
    start_command: "python manage.py runserver 0.0.0.0:{BE_PORT}"
    start_post_command: "python manage.py migrate"

  frontend:
    executor: process
    dir: frontend
    main_branch: main
    start_command: "npm start"
    claude_working_dir: true

presets:
  fullstack:
    projects: [backend, frontend]
    description: "Backend + Frontend"

env_variables:
  # All names are examples — use whatever your project already calls these ports
  BE_PORT:
    name: "Django Backend"
    url: "http://{host}:{port}"
    port: "8000"
    env: "BE_PORT"
    range: [8000, 8100]

  FE_PORT:
    name: "React Frontend"
    url: "http://{host}:{port}"
    port: "3000"
    env: "FE_PORT"
    range: [3000, 3100]

  POSTGRES_PORT:
    name: "PostgreSQL"
    url: "postgresql://{host}:{port}/dbname"
    port: "5432"
    env: "POSTGRES_PORT"
    range: [5432, 5532]

  REACT_APP_API_BASE_URL:
    value: "http://localhost:{BE_PORT}"
    env: "REACT_APP_API_BASE_URL"

  DATABASE_URL:
    value: "postgresql://user:pass@localhost:{POSTGRES_PORT}/mydb"
    env: "DATABASE_URL"
```

### 6.4 Pattern 4: Complex Stack

**Scenario:** Backend + Frontend + Worker + Multiple DBs + Message Queue

```yaml
project_name: "complex-app"
hostname: localhost
default_preset: fullstack

projects:
  backend:
    executor: docker
    dir: backend
    main_branch: main
    start_command: "docker-compose up -d"
    start_post_command: "make migrate && make seed"
    claude_working_dir: true

  frontend:
    executor: process
    dir: frontend
    main_branch: main
    start_command: "npm start"

  worker:
    executor: process
    dir: worker
    main_branch: main
    start_command: "python worker.py"

presets:
  fullstack:
    projects: [backend, frontend]
    description: "Backend + Frontend"
  all:
    projects: [backend, frontend, worker]
    description: "All services"

env_variables:
  # All names are examples — use whatever your project already calls these ports
  BE_PORT:
    name: "Backend API"
    url: "http://{host}:{port}"
    port: "8080"
    env: "BE_PORT"
    range: [8080, 8180]

  FE_PORT:
    name: "Frontend"
    url: "http://{host}:{port}"
    port: "3000"
    env: "FE_PORT"
    range: [3000, 3100]

  POSTGRES_PORT:
    name: "PostgreSQL"
    url: "postgresql://{host}:{port}/dbname"
    port: "5432"
    env: "POSTGRES_PORT"
    range: [5432, 5532]

  REDIS_PORT:
    name: "Redis"
    url: "redis://{host}:{port}"
    port: "6379"
    env: "REDIS_PORT"
    range: [6379, 6479]

  RABBITMQ_PORT:
    name: "RabbitMQ"
    url: "amqp://{host}:{port}"
    port: "5672"
    env: "RABBITMQ_PORT"
    range: [5672, 5772]

  RABBITMQ_UI_PORT:
    name: "RabbitMQ Management UI"
    url: "http://{host}:{port}"
    port: "15672"
    env: "RABBITMQ_UI_PORT"
    range: [15672, 15772]

  MAILPIT_SMTP_PORT:
    name: "Mailpit SMTP"
    url: "smtp://{host}:{port}"
    port: "1025"
    env: "MAILPIT_SMTP_PORT"
    range: [1025, 1125]

  MAILPIT_UI_PORT:
    name: "Mailpit UI"
    url: "http://{host}:{port}"
    port: "8025"
    env: "MAILPIT_UI_PORT"
    range: [8025, 8125]

  REACT_APP_API_BASE_URL:
    value: "http://localhost:{BE_PORT}"
    env: "REACT_APP_API_BASE_URL"

  DATABASE_URL:
    value: "postgresql://user:pass@localhost:{POSTGRES_PORT}/mydb"
    env: "DATABASE_URL"

  REDIS_URL:
    value: "redis://localhost:{REDIS_PORT}"
    env: "REDIS_URL"

  RABBITMQ_URL:
    value: "amqp://guest:guest@localhost:{RABBITMQ_PORT}/"
    env: "RABBITMQ_URL"

  swagger:
    name: "Swagger UI"
    url: "http://{host}:{port}/swagger"
    port: "8080"
    env: null

  rabbitmq_ui:
    name: "RabbitMQ Management"
    url: "http://{host}:{port}"
    port: "15672"
    env: null
```

---

## 7. Troubleshooting

AI-actionable solutions for common problems.

### 7.1 Port Conflicts

**Symptom:** Error "port already in use"

**AI Diagnostic:**

```bash
# Run health check
worktree doctor

# Check specific port
lsof -i :8080
netstat -an | grep 8080  # Linux
```

**AI Action:**

```
STEP 1: Run worktree doctor
Run: worktree doctor

IF output shows "Port conflict detected":
  → Continue to Step 2
ELSE:
  → Port conflict is external (not worktree-managed)
  → Continue to Step 3

STEP 2: Try auto-fix
Run: worktree doctor --fix

IF successful:
  → Problem solved ✓
ELSE:
  → Continue to Step 3

STEP 3: Check ranges
Read .worktree.yml env_variables ranges

IF range exhausted (all ports allocated):
  SUGGEST: Expand range
  Example: [8080, 8130] → [8080, 8180] (+50 ports)

IF ranges overlap:
  SUGGEST: Separate ranges
  Example: BE [8080, 8180], FE [8080, 8180] (WRONG!)
         → BE [8080, 8180], FE [3000, 3100] (CORRECT)

STEP 4: Check external conflicts
Run: lsof -i :8080

IF port used by external service:
  SUGGEST: Different base port
  Example: Change BE_PORT from 8080 to 9080
```

**Example:**

```
AI: I see the error "port 8080 already in use". Let me investigate.

Run: worktree doctor
Output: "Port conflict: BE_PORT 8080 already allocated to feature-a"

AI: The port range is exhausted. Current range: [8080, 8130] (50 ports).
    You have 50 instances using all ports.

    I'll expand the range to [8080, 8180] (100 ports) in .worktree.yml.

    [Edit .worktree.yml]

AI: Fixed! Range expanded. Try creating the worktree again.
```

### 7.2 Git Worktree Creation Failed

**Symptom:** Error "fatal: not a git repository"

**AI Diagnostic:**

```bash
# Check if git repo
cd backend && git status

# Check main branch
git branch --list main develop master

# Check commit history
git log --oneline -n 1
```

**AI Action:**

```
STEP 1: Check if git repo
Run: cd {project_dir} && git status

IF "not a git repository":
  → Auto-fix: git init
  → Continue to Step 2
ELSE:
  → Is git repo ✓
  → Continue to Step 2

STEP 2: Initialize git if needed
IF not git repo:
  Run:
    git init
    git add .
    git commit -m "initial commit"

STEP 3: Check main branch exists
Run: git branch

IF main branch doesn't exist:
  Run: git branch main && git checkout main

STEP 4: Check ≥1 commit
Run: git log --oneline -n 1

IF no commits:
  Run: git commit --allow-empty -m "init"

STEP 5: Retry worktree creation
Run: worktree new-feature {feature-name}
```

**Example:**

```
AI: Error creating worktree: "not a git repository"

AI: Checking backend/ directory...
Run: cd backend && git status
Output: "fatal: not a git repository"

AI: backend/ is not a git repository. I'll initialize it.

Run:
  cd backend
  git init
  git add .
  git commit -m "initial commit"

AI: Git repository initialized ✓
    Try creating the worktree again:
    worktree new-feature feature-auth
```

### 7.3 Docker Compose Fails

**Symptom:** Services don't start, docker-compose errors

**AI Diagnostic:**

```bash
# Check Docker running
docker ps

# Check env vars exported
worktree ports feature-name

# Check docker-compose.yml syntax
cd worktrees/feature-name/backend
docker-compose config
```

**AI Action:**

```
STEP 1: Check Docker daemon
Run: docker ps

IF "Cannot connect to Docker daemon":
  ASK USER: "Please start Docker Desktop or Docker daemon"
  WAIT: User confirms Docker started
ELSE:
  → Docker is running ✓

STEP 2: Check environment variables
Run: worktree ports feature-name

Verify exported vars:
  BE_PORT=8081
  FE_PORT=3001
  POSTGRES_PORT=5433
  ...

STEP 3: Check docker-compose.yml uses env vars
Read: worktrees/feature-name/backend/docker-compose.yml

LOOK FOR: ${BE_PORT}, ${POSTGRES_PORT}, etc.

IF docker-compose.yml has hardcoded ports:
  PROBLEM: docker-compose.yml doesn't use env vars
  SUGGEST: Update docker-compose.yml to use ${PORT_VAR}

  Example:
    ports:
      - "8080:8080"  # WRONG (hardcoded)

    Should be:
      - "${BE_PORT}:8080"  # CORRECT (uses env var)

STEP 4: Check docker-compose syntax
Run: cd worktrees/feature-name/backend && docker-compose config

IF syntax error:
  SHOW: Error message
  SUGGEST: Fix YAML syntax

STEP 5: Check container logs
Run: docker-compose logs

ANALYZE: Logs for errors (database connection, missing env vars, etc.)
```

**Example:**

```
AI: Docker compose failed to start. Let me investigate.

Run: docker ps
Output: [list of containers] ✓

Run: worktree ports feature-auth
Output:
  BE_PORT=8081
  POSTGRES_PORT=5433

AI: Environment variables are exported correctly ✓

Read: worktrees/feature-auth/backend/docker-compose.yml
Found:
  ports:
    - "8080:8080"  # Hardcoded port!

AI: Problem found! docker-compose.yml uses hardcoded port 8080.
    It should use ${BE_PORT} instead.

    I'll update docker-compose.yml:

    ports:
      - "${BE_PORT}:8080"

    [Edit docker-compose.yml]

AI: Fixed! Try starting again: worktree start feature-auth
```

### 7.4 Process Executor Doesn't Stop

**Symptom:** Service still running after `worktree stop`

**AI Diagnostic:**

```bash
# Check PID file
ls worktrees/feature-name/*.pid

# Check process
ps aux | grep <process-name>

# Check process by PID
cat worktrees/feature-name/backend.pid
ps -p <PID>
```

**AI Action:**

```
STEP 1: Check PID file exists
Run: ls worktrees/feature-name/*.pid

IF PID file doesn't exist:
  PROBLEM: Process didn't start correctly
  SUGGEST: Check start_command in .worktree.yml
ELSE:
  → PID file exists
  → Continue to Step 2

STEP 2: Check process is running
Run: cat worktrees/feature-name/backend.pid
     ps -p <PID>

IF process is running:
  → Process didn't stop
  → Continue to Step 3
ELSE:
  → Process stopped but PID file remains (stale)
  → Run: rm worktrees/feature-name/backend.pid
  → Done ✓

STEP 3: Manually kill process
Run: kill <PID>
Wait: 5 seconds
Run: ps -p <PID>

IF still running:
  Run: kill -9 <PID>  # Force kill

Run: rm worktrees/feature-name/backend.pid

STEP 4: Verify stopped
Run: worktree status feature-name
Output should show: "Stopped"
```

**Example:**

```
AI: Process still running after stop. Let me investigate.

Run: ls worktrees/feature-auth/*.pid
Output: backend.pid frontend.pid

Run: cat worktrees/feature-auth/backend.pid
Output: 12345

Run: ps -p 12345
Output: [process details] ← Still running!

AI: Backend process (PID 12345) is still running. I'll stop it manually.

Run: kill 12345
Wait: 5 seconds
Run: ps -p 12345
Output: No such process ✓

Run: rm worktrees/feature-auth/backend.pid

AI: Process stopped and PID file removed ✓
```

### 7.5 Registry Corruption

**Symptom:** `worktree list` shows wrong data, commands fail

**AI Diagnostic:**

```bash
# Check registry file
cat worktrees/.registry.json

# Validate JSON syntax
python3 -m json.tool worktrees/.registry.json

# Run health check
worktree doctor
```

**AI Action:**

```
STEP 1: Backup registry
Run: cp worktrees/.registry.json worktrees/.registry.json.bak

STEP 2: Validate JSON
Run: python3 -m json.tool worktrees/.registry.json

IF JSON invalid:
  PROBLEM: Registry file is corrupted
  → Continue to Step 3 (fix)
ELSE:
  → JSON is valid
  → Continue to Step 4 (data inconsistency)

STEP 3: Fix JSON syntax
IF JSON invalid:
  READ: worktrees/.registry.json
  IDENTIFY: Syntax error (missing comma, bracket, quote)
  FIX: Correct syntax
  SAVE: worktrees/.registry.json

STEP 4: Run doctor
Run: worktree doctor

IF issues detected:
  Run: worktree doctor --fix

STEP 5: Verify
Run: worktree list

IF still broken:
  SUGGEST: Restore backup or rebuild registry
  "I can restore the backup:
   cp worktrees/.registry.json.bak worktrees/.registry.json

   Or rebuild registry from scratch (loses data):
   rm worktrees/.registry.json
   worktree list  # Rebuilds from actual worktrees"
```

**Example:**

```
AI: Registry appears corrupted. Let me investigate.

Run: cp worktrees/.registry.json worktrees/.registry.json.bak
AI: Backup created ✓

Run: python3 -m json.tool worktrees/.registry.json
Output: Expecting ',' delimiter: line 15 column 5

AI: JSON syntax error found at line 15. Let me fix it.

Read: worktrees/.registry.json
Found: Missing comma after entry

[Edit worktrees/.registry.json - add comma]

Run: python3 -m json.tool worktrees/.registry.json
Output: [Valid JSON] ✓

Run: worktree doctor
Output: All checks passed ✓

AI: Registry fixed! Try your command again.
```

---

## 8. Quick Reference

Fast lookup for AI during execution.

### 8.1 Command Reference

#### Lifecycle Commands

```bash
# Create worktree with default preset
worktree new-feature <branch>

# Create with specific preset
worktree new-feature <branch> <preset>

# Create with YOLO mode
worktree new-feature <branch> --yolo

# Create without post-commands (no migrations, seed)
worktree new-feature <branch> --no-fixtures

# Start services
worktree start <feature-name>

# Stop services
worktree stop <feature-name>

# Restart services
worktree restart <feature-name>

# Remove worktree
worktree remove <feature-name>
```

#### Status Commands

```bash
# List all worktrees
worktree list

# Show detailed status
worktree status <feature-name>

# Show port allocations
worktree ports <feature-name>

# Health check
worktree doctor

# Fix issues
worktree doctor --fix
```

#### YOLO Mode

```bash
# Enable YOLO mode
worktree yolo <feature-name>

# Disable YOLO mode
worktree yolo <feature-name> --disable

# Check YOLO status
worktree status <feature-name>  # Shows YOLO mode status
```

#### Agent Commands (Scheduled Tasks)

```bash
# List all agents
worktree agent list

# Validate agent config
worktree agent validate <agent-name>

# Run agent manually
worktree agent run <agent-name>

# Schedule agent (cron/launchd)
worktree agent schedule <agent-name>

# Schedule all agents
worktree agent schedule --all
```

### 8.2 File Locations

```
Project root/
├── .worktree.yml              # Main configuration (YOU create this)
├── worktrees/                 # All feature instances
│   ├── .registry.json         # Port allocation registry (AUTO-GENERATED)
│   ├── feature-a/             # Feature directory
│   │   ├── .worktree-instance # Instance marker (AUTO-GENERATED)
│   │   ├── backend/           # Git worktree
│   │   └── frontend/          # Git worktree
│   └── feature-b/             # Another feature
│       ├── .worktree-instance
│       ├── backend/
│       └── frontend/
├── backend/                   # Main backend repo
│   ├── .git/                  # Main git directory
│   ├── docker-compose.yml
│   └── ...
└── frontend/                  # Main frontend repo
    ├── .git/                  # Main git directory
    ├── package.json
    └── ...
```

### 8.3 Configuration Fields Quick Reference

**Required Top-Level Fields:**

```yaml
project_name: "myproject"      # Alphanumeric + hyphens, no start/end hyphen
hostname: localhost            # Default: localhost
default_preset: "fullstack"    # Default preset name
```

**Projects (at least one required):**

```yaml
projects:
  <project-name>:
    dir: <path>                # Required: relative path from project root
    main_branch: <branch>      # Required: main/develop/master
    executor: docker|process   # Optional: default is docker
    start_command: <cmd>       # Required: command to start service
    start_pre_command: <cmd>   # Optional: before start
    start_post_command: <cmd>  # Optional: after start (migrations, seed)
    stop_pre_command: <cmd>    # Optional: before stop
    stop_post_command: <cmd>   # Optional: after stop
    restart_pre_command: <cmd> # Optional: before restart
    restart_post_command: <cmd># Optional: after restart
    claude_working_dir: true   # Optional: default false
```

**Presets (at least one required):**

```yaml
presets:
  <preset-name>:
    projects: [list]           # Required: list of project names
    description: <text>        # Optional: human-readable description
```

**Port Variables (Type 1: Allocated):**

```yaml
env_variables:
  <VAR_NAME>:
    name: <display-name>       # Required: "Backend API"
    url: <url-template>        # Required: "http://{host}:{port}"
    port: <base-port>          # Required: "8080"
    env: <VAR_NAME>            # Required: "BE_PORT"
    range: [min, max]          # Required: [8080, 8180]
```

**Port Variables (Type 2: Calculated):**

```yaml
env_variables:
  <VAR_NAME>:
    port: <expression>         # Required: "4510 + {instance} * 50"
    env: <VAR_NAME>            # Required: "LOCALSTACK_EXT_START"
    # NO range field
```

**Port Variables (Type 3: String Template):**

```yaml
env_variables:
  <VAR_NAME>:
    value: <template>          # Required: "http://localhost:{BE_PORT}"
    env: <VAR_NAME>            # Required: "REACT_APP_API_BASE_URL"
    # NO port or range
```

**Port Variables (Type 4: Display-Only):**

```yaml
env_variables:
  <name>:
    name: <display-name>       # Required: "Swagger UI"
    url: <url-template>        # Required: "http://{host}:{port}/swagger"
    port: <reference>          # Required: "8080" (base port reference)
    env: null                  # Required: MUST be null
```

### 8.4 Common Errors and Quick Fixes

| Error | Quick Fix |
|-------|-----------|
| "command not found: worktree" | `go install github.com/braunmar/worktree@latest` |
| "port already in use" | `worktree doctor --fix` |
| "not a git repository" | `git init && git commit --allow-empty` |
| "Cannot connect to Docker daemon" | Start Docker Desktop |
| "range exhausted" | Expand range in .worktree.yml |
| JSON syntax error | `python3 -m json.tool .registry.json` |

---

## 9. Validation Workflow

Help AI verify setup is correct.

### 9.1 Configuration Validation Steps

**AI should run these steps after creating .worktree.yml:**

```bash
# Step 1: Syntax check
cat .worktree.yml
# → Valid YAML? No syntax errors?

# Step 2: Test configuration (dry run if available, or just try)
# Note: --dry-run flag may not exist, just document what would happen
worktree new-feature test/validation

# Step 3: Check status
worktree status test-validation
# → Shows: Running, Stopped, or Error?

# Step 4: Check port allocations
worktree ports test-validation
# → All ports allocated correctly?
# → Env vars exported?

# Step 5: Verify services accessible
# Visit URLs from `worktree ports` output
# Example:
curl http://localhost:8080/health    # Backend health check
curl http://localhost:3000            # Frontend loads

# Step 6: Health check
worktree doctor
# → All checks pass?

# Step 7: Clean up
worktree remove test-validation
```

### 9.2 AI Validation Checklist

**Before proceeding, AI should verify:**

```
Configuration Validation:
□ .worktree.yml is valid YAML (no syntax errors)
□ project_name is alphanumeric + hyphens
□ All required fields present (project_name, hostname, projects, presets)
□ At least one project defined
□ At least one preset defined
□ default_preset references existing preset

Projects Validation:
□ All projects have dir field
□ All projects have main_branch field
□ All projects have start_command field
□ All project directories exist
□ All project directories are git repos
□ All project git repos have ≥1 commit
□ executor types are correct (docker vs process)
□ claude_working_dir set for ≤1 project

Port Validation:
□ All Type 1 ports have range field
□ Port ranges don't overlap
□ Port ranges are sufficient size (≥20 ports)
□ Port ranges don't conflict with system ports (<1024)
□ Type 2 ports have valid expressions
□ Type 3 placeholders reference existing port vars
□ Type 4 ports have env: null
□ No duplicate env variable names

Prerequisites Validation:
□ worktree-manager installed (worktree --version works)
□ Go 1.22+ installed (if needed)
□ Docker installed and running (if executor: docker)
□ All project directories exist

Health Check:
□ worktree doctor passes
□ No port conflicts
□ No stale worktrees
```

### 9.3 Example Validation Session

```
AI: Configuration created! Let me validate the setup.

Step 1: Check YAML syntax
Run: cat .worktree.yml
Output: [Valid YAML] ✓

Step 2: Create test worktree
Run: worktree new-feature test/validation
Output:
  ✓ Allocated ports: BE=8080, FE=3000, PG=5432
  ✓ Created worktree: backend/
  ✓ Created worktree: frontend/
  ✓ Started services

✓ Test worktree created successfully

Step 3: Check status
Run: worktree status test-validation
Output:
  Feature: test-validation
  Instance: 0
  Status: Running
  Projects: backend (✓), frontend (✓)

✓ Status looks good

Step 4: Check ports
Run: worktree ports test-validation
Output:
  Backend API: http://localhost:8080
  Frontend: http://localhost:3000
  PostgreSQL: postgresql://localhost:5432
  Swagger UI: http://localhost:8080/swagger

✓ All ports allocated correctly

Step 5: Verify services
Run: curl -s http://localhost:8080/health
Output: {"status": "healthy"}

Run: curl -s http://localhost:3000
Output: [HTML content]

✓ Services are accessible

Step 6: Health check
Run: worktree doctor
Output:
  ✓ Git repositories healthy
  ✓ No port conflicts
  ✓ No stale worktrees
  ✓ Registry consistent

✓ All health checks passed

Step 7: Clean up
Run: worktree remove test-validation
Output: ✓ Removed test-validation

✓ Validation complete! Configuration is correct.

You can now create real feature worktrees:
  worktree new-feature feature/my-feature
```

---

## 10. Additional Resources

### 10.1 Documentation

- **[README.md](README.md)** - Quick start, installation, overview
- **[.worktree.example.yml](.worktree.example.yml)** - Complete example configuration with inline comments
  - Fetch via: `curl -s "https://raw.githubusercontent.com/braunmar/worktree/main/.worktree.example.yml"`
- **[.worktree.example-real.yml](.worktree.example-real.yml)** - Real-world configuration from project
  - Fetch via: `curl -s "https://raw.githubusercontent.com/braunmar/worktree/main/.worktree.example-real.yml"`
- **[AGENTS.md](AGENTS.md)** - Architecture, development patterns, package organization (for developers)
- **[CONTRIBUTING.md](CONTRIBUTING.md)** - Contribution guidelines, code style, architecture
- **[examples/](examples/)** - Practical examples and tutorials

### 10.2 When to Reference Other Docs

**Use SKILL.md (this file) for:**
- ✅ Setting up worktree-manager for a new project
- ✅ Configuration wizard (creating .worktree.yml)
- ✅ Decision trees (executor, ports)
- ✅ Troubleshooting common issues
- ✅ Quick reference (commands, fields)

**Use .worktree.example.yml for:**
- ✅ Copy-paste starting point
- ✅ All configuration options with inline comments
- ✅ Complex examples (scheduled agents, generated files)

**Use AGENTS.md for:**
- ✅ Developing worktree-manager itself (contributing code)
- ✅ Understanding internal architecture
- ✅ Package organization, registry system, port allocation logic

**Use README.md for:**
- ✅ Quick start (if user already knows what they want)
- ✅ Installation methods
- ✅ Project overview

### 10.3 Help and Support

**For Users:**
- GitHub Issues: https://github.com/braunmar/worktree/issues
- GitHub Discussions: https://github.com/braunmar/worktree/discussions

**For Contributors:**
- See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines
- See [AGENTS.md](AGENTS.md) for architecture

**Version Information:**
```bash
worktree --version
```

---

## Appendix: AI Workflow Summary

> **Starting fresh?** Go to the **⚡ AGENT ENTRY POINT** at the very top of this file. It tells you exactly what to do without reading everything.

**High-Level AI Workflow for Setting Up Worktree-Manager:**

```
1. UNDERSTAND PROBLEM
   └─> Read Section 1 (Overview)
   └─> Confirm project fits criteria

2. CHECK PREREQUISITES
   └─> Read Section 2 (Prerequisites)
   └─> Run diagnostic commands
   └─> Fix any issues (git init, install worktree, etc.)

3. CONFIGURATION WIZARD
   └─> Read Section 3 (Configuration Wizard)
   └─> Ask user basic questions (project name, hostname)
   └─> Discover projects (ls, detect types)
   └─> Configure each project (branch, start command, executor)
   └─> Design presets (groups of projects)
   └─> Configure ports (CRITICAL - use decision trees)
       └─> Type 1: Allocated (most common)
       └─> Type 2: Calculated (rare)
       └─> Type 3: String templates
       └─> Type 4: Display-only

4. ADVANCED (OPTIONAL)
   └─> Read Section 4 (Advanced)
   └─> Configure symlinks/copies if needed
   └─> Configure generated files if needed
   └─> Configure lifecycle hooks if needed

5. CLAUDE CODE INTEGRATION (IF APPLICABLE)
   └─> Read Section 5 (Claude Code)
   └─> Enable YOLO mode if user trusts autonomous operation
   └─> Set claude_working_dir for main project

6. VALIDATION
   └─> Read Section 9 (Validation)
   └─> Create test worktree
   └─> Verify services start
   └─> Run health check (worktree doctor)
   └─> Clean up test worktree

7. TROUBLESHOOTING (IF NEEDED)
   └─> Read Section 7 (Troubleshooting)
   └─> Follow diagnostic steps
   └─> Apply fixes

8. USER HANDOFF
   └─> Show common commands (Section 8)
   └─> Show how to create real feature worktree
   └─> Reference additional docs (Section 10)
```

**Success Criteria:**

- ✅ `.worktree.yml` created and valid
- ✅ Test worktree created successfully
- ✅ Services start without errors
- ✅ `worktree doctor` passes all checks
- ✅ User understands basic commands

**Time Estimate:**

- Simple setup (backend + frontend): 3-5 conversation turns
- Complex setup (multiple services, DBs, queues): 5-10 conversation turns
- With troubleshooting: +2-5 turns per issue

---

**End of SKILL.md**

**Version:** 1.0.0
**Last Updated:** 2026-02-23
**Maintained By:** worktree-manager project
**Feedback:** https://github.com/braunmar/worktree/issues
