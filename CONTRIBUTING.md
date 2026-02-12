# Contributing to Worktree Manager

Thank you for your interest in contributing! This document outlines how to get started.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/worktree`
3. Create a feature branch: `git checkout -b feature/my-change`
4. Make your changes
5. Run tests: `make test`
6. Submit a pull request

## Pull Request Guidelines

- Keep PRs focused — one change per PR
- Write tests for new functionality
- Run `make test` and `make vet` before submitting
- Describe what the PR does and why in the description
- Reference any related issues

## Reporting Bugs

Use [GitHub Issues](https://github.com/braunmar/worktree/issues) with the **bug** label.

Include:
- OS and Go version
- Steps to reproduce
- Expected vs actual behavior
- Relevant `.worktree.yml` snippet (sanitized)

## Requesting Features

Open a [GitHub Discussion](https://github.com/braunmar/worktree/discussions) or issue with the **enhancement** label.

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Keep functions small and focused
- Prefer clear naming over comments

## Architecture

```
.
├── cmd/                   # CLI commands
│   ├── root.go            # Root command setup
│   ├── create.go          # Create worktrees
│   ├── remove.go          # Remove worktrees
│   ├── start.go           # Start instance with fixtures
│   └── list.go            # List worktrees
├── pkg/                    # Shared packages
│   ├── config/            # Configuration and port calculations
│   ├── git/               # Git worktree operations
│   ├── docker/            # Docker instance checks
│   └── ui/                # Colored terminal output
├── main.go                # Entry point
├── Makefile               # Build system
└── go.mod                 # Dependencies
```

## Building

```bash
make build          # Build binary → ./worktree
make install        # Install via go install → ~/go/bin/worktree
make install-user   # Install to ~/.local/bin/worktree (no sudo)
make install-global # Install to /usr/local/bin/worktree (requires sudo)
make uninstall      # Remove from all install locations
make clean          # Remove local ./worktree binary
make test           # Run tests
make fmt            # Format code
make vet            # Vet code
```

## License

By contributing, you agree your contributions are licensed under the [MIT License](LICENSE).
