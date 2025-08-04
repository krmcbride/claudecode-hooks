# Development workflow makefile
# This file contains all development-related targets: testing, code quality, and dev workflows

##@ Test

# Test output format (can be overridden with GOTESTSUM_FORMAT env var)
GOTESTSUM_FORMAT ?= testdox

.PHONY: test
test: tools ## Run tests
	@printf "$(YELLOW)Running tests...$(NC)\n"
	@$(GOTESTSUM) --format $(GOTESTSUM_FORMAT) -- -v ./...
	@printf "$(GREEN)✓ Tests passed$(NC)\n"

.PHONY: test-cover
test-cover: tools ## Run tests with coverage
	@printf "$(YELLOW)Running tests with coverage...$(NC)\n"
	@$(GOTESTSUM) --format $(GOTESTSUM_FORMAT) -- -race -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -html=coverage.out -o coverage.html
	@printf "$(GREEN)✓ Coverage report generated: coverage.html$(NC)\n"

##@ Code Quality

.PHONY: lint
lint: tools ## Run linter
	@printf "$(YELLOW)Running linter...$(NC)\n"
	@$(GOLANGCI_LINT) run
	@printf "$(GREEN)✓ Linting passed$(NC)\n"

.PHONY: fmt
fmt: tools ## Format code
	@printf "$(YELLOW)Formatting code...$(NC)\n"
	@$(GOIMPORTS_REVISER) -rm-unused -set-alias -format ./...
	@$(GOLANGCI_LINT) run --fix
	@printf "$(GREEN)✓ Code formatted$(NC)\n"

.PHONY: fmt-file
fmt-file: tools ## Format a specific file (usage: make fmt-file FILE=path/to/file.go)
	@if [ -z "$(FILE)" ]; then \
		printf "$(RED)Error: FILE parameter is required$(NC)\n"; \
		printf "Usage: make fmt-file FILE=path/to/file.go\n"; \
		exit 1; \
	fi
	@printf "$(YELLOW)Formatting $(FILE)...$(NC)\n"
	@$(GOIMPORTS_REVISER) -rm-unused -set-alias -format "$(FILE)"
	@$(GOLANGCI_LINT) run --fix "$(FILE)"
	@printf "$(GREEN)✓ Formatted $(FILE)$(NC)\n"

.PHONY: check
check: fmt lint test ## Run all checks (format, lint, test)
	@printf "$(GREEN)✓ All checks passed$(NC)\n"

##@ Go Modules

.PHONY: mod-tidy
mod-tidy: ## Tidy go modules
	@printf "$(YELLOW)Tidying modules...$(NC)\n"
	@go mod tidy
	@printf "$(GREEN)✓ Modules tidied$(NC)\n"

.PHONY: mod-verify
mod-verify: ## Verify go modules
	@printf "$(YELLOW)Verifying modules...$(NC)\n"
	@go mod verify
	@printf "$(GREEN)✓ Modules verified$(NC)\n"

##@ Cleanup

.PHONY: clean
clean: ## Clean build artifacts
	@printf "$(YELLOW)Cleaning build artifacts...$(NC)\n"
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@printf "$(GREEN)✓ Cleaned$(NC)\n"

.PHONY: clean-all
clean-all: clean clean-tools ## Clean everything (build artifacts and tools)
	@printf "$(GREEN)✓ Everything cleaned$(NC)\n"

##@ Development

.PHONY: dev
dev: tools check build ## Full development workflow (tools, check, build)
	@printf "$(GREEN)✓ Development workflow complete$(NC)\n"
