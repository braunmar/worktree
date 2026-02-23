# Example 4: Polyglot Services (Node API + Python Worker + Frontend)

Learn how to work with multiple services in different languages and use presets to run exactly what you need.

## Use Case

You have:
- **API** (Node.js) - REST API for frontend
- **Worker** (Python) - Background job processor
- **Frontend** (React) - User interface

Different workflows need different combinations:
- UI development → API + Frontend (no worker)
- Backend development → API + Worker (no frontend)
- Full-stack features → Everything
- API testing → API only

**The Problem**: Running ALL services ALL the time wastes resources and slows down startup.

**The Solution**: Use presets to run exactly what you need!

## What You'll Learn

- How to configure multiple services with different languages
- How to use presets for different workflow combinations
- When to use `fullstack` vs `api-worker` vs `all`
- How services communicate (API → Worker via Redis)
- Real-world patterns for microservices development

## Project Structure

See [project-structure.txt](project-structure.txt) for detailed layout.

**Key characteristics**:
- Three separate Git repositories (api, worker, frontend)
- Each repository can use different language/stack
- Services communicate via Docker network + Redis queue
- Presets control which services run

---

## Configuration Walkthrough

Let's break down the [.worktree.yml](.worktree.yml) file.

### Projects Section

```yaml
projects:
  api:
    executor: docker
    dir: api
    main_branch: main
    start_command: "docker compose up -d"

  worker:
    executor: docker
    dir: worker
    main_branch: main
    start_command: "docker compose up -d"

  frontend:
    executor: docker
    dir: frontend
    main_branch: main
    start_command: "docker compose up -d"
```

**Three projects!** Each is a separate git repository with its own docker-compose.yml.

### Presets Section (The Power of Combinations)

```yaml
presets:
  fullstack:
    projects: [api, frontend]
    description: "Frontend + API only (no worker)"

  api-worker:
    projects: [api, worker]
    description: "API + Worker (no UI)"

  frontend-all:
    projects: [frontend, api, worker]
    description: "Full stack with worker"

  api:
    projects: [api]
    description: "API only"

  all:
    projects: [api, worker, frontend]
    description: "All services"
```

**Five presets for different workflows!**

**See [COMBINATIONS.md](COMBINATIONS.md)** for detailed scenarios and when to use each preset.

### Environment Variables

```yaml
env_variables:
  API_PORT:
    name: "Node.js API"
    url: "http://{host}:{port}"
    port: "8080"
    env: "API_PORT"
    range: [8080, 8180]

  WORKER_PORT:
    name: "Python Worker (Health Check)"
    url: "http://{host}:{port}/health"
    port: "8081"
    env: "WORKER_PORT"
    range: [8081, 8181]

  # ... more ports ...

  REACT_APP_API_BASE_URL:
    name: "API Base URL (for frontend)"
    url: "{value}"
    value: "http://{host}:{API_PORT}"
    env: "REACT_APP_API_BASE_URL"
```

**Each service gets its own port**. Frontend knows backend URL via string template.

---

## How Services Communicate

### 1. Frontend → API (HTTP)

**Frontend** (React):
```javascript
const API_BASE_URL = process.env.REACT_APP_API_BASE_URL; // http://localhost:8080

// User clicks "Generate Report"
fetch(`${API_BASE_URL}/api/reports/generate`, {
  method: 'POST',
  body: JSON.stringify({ type: 'monthly' })
});
```

**API** (Node.js):
```javascript
app.post('/api/reports/generate', async (req, res) => {
  // Enqueue job for worker
  await redis.lpush('jobs', JSON.stringify({
    type: 'generate_report',
    data: req.body
  }));
  res.json({ status: 'queued' });
});
```

**Communication**: HTTP over localhost (browser → API)

### 2. API → Worker (Redis Queue)

**API** enqueues job:
```javascript
await redis.lpush('jobs', JSON.stringify({ type: 'generate_report' }));
```

**Worker** processes job:
```python
import redis
r = redis.Redis.from_url(os.getenv('REDIS_URL'))

while True:
    _, job_data = r.brpop('jobs')  # Block until job available
    job = json.loads(job_data)

    if job['type'] == 'generate_report':
        # Process report...
        print(f"Processing: {job['data']}")
```

**Communication**: Redis queue (API pushes, Worker pulls)

### 3. Worker → Database (PostgreSQL)

