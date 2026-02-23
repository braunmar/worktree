# Migration Guide: From Hardcoded Ports to Worktree-Manager

So you've got a project with hardcoded ports in docker-compose.yml and you want to run multiple instances? Here's your step-by-step guide to migrate to worktree-manager.

## What You're Starting With

Typical setup before worktree-manager:
- `backend/docker-compose.yml` with `ports: "8080:8080"`
- `frontend/docker-compose.yml` with `ports: "3000:3000"`
- Can only run ONE instance at a time
- Port conflicts when trying to run multiple features
- Manual worktree management
- Environment variable chaos

**The pain**:
```bash
# Working on feature A
cd backend && docker compose up  # Binds to port 8080

# Need to work on feature B urgently
cd ../feature-b/backend && docker compose up
# ❌ ERROR: Port 8080 already in use!

# You have to stop feature A first
cd ../../feature-a/backend && docker compose down
# Now feature A is stopped - can't compare behavior!
```

## What You'll End Up With

After migration to worktree-manager:
- `backend/docker-compose.yml` with `ports: "${APP_PORT:-8080}:8080"`
- `frontend/docker-compose.yml` with `ports: "${FE_PORT:-3000}:3000"`
- `.worktree.yml` at project root
- Can run UNLIMITED instances simultaneously
- Each instance gets unique ports automatically
- Simple commands: `worktree new-feature`, `worktree start`, `worktree stop`

**The joy**:
```bash
# Working on feature A
worktree new-feature feature/user-auth
# ✅ Running on ports 8080, 3000

# Need to work on feature B urgently
worktree new-feature feature/urgent-fix
# ✅ Running on ports 8081, 3001

# Both running! Compare behavior side by side!
curl http://localhost:8080/api/users  # Feature A
curl http://localhost:8081/api/users  # Feature B
open http://localhost:3000  # Feature A UI
open http://localhost:3001  # Feature B UI
```

---

## Migration Steps

### Step 1: Audit Current Port Usage

**Find all hardcoded ports**:

```bash
cd /path/to/your/project
grep -r ":[0-9]\{4\}:" . --include="docker-compose*.yml"
```

**Example output**:
```
./backend/docker-compose.yml:    - "8080:8080"
./backend/docker-compose.yml:    - "5432:5432"
./backend/docker-compose.yml:    - "6379:6379"
./backend/docker-compose.yml:    - "8025:8025"
./frontend/docker-compose.yml:    - "3000:3000"
```

**Make a list**:
- Backend API: 8080
- Frontend: 3000
- PostgreSQL: 5432
- Redis: 6379
- Mailpit UI: 8025
- Mailpit SMTP: 1025

### Step 2: Update docker-compose.yml Files

For each hardcoded port, replace with the `${VAR:-default}` pattern.

#### Backend: docker-compose.yml

**Before** (hardcoded ports):

```yaml
version: '3.8'

services:
  api:
    build: .
    ports:
      - "8080:8080"  # ❌ HARDCODED
    environment:
      - PORT=8080
      - DATABASE_URL=postgresql://postgres:postgres@postgres:5432/app

  postgres:
    image: postgres:15
    ports:
      - "5432:5432"  # ❌ HARDCODED

  redis:
    image: redis:7
    ports:
      - "6379:6379"  # ❌ HARDCODED

  mailpit:
    image: axllent/mailpit:latest
    ports:
      - "8025:8025"  # ❌ HARDCODED
      - "1025:1025"  # ❌ HARDCODED
```

**After** (dynamic ports with fallback):

