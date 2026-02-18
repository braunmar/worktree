# Worktree Manager

A Go CLI tool for managing git worktrees in multi-instance development environments.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

> **Disclaimer:** This software is provided "as is", without warranty of any kind. See [LICENSE](LICENSE) for full terms.

## Features

- **Coordinated Worktrees**: Creates backend + frontend worktrees together
- **Instance Integration**: Works seamlessly with multi-instance Docker setup
- **Automatic Fixtures**: Runs database fixtures after backend starts
- **Frontend Docker**: Runs frontend in Docker with dynamic environment variables
- **Safety Checks**: Warns about uncommitted changes and running instances
- **Status Tracking**: Shows running instances and git status
- **Dynamic Port Allocation**: Automatically finds and allocates available ports per feature
- **Registry System**: Tracks feature worktrees and their allocated resources

## Architecture

```
tools/worktree-manager/
├── cmd/                    # CLI commands
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

## Usage

### Quick Reference

```bash
# Create new feature environment
worktree new-feature feature/my-feature

# Start existing feature
worktree start feature-my-feature

# Start without post-startup commands
worktree start feature-my-feature --no-fixtures

# List all features
worktree list

# Show port allocations
worktree ports

# Check environment health
worktree doctor

# Remove feature
worktree remove feature/my-feature
```

## Dependencies

- **cobra**: CLI framework
- **color**: Colored terminal output
- **Go 1.21+**: Required Go version

## Port Expression Syntax

Port values in `.worktree.yml` support dynamic calculation using `{instance}` placeholder:

### Supported Formats

- **Simple offset**: `"3000 + {instance}"` → 3000, 3001, 3002, ...
- **Multiplication**: `"4510 + {instance} * 50"` → 4510, 4560, 4610, ...
- **Static value**: `"8080"` → always 8080

### Examples

```yaml
ports:
  FE_PORT:
    name: "Frontend"
    url: "http://{host}:{port}"
    port: "3000 + {instance}"
    # Instance 0 → 3000, Instance 1 → 3001, Instance 2 → 3002

  LOCALSTACK_EXT_START:
    name: "LocalStack Start"
    url: ""
    port: "4510 + {instance} * 50"
    # Instance 0 → 4510, Instance 1 → 4560, Instance 2 → 4610
```

### Constraints

- Results must be in valid port range (1-65535)
- Only integers supported
- Instance number derived from APP_PORT allocation
- Expression evaluation happens at feature creation time

## Configuration

The tool automatically discovers:
- Project root (by finding `.worktree.yml` configuration)
- Available ports (dynamically allocated from configured ranges)
- Worktree paths (`worktrees/<feature-name>/<project>`)

## Fixture Support

When you run `worktree start <instance>`, it automatically:
1. Starts backend services
2. Waits for services to be ready
3. Runs `make dev-fixture INSTANCE=N` to load seed data
4. Shows frontend start command

Skip fixtures with `--no-fixtures`:
```bash
worktree start 1 --no-fixtures
```

## Frontend Docker

Frontend runs in Docker with dynamic environment variables:
- `REACT_APP_API_BASE_URL` is set based on instance number
- Container connects to backend's Docker network
- Hot reload works via polling

Start with:
```bash
cd worktrees/1/frontend
make up-docker INSTANCE=1
```

## Development

### Adding a New Command

1. Create `cmd/<command>.go`
2. Implement cobra command structure
3. Register in `cmd/root.go` init()
4. Rebuild with `make build`

### Adding a New Package

1. Create `pkg/<package>/<file>.go`
2. Export functions with capital letters
3. Import in commands as needed

### Testing

```bash
# Run tests
make test

# Test specific package
go test ./pkg/config -v
```

## Command Reference

### new-feature
Create a new feature worktree with automatic port allocation.

```bash
worktree new-feature <branch> [preset]

Arguments:
  branch               Branch name (e.g., feature/user-auth)
  preset               Optional preset name (defaults to default_preset from config)

Flags:
  --no-fixtures        Skip post-startup commands
```

### start
Start services for an existing feature.

```bash
worktree start <feature-name>

Flags:
  --preset string      Override preset (defaults to preset used at creation)
  --no-fixtures        Skip post-startup commands
```

### stop
Stop services for a feature.

```bash
worktree stop <feature-name>
```

### stopall
Stop all running features.

```bash
worktree stopall
```

### remove
Remove a feature worktree.

```bash
worktree remove <feature-name-or-branch>

