# Worktree Manager

A Go CLI tool for managing git worktrees in multi-instance development environments.

## Features

- **Coordinated Worktrees**: Creates backend + frontend worktrees together
- **Instance Integration**: Works seamlessly with multi-instance Docker setup
- **Automatic Fixtures**: Runs database fixtures after backend starts
- **Frontend Docker**: Runs frontend in Docker with dynamic environment variables
- **Safety Checks**: Warns about uncommitted changes and running instances
- **Status Tracking**: Shows running instances and git status

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
make build          # Build binary to ../../scripts/worktree
make clean          # Remove binary
make test           # Run tests
make fmt            # Format code
make vet            # Vet code
```

## Usage

See [WORKTREE-WORKFLOW.md](../../WORKTREE-WORKFLOW.md) for complete documentation.

### Quick Reference

```bash
# Create worktrees
scripts/worktree create feature/my-feature 1

# Start backend with fixtures
scripts/worktree start 1

# Start backend without fixtures
scripts/worktree start 1 --no-fixtures

# List all worktrees
scripts/worktree list

# Remove worktrees
scripts/worktree remove 1
```

## Dependencies

- **cobra**: CLI framework
- **color**: Colored terminal output
- **Go 1.21+**: Required Go version

## Configuration

The tool automatically discovers:
- Project root (by finding `backend/` and `frontend/` directories)
- Instance ports (calculated from instance number)
- Worktree paths (`worktrees/N/backend`, `worktrees/N/frontend`)

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

## Troubleshooting

### "Could not find project root"

Make sure you're running from within your project directory and that a `.worktree.yml` configuration file exists in the project root.

### "Failed to run fixtures"

Check that `make dev-fixture` exists in the backend Makefile and works correctly:
```bash
cd backend
make dev-fixture INSTANCE=1
```

### Frontend not connecting to backend

Verify the Docker network exists and backend is running:
```bash
# Replace <project-name> with your project_name from .worktree.yml
docker network ls | grep <project-name>
docker ps | grep <project-name>
```
