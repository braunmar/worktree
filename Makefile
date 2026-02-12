.PHONY: build install install-global install-user uninstall clean test test-coverage test-system test-system-coverage test-all test-all-coverage help tidy fmt vet

# Default target
.DEFAULT_GOAL := help

build: ## Build the worktree binary
	@echo "🔨 Building worktree manager..."
	@go build -o ./worktree .
	@echo "✅ Binary built: worktree"

install: ## Install using go install (installs to $GOBIN or ~/go/bin)
	@echo "🔧 Installing worktree using go install..."
	@go install .
	@GOBIN=$${GOBIN:-$$(go env GOPATH)/bin}; \
	echo "✅ Installed to: $$GOBIN/worktree"; \
	echo ""; \
	if echo $$PATH | grep -q "$$GOBIN"; then \
		echo "You can now use: worktree <command>"; \
	else \
		echo "⚠️  $$GOBIN is not in your PATH"; \
		echo "Add this to your ~/.zshrc:"; \
		echo "  export PATH=\"\$$PATH:$$GOBIN\""; \
		echo "Then run: source ~/.zshrc"; \
	fi

install-global: ## Install binary to /usr/local/bin (requires sudo)
	@echo "🔧 Installing worktree to /usr/local/bin..."
	@sudo go build -o /usr/local/bin/worktree .
	@echo "✅ Installed: /usr/local/bin/worktree"
	@echo ""
	@echo "You can now use: worktree <command>"

install-user: ## Install binary to ~/.local/bin (no sudo required)
	@echo "🔧 Installing worktree to ~/.local/bin..."
	@mkdir -p ~/.local/bin
	@go build -o ~/.local/bin/worktree .
	@echo "✅ Installed: ~/.local/bin/worktree"
	@echo ""
	@if echo $$PATH | grep -q "$$HOME/.local/bin"; then \
		echo "You can now use: worktree <command>"; \
	else \
		echo "⚠️  ~/.local/bin is not in your PATH"; \
		echo "Add this to your ~/.bashrc or ~/.zshrc:"; \
		echo '  export PATH="$$HOME/.local/bin:$$PATH"'; \
		echo "Then run: source ~/.bashrc  (or ~/.zshrc)"; \
	fi

uninstall: ## Uninstall binary from system
	@echo "🗑️  Uninstalling worktree..."
	@GOBIN=$${GOBIN:-$$(go env GOPATH)/bin}; \
	if [ -f "$$GOBIN/worktree" ]; then \
		rm "$$GOBIN/worktree" && echo "✅ Removed: $$GOBIN/worktree"; \
	fi; \
	if [ -f "/usr/local/bin/worktree" ]; then \
		sudo rm /usr/local/bin/worktree && echo "✅ Removed: /usr/local/bin/worktree"; \
	fi; \
	if [ -f "$$HOME/.local/bin/worktree" ]; then \
		rm "$$HOME/.local/bin/worktree" && echo "✅ Removed: ~/.local/bin/worktree"; \
	fi; \
	echo "✅ Uninstall complete"

clean: ## Remove built binary
	@echo "🧹 Cleaning up..."
	@rm -f ./worktree
	@echo "✅ Cleaned"

test: ## Run unit tests (pkg/ only)
	@echo "🧪 Running unit tests..."
	@go test -v ./pkg/...

test-coverage: ## Run unit tests with coverage
	@echo "🧪 Running unit tests with coverage..."
	@go test -v -race -coverprofile=coverage-unit.out -covermode=atomic ./pkg/...
	@echo "📊 Coverage saved to: coverage-unit.out"
	@go tool cover -func=coverage-unit.out | tail -1

test-system: ## Run system/integration tests (builds binary, requires git)
	@echo "🧪 Running system tests..."
	@go test -v -timeout 120s ./test/system/...

test-system-coverage: ## Run system tests with coverage
	@echo "🧪 Running system tests with coverage..."
	@go test -v -race -timeout 120s -coverprofile=coverage-system.out -covermode=atomic ./test/system/...
	@echo "📊 Coverage saved to: coverage-system.out"
	@go tool cover -func=coverage-system.out | tail -1

test-all: test test-system ## Run all tests (unit + system, no coverage)

test-all-coverage: ## Run all tests with merged coverage (matches CI workflow)
	@echo "🧪 Running unit tests with coverage..."
	@go test -v -race -coverprofile=coverage-unit.out -covermode=atomic ./pkg/...
	@echo ""
	@echo "🧪 Running system tests with coverage..."
	@go test -v -race -timeout 120s -coverprofile=coverage-system.out -covermode=atomic ./test/system/...
	@echo ""
	@echo "📊 Merging coverage profiles..."
	@echo "mode: atomic" > coverage.out
	@tail -q -n +2 coverage-unit.out coverage-system.out >> coverage.out
	@echo "✅ Merged coverage saved to: coverage.out"
	@go tool cover -func=coverage.out | tail -1
	@echo ""
	@echo "💡 View HTML coverage: go tool cover -html=coverage.out"

tidy: ## Tidy go modules
	@echo "📦 Tidying go modules..."
	@go mod tidy
	@echo "✅ Done"

fmt: ## Format code
	@echo "🎨 Formatting code..."
	@go fmt ./...
	@echo "✅ Done"

vet: ## Vet code
	@echo "🔍 Vetting code..."
	@go vet ./...
	@echo "✅ Done"

help: ## Show this help
	@echo "Worktree Manager - Makefile targets:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'
	@echo ""
