# Example 2: Fullstack Basic (No Docker)

Learn how to work with multiple projects using process executors. Perfect for dev servers like `npm start` and `go run`.

## Use Case

You have:
- A frontend (React, Vue, Angular, etc.)
- A backend API (Node.js, Go, Python, etc.)
- Both run with simple commands (`npm start`, `go run main.go`)
- No Docker (yet - see Example 3 for Docker)

You want to:
- Run both frontend and backend for each feature
- Or sometimes just backend (API testing)
- Or sometimes just frontend (mock API)

## What You'll Learn

- How to configure multiple projects
- How presets let you run different combinations
- How env vars reference each other (frontend needs to know backend URL!)
- When to use `--preset backend` vs `--preset fullstack`

## Project Structure

See [project-structure.txt](project-structure.txt) for detailed layout.

**Key points**:
- Both `backend/` and `frontend/` are separate git repositories
- Each has its own `.git` directory
- Worktrees are created for BOTH projects simultaneously
- Both run as background processes (PID files track them)

---

## Configuration Walkthrough

### Projects Section

```yaml
projects:
  backend:
    executor: process
    dir: backend
    main_branch: main
    start_command: "go run main.go"
    claude_working_dir: true  # Claude starts here

  frontend:
    executor: process
    dir: frontend
    main_branch: main
    start_command: "npm start"
```

**Two projects!** Each with:
- `executor: process` - Runs as background process (no Docker)
- `start_command` - How to start the service

**`claude_working_dir: true`** means if you're using Claude Code, it'll navigate to the backend directory after creating the worktree.

### Presets Section (This is Important!)

```yaml
presets:
  fullstack:
    projects: [backend, frontend]
    description: "Full stack (backend + frontend)"

  backend:
    projects: [backend]
    description: "Backend API only"

  frontend:
    projects: [frontend]
    description: "Frontend only"

default_preset: fullstack
```

**Presets let you choose what to run!**

| Preset | What Runs | When to Use |
|--------|-----------|-------------|
| `fullstack` | Backend + Frontend | Working on features that need both |
| `backend` | Backend only | API development, testing endpoints |
| `frontend` | Frontend only | UI work with mocked API |

**Usage**:
```bash
# Run everything (default)
worktree new-feature feature/user-profile

# Run backend only
worktree new-feature feature/new-endpoint --preset backend

# Run frontend only
worktree new-feature feature/ui-tweaks --preset frontend
```

### Environment Variables (The Magic Part)

```yaml
env_variables:
  APP_PORT:
    name: "Backend API"
    url: "http://{host}:{port}"
    port: "8080"
    env: "APP_PORT"
    range: [8080, 8180]

  FE_PORT:
    name: "Frontend"
    url: "http://{host}:{port}"
    port: "3000"
    env: "FE_PORT"
    range: [3000, 3100]

  REACT_APP_API_BASE_URL:
    name: "API Base URL (for frontend)"
    url: "{value}"
    value: "http://{host}:{APP_PORT}"
    env: "REACT_APP_API_BASE_URL"
```

**How this works**:

1. **First pass**: Allocate ports
   - `APP_PORT=8080` for instance 0
   - `FE_PORT=3000` for instance 0

2. **Second pass**: Calculate string templates
   - `REACT_APP_API_BASE_URL` = `"http://localhost:8080"` (uses `{APP_PORT}` value)

3. **Export all env vars** before running start commands
   - Backend gets: `APP_PORT=8080`
   - Frontend gets: `FE_PORT=3000`, `REACT_APP_API_BASE_URL=http://localhost:8080`

**Your application code**:

Backend (Go):
```go
port := os.Getenv("APP_PORT") // "8080"
if port == "" {
    port = "8080" // Fallback
}
```

Frontend (React):
```javascript
const API_BASE_URL = process.env.REACT_APP_API_BASE_URL || 'http://localhost:8080';
// Uses the dynamically allocated port!
```

---

## Try It Out

### Prerequisites

1. **Two git repositories with at least one commit each**:
   ```bash
   cd backend && git init && git add . && git commit -m "init" && cd ..
   cd frontend && git init && git add . && git commit -m "init" && cd ..
   ```

2. **Your backend should read APP_PORT env var**:
   ```go
   // Go example
   port := os.Getenv("APP_PORT")
   ```

3. **Your frontend should read REACT_APP_API_BASE_URL env var**:
   ```javascript
   // React example
   const API_URL = process.env.REACT_APP_API_BASE_URL;
   ```

### Step 1: Copy Configuration

```bash
cp examples/02-fullstack-basic/.worktree.yml .
```

### Step 2: Create Fullstack Instance

```bash
worktree new-feature feature/user-profile
```

**What happens**:
1. ✅ Allocates APP_PORT=8080, FE_PORT=3000
2. ✅ Calculates REACT_APP_API_BASE_URL=http://localhost:8080
3. ✅ Creates backend worktree at `worktrees/feature-user-profile/backend/`
4. ✅ Creates frontend worktree at `worktrees/feature-user-profile/frontend/`
5. ✅ Exports env vars
6. ✅ Runs `go run main.go` in backend (on port 8080)
7. ✅ Runs `npm start` in frontend (on port 3000)
8. ✅ Frontend knows to call API at http://localhost:8080

