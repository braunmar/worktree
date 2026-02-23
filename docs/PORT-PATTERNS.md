# Docker Compose Port Patterns with Worktree-Manager

A comprehensive guide to integrating worktree-manager with Docker Compose. Learn the patterns, avoid the pitfalls, and run unlimited instances without port conflicts.

## Table of Contents

1. [The Problem](#the-problem)
2. [The Solution](#the-solution)
3. [Common Patterns](#common-patterns)
4. [Configuring .worktree.yml](#configuring-worktreeyml)
5. [Inter-Service Communication](#inter-service-communication)
6. [Migration Checklist](#migration-checklist)
7. [Troubleshooting](#troubleshooting)
8. [Advanced Patterns](#advanced-patterns)

---

## The Problem

Hard coded ports cause conflicts when running multiple instances.

**Example** (won't work for multiple instances):

```yaml
# docker-compose.yml
services:
  api:
    ports:
      - "8080:8080"  # ❌ HARDCODED

  postgres:
    ports:
      - "5432:5432"  # ❌ HARDCODED
```

**What happens**:

```bash
# First instance
worktree new-feature feature/user-auth
# ✅ Works - binds to ports 8080, 5432

# Second instance
worktree new-feature feature/payments
# ❌ ERROR! Ports 8080 and 5432 already in use
# Error: bind: address already in use
```

**You can only run ONE instance at a time.** 😞

---

## The Solution

Use environment variables with fallback defaults in your docker-compose.yml files.

**Pattern**: `"${ENV_VAR:-default}"`

```yaml
# docker-compose.yml
services:
  api:
    ports:
      - "${APP_PORT:-8080}:8080"  # ✅ DYNAMIC with fallback
    environment:
      - APP_PORT=${APP_PORT}

  postgres:
    ports:
      - "${POSTGRES_PORT:-5432}:5432"  # ✅ DYNAMIC with fallback
```

**How it works**:

1. **Worktree-manager allocates unique ports**:
   - Instance 0: `APP_PORT=8080`, `POSTGRES_PORT=5432`
   - Instance 1: `APP_PORT=8081`, `POSTGRES_PORT=5433`
   - Instance 2: `APP_PORT=8082`, `POSTGRES_PORT=5434`

2. **Exports env vars before running `docker compose up`**:
   ```bash
   export APP_PORT=8080
   export POSTGRES_PORT=5432
   docker compose up -d
   ```

3. **Docker Compose reads env vars from environment**:
   - `"${APP_PORT:-8080}"` → If `APP_PORT=8081`, becomes `"8081:8080"`
   - `"${POSTGRES_PORT:-5432}"` → If `POSTGRES_PORT=5433`, becomes `"5433:5432"`

4. **Falls back to default if env var not set**:
   - Useful for manual `docker compose up` without worktree-manager
   - `"${APP_PORT:-8080}"` → If `APP_PORT` not set, becomes `"8080:8080"`

**Now you can run UNLIMITED instances!** 🎉

---

## Common Patterns

### Pattern 1: Web Server (Backend API)

**Use case**: Backend API (Node.js, Go, Python, etc.)

```yaml
services:
  api:
    build: .
    ports:
      - "${APP_PORT:-8080}:8080"
    environment:
      - PORT=8080                                    # Internal port (fixed)
      - DATABASE_URL=postgresql://postgres:postgres@postgres:5432/app
      - REDIS_URL=redis://redis:6379
      - CORS_ALLOWED_ORIGINS=http://localhost:${FE_PORT}
    depends_on:
      - postgres
```

**Key points**:
- External port (`${APP_PORT}`) is dynamic
- Internal port (`:8080`) is fixed
- Application inside container always listens on port 8080
- `CORS_ALLOWED_ORIGINS` references `${FE_PORT}` to allow frontend to connect

**Application code** (example in Node.js):

```javascript
// ✅ Correct - always listen on internal port (8080)
const PORT = 8080;  // Fixed, matches docker-compose.yml
app.listen(PORT, () => {
  console.log(`Server running on port ${PORT}`);
});
```

**Don't do this**:
```javascript
// ❌ Wrong - don't read APP_PORT inside container
const PORT = process.env.APP_PORT || 8080;
```

**Why**: Inside container, port is always 8080. `APP_PORT` is for HOST port mapping only.

### Pattern 2: Frontend (React/Vue/Angular)

**Use case**: Frontend dev server with hot reload

```yaml
services:
  app:
    build: .
    ports:
      - "${FE_PORT:-3000}:3000"
    environment:
      - REACT_APP_API_BASE_URL=${REACT_APP_API_BASE_URL}  # Where to call API
      - WDS_SOCKET_PORT=${FE_PORT}                         # For Webpack hot reload
      - VITE_API_BASE_URL=${REACT_APP_API_BASE_URL}      # For Vite projects
    volumes:
      - ./src:/app/src                                     # Hot reload
      - /app/node_modules                                  # Don't overwrite
```

**Key points**:
- `REACT_APP_API_BASE_URL` tells frontend where backend is (uses `{APP_PORT}`)
- `WDS_SOCKET_PORT` tells Webpack Dev Server to use dynamic port for hot reload
- Volumes enable hot reload (code changes reflect immediately)

**Frontend code** (React):

```javascript
// ✅ Correct - read from env var
const API_BASE_URL = process.env.REACT_APP_API_BASE_URL || 'http://localhost:8080';

// Make API calls
fetch(`${API_BASE_URL}/api/users`)
  .then(res => res.json())
  .then(data => console.log(data));
```

**For Vite**:
```javascript
// Vite uses import.meta.env instead of process.env
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';
```

### Pattern 3: Database (PostgreSQL)

**Use case**: PostgreSQL database

```yaml
services:
  postgres:
    image: postgres:15-alpine
    ports:
      - "${POSTGRES_PORT:-5432}:5432"
    environment:
      - POSTGRES_PASSWORD=postgres
      - POSTGRES_DB=app
      - POSTGRES_USER=postgres
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  postgres_data:
```

**Key points**:
- External port (`${POSTGRES_PORT}`) is dynamic (5432, 5433, 5434, ...)
- Internal port (`:5432`) is always 5432
- Health check ensures database is ready before dependent services start
- Volume persists data across container restarts

**Connection string** (from API service):

```yaml
services:
  api:
    environment:
      # ✅ Correct - use service name and internal port
      - DATABASE_URL=postgresql://postgres:postgres@postgres:5432/app
```

**Don't do this**:
```yaml
services:
  api:
    environment:
      # ❌ Wrong - don't use ${POSTGRES_PORT} for inter-container communication
      - DATABASE_URL=postgresql://postgres:postgres@postgres:${POSTGRES_PORT}/app
```

**Why**: Inside Docker network, containers communicate using service names and internal ports.

### Pattern 4: Multiple Databases

**Use case**: PostgreSQL + Redis + MongoDB

```yaml
services:
  postgres:
    image: postgres:15-alpine
    ports:
      - "${POSTGRES_PORT:-5432}:5432"
    environment:
      - POSTGRES_PASSWORD=postgres

  redis:
    image: redis:7-alpine
    ports:
      - "${REDIS_PORT:-6379}:6379"

  mongodb:
    image: mongo:7
    ports:
      - "${MONGO_PORT:-27017}:27017"
    environment:
      - MONGO_INITDB_ROOT_USERNAME=admin
      - MONGO_INITDB_ROOT_PASSWORD=password

volumes:
  postgres_data:
  mongo_data:
```

**Configuration in .worktree.yml**:

```yaml
env_variables:
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

  MONGO_PORT:
    name: "MongoDB"
    url: "{host}:{port}"
    port: "27017"
    env: "MONGO_PORT"
    range: [27017, 27117]
```

**Connection strings** (from API):

```yaml
services:
  api:
    environment:
      - DATABASE_URL=postgresql://postgres:postgres@postgres:5432/app
      - REDIS_URL=redis://redis:6379
      - MONGO_URL=mongodb://admin:password@mongodb:27017/app
```

**All internal ports (5432, 6379, 27017)!**

### Pattern 5: Background Worker

**Use case**: Python worker for async tasks (Celery, RQ, custom)

```yaml
services:
  worker:
    build: .
    ports:
      - "${WORKER_PORT:-8081}:8081"  # Health check endpoint
    environment:
      - WORKER_PORT=8081
      - REDIS_URL=redis://redis:6379
      - DATABASE_URL=postgresql://postgres:postgres@postgres:5432/app
    depends_on:
      - redis
      - postgres
    command: python worker.py
```

**Key points**:
- Worker exposes health check endpoint on `${WORKER_PORT}` (optional)
- Connects to Redis for job queue
- Connects to PostgreSQL for data storage

### Pattern 6: Email Testing (Mailpit)

**Use case**: Local email testing without sending real emails

```yaml
services:
  mailpit:
    image: axllent/mailpit:latest
    ports:
      - "${MAILPIT_PORT:-8025}:8025"       # Web UI
      - "${MAILPIT_SMTP_PORT:-1025}:1025"  # SMTP server
    environment:
      - MP_MAX_MESSAGES=500
      - MP_SMTP_AUTH_ACCEPT_ANY=1
```

**Key points**:
- Port 8025: Web UI to view captured emails
- Port 1025: SMTP server for your app to send emails to

**Application configuration**:

```yaml
services:
  api:
    environment:
      - SMTP_HOST=mailpit
      - SMTP_PORT=1025  # Internal port (fixed)
```

**View emails**: http://localhost:8025 (or 8026, 8027, etc. for other instances)

---

## Configuring .worktree.yml

For each port in your docker-compose.yml, add an entry to `.worktree.yml`:

```yaml
env_variables:
  APP_PORT:
    name: "Backend API"                # Display name
    url: "http://{host}:{port}"       # URL template (for worktree ports command)
    port: "8080"                       # Base port (hint for first instance)
    env: "APP_PORT"                    # Env var name
    range: [8080, 8180]                # Allocation range (100 instances)

  FE_PORT:
    name: "Frontend"
    url: "http://{host}:{port}"
    port: "3000"
    env: "FE_PORT"
    range: [3000, 3100]

  POSTGRES_PORT:
    name: "PostgreSQL"
    url: "{host}:{port}"               # No http:// for database
    port: "5432"
    env: "POSTGRES_PORT"
    range: [5432, 5532]

  # String template that references other env vars
  REACT_APP_API_BASE_URL:
    name: "API Base URL (for frontend)"
    url: "{value}"
    value: "http://{host}:{APP_PORT}"  # Uses {APP_PORT} placeholder
    env: "REACT_APP_API_BASE_URL"
```

**Key concepts**:

1. **Port ranges**: `[start, end]` - worktree-manager allocates from this range
2. **Base port**: `port: "8080"` - hint for first instance (not enforced)
3. **String templates**: `value: "http://{host}:{APP_PORT}"` - references other env vars
4. **URL templates**: For display in `worktree ports` command

**String template evaluation order**:
1. First pass: Allocate all ports (APP_PORT, FE_PORT, POSTGRES_PORT)
2. Second pass: Calculate string templates (REACT_APP_API_BASE_URL)

---

## Inter-Service Communication

Understanding how services communicate is critical for correct configuration.

### Inside Docker Network

**Rule**: Use service names and internal (fixed) ports.

```yaml
services:
  api:
    environment:
      # ✅ Correct - service name + internal port
      - DATABASE_URL=postgresql://postgres:postgres@postgres:5432/app
      - REDIS_URL=redis://redis:6379
      - WORKER_URL=http://worker:8081

  worker:
    environment:
      # ✅ Correct - service name + internal port
      - DATABASE_URL=postgresql://postgres:postgres@postgres:5432/app
      - REDIS_URL=redis://redis:6379
```

**Why**: Docker creates an internal network where containers can reach each other by service name. Port mapping (HOST:CONTAINER) doesn't apply here.

### Outside Docker Network (Browser → Backend)

**Rule**: Use localhost and external (dynamic) ports.

```yaml
services:
  frontend:
    environment:
      # ✅ Correct - localhost + external port
      - REACT_APP_API_BASE_URL=http://localhost:${APP_PORT}
```

**Why**: Frontend code runs in the browser (not inside Docker). Browser makes HTTP requests to `localhost:8080` (external port), which Docker maps to container's internal port 8080.

### Diagram: Port Mapping Flow

```
Browser                 Docker Host            Docker Container
--------                -----------            ----------------
Request to       --->   Port mapping    --->   Application
localhost:8080          (8080:8080)            listening on :8080

                        For instance 1:
                        8081:8080

Browser request  --->   Host port 8081  --->   Container port 8080
localhost:8081
```

### Common Mistake: Using External Port Inside Containers

```yaml
# ❌ WRONG
services:
  api:
    environment:
      - DATABASE_URL=postgresql://postgres:postgres@postgres:${POSTGRES_PORT}/app
      # This will use 5433 for instance 1, which doesn't exist inside Docker network!
```

**Result**: Connection refused errors because postgres:5433 doesn't exist (only postgres:5432 exists inside the network).

---

## Migration Checklist

Migrating from hardcoded ports? Follow these steps:

### Step 1: Audit Current Ports

Find all hardcoded ports in your docker-compose.yml files:

```bash
grep -r ":[0-9]\{4\}:" . --include="docker-compose*.yml"
```

Make a list:
- Backend API: 8080
- Frontend: 3000
- PostgreSQL: 5432
- Redis: 6379
- etc.

### Step 2: Update docker-compose.yml Files

For each hardcoded port, replace with the `${VAR:-default}` pattern:

**Before**:
```yaml
services:
  api:
    ports:
      - "8080:8080"  # ❌ Hardcoded
```

**After**:
```yaml
services:
  api:
    ports:
      - "${APP_PORT:-8080}:8080"  # ✅ Dynamic with fallback
    environment:
      - APP_PORT=${APP_PORT}      # Pass env var to container (optional but recommended)
```

**Do this for ALL services**: web servers, databases, caches, etc.

### Step 3: Add env_variables to .worktree.yml

For each port you found, add an entry:

```yaml
env_variables:
  APP_PORT:
    name: "Backend API"
    url: "http://{host}:{port}"
    port: "8080"
    env: "APP_PORT"
    range: [8080, 8180]  # Adjust range based on how many instances you need

  # Add entries for all other ports...
```

### Step 4: Test Single Instance

```bash
worktree new-feature feature/test-migration
worktree ports feature-test-migration
worktree status feature-test-migration
```

Visit URLs and verify everything works.

### Step 5: Test Multiple Instances

```bash
worktree new-feature feature/test-2
worktree list
```

Both should be running with different ports!

### Step 6: Verify Inter-Service Communication

Check that services can connect to each other:

- Backend → Database
- Backend → Redis
- Frontend → Backend (in browser)
- Worker → Database
- Worker → Redis

### Step 7: Clean Up

```bash
worktree remove feature-test-migration
worktree remove feature-test-2
```

### Step 8: Document

Add to your project README:
- How to create instances (`worktree new-feature`)
- How to check ports (`worktree ports`)
- How to manage instances (start/stop/remove)

---

## Troubleshooting

### Problem: Port conflict "address already in use"

**Full error**:
```
Error starting userland proxy: listen tcp4 0.0.0.0:8080: bind: address already in use
```

**Solutions**:

1. **Check what's using the port**:
   ```bash
   lsof -i :8080
   ```

2. **Check other worktree instances**:
   ```bash
   worktree list
   ```

3. **Run health check**:
   ```bash
   worktree doctor
   ```

4. **Stop conflicting instance**:
   ```bash
   worktree stop feature-old
   # Or
   docker compose down  # From the conflicting directory
   ```

### Problem: Frontend can't reach backend

**Symptom**: Console errors like "Failed to fetch" or "Network error"

**Causes and Solutions**:

1. **Frontend isn't reading env var**:
   ```javascript
   // ❌ Wrong
   const API_URL = 'http://localhost:8080';  // Hardcoded

   // ✅ Correct
   const API_URL = process.env.REACT_APP_API_BASE_URL || 'http://localhost:8080';
   ```

2. **Env var not set in docker-compose.yml**:
   ```yaml
   services:
     app:
       environment:
         - REACT_APP_API_BASE_URL=${REACT_APP_API_BASE_URL}  # Add this
   ```

3. **CORS not configured for dynamic port**:
   ```yaml
   # In backend docker-compose.yml
   services:
     api:
       environment:
         - CORS_ALLOWED_ORIGINS=http://localhost:${FE_PORT}  # Dynamic!
   ```

### Problem: Backend can't connect to database

**Symptom**: "connection refused" or "could not connect to server"

**Cause**: Using external port instead of internal port

**Solution**:
```yaml
# ✅ Correct
services:
  api:
    environment:
      - DATABASE_URL=postgresql://postgres:postgres@postgres:5432/app

# ❌ Wrong
services:
  api:
    environment:
      - DATABASE_URL=postgresql://postgres:postgres@postgres:${POSTGRES_PORT}/app
```

**Remember**: Inside Docker network, always use internal ports!

### Problem: Hot reload not working in frontend

**Cause**: WDS_SOCKET_PORT not set

**Solution**:
```yaml
services:
  app:
    environment:
      - WDS_SOCKET_PORT=${FE_PORT}  # For Webpack Dev Server
```

This tells Webpack to use the dynamic frontend port for hot reload WebSocket connection.

### Problem: Migrations fail in start_post_command

**Symptom**: "No such container" or "Container not running"

**Cause**: Container not fully started yet

**Solution**: Add healthcheck to wait for readiness:

```yaml
services:
  api:
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 5s
      timeout: 5s
      retries: 5
      start_period: 10s
```

Or add a delay:
```yaml
projects:
  backend:
    start_post_command: "sleep 5 && docker compose exec -T api make migrate"
```

---

## Advanced Patterns

### Pattern 1: Shared Services Across Instances

For services like databases, you might want to share across instances to save resources:

**Option A**: Run database outside worktree-manager (manually), use same port for all instances:

```yaml
# .worktree.yml - no POSTGRES_PORT (use hardcoded)
env_variables:
  APP_PORT:
    # ...
  # No POSTGRES_PORT entry
```

```yaml
# docker-compose.yml - hardcoded database port
services:
  api:
    environment:
      - DATABASE_URL=postgresql://postgres:postgres@localhost:5432/app_{INSTANCE}
```

**Drawback**: Requires manual database setup, can't use Docker Compose for database.

**Option B**: Use database per instance (recommended):

Each instance gets its own isolated database. Clean separation, no shared state.

### Pattern 2: Fixed Port for One Service

If ONE service needs a fixed port (e.g., OAuth callback URL):

```yaml
# .worktree.yml
env_variables:
  API_PORT:
    name: "API (OAuth callback)"
    url: "http://{host}:{port}"
    port: "8080"          # Fixed
    env: "API_PORT"
    range: [8080, 8080]   # Range of 1 = always 8080
```

**Drawback**: Can only run one instance using this preset. Consider using different presets for different use cases.

### Pattern 3: Display-Only Ports

Some ports don't need to be allocated (just calculated):

```yaml
# .worktree.yml
env_variables:
  MAILPIT_SMTP_PORT:
    name: "Mailpit SMTP"
    url: "{host}:{port}"
    value: "{MAILPIT_PORT} + 1000"  # Calculated: If MAILPIT_PORT=8025, this is 9025
    env: "MAILPIT_SMTP_PORT"
```

**Use case**: Related ports that follow a pattern (HTTP port + 1000 = SMTP port).

---

## Key Takeaways

✅ **Always use `${VAR:-default}` pattern** for all ports in docker-compose.yml
✅ **Port mapping format**: `"HOST:CONTAINER"` - left (HOST) is dynamic, right (CONTAINER) is fixed
✅ **Inside Docker network**: Use service names and internal ports (e.g., `postgres:5432`)
✅ **Outside Docker network**: Use localhost and external ports (e.g., `localhost:${APP_PORT}`)
✅ **Worktree-manager exports env vars** before running `docker compose up -d`
✅ **Fallback defaults (`:-8080`)** let you run `docker compose up` manually without worktree-manager
✅ **String templates** let env vars reference each other (frontend knows backend URL)
✅ **Each instance is completely isolated** with unique ports

## Related Resources

- **[Example 3: Fullstack Docker](../examples/03-fullstack-docker/)** - Complete walkthrough with code examples
- **[Migration Guide](../examples/MIGRATION-GUIDE.md)** - Step-by-step migration from hardcoded ports
- **[Example 4: Polyglot Services](../examples/04-polyglot-services/)** - Multiple services with different languages
- **[Complete Config Reference](../.worktree.example.yml)** - All configuration options

**This pattern is the foundation for multi-instance Docker development!** 🚀
