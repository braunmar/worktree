# Worktree Manager

[![CI](https://github.com/braunmar/worktree/actions/workflows/ci.yml/badge.svg)](https://github.com/braunmar/worktree/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/braunmar/worktree/branch/main/graph/badge.svg)](https://codecov.io/gh/braunmar/worktree)
[![Go Report Card](https://goreportcard.com/badge/github.com/braunmar/worktree)](https://goreportcard.com/report/github.com/braunmar/worktree)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

> **Disclaimer:** This software is provided "as is", without warranty of any kind. See [LICENSE](LICENSE) for full terms.

> **For AI Agents:** Setting up worktree-manager for a new project? See [SKILL.md](SKILL.md) for comprehensive AI-friendly setup guidance with configuration wizards and decision trees.

## Motivation

A CLI tool for managing multiple git worktrees with coordinated Docker environments and dynamic port allocation.

This is not a replacement of your setup of Git worktrees, [OpenClaw](https://openclaw.ai/) or [Vibe-Kanban](https://www.vibekanban.com/). It is meant to extend it.

Keep your main .git repo for hotfixes and code reviews. Configure, run and develop 1-N features separately with your preferred tooling in git worktrees. It is meant for humans when you want develop N features/fixes simultaneously, but it can be used as tool for agents.

## How it works

Simply configure your needs, wrapping git worktrees, allocating ports, setting correct ENVironment and executing pre-command, command, post-command to manage multiple git worktrees environments.

## Quick Start

**For AI Agents:** See [SKILL.md](SKILL.md) for step-by-step configuration wizard and decision trees.

**For Humans:**

### 1. Install

**Option 1: Go Install (Recommended)**
```bash
go install github.com/braunmar/worktree@latest
```

**Option 2: Download Release**

Download the latest binary from [GitHub Releases](https://github.com/braunmar/worktree/releases).

**Option 3: Build from Source**
```bash
make build
make install
```

### 2. Create Configuration

Create a `.worktree.yml` file in your project root. See [.worktree.example.yml](.worktree.example.yml) for a complete example or real project configuration [.worktree.example-real.yml](.worktree.example-real.yml).

### 3. Create Your First Feature

```bash
worktree new-feature feature/my-feature
```

That's it! The tool will create worktrees, allocate ports, and start services.

## Common Commands

```bash
worktree list                    # List all features
worktree start <feature-name>    # Start a feature
worktree stop <feature-name>     # Stop a feature
worktree remove <feature-name>   # Remove a feature
worktree doctor                  # Check health
```

## Documentation

### Getting Started

New to worktree-manager? Start with a hands-on example:

- **[Fullstack Docker Guide](examples/03-fullstack-docker/)** - Frontend + Backend with Docker Compose ⭐ Most Common
  - Learn the `${APP_PORT:-8080}` pattern
  - Set up multi-repo structure
  - Run multiple instances without port conflicts
- **[All Examples](examples/)** - Browse all examples (minimal → production)

### Configuration Reference

- **[Docker Port Patterns](docs/PORT-PATTERNS.md)** - How to integrate with Docker Compose ⭐
- **[Complete Config](.worktree.example.yml)** - All options documented (492 lines)
- **[Real Project Config](.worktree.example-real.yml)** - Production setup (652 lines)

### Guides

- **[Migration Guide](examples/MIGRATION-GUIDE.md)** - From hardcoded ports to dynamic allocation

### Developer Reference

- **[Architecture](CLAUDE.md)** - For contributors and AI agents
- **[AI Agent Setup](SKILL.md)** - Configuration wizard for AI agents

## TODO

Simple agents workflow configuration is on the roadmap, it is meant to run simple agents for simple needs like housekeeping jobs (npm audit, security audit, code coverage, ...). This feature MAY or MAY NOT be separated in the future. You can experiment and help to evolve it.

- [ ] Battletest simple agents setup
- [ ] Review Github actions

## Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Security

To report a vulnerability, see [SECURITY.md](SECURITY.md). Do not open public issues for security problems.

## Code of Conduct

This project follows the [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you agree to uphold it.

## License

MIT License — see [LICENSE](LICENSE).
