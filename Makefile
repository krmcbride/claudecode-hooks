# Go Claude Code Hook Project Makefile

# Project configuration
BIN_DIR := bin
BUILD_DIR := build
HOOK_DIR := .claude/hooks
USER_HOOK_DIR := $(or $(CLAUDE_CONFIG_DIR),$(HOME)/.claude)/hooks

# Hook definitions (name:package)
HOOKS := git-block:cmd/git-block aws-block:cmd/aws-block kubectl-block:cmd/kubectl-block
HOOK_BINARIES := $(foreach hook,$(HOOKS),krmcbride-$(word 1,$(subst :, ,$(hook))))

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
build: ## Build all hook binaries
	@printf "$(YELLOW)Building all hooks...$(NC)\n"
	@mkdir -p $(BUILD_DIR)
	@$(foreach hook,$(HOOKS), \
		printf "$(YELLOW)Building krmcbride-$(word 1,$(subst :, ,$(hook)))...$(NC)\n"; \
		go build -ldflags="-w -s" -o $(BUILD_DIR)/krmcbride-$(word 1,$(subst :, ,$(hook))) ./$(word 2,$(subst :, ,$(hook))) || exit 1; \
		printf "$(GREEN)✓ Built $(BUILD_DIR)/krmcbride-$(word 1,$(subst :, ,$(hook)))$(NC)\n"; \
	)
	@printf "$(GREEN)✓ All hooks built$(NC)\n"

# Individual hook build targets
.PHONY: build-git-block
build-git-block: ## Build git-block hook
	@printf "$(YELLOW)Building krmcbride-git-block...$(NC)\n"
	@mkdir -p $(BUILD_DIR)
	@go build -ldflags="-w -s" -o $(BUILD_DIR)/krmcbride-git-block ./cmd/git-block
	@printf "$(GREEN)✓ Built $(BUILD_DIR)/krmcbride-git-block$(NC)\n"

.PHONY: build-aws-block
build-aws-block: ## Build aws-block hook
	@printf "$(YELLOW)Building krmcbride-aws-block...$(NC)\n"
	@mkdir -p $(BUILD_DIR)
	@go build -ldflags="-w -s" -o $(BUILD_DIR)/krmcbride-aws-block ./cmd/aws-block
	@printf "$(GREEN)✓ Built $(BUILD_DIR)/krmcbride-aws-block$(NC)\n"

.PHONY: build-kubectl-block
build-kubectl-block: ## Build kubectl-block hook
	@printf "$(YELLOW)Building krmcbride-kubectl-block...$(NC)\n"
	@mkdir -p $(BUILD_DIR)
	@go build -ldflags="-w -s" -o $(BUILD_DIR)/krmcbride-kubectl-block ./cmd/kubectl-block
	@printf "$(GREEN)✓ Built $(BUILD_DIR)/krmcbride-kubectl-block$(NC)\n"

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
install: build ## Install all hook binaries to project .claude/hooks/ directory
	@printf "$(YELLOW)Installing all hooks to project...$(NC)\n"
	@mkdir -p $(HOOK_DIR)
	@$(foreach hook,$(HOOKS), \
		cp $(BUILD_DIR)/krmcbride-$(word 1,$(subst :, ,$(hook))) $(HOOK_DIR)/krmcbride-$(word 1,$(subst :, ,$(hook))); \
		chmod +x $(HOOK_DIR)/krmcbride-$(word 1,$(subst :, ,$(hook))); \
		printf "$(GREEN)✓ Installed krmcbride-$(word 1,$(subst :, ,$(hook))) to $(HOOK_DIR)/$(NC)\n"; \
	)