```yaml
version: '3.8'

services:
  api:
    build: .
    ports:
      - "${APP_PORT:-8080}:8080"  # ✅ DYNAMIC
    environment:
      - PORT=8080  # Internal port (always 8080 in container)
      - APP_PORT=${APP_PORT}  # External port (for reference)
      - DATABASE_URL=postgresql://postgres:postgres@postgres:5432/app  # Internal port!
      - REDIS_URL=redis://redis:6379  # Internal port!
      - CORS_ALLOWED_ORIGINS=http://localhost:${FE_PORT}  # External port (for browser)

  postgres:
    image: postgres:15
    ports:
      - "${POSTGRES_PORT:-5432}:5432"  # ✅ DYNAMIC

  redis:
    image: redis:7
    ports:
      - "${REDIS_PORT:-6379}:6379"  # ✅ DYNAMIC

  mailpit:
    image: axllent/mailpit:latest
    ports:
      - "${MAILPIT_PORT:-8025}:8025"  # ✅ DYNAMIC
      - "${MAILPIT_SMTP_PORT:-1025}:1025"  # ✅ DYNAMIC
```

**Key changes**:
1. Replace all `"PORT:PORT"` with `"${VAR:-PORT}:PORT"`
2. Add env vars to `api` service (`APP_PORT`, `DATABASE_URL`, `REDIS_URL`, `CORS_ALLOWED_ORIGINS`)
3. Use EXTERNAL port (`${FE_PORT}`) for CORS (browser makes the call)
4. Use INTERNAL port (`:5432`, `:6379`) for service-to-service communication

#### Frontend: docker-compose.yml

**Before**:

```yaml
version: '3.8'

services:
  app:
    build: .
    ports:
      - "3000:3000"  # ❌ HARDCODED
    environment:
      - REACT_APP_API_BASE_URL=http://localhost:8080  # ❌ HARDCODED
```

**After**:

```yaml
version: '3.8'

services:
  app:
    build: .
    ports:
      - "${FE_PORT:-3000}:3000"  # ✅ DYNAMIC
    environment:
      - REACT_APP_API_BASE_URL=${REACT_APP_API_BASE_URL}  # ✅ DYNAMIC
      - WDS_SOCKET_PORT=${FE_PORT}  # For Webpack hot reload
      - VITE_API_BASE_URL=${REACT_APP_API_BASE_URL}  # For Vite (if applicable)
```

### Step 3: Create .worktree.yml

**Copy from example**:

```bash
cp examples/03-fullstack-docker/.worktree.yml .
```

**Or create from scratch**:

```yaml
project_name: "myproject"
hostname: localhost

projects:
  backend:
    executor: docker
    dir: backend
    main_branch: main
    start_command: "docker compose up -d"
    start_post_command: "docker compose exec -T api make migrate"  # Optional
    claude_working_dir: true

  frontend:
    executor: docker
    dir: frontend
    main_branch: main
    start_command: "docker compose up -d"

presets:
  fullstack:
    projects: [backend, frontend]
    description: "Full application (backend + frontend)"

  backend:
    projects: [backend]
    description: "Backend only"

  frontend:
    projects: [frontend]
    description: "Frontend only"

default_preset: fullstack

env_variables:
  APP_PORT:
    name: "Backend API"
    url: "http://{host}:{port}"
    port: "8080"
    env: "APP_PORT"
    range: [8080, 8180]  # Supports 100 instances

  FE_PORT:
    name: "Frontend"
    url: "http://{host}:{port}"
    port: "3000"
    env: "FE_PORT"
    range: [3000, 3100]

  POSTGRES_PORT:
    name: "PostgreSQL"
    url: "{host}:{port}"
    port: "5432"
    env: "POSTGRES_PORT"
    range: [5432, 5532]

  REDIS_PORT:
    name: "Redis"
    url: "{host}:{port}"
    port: "6379"
    env: "REDIS_PORT"
    range: [6379, 6479]

  MAILPIT_PORT:
    name: "Mailpit UI"
    url: "http://{host}:{port}"
    port: "8025"
    env: "MAILPIT_PORT"
    range: [8025, 8125]

  MAILPIT_SMTP_PORT:
    name: "Mailpit SMTP"
    url: "{host}:{port}"
    port: "1025"
    env: "MAILPIT_SMTP_PORT"
    range: [1025, 1125]

  # String template that references APP_PORT
  REACT_APP_API_BASE_URL:
    name: "API Base URL (for frontend)"
    url: "{value}"
    value: "http://{host}:{APP_PORT}"
    env: "REACT_APP_API_BASE_URL"
```

