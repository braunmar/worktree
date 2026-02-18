# Contributing to Worktree Manager

Thank you for your interest in contributing! This document outlines how to get started.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/worktree`
3. Create a feature branch: `git checkout -b feature/my-change`
4. Make your changes
5. Run tests: `make test`
6. Submit a pull request

## Development Setup

```bash
# Build
make build

# Run tests
make test

# Format code
make fmt

# Vet code
make vet
```

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

## License

By contributing, you agree your contributions are licensed under the [MIT License](LICENSE).