.PHONY: install-user
install-user: build ## Install all hook binaries to user ~/.claude/hooks/ directory
	@printf "$(YELLOW)Installing all hooks to user config...$(NC)\n"
	@mkdir -p $(USER_HOOK_DIR)
	@$(foreach hook,$(HOOKS), \
		cp $(BUILD_DIR)/krmcbride-$(word 1,$(subst :, ,$(hook))) $(USER_HOOK_DIR)/krmcbride-$(word 1,$(subst :, ,$(hook))); \
		chmod +x $(USER_HOOK_DIR)/krmcbride-$(word 1,$(subst :, ,$(hook))); \
		printf "$(GREEN)✓ Installed krmcbride-$(word 1,$(subst :, ,$(hook))) to $(USER_HOOK_DIR)/$(NC)\n"; \
	)

# Individual hook install targets
.PHONY: install-git-block
install-git-block: build-git-block ## Install git-block hook to project
	@printf "$(YELLOW)Installing krmcbride-git-block to project...$(NC)\n"
	@mkdir -p $(HOOK_DIR)
	@cp $(BUILD_DIR)/krmcbride-git-block $(HOOK_DIR)/krmcbride-git-block
	@chmod +x $(HOOK_DIR)/krmcbride-git-block
	@printf "$(GREEN)✓ Installed krmcbride-git-block to $(HOOK_DIR)/$(NC)\n"

.PHONY: install-user-git-block
install-user-git-block: build-git-block ## Install git-block hook to user config
	@printf "$(YELLOW)Installing krmcbride-git-block to user config...$(NC)\n"
	@mkdir -p $(USER_HOOK_DIR)
	@cp $(BUILD_DIR)/krmcbride-git-block $(USER_HOOK_DIR)/krmcbride-git-block
	@chmod +x $(USER_HOOK_DIR)/krmcbride-git-block
	@printf "$(GREEN)✓ Installed krmcbride-git-block to $(USER_HOOK_DIR)/$(NC)\n"

.PHONY: install-aws-block
install-aws-block: build-aws-block ## Install aws-block hook to project
	@printf "$(YELLOW)Installing krmcbride-aws-block to project...$(NC)\n"
	@mkdir -p $(HOOK_DIR)
	@cp $(BUILD_DIR)/krmcbride-aws-block $(HOOK_DIR)/krmcbride-aws-block
	@chmod +x $(HOOK_DIR)/krmcbride-aws-block
	@printf "$(GREEN)✓ Installed krmcbride-aws-block to $(HOOK_DIR)/$(NC)\n"

.PHONY: install-user-aws-block
install-user-aws-block: build-aws-block ## Install aws-block hook to user config
	@printf "$(YELLOW)Installing krmcbride-aws-block to user config...$(NC)\n"
	@mkdir -p $(USER_HOOK_DIR)
	@cp $(BUILD_DIR)/krmcbride-aws-block $(USER_HOOK_DIR)/krmcbride-aws-block
	@chmod +x $(USER_HOOK_DIR)/krmcbride-aws-block
	@printf "$(GREEN)✓ Installed krmcbride-aws-block to $(USER_HOOK_DIR)/$(NC)\n"

.PHONY: install-kubectl-block
install-kubectl-block: build-kubectl-block ## Install kubectl-block hook to project
	@printf "$(YELLOW)Installing krmcbride-kubectl-block to project...$(NC)\n"
	@mkdir -p $(HOOK_DIR)
	@cp $(BUILD_DIR)/krmcbride-kubectl-block $(HOOK_DIR)/krmcbride-kubectl-block
	@chmod +x $(HOOK_DIR)/krmcbride-kubectl-block
	@printf "$(GREEN)✓ Installed krmcbride-kubectl-block to $(HOOK_DIR)/$(NC)\n"