**Add env_variables for each port** you found in Step 1!

### Step 4: Test Single Instance

**Create test instance**:

```bash
worktree new-feature feature/test-migration
```

**What should happen**:
```
✓ Allocating ports: APP_PORT=8080, FE_PORT=3000, POSTGRES_PORT=5432, ...
✓ Creating backend worktree at worktrees/feature-test-migration/backend/
✓ Creating frontend worktree at worktrees/feature-test-migration/frontend/
✓ Exporting environment variables
✓ Starting backend services (docker compose up -d)
✓ Running migrations (docker compose exec -T api make migrate)
✓ Starting frontend services (docker compose up -d)
✓ Feature environment ready!
```

**Check allocated ports**:

```bash
worktree ports feature-test-migration
```

**Output**:
```
✓ Backend API           http://localhost:8080
✓ Frontend              http://localhost:3000
✓ PostgreSQL            localhost:5432
✓ Redis                 localhost:6379
✓ Mailpit UI            http://localhost:8025
✓ Mailpit SMTP          localhost:1025
✓ API Base URL          http://localhost:8080
```

**Verify services are running**:

```bash
worktree status feature-test-migration

# Or check Docker containers directly
cd worktrees/feature-test-migration/backend
docker compose ps
```

**Test the services**:

```bash
# Test backend API
curl http://localhost:8080/api/health
# Should return 200 OK

# Test database connection
psql -h localhost -p 5432 -U postgres -d app
# Should connect successfully

# Open frontend in browser
open http://localhost:3000
# Should load and make API calls to localhost:8080
```

