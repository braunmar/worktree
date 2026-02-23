# Example 1: Minimal Setup

The simplest possible worktree-manager setup. Get started in 5 minutes!

## Use Case

You're a solo developer (or just want to try the tool) and you have:
- A single backend API project
- No Docker (just `go run`, `npm start`, or `python app.py`)
- Want to run multiple feature branches simultaneously

## What You'll Learn

- How to create a basic `.worktree.yml` file
- How port allocation works
- How to run your first `worktree new-feature` command
- The core concepts without complexity

## Project Structure

```
myapi/                    # Project root
├── .worktree.yml         # This example config
├── backend/              # Your backend code
│   ├── .git/             # Git repository (must exist!)
│   └── main.go           # Or index.js, app.py, etc.
└── worktrees/            # Created by worktree-manager
    └── .registry.json    # Port allocation registry
```

**Key Point**: The `backend/` directory MUST be a git repository with at least one commit.

---

## Configuration Walkthrough

Let's break down the [.worktree.yml](.worktree.yml) file:

### Project Basics

```yaml
project_name: "myapi"
hostname: localhost
```

- `project_name`: Used in Docker container names (not critical for process executor)
- `hostname`: Where your services run (usually `localhost`)

### Projects Section

```yaml
projects:
  backend:
    executor: process  # No Docker needed!
    dir: backend
    main_branch: main
    start_command: "go run main.go"
```

**What this means**:
- `executor: process` - Runs your start_command as a background process (like running it in a terminal)
- `dir: backend` - Where the project lives (relative to project root)
- `main_branch: main` - The main branch for this repo
- `start_command` - What to run to start your service

**Supported commands**:
- Go: `go run main.go` or `go run cmd/api/main.go`
- Node.js: `npm start` or `node index.js`
- Python: `python app.py` or `python -m uvicorn main:app --reload`

### Presets Section

```yaml
presets:
  backend:
    projects: [backend]

default_preset: backend
```

A preset is just a group of projects. In this example, we only have one project, so the preset only contains that one project. Presets become more useful when you have multiple projects (see Example 2).

### Environment Variables (Ports)

```yaml
env_variables:
  APP_PORT:
    name: "Backend API"
    url: "http://{host}:{port}"
    port: "8080"
    env: "APP_PORT"
    range: [8080, 8180]
```

**This is the magic!**

When you run `worktree new-feature feature/my-feature`:
1. Worktree-manager allocates an available port from the range (8080-8180)
2. First instance gets 8080
3. Second instance gets 8081
4. And so on...
5. Exports `APP_PORT` as an environment variable before running `start_command`

**Your application should**:
- Read the `APP_PORT` env var to know which port to use
- Example in Go: `port := os.Getenv("APP_PORT")` (or default to 8080 if not set)
- Example in Node: `const port = process.env.APP_PORT || 8080`

---

## Try It Out

### Prerequisites

1. **Git repository with at least one commit**:
   ```bash
   cd backend
   git init
   git add .
   git commit -m "initial commit"
   ```

2. **Worktree-manager installed**:
   ```bash
   worktree --version
   ```

### Step 1: Copy the Configuration

```bash
# From your project root (where backend/ lives)
cp examples/01-minimal/.worktree.yml .
```

### Step 2: Create Your First Feature

```bash
worktree new-feature feature/my-first-test
```

**What happens**:
1. ✅ Allocates APP_PORT=8080 from registry
2. ✅ Creates git worktree at `worktrees/feature-my-first-test/backend/`
3. ✅ Creates branch `feature/my-first-test` in that worktree
4. ✅ Exports `APP_PORT=8080` to environment
5. ✅ Runs `go run main.go` (or your start_command) in the worktree directory
6. ✅ Your API is now running on port 8080!

### Step 3: Create a Second Feature

```bash
worktree new-feature feature/another-test
```

**What happens**:
1. ✅ Allocates APP_PORT=8081 (8080 is taken)
2. ✅ Creates git worktree at `worktrees/feature-another-test/backend/`
3. ✅ Creates branch `feature/another-test`
4. ✅ Exports `APP_PORT=8081`
5. ✅ Runs `go run main.go` in the new worktree
6. ✅ Second API running on port 8081!

**Now you have TWO feature branches running simultaneously with NO port conflicts!**

---

## Verification

### Check Running Instances

```bash
worktree list
```

You should see:
```
Feature                  Projects  Status   Ports
feature-my-first-test   backend   Running  8080
feature-another-test    backend   Running  8081
```

### Check Ports

```bash
worktree ports feature-my-first-test
```

Shows:
```
✓ Backend API  http://localhost:8080
```

### Test Your APIs

```bash
# Test first instance
curl http://localhost:8080

# Test second instance
curl http://localhost:8081
```

Both should respond! 🎉

---

## Managing Instances

### Stop an Instance

```bash
worktree stop feature-my-first-test
```

Sends SIGTERM to the process (graceful shutdown).

### Start a Stopped Instance

```bash
worktree start feature-my-first-test
```

Restarts the service on the same port (8080).

### Remove an Instance

```bash
worktree remove feature-my-first-test
```

**Warning**: This deletes the git worktree and any uncommitted changes!

### Check Instance Status

```bash
worktree status feature-my-first-test
```

Shows whether the process is running and PID information.

---

## Troubleshooting

### Problem: "Port 8080 already in use"

**Solution**: Something else is using port 8080.

```bash
# Find what's using the port
lsof -i :8080

# Or use worktree doctor
worktree doctor

# Stop the conflicting process or change the port range in .worktree.yml
```

### Problem: "directory is not a git repository"

**Solution**: The backend directory needs to be a git repo:

```bash
cd backend
git init
git add .
git commit -m "initial commit"
cd ..
worktree new-feature feature/test
```

### Problem: Start command fails

**Solution**: Check the logs in the worktree directory:

```bash
cd worktrees/feature-test/backend
# Check if there's an error log or output file
# Manually test the start command:
go run main.go  # Or your start_command
```

---

## What's Next?

This example showed the absolute basics. Now you can:

- **[Example 2: Fullstack Basic](../02-fullstack-basic/)** - Learn how to work with multiple projects (frontend + backend)
- **[Example 3: Fullstack Docker](../03-fullstack-docker/)** - Learn Docker integration (most common use case)
- **[Docker Port Patterns](../../docs/PORT-PATTERNS.md)** - Deep dive into Docker Compose patterns

## Key Takeaways

✅ Worktree-manager lets you run multiple feature branches simultaneously
✅ Each instance gets a unique port automatically
✅ Process executor is the simplest way to get started (no Docker needed)
✅ Configuration is just a simple YAML file
✅ Git worktrees keep your branches organized
