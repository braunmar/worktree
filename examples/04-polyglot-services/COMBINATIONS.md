# Preset Combinations Guide

Learn when to use each preset for maximum efficiency. The key idea: **run only what you need**!

## Available Presets

| Preset | Services | Use When |
|--------|----------|----------|
| `fullstack` | API + Frontend | Working on UI features that call APIs |
| `api-worker` | API + Worker | Working on backend/async processing |
| `frontend-all` | Frontend + API + Worker | Working on features that need everything |
| `api` | API only | Testing API endpoints, writing integration tests |
| `all` | API + Worker + Frontend | Same as frontend-all (everything) |

---

## Scenario 1: Building a New API Endpoint (No UI Needed)

**What you're doing**: Adding `/api/export-data` endpoint to generate CSV exports

**What you need**:
- ✅ API (to test the endpoint)
- ✅ Database (to query data)
- ✅ Redis (not strictly needed, but typically shared)
- ❌ Worker (not needed for this endpoint)
- ❌ Frontend (testing with curl/Postman)

**Command**:
```bash
worktree new-feature feature/export-endpoint --preset api
```

**What starts**:
- Node.js API on `API_PORT` (e.g., 8080)
- PostgreSQL on `POSTGRES_PORT` (e.g., 5432)
- Redis on `REDIS_PORT` (e.g., 6379)

**What doesn't start** (saves resources):
- Python Worker
- Frontend

**Testing**:
```bash
curl -X POST http://localhost:8080/api/export-data \
  -H "Content-Type: application/json" \
  -d '{"format": "csv", "start_date": "2024-01-01"}'
```

**Why this is useful**:
- Faster startup (fewer containers)
- Less memory usage
- Clearer mental model (only API running)
- No need to wait for frontend to build

---

## Scenario 2: Building a User Profile Page

**What you're doing**: Creating a new UI page to display user profile

