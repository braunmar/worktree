# Worktree Manager Examples

Practical, copy-paste ready examples for common use cases. Each example includes:
- Complete `.worktree.yml` configuration
- Project structure diagram
- Step-by-step setup guide
- Verification steps
- Troubleshooting tips

## Quick Start: Choose Your Path

### 📦 Example 1: Minimal Setup
**[examples/01-minimal/](01-minimal/)** - Single project, no Docker

Perfect for getting started in 5 minutes! Learn the basics:
- How to create your first `.worktree.yml`
- How port allocation works
- How to run `worktree new-feature`

**Time**: 5 minutes | **Complexity**: ⭐

---

### 🔧 Example 2: Fullstack Basic
**[examples/02-fullstack-basic/](02-fullstack-basic/)** - Frontend + Backend without Docker

Learn how to work with multiple projects using process executors (npm start, go run):
- How presets group projects
- How env vars reference each other
- Run different combinations (backend-only, frontend-only, fullstack)

**Time**: 10 minutes | **Complexity**: ⭐⭐

---

### 🚀 Example 3: Fullstack Docker (Most Common)
**[examples/03-fullstack-docker/](03-fullstack-docker/)** - Frontend + Backend with Docker Compose

This is what most teams need! Learn how to:
- Configure `docker-compose.yml` with `${APP_PORT:-8080}` pattern
- Run multiple instances without port conflicts
- Integrate with existing Docker setups
- Set up multi-repo structure with shared `.claude/` directory

**Time**: 15 minutes | **Complexity**: ⭐⭐⭐

---

### 🏗️ Example 4: Polyglot Services
**[examples/04-polyglot-services/](04-polyglot-services/)** - Node API + Python Worker + Frontend

Learn how to work with multiple languages and services:
- Run different project combinations with presets
- Frontend + API only (no worker)
- API + Worker only (no frontend)
- Everything together

**Time**: 20 minutes | **Complexity**: ⭐⭐⭐⭐

---

### 🏭 Example 5: Real-World Production
**[examples/05-real-world/](05-real-world/)** - Production Configuration

See how a real production project uses worktree-manager:
- Multiple repos (backend, frontend, web, infrastructure, ai-config)
- Scheduled agents for housekeeping
- Lifecycle hooks
- Complex presets

**Time**: Reference material | **Complexity**: ⭐⭐⭐⭐⭐

---

## Additional Resources

- **[Docker Port Patterns](../docs/PORT-PATTERNS.md)** - Comprehensive guide to Docker Compose integration ⭐
- **[Migration Guide](MIGRATION-GUIDE.md)** - Migrating from hardcoded ports to dynamic allocation
- **[Complete Config Reference](../.worktree.example.yml)** - All 492 options documented
- **[Real Project Config](../.worktree.example-real.yml)** - Production setup (652 lines)

## Contributing Examples

Have a useful example? See [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines.

Examples should:
- Solve a real-world problem
- Be copy-paste ready
- Include verification steps
- Show actual file structure
- Use conversational, friendly tone