**Worker** queries data:
```python
import psycopg2
conn = psycopg2.connect(os.getenv('DATABASE_URL'))
cursor = conn.cursor()
cursor.execute("SELECT * FROM users WHERE active = true")
users = cursor.fetchall()
```

**Communication**: Direct PostgreSQL connection over Docker network

---

## Try It Out

### Prerequisites

1. **Three git repositories with ≥1 commit each**:
   ```bash
   cd api && git init && git add . && git commit -m "init" && cd ..
   cd worker && git init && git add . && git commit -m "init" && cd ..
   cd frontend && git init && git add . && git commit -m "init" && cd ..
   ```

2. **docker-compose.yml files updated** with `${VAR:-default}` pattern:
   - Copy [api/docker-compose.example.yml](api/docker-compose.example.yml)
   - Copy [worker/docker-compose.example.yml](worker/docker-compose.example.yml)
   - Copy [frontend/docker-compose.example.yml](frontend/docker-compose.example.yml)

3. **Copy configuration**:
   ```bash
   cp examples/04-polyglot-services/.worktree.yml .
   ```

### Scenario 1: UI Development (Frontend + API)

```bash
worktree new-feature feature/user-profile --preset fullstack
```

**What starts**:
- Frontend on port 3000
- API on port 8080
- PostgreSQL on port 5432
- Redis on port 6379

**What doesn't start**:
- Worker (not needed for UI work)

**Test it**:
```bash
# Check ports
worktree ports feature-user-profile

# Open frontend
open http://localhost:3000

# Test API
curl http://localhost:8080/api/users
```

### Scenario 2: Backend Development (API + Worker)

```bash
worktree new-feature feature/optimize-jobs --preset api-worker
```

**What starts**:
- API on port 8081 (8080 is taken)
- Worker on port 8082
- PostgreSQL on port 5433
- Redis on port 6380

**What doesn't start**:
- Frontend (not needed)

**Test it**:
```bash
# Enqueue test job
curl -X POST http://localhost:8081/api/process-report \
  -H "Content-Type: application/json" \
  -d '{"type": "monthly"}'

# Check worker logs
docker logs -f polyglot-app-feature-optimize-jobs-worker-1
```

### Scenario 3: Full-Stack Feature (Everything)

```bash
worktree new-feature feature/async-reports --preset frontend-all
```

**What starts**:
- Frontend on port 3002
- API on port 8083
- Worker on port 8084
- PostgreSQL on port 5435
- Redis on port 6382

**All services running!**

**Test end-to-end**:
1. Open http://localhost:3002
2. Click "Generate Report" button
3. Frontend POSTs to http://localhost:8083/api/reports/generate
4. API enqueues job to Redis
5. Worker picks up job and processes it
6. Check worker logs to see processing

---

## Verification

### Check All Instances

```bash
worktree list
```

Shows:
```
Feature                  Preset         Projects                Status   Ports
feature-user-profile    fullstack      api, frontend           Running  8080, 3000
feature-optimize-jobs   api-worker     api, worker             Running  8081, 8082
feature-async-reports   frontend-all   api, worker, frontend   Running  8083, 8084, 3002
```

**Three instances, different presets, different ports!**

### Check Service Communication

**Test API → Worker communication**:

```bash
# From feature-async-reports instance
cd worktrees/feature-async-reports/api

# Enqueue test job
docker compose exec -T api node -e "
const Redis = require('ioredis');
const redis = new Redis('redis://redis:6379');
redis.lpush('jobs', JSON.stringify({ type: 'test' }));
process.exit(0);
"

# Check worker picked it up
cd ../worker
docker compose logs -f worker
# Should show: "Processing: {'type': 'test'}"
```

**Test Frontend → API communication**:

```bash
# Open browser devtools
open http://localhost:3002

# Make API call from console
fetch('http://localhost:8083/api/users').then(r => r.json()).then(console.log)

# Should see: [{"id": 1, "name": "..."}]
```

---

## Preset Decision Tree

**Use this to decide which preset**:

```
Do you need the UI (frontend)?
├─ YES: Do you need background job processing?
│   ├─ YES → Use "frontend-all" or "all"
│   └─ NO → Use "fullstack"
└─ NO: Do you need background job processing?
    ├─ YES → Use "api-worker"
    └─ NO: Do you just need to test API endpoints?
        ├─ YES → Use "api"
        └─ NO → Use "fullstack" (default)
```

**Or see [COMBINATIONS.md](COMBINATIONS.md)** for detailed scenarios!