**If something fails**: See [Troubleshooting](#troubleshooting) section below.

### Step 5: Test Multiple Instances

**Create second instance**:

```bash
worktree new-feature feature/second-test
```

**What should happen**:
```
✓ Allocating ports: APP_PORT=8081, FE_PORT=3001, POSTGRES_PORT=5433, ...
✓ Creating worktrees...
✓ Starting services...
✓ Feature environment ready!
```

**Different ports!** 🎉

**Check both instances**:

```bash
worktree list
```

**Output**:
```
Feature                  Preset      Projects            Status   Ports
feature-test-migration  fullstack   backend, frontend   Running  8080, 3000
feature-second-test     fullstack   backend, frontend   Running  8081, 3001
```

**Compare ports**:

```bash
worktree ports feature-test-migration
# Shows: 8080, 3000, 5432, 6379, ...

worktree ports feature-second-test
# Shows: 8081, 3001, 5433, 6380, ...
```

**Test both simultaneously**:

```bash
# Test first backend
curl http://localhost:8080/api/users
# Returns users from first instance

# Test second backend
curl http://localhost:8081/api/users
# Returns users from second instance (different database!)

# Open both frontends
open http://localhost:3000  # First instance
open http://localhost:3001  # Second instance
```

**Both work!** No conflicts! 🚀

### Step 6: Clean Up Test Instances

```bash
worktree remove feature-test-migration
worktree remove feature-second-test
```

**What happens**:
- Stops Docker containers
- Removes worktrees
- Frees allocated ports

### Step 7: Update Team Documentation

Add to your project README:

```markdown
## Development with Worktree-Manager

### Creating a Feature Environment

Create an isolated environment for your feature:

\`\`\`bash
worktree new-feature feature/my-feature
\`\`\`

This creates worktrees, allocates ports, and starts all services.

### Checking Your Ports

\`\`\`bash
worktree ports feature/my-feature
\`\`\`

### Managing Instances

\`\`\`bash
worktree list                      # List all instances
worktree start feature/my-feature  # Start a stopped instance
worktree stop feature/my-feature   # Stop a running instance
worktree remove feature/my-feature # Remove instance completely
\`\`\`

### Running Multiple Features

You can run unlimited instances simultaneously! Each gets unique ports:

\`\`\`bash
worktree new-feature feature/user-auth    # Instance 1: 8080, 3000
worktree new-feature feature/payments     # Instance 2: 8081, 3001
worktree new-feature feature/analytics    # Instance 3: 8082, 3002
\`\`\`

All running at the same time - no conflicts!

### Auto-Detection

From any directory within a worktree, commands auto-detect the instance:

\`\`\`bash
cd worktrees/feature-user-auth/backend
worktree ports          # Auto-detects: feature-user-auth
worktree stop           # Auto-detects: feature-user-auth
\`\`\`
```

---

## Common Migration Issues

### Issue 1: Service Can't Connect to Database

**Symptom**: Backend logs show "connection refused" or "could not connect to server"

**Cause**: Using external port instead of internal port in DATABASE_URL

**Wrong**:
```yaml
services:
  api:
    environment:
      - DATABASE_URL=postgresql://postgres:postgres@postgres:${POSTGRES_PORT}/app
```

**Correct**:
```yaml
services:
  api:
    environment:
      - DATABASE_URL=postgresql://postgres:postgres@postgres:5432/app  # Use 5432, not ${POSTGRES_PORT}
```

**Why**: Inside Docker network, containers communicate using internal ports. `postgres:5432` always works, regardless of external port mapping.

### Issue 2: Frontend Can't Reach Backend

**Symptom**: Console errors like "Failed to fetch" or "Network error"

**Cause 1**: Frontend not reading REACT_APP_API_BASE_URL env var

**Fix**:
```javascript
// ❌ Wrong - hardcoded
const API_URL = 'http://localhost:8080';

// ✅ Correct - read from env
const API_URL = process.env.REACT_APP_API_BASE_URL || 'http://localhost:8080';
```

**Cause 2**: Env var not passed to frontend container

**Fix** (in frontend/docker-compose.yml):
```yaml
services:
  app:
    environment:
      - REACT_APP_API_BASE_URL=${REACT_APP_API_BASE_URL}  # Must pass env var!
```

**Cause 3**: CORS not configured for dynamic frontend port

**Fix** (in backend/docker-compose.yml):
```yaml
services:
  api:
    environment:
      - CORS_ALLOWED_ORIGINS=http://localhost:${FE_PORT}  # Use ${FE_PORT}, not hardcoded 3000
```

### Issue 3: Port Still Showing as "In Use"

**Symptom**: `worktree new-feature` fails with "address already in use"

**Cause**: Another process (or orphaned Docker container) is using the port

**Solution 1**: Check what's using the port:
```bash
lsof -i :8080
```

**Solution 2**: Run health check:
```bash
worktree doctor
```

**Solution 3**: Check other worktree instances:
```bash
worktree list
```

**Solution 4**: Stop old containers:
```bash
cd backend  # Your main project directory
docker compose down
```

### Issue 4: Hot Reload Not Working (Frontend)

**Symptom**: Code changes don't reflect in browser

**Cause**: Webpack Dev Server not configured for dynamic port

**Fix** (in frontend/docker-compose.yml):
```yaml
services:
  app:
    environment:
      - WDS_SOCKET_PORT=${FE_PORT}  # Tell Webpack to use dynamic port
```

For Vite:
```yaml
services:
  app:
    environment:
      - VITE_HMR_PORT=${FE_PORT}  # Vite uses different env var
```

### Issue 5: Migrations Fail in start_post_command

**Symptom**: "No such container" or "Container not running"

**Cause**: Container not fully started yet

**Fix**: Add health check (in backend/docker-compose.yml):
```yaml
services:
  api:
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 5s
      timeout: 5s
      retries: 5
      start_period: 10s

  postgres:
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 5s
      retries: 5
```

Or add a delay:
```yaml
# In .worktree.yml
projects:
  backend:
    start_post_command: "sleep 5 && docker compose exec -T api make migrate"
```

### Issue 6: Generated .env Files Not Working

**Symptom**: .env.worktree files not created or empty

**Cause**: `generated_files` section missing in .worktree.yml

**Fix**: Add to .worktree.yml:
```yaml
generated_files:
  - path: "{project}/.env.worktree"
    content: |
      APP_PORT={APP_PORT}
      FE_PORT={FE_PORT}
      POSTGRES_PORT={POSTGRES_PORT}
      REDIS_PORT={REDIS_PORT}
      MAILPIT_PORT={MAILPIT_PORT}
      REACT_APP_API_BASE_URL={REACT_APP_API_BASE_URL}
```

Then update docker-compose.yml to use it:
```yaml
services:
  api:
    env_file:
      - .env.worktree  # Load from generated file
```

---

## Migration Checklist

Use this checklist to track your migration progress:

- [ ] Step 1: Audited all hardcoded ports (created list)
- [ ] Step 2: Updated backend/docker-compose.yml with `${VAR:-default}` pattern
- [ ] Step 2: Updated frontend/docker-compose.yml with `${VAR:-default}` pattern
- [ ] Step 2: Updated inter-service communication (use internal ports)
- [ ] Step 2: Updated CORS configuration (use `${FE_PORT}`)
- [ ] Step 3: Created `.worktree.yml` with all env_variables
- [ ] Step 3: Added presets (fullstack, backend, frontend)
- [ ] Step 3: Configured generated_files (if needed)
- [ ] Step 3: Configured symlinks (if needed)
- [ ] Step 4: Tested single instance creation
- [ ] Step 4: Verified all services start successfully
- [ ] Step 4: Tested service-to-service communication
- [ ] Step 4: Tested frontend can reach backend
- [ ] Step 5: Tested multiple instances simultaneously
- [ ] Step 5: Verified no port conflicts
- [ ] Step 5: Tested that both instances work independently
- [ ] Step 6: Cleaned up test instances
- [ ] Step 7: Updated team documentation (README)
- [ ] Step 7: Shared migration guide with team
- [ ] Step 7: Conducted team training session (optional)

**All checked?** Congratulations! Your migration is complete! 🎉

---

## Next Steps

Now that you've migrated, explore advanced features:

### 1. Explore Different Presets

```bash
# Backend only (for API development)
worktree new-feature feature/api-endpoint --preset backend

# Frontend only (for UI work)
worktree new-feature feature/ui-redesign --preset frontend
```

**See**: [Example 4: Polyglot Services](04-polyglot-services/COMBINATIONS.md) for preset strategies.

### 2. Set Up Scheduled Agents

Automate housekeeping tasks (npm audit, dependency updates, etc.):

**See**: [Example 5: Real-World](05-real-world/) for scheduled agents configuration.

### 3. Add Lifecycle Hooks

Run custom commands before/after start/stop/restart:

```yaml
projects:
  backend:
    start_pre_command: "make check-deps"
    start_post_command: "make migrate && make seed"
    stop_pre_command: "make drain-connections"
    stop_post_command: "make cleanup-temp-files"
```

**See**: [Example 5: Real-World](05-real-world/) for production examples.

### 4. Deep Dive into Docker Patterns

**See**: [docs/PORT-PATTERNS.md](../docs/PORT-PATTERNS.md) for advanced patterns, troubleshooting, and best practices.

---

## Additional Resources

- **[Example 3: Fullstack Docker](03-fullstack-docker/)** - Complete Docker integration guide
- **[Docker Port Patterns](../docs/PORT-PATTERNS.md)** - Deep dive into Docker Compose patterns
- **[Example 4: Polyglot Services](04-polyglot-services/)** - Multiple services with different languages
- **[Example 5: Real-World](05-real-world/)** - Production configuration reference
- **[Complete Config Reference](../.worktree.example.yml)** - All options documented

---

## Key Takeaways

✅ **Replace hardcoded ports** with `${VAR:-default}` pattern in docker-compose.yml
✅ **Use internal ports** for service-to-service communication (e.g., `postgres:5432`)
✅ **Use external ports** for browser-to-service communication (e.g., `localhost:${APP_PORT}`)
✅ **Test incrementally** - single instance first, then multiple instances
✅ **Document for your team** - update README with worktree-manager usage
✅ **Start simple** - add advanced features (agents, hooks) later
✅ **Use health checks** to avoid race conditions in start_post_command

**Migration complete! Enjoy running unlimited parallel instances!** 🚀