.PHONY: install-user-kubectl-block
install-user-kubectl-block: build-kubectl-block ## Install kubectl-block hook to user config
	@printf "$(YELLOW)Installing krmcbride-kubectl-block to user config...$(NC)\n"
	@mkdir -p $(USER_HOOK_DIR)
	@cp $(BUILD_DIR)/krmcbride-kubectl-block $(USER_HOOK_DIR)/krmcbride-kubectl-block
	@chmod +x $(USER_HOOK_DIR)/krmcbride-kubectl-block
	@printf "$(GREEN)✓ Installed krmcbride-kubectl-block to $(USER_HOOK_DIR)/$(NC)\n"

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
uninstall: ## Remove all installed project hooks
	@printf "$(YELLOW)Removing all project hooks...$(NC)\n"
	@$(foreach hook,$(HOOKS), \
		rm -f $(HOOK_DIR)/krmcbride-$(word 1,$(subst :, ,$(hook))); \
		printf "$(GREEN)✓ Removed krmcbride-$(word 1,$(subst :, ,$(hook))) from $(HOOK_DIR)/$(NC)\n"; \
	)

.PHONY: uninstall-user
uninstall-user: ## Remove all installed user hooks
	@printf "$(YELLOW)Removing all user hooks...$(NC)\n"
	@$(foreach hook,$(HOOKS), \
		rm -f $(USER_HOOK_DIR)/krmcbride-$(word 1,$(subst :, ,$(hook))); \
		printf "$(GREEN)✓ Removed krmcbride-$(word 1,$(subst :, ,$(hook))) from $(USER_HOOK_DIR)/$(NC)\n"; \
	)

.PHONY: uninstall-all
uninstall-all: uninstall uninstall-user ## Remove all installed hooks (project and user)
	@printf "$(GREEN)✓ All hooks removed$(NC)\n"

# Individual hook uninstall targets
.PHONY: uninstall-git-block
uninstall-git-block: ## Remove git-block hook from project
	@printf "$(YELLOW)Removing krmcbride-git-block from project...$(NC)\n"
	@rm -f $(HOOK_DIR)/krmcbride-git-block
	@printf "$(GREEN)✓ Removed krmcbride-git-block$(NC)\n"

.PHONY: uninstall-user-git-block
uninstall-user-git-block: ## Remove git-block hook from user config
	@printf "$(YELLOW)Removing krmcbride-git-block from user config...$(NC)\n"
	@rm -f $(USER_HOOK_DIR)/krmcbride-git-block
	@printf "$(GREEN)✓ Removed krmcbride-git-block$(NC)\n"

.PHONY: uninstall-aws-block
uninstall-aws-block: ## Remove aws-block hook from project
	@printf "$(YELLOW)Removing krmcbride-aws-block from project...$(NC)\n"
	@rm -f $(HOOK_DIR)/krmcbride-aws-block
	@printf "$(GREEN)✓ Removed krmcbride-aws-block$(NC)\n"

.PHONY: uninstall-user-aws-block
uninstall-user-aws-block: ## Remove aws-block hook from user config
	@printf "$(YELLOW)Removing krmcbride-aws-block from user config...$(NC)\n"
	@rm -f $(USER_HOOK_DIR)/krmcbride-aws-block
	@printf "$(GREEN)✓ Removed krmcbride-aws-block$(NC)\n"

.PHONY: uninstall-kubectl-block
uninstall-kubectl-block: ## Remove kubectl-block hook from project
	@printf "$(YELLOW)Removing krmcbride-kubectl-block from project...$(NC)\n"
	@rm -f $(HOOK_DIR)/krmcbride-kubectl-block
	@printf "$(GREEN)✓ Removed krmcbride-kubectl-block$(NC)\n"

.PHONY: uninstall-user-kubectl-block
uninstall-user-kubectl-block: ## Remove kubectl-block hook from user config
	@printf "$(YELLOW)Removing krmcbride-kubectl-block from user config...$(NC)\n"
	@rm -f $(USER_HOOK_DIR)/krmcbride-kubectl-block
	@printf "$(GREEN)✓ Removed krmcbride-kubectl-block$(NC)\n"

.PHONY: clean-all
clean-all: clean clean-tools ## Clean everything (build artifacts and tools)
	@printf "$(GREEN)✓ Everything cleaned$(NC)\n"

.PHONY: dev
dev: tools check build ## Full development workflow (tools, check, build)
	@printf "$(GREEN)✓ Development workflow complete$(NC)\n"

# Default target
.DEFAULT_GOAL := help