---

## Key Patterns

### Pattern 1: API Enqueues, Worker Processes

**API** (lightweight, fast):
```javascript
app.post('/api/export-data', async (req, res) => {
  // Enqueue job (fast, returns immediately)
  await redis.lpush('jobs', JSON.stringify({
    type: 'export',
    user_id: req.user.id,
    format: req.body.format
  }));
  res.json({ status: 'queued', job_id: '...' });
});
```

**Worker** (heavyweight, slow):
```python
while True:
    _, job_data = r.brpop('jobs')
    job = json.loads(job_data)

    if job['type'] == 'export':
        # This might take 30 seconds or 5 minutes!
        data = query_database_for_export(job['user_id'])
        generate_csv(data, job['format'])
        upload_to_s3(csv_file)
```

**Why separate**:
- API stays responsive (doesn't block on slow operations)
- Worker can be scaled independently (add more worker containers)
- Failure isolation (worker crash doesn't affect API)

### Pattern 2: Shared Database, Separate Services

Both API and Worker connect to the same database:

**docker-compose.yml** (same in both api/ and worker/):
```yaml
services:
  postgres:
    image: postgres:15
    ports:
      - "${POSTGRES_PORT:-5432}:5432"
```

**Why share database**:
- Single source of truth
- Worker can update job status in database
- API can query job status from database

### Pattern 3: Frontend Polls for Job Status

**Frontend**:
```javascript
const [jobStatus, setJobStatus] = useState('pending');

useEffect(() => {
  const interval = setInterval(async () => {
    const res = await fetch(`${API_BASE_URL}/api/jobs/${jobId}`);
    const data = await res.json();
    setJobStatus(data.status);
    if (data.status === 'completed') clearInterval(interval);
  }, 2000);  // Poll every 2 seconds
  return () => clearInterval(interval);
}, [jobId]);
```

**Alternative**: Use WebSockets for real-time updates.

---

## Common Workflows

### Workflow 1: Add New API Endpoint

```bash
# API only (test with curl)
worktree new-feature feature/new-endpoint --preset api

# Implement endpoint
cd worktrees/feature-new-endpoint/api
vim src/routes/users.js

# Test
curl http://localhost:8080/api/users/new-endpoint

# Commit when done
git add . && git commit -m "Add new endpoint"
```

### Workflow 2: Add New Worker Job Type

```bash
# API + Worker (test job processing)
worktree new-feature feature/new-job --preset api-worker

# Implement job type in worker
cd worktrees/feature-new-job/worker
vim worker.py

# Update API to enqueue new job type
cd ../api
vim src/routes/jobs.js

# Test
curl -X POST http://localhost:8081/api/trigger-new-job
docker logs -f polyglot-app-feature-new-job-worker-1
```

### Workflow 3: Add Full-Stack Feature

```bash
# Everything
worktree new-feature feature/reports --preset frontend-all

# Implement UI
cd worktrees/feature-reports/frontend
vim src/components/ReportGenerator.jsx

# Implement API endpoint
cd ../api
vim src/routes/reports.js

# Implement worker processing
cd ../worker
vim worker.py

# Test end-to-end
open http://localhost:3000
```

---

## What's Next?

### For Reference

- **[Example 5: Real-World](../05-real-world/)** - See production config with 5+ projects
- **[COMBINATIONS.md](COMBINATIONS.md)** - Deep dive into preset strategies
- **[docs/PORT-PATTERNS.md](../../docs/PORT-PATTERNS.md)** - Docker integration patterns

### For Migration

- **[Migration Guide](../MIGRATION-GUIDE.md)** - Step-by-step migration from hardcoded ports

### For Basics

- **[Example 3: Fullstack Docker](../03-fullstack-docker/)** - Learn Docker integration basics

---

## Key Takeaways

✅ **Presets are powerful** - Run only what you need (API, Worker, Frontend, or combinations)
✅ **Polyglot is easy** - Node + Python + React all work together seamlessly
✅ **Services communicate** via Docker network (internal ports) and Redis (job queue)
✅ **Each instance is isolated** - Different ports, different branches, no conflicts
✅ **Default preset matters** - Most teams use `fullstack` (API + Frontend, no Worker)
✅ **You can switch presets** - Start with `fullstack`, later use `frontend-all` if needed
✅ **Background jobs** should be processed by Worker, not API (keeps API responsive)

**Presets let you run exactly what you need - no more, no less!** 🚀