### Step 3: Create Backend-Only Instance

```bash
worktree new-feature feature/new-endpoint --preset backend
```

**What happens**:
1. ✅ Allocates APP_PORT=8081 (8080 is taken)
2. ✅ Creates backend worktree only (no frontend!)
3. ✅ Runs `go run main.go` on port 8081
4. ✅ You can test your API with curl/Postman at http://localhost:8081

**Why is this useful?**
- Backend developers don't need to run the frontend
- Saves resources (no npm process)
- Faster startup
- Clearer mental model

---

## Verification

### Check What's Running

```bash
worktree list
```

Shows:
```
Feature                 Preset      Projects            Status   Ports
feature-user-profile   fullstack   backend, frontend   Running  8080, 3000
feature-new-endpoint   backend     backend             Running  8081
```

### Check Ports for Each Instance

```bash
worktree ports feature-user-profile
```

Shows:
```
✓ Backend API           http://localhost:8080
✓ Frontend              http://localhost:3000
✓ API Base URL          http://localhost:8080
```

```bash
worktree ports feature-new-endpoint
```

Shows:
```
✓ Backend API           http://localhost:8081
```

### Test the Services

```bash
# Test first backend (from fullstack instance)
curl http://localhost:8080/api/health

# Test second backend (backend-only instance)
curl http://localhost:8081/api/health

# Open first frontend in browser
open http://localhost:3000
```

---

## Common Patterns

### Pattern 1: Full-Stack Feature Development

```bash
worktree new-feature feature/user-authentication
# Runs: Backend (8080) + Frontend (3000)
```

**Use when**: Building features that need both UI and API

### Pattern 2: Backend API Development

```bash
worktree new-feature feature/payment-endpoint --preset backend
# Runs: Backend only (8081)
```

**Use when**:
- Building new API endpoints
- Writing integration tests
- Backend-only work

**Test with**:
```bash
curl -X POST http://localhost:8081/api/payments \
  -H "Content-Type: application/json" \
  -d '{"amount": 100}'
```

### Pattern 3: Frontend UI Work

```bash
worktree new-feature feature/redesign-navbar --preset frontend
# Runs: Frontend only (3001)
```

**Use when**:
- UI tweaks, styling
- Component development
- Frontend has mock API or can work standalone

---

## Troubleshooting

### Problem: Frontend can't reach backend

**Symptom**: Console errors like "Failed to fetch http://localhost:undefined"

**Cause**: Frontend isn't reading REACT_APP_API_BASE_URL

**Solution**: Make sure your frontend code reads the env var:

```javascript
// In your API client
const API_BASE_URL = process.env.REACT_APP_API_BASE_URL || 'http://localhost:8080';

fetch(`${API_BASE_URL}/api/users`)
```

**Note**: In React (Create React App), env vars MUST start with `REACT_APP_` prefix!

### Problem: "Port 3000 already in use"

**Cause**: Another frontend instance (or different app) is using port 3000

**Solution**: Check what's running:

```bash
lsof -i :3000
# Or
worktree list
```

Stop conflicting instance:
```bash
worktree stop feature-old-feature
```

### Problem: Backend starts but immediately exits

**Cause**: Backend might not be reading APP_PORT correctly

**Debug**:
```bash
cd worktrees/feature-test/backend
# Check the PID file
cat backend.pid
# Check if process is actually running
ps aux | grep <PID>
# Manually test start command
go run main.go
```

**Fix**: Make sure backend reads APP_PORT:
```go
port := os.Getenv("APP_PORT")
if port == "" {
    port = "8080"
}
log.Printf("Starting server on port %s", port)
http.ListenAndServe(":"+port, router)
```

---

## Key Concepts Recap

### 1. Multiple Projects
You can have as many projects as you want. Each gets its own worktree.

### 2. Presets Are Powerful
Presets let you run exactly what you need:
- Full stack (both)
- Backend only
- Frontend only
- Custom combinations

### 3. Env Vars Connect Services
- Backend gets `APP_PORT`
- Frontend gets `FE_PORT` and `REACT_APP_API_BASE_URL`
- String templates let vars reference each other

### 4. Each Instance Is Isolated
- Instance 0: Backend on 8080, Frontend on 3000
- Instance 1: Backend on 8081, Frontend on 3001
- No conflicts!

---

## What's Next?

Ready for Docker? Most teams use Docker for local development.

**[Example 3: Fullstack Docker](../03-fullstack-docker/)** - Learn how to integrate worktree-manager with Docker Compose, including the `${APP_PORT:-8080}` pattern that prevents port conflicts.

Or explore advanced patterns:

- **[Docker Port Patterns](../../docs/PORT-PATTERNS.md)** - Deep dive into Docker Compose integration
- **[Example 4: Polyglot Services](../04-polyglot-services/)** - Multiple services with different languages

## Key Takeaways

✅ Presets let you run different project combinations
✅ String template env vars let services find each other
✅ Process executor is great for dev servers (no Docker needed)
✅ Each instance gets isolated ports automatically
✅ You can run backend-only or frontend-only instances to save resources
