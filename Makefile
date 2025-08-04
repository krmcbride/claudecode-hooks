# Go Claude Code Hook Project Makefile

# Project configuration
BINARY_NAME := krmcbride-claude-hook
MAIN_PACKAGE := .
BIN_DIR := bin
BUILD_DIR := build
HOOK_DIR := .claude/hooks
USER_HOOK_DIR := $(or $(CLAUDE_CONFIG_DIR),$(HOME)/.claude)/hooks

# Tool versions (pinned)
GOLANGCI_LINT_VERSION := v2.3.1
GOIMPORTS_REVISER_VERSION := v3.8.2

# Go configuration
export CGO_ENABLED=0
export GOOS=$(shell go env GOOS)
export GOARCH=$(shell go env GOARCH)

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[0;33m
NC := \033[0m # No Color

# Helper function for colored output
define print_color
	@printf "$(1)%s$(NC)\n" "$(2)"
endef

# Function to install a Go tool with pinned version
# Usage: $(call install-tool,tool-name,module@version,binary-name)
define install-tool
	@printf "$(YELLOW)Installing $(1) $(2)...$(NC)\n"
	@mkdir -p $(BIN_DIR)
	@if [ ! -f "$(BIN_DIR)/$(3)" ] || [ "$$($(BIN_DIR)/$(3) --version 2>/dev/null | grep -o '$(2)' || echo '')" != "$(2)" ]; then \
		GOBIN=$$(pwd)/$(BIN_DIR) go install $(2); \
		printf "$(GREEN)✓ $(1) installed$(NC)\n"; \
	else \
		printf "$(GREEN)✓ $(1) already installed$(NC)\n"; \
	fi
endef

.PHONY: help
help: ## Show this help message
	@printf "$(GREEN)Available targets:$(NC)\n"
	@awk 'BEGIN {FS = ":.*?## "; YELLOW="\033[0;33m"; NC="\033[0m"} /^[a-zA-Z_-]+:.*?## / {printf "  " YELLOW "%-15s" NC " %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: tools
tools: ## Install all development tools
	$(call install-tool,golangci-lint,github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION),golangci-lint)
	$(call install-tool,goimports-reviser,github.com/incu6us/goimports-reviser/v3@$(GOIMPORTS_REVISER_VERSION),goimports-reviser)

.PHONY: clean-tools
clean-tools: ## Remove all installed tools
	@printf "$(YELLOW)Removing tools...$(NC)\n"
	@rm -rf $(BIN_DIR)
	@printf "$(GREEN)✓ Tools removed$(NC)\n"

.PHONY: build
build: ## Build the binary
	@printf "$(YELLOW)Building $(BINARY_NAME)...$(NC)\n"
	@mkdir -p $(BUILD_DIR)
	@go build -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@printf "$(GREEN)✓ Built $(BUILD_DIR)/$(BINARY_NAME)$(NC)\n"

.PHONY: test
test: ## Run tests
	@printf "$(YELLOW)Running tests...$(NC)\n"
	@go test -v ./...
	@printf "$(GREEN)✓ Tests passed$(NC)\n"

.PHONY: test-cover
test-cover: ## Run tests with coverage
	@printf "$(YELLOW)Running tests with coverage...$(NC)\n"
	@go test -race -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -html=coverage.out -o coverage.html
	@printf "$(GREEN)✓ Coverage report generated: coverage.html$(NC)\n"

.PHONY: lint
lint: tools ## Run linter
	@printf "$(YELLOW)Running linter...$(NC)\n"
	@$(BIN_DIR)/golangci-lint run
	@printf "$(GREEN)✓ Linting passed$(NC)\n"

.PHONY: fmt
fmt: tools ## Format code
	@printf "$(YELLOW)Formatting code...$(NC)\n"
	@$(BIN_DIR)/golangci-lint run --fix
	@printf "$(GREEN)✓ Code formatted$(NC)\n"

.PHONY: check
check: fmt lint test ## Run all checks (format, lint, test)
	@printf "$(GREEN)✓ All checks passed$(NC)\n"

.PHONY: install
install: build ## Install the hook binary to project .claude/hooks/ directory
	@printf "$(YELLOW)Installing $(BINARY_NAME) hook to project...$(NC)\n"
	@mkdir -p $(HOOK_DIR)
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(HOOK_DIR)/$(BINARY_NAME)
	@chmod +x $(HOOK_DIR)/$(BINARY_NAME)
	@printf "$(GREEN)✓ Installed $(BINARY_NAME) to $(HOOK_DIR)/$(NC)\n"

.PHONY: install-user
install-user: build ## Install the hook binary to user ~/.claude/hooks/ directory
	@printf "$(YELLOW)Installing $(BINARY_NAME) hook to user config...$(NC)\n"
	@mkdir -p $(USER_HOOK_DIR)
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(USER_HOOK_DIR)/$(BINARY_NAME)
	@chmod +x $(USER_HOOK_DIR)/$(BINARY_NAME)
	@printf "$(GREEN)✓ Installed $(BINARY_NAME) to $(USER_HOOK_DIR)/$(NC)\n"

.PHONY: install-global
install-global: build ## Install the binary to $GOPATH/bin for global use
	@printf "$(YELLOW)Installing $(BINARY_NAME) globally...$(NC)\n"
	@go install .
	@printf "$(GREEN)✓ Installed $(BINARY_NAME) to $$(go env GOPATH)/bin$(NC)\n"

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

.PHONY: clean
clean: ## Clean build artifacts
	@printf "$(YELLOW)Cleaning build artifacts...$(NC)\n"
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@printf "$(GREEN)✓ Cleaned$(NC)\n"

.PHONY: uninstall
uninstall: ## Remove installed project hook
	@printf "$(YELLOW)Removing project hook...$(NC)\n"
	@rm -f $(HOOK_DIR)/$(BINARY_NAME)
	@printf "$(GREEN)✓ Project hook removed$(NC)\n"

.PHONY: uninstall-user
uninstall-user: ## Remove installed user hook
	@printf "$(YELLOW)Removing user hook...$(NC)\n"
	@rm -f $(USER_HOOK_DIR)/$(BINARY_NAME)
	@printf "$(GREEN)✓ User hook removed$(NC)\n"

.PHONY: uninstall-all
uninstall-all: uninstall uninstall-user ## Remove all installed hooks
	@printf "$(GREEN)✓ All hooks removed$(NC)\n"

.PHONY: clean-all
clean-all: clean clean-tools ## Clean everything (build artifacts and tools)
	@printf "$(GREEN)✓ Everything cleaned$(NC)\n"

.PHONY: dev
dev: tools check build ## Full development workflow (tools, check, build)
	@printf "$(GREEN)✓ Development workflow complete$(NC)\n"

# Default target
.DEFAULT_GOAL := help