# Worktree Manager - Open Source Package

Generic git worktree management tool for multi-instance development environments.

## Overview

Worktree Manager is a CLI tool that helps teams work on multiple features in parallel by:
- Creating coordinated git worktrees across multiple repositories
- Managing isolated development instances with Docker
- Running post-startup commands automatically
- Providing one-command feature setup

**Perfect for:**
- Microservice architectures
- Multi-repo projects (frontend + backend + services)
- Teams working on multiple features simultaneously
- Projects with complex Docker setups

---

## Installation

### From Source

```bash
git clone https://github.com/yourusername/worktree-manager
cd worktree-manager
make build
```

Binary is created at `scripts/worktree`.

### Pre-built Binaries

Download from [Releases](https://github.com/yourusername/worktree-manager/releases).

---

## Quick Start

### 1. Create Configuration

Create `.worktree.yml` in your project root:

```yaml
projects:
  backend:
    dir: backend
    main_branch: main
    start_command: "docker-compose up -d"
    post_command: "npm run migrate && npm run seed"
    claude_working_dir: true

  frontend:
    dir: frontend
    main_branch: main
    start_command: "npm start"

presets:
  fullstack:
    projects: [backend, frontend]
    description: "Full-stack development"

default_preset: fullstack
max_instances: 5
auto_fixtures: true
```

### 2. One-Command Setup

```bash
worktree new-feature "Add user authentication" 1
```

This:
1. Creates worktrees for backend + frontend
2. Starts all services
3. Runs post-startup commands (migrations, seed data)
4. Ready to code!

### 3. Work on Feature

Everything is isolated:
- `worktrees/1/backend` - Backend on `feature/add-user-authentication`
- `worktrees/1/frontend` - Frontend on same branch
- Separate Docker network, database, ports

### 4. Clean Up

```bash
worktree remove 1
```

---

## Configuration

### Project Configuration

Each project in `.worktree.yml` can have:

```yaml
projects:
  myproject:
    # Required
    dir: path/to/project              # Directory relative to root
    main_branch: main                 # Main branch name

    # Optional
    start_command: "..."              # Start services ({instance} placeholder)
    post_command: "..."               # Run after start (migrations, fixtures, seed)
    claude_working_dir: true          # Set as working directory
```

### Presets

Define common setups:

```yaml
presets:
  fullstack:
    projects: [backend, frontend]
    description: "Backend + Frontend"

  backend-only:
    projects: [backend]
    description: "Backend API only"
```

### Placeholders

Commands support `{instance}` placeholder:

```yaml
start_command: "docker-compose up -d --project-name myapp-{instance}"
post_command: "DB_PORT=543{instance} npm run migrate && npm run seed"
```

---

## Commands

### `worktree new-feature <branch> <instance> [preset]`

One-command feature setup.

```bash
worktree new-feature "Add search" 1 fullstack
worktree new-feature "Fix bug" 2 backend --no-fixtures
```

**Flags:**
- `--no-fixtures` - Skip post-commands

### `worktree create <branch> <instance>`

Create worktrees only (no services).

```bash
worktree create feature/my-feature 1
```

### `worktree start <instance>`

Start services for existing worktree.

```bash
worktree start 1
worktree start 1 --no-fixtures
```

### `worktree list`

List all worktrees with status.

```bash
worktree list
```

### `worktree remove <instance>`

Remove worktrees safely.

```bash
worktree remove 1
worktree remove 1 --force
```

---

## Integration with Claude AI

For teams using [Claude Code](https://claude.com/claude-code):

1. Create skill directory: `.claude/skills/new-feature/`
2. Add `skill.md` and `handler.sh` (see examples in repo)
3. Use `/new-feature` command to set up environment

Claude automatically navigates to the worktree and is ready to code.

---

## Use Cases

### Multi-Repo Monorepo

```yaml
projects:
  api:
    dir: services/api
    start_command: "cd services/api && npm start"

  web:
    dir: apps/web
    start_command: "cd apps/web && npm run dev"

  mobile:
    dir: apps/mobile
    start_command: "cd apps/mobile && expo start"
```

### Microservices

```yaml
projects:
  users-service:
    dir: microservices/users
    start_command: "docker-compose -f microservices/users/docker-compose.yml up"

  orders-service:
    dir: microservices/orders
    start_command: "docker-compose -f microservices/orders/docker-compose.yml up"
```

### Different Languages

```yaml
projects:
  backend-go:
    dir: backend
    start_command: "go run main.go --port 808{instance}"

  frontend-react:
    dir: frontend
    start_command: "PORT=300{instance} npm start"

  worker-python:
    dir: worker
    start_command: "python worker.py --instance {instance}"
```

---

## Architecture

```
worktree-manager/
├── cmd/                # CLI commands
│   ├── create.go      # Create worktrees
│   ├── remove.go      # Remove worktrees
│   ├── start.go       # Start services
│   ├── list.go        # List worktrees
│   └── newfeature.go  # One-command setup
├── pkg/
│   ├── config/        # Configuration & port allocation
│   ├── git/           # Git worktree operations
│   ├── docker/        # Docker instance checks
│   └── ui/            # Terminal output
└── main.go
```

**Dependencies:**
- cobra - CLI framework
- yaml.v3 - Config parsing
- color - Terminal colors

---

## Port Allocation

Automatic port calculation based on instance:

```
Instance 0: 3000, 8080, 5432
Instance 1: 3001, 8081, 5433
Instance 2: 3002, 8082, 5434
...
```

Configure in your commands:
```yaml
start_command: "docker-compose up -p myapp-{instance} --env-file .env.{instance}"
```

---

## Best Practices

1. **Use presets** - Define common setups (fullstack, backend-only, etc.)
2. **Placeholder everything** - Use `{instance}` for ports, database names, etc.
3. **Keep post_command idempotent** - Fixtures should be safe to re-run
4. **Clean up regularly** - Remove unused worktrees
5. **Version control config** - Commit `.worktree.yml` to git

---

## Comparison

| Feature | Worktree Manager | Git Worktree | Docker Compose |
|---------|------------------|--------------|----------------|
| Multi-repo | ✅ | ❌ | ❌ |
| Auto-start services | ✅ | ❌ | Partial |
| Port allocation | ✅ | ❌ | Manual |
| Post-commands | ✅ | ❌ | Manual |
| One command | ✅ | ❌ | ❌ |

---

## Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md).

1. Fork the repo
2. Create feature branch
3. Make changes
4. Run tests: `make test`
5. Submit PR

---

## License

MIT License - See [LICENSE](LICENSE)

---

## Support

- **Issues**: [GitHub Issues](https://github.com/yourusername/worktree-manager/issues)
- **Discussions**: [GitHub Discussions](https://github.com/yourusername/worktree-manager/discussions)
- **Docs**: [Full Documentation](https://worktree-manager.dev)

---

## Roadmap

- [ ] GitHub Actions integration
- [ ] Remote worktrees (SSH)
- [ ] Template repository
- [ ] VS Code extension
- [ ] Auto-cleanup stale worktrees
- [ ] Worktree snapshots

---

**Made with ❤️ for developers who work on multiple features**