**What you need**:
- ✅ Frontend (to see the UI)
- ✅ API (to fetch user data)
- ✅ Database (API needs data)
- ❌ Worker (profile page doesn't trigger background jobs)

**Command**:
```bash
worktree new-feature feature/user-profile --preset fullstack
```

**What starts**:
- Frontend on `FE_PORT` (e.g., 3000)
- Node.js API on `API_PORT` (e.g., 8080)
- PostgreSQL on `POSTGRES_PORT` (e.g., 5432)
- Redis on `REDIS_PORT` (e.g., 6379)

**What doesn't start**:
- Python Worker (not needed)

**Testing**:
```bash
# Open frontend
open http://localhost:3000/profile

# Frontend makes API call to:
# http://localhost:8080/api/users/123
```

**Why this is useful**:
- Most frontend work doesn't need the worker
- Worker adds startup time and memory overhead
- Keep it simple: only run what you need

---

## Scenario 3: Building a Report Generation Feature (Needs Background Jobs)

**What you're doing**: Adding a feature to generate monthly reports (async processing, can take minutes)

**What you need**:
- ✅ Frontend (to trigger report generation)
- ✅ API (to receive request and enqueue job)
- ✅ Worker (to process report in background)
- ✅ Database (to query data for report)
- ✅ Redis (job queue)

**Command**:
```bash
worktree new-feature feature/monthly-reports --preset frontend-all
```

**What starts**:
- Frontend on `FE_PORT` (e.g., 3000)
- Node.js API on `API_PORT` (e.g., 8080)
- Python Worker on `WORKER_PORT` (e.g., 8081)
- PostgreSQL on `POSTGRES_PORT` (e.g., 5432)
- Redis on `REDIS_PORT` (e.g., 6379)

**All services!** This is when you need everything.

**Workflow**:
1. User clicks "Generate Report" in frontend (port 3000)
2. Frontend POSTs to API (port 8080): `/api/reports/generate`
3. API enqueues job to Redis: `LPUSH jobs '{"type": "monthly_report"}'`
4. API responds: `{"status": "queued", "job_id": "abc123"}`
5. Worker (port 8081) pulls job from Redis: `BRPOP jobs`
6. Worker processes report (queries database, generates PDF, saves to S3)
7. Worker updates job status in database
8. Frontend polls API for status: `/api/jobs/abc123`

**Why you need everything**:
- Frontend: User interface
- API: Receive request, enqueue job, check status
- Worker: Actually generate the report
- Database: Query data, store job status
- Redis: Job queue

---

## Scenario 4: Working on the Python Worker Itself

**What you're doing**: Optimizing the worker's report generation logic

**What you need**:
- ✅ API (to enqueue test jobs)
- ✅ Worker (what you're optimizing)
- ✅ Database (worker needs data)
- ✅ Redis (job queue)
- ❌ Frontend (not needed for worker development)

**Command**:
```bash
worktree new-feature feature/optimize-worker --preset api-worker
```

**What starts**:
- Node.js API on `API_PORT` (to enqueue jobs)
- Python Worker on `WORKER_PORT` (what you're working on)
- PostgreSQL + Redis (worker dependencies)

**What doesn't start**:
- Frontend (not needed)

**Testing**:
```bash
# Enqueue test job via API
curl -X POST http://localhost:8080/api/process-report \
  -H "Content-Type: application/json" \
  -d '{"type": "monthly", "month": "2024-01"}'

# Check worker logs
docker logs -f polyglot-app-feature-optimize-worker-worker-1

# Check worker health
curl http://localhost:8081/health
```

**Why this is useful**:
- Faster startup (no frontend build)
- Focus on backend only
- Can enqueue many test jobs quickly
- Monitor worker performance directly

---

## Scenario 5: Full-Stack Feature (API + Worker + Frontend)

**What you're doing**: Implementing end-to-end feature (e.g., async image processing)

**What you need**:
- ✅ Everything

**Commands** (all equivalent):
```bash
worktree new-feature feature/image-processing --preset frontend-all
# OR
worktree new-feature feature/image-processing --preset all
```

Both start all services.

---

## Quick Reference Table

| Scenario | Preset | Services Running | Typical Use |
|----------|--------|------------------|-------------|
| API endpoint development | `api` | API, DB, Redis | Testing with curl/Postman |
| UI development (simple) | `fullstack` | API, Frontend, DB, Redis | Most frontend work |
| Async features | `frontend-all` or `all` | Everything | Report generation, email sending, image processing |
| Backend job optimization | `api-worker` | API, Worker, DB, Redis | Worker development |

---

## Key Insights

### 1. Default Preset Matters

The default preset is `fullstack` because most development is frontend + API (no worker needed).

```yaml
default_preset: fullstack
```

If you run:
```bash
worktree new-feature feature/user-settings
# No --preset flag → uses fullstack
```

### 2. Presets Save Time and Resources

**Without presets** (always running everything):
- Frontend: ~30s startup + 500MB RAM
- API: ~10s startup + 300MB RAM
- Worker: ~15s startup + 400MB RAM
- Database: ~5s startup + 300MB RAM
- Redis: ~2s startup + 50MB RAM
- **Total**: ~62s startup, ~1.55GB RAM

**With presets** (run only what you need):
- `fullstack` (no worker): ~47s startup, ~1.15GB RAM
- `api` (just API): ~17s startup, ~650MB RAM
- **Savings**: 25-75% faster startup, 30-60% less memory

### 3. You Can Switch Presets

**Start with fullstack**:
```bash
worktree new-feature feature/test --preset fullstack
# Running: API, Frontend
```

**Later, need to test worker**:
```bash
worktree stop feature-test
worktree start feature-test --preset frontend-all
# Now running: API, Frontend, Worker
```

**Note**: Port allocations stay the same (API_PORT, FE_PORT, etc. don't change).

### 4. Each Preset Is Independent

Instance 1 with `fullstack`:
```bash
worktree new-feature feature/ui-work --preset fullstack
# Runs: API (8080), Frontend (3000)
```

Instance 2 with `api-worker`:
```bash
worktree new-feature feature/backend-work --preset api-worker
# Runs: API (8081), Worker (8082)
```

Instance 3 with `all`:
```bash
worktree new-feature feature/e2e-test --preset all
# Runs: API (8083), Frontend (3003), Worker (8084)
```

All running simultaneously with different port allocations!

---

## Best Practices

### 1. Start with the Simplest Preset

Don't default to `all`. Start with the minimum:
- **API endpoint?** → Use `api`
- **UI work?** → Use `fullstack`
- **Async feature?** → Use `frontend-all`

### 2. Name Presets by What They Include, Not by Use Case

**Good**:
- `fullstack` (clear: frontend + api)
- `api-worker` (clear: api + worker)

**Bad**:
- `development` (unclear: what's included?)
- `testing` (unclear: what's included?)

### 3. Document Preset Usage in Your Project README

```markdown
## Preset Guide

- `worktree new-feature <name>` → fullstack (API + Frontend)
- `worktree new-feature <name> --preset api` → API only
- `worktree new-feature <name> --preset api-worker` → API + Worker
- `worktree new-feature <name> --preset all` → Everything
```

### 4. Consider Your Team's Workflow

If most developers work on frontend:
- Default preset: `fullstack`

If most developers work on backend:
- Default preset: `api-worker`

If you do lots of E2E testing:
- Default preset: `all`

---

## Next Steps

- **[Example 3: Fullstack Docker](../03-fullstack-docker/)** - Learn Docker integration basics
- **[docs/PORT-PATTERNS.md](../../docs/PORT-PATTERNS.md)** - Deep dive into port patterns
- **[Example 5: Real-World](../05-real-world/)** - See production configuration with 6 presets

**Key Takeaway**: Presets are powerful! They let you run exactly what you need, saving time and resources. 🚀