Flags:
  -f, --force          Skip confirmation prompts
```

### list
List all feature worktrees and their status.

```bash
worktree list
```

### status
Show detailed status for a specific feature.

```bash
worktree status <feature-name>
```

### logs
Show logs for a feature's services.

```bash
worktree logs <feature-name> [project-name]

Arguments:
  feature-name         Feature to show logs for
  project-name         Optional project (defaults to first project in preset)
```

### rebase
Update main branch and rebase feature on top of it.

```bash
worktree rebase <feature-name>
```

### doctor
Check worktree environment health and identify issues.

```bash
worktree doctor

Flags:
  --feature string     Check specific feature only
  --no-fetch          Skip git fetch (faster)
  --fix               Auto-fix safe issues
  --json              JSON output for scripting
```

### ports
Show port allocation for all features.

```bash
worktree ports
```

## Troubleshooting

### Port Issues

**"No available ports in range X-Y for service Z"**
- Check what features are using ports: `worktree list`
- Expand range in `.worktree.yml`: Update the `port` expression or allocation strategy
- Remove unused features: `worktree remove <feature-name>`
- Check for port conflicts with other applications

**"Port already allocated"**
- Find which feature uses it: `worktree list` or `worktree ports`
- Stop that feature: `worktree stop <feature-name>`
- Or adjust port ranges in `.worktree.yml` to avoid conflicts

**LocalStack port range errors**
- Ensure LOCALSTACK_EXT_START uses expressions: `"4510 + {instance} * 50"`
- Verify range is sufficient (typically 50 ports needed)
- Check instance calculation: `worktree ports` shows allocated ports
- Make sure END port = START port + range size

### Git Worktree Issues

**"Worktree already exists"**
- Remove existing: `git worktree remove worktrees/<feature-name>`
- Or use: `worktree remove <feature-name> --force`
- Check for leftover directories: `ls worktrees/`

**"Branch already exists"**
- Use different branch name
- Or delete remote branch first: `git push origin --delete <branch-name>`
- Or checkout and delete local branch: `git branch -D <branch-name>`

**"Uncommitted changes" warnings**
- Commit or stash changes before creating new worktrees
- Use `git status` to see what's uncommitted
- Use `worktree doctor` to check all worktrees

### Docker Issues

**Services not starting**
- Check Docker is running: `docker ps`
- Verify network exists: `docker network ls`
- Check logs: `worktree logs <feature-name>`
- Ensure no port conflicts with host system

**"Container already exists"**
- Stop existing containers: `worktree stop <feature-name>`
- Or stop all: `worktree stopall`
- Check for orphaned containers: `docker ps -a | grep <project-name>`

### Configuration Issues

**"Could not find project root"**
- Make sure you're running from within your project directory
- Verify `.worktree.yml` exists in the project root
- Check file permissions: `ls -la .worktree.yml`

**"Failed to run fixtures"**
- Check that post_command is defined for project in `.worktree.yml`
- Verify the command works manually in the project directory
- Use `--no-fixtures` flag to skip if not needed

**"Invalid port expression"**
- Port expressions must be valid arithmetic: `"base + {instance}"` or `"base + {instance} * multiplier"`
- Results must be within 1-65535 range
- Check `.worktree.yml` syntax

### Registry Issues

**"Feature not found in registry"**
- List available features: `worktree list`
- Feature name is normalized: `feature/user-auth` becomes `feature-user-auth`
- Use either normalized or original branch name

**Orphaned registry entries**
- Run diagnostics: `worktree doctor`
- Auto-fix: `worktree doctor --fix`
- Manual cleanup: Remove from `worktrees/.registry.json`

### General Troubleshooting

**Use the doctor command**
```bash
worktree doctor              # Check all issues
worktree doctor --fix        # Auto-fix safe issues
worktree doctor --feature X  # Check specific feature
```

**Check system status**
```bash
worktree list                # Show all features and status
worktree ports               # Show port allocations
worktree status <feature>    # Detailed feature status
```

---

## Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Security

To report a vulnerability, see [SECURITY.md](SECURITY.md). Do not open public issues for security problems.

## Code of Conduct

This project follows the [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you agree to uphold it.

## License

MIT License — see [LICENSE](LICENSE).

This software is provided **"as is"**, without warranty of any kind, express or implied.
