# Go Claude Code Hook Project Makefile

# Include modular makefiles
include makefiles/tools.mk
include makefiles/build.mk
include makefiles/dev.mk

# Project configuration
BUILD_DIR := build
HOOK_DIR := .claude/hooks
USER_HOOK_DIR := $(or $(CLAUDE_CONFIG_DIR),$(HOME)/.claude)/hooks

# Binary prefix for hook executables (overridable)
HOOK_PREFIX ?= krmcbride-

# Hook management is handled by the HOOKS variable in makefiles/build.mk

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

##@ Help

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

# Default target
.DEFAULT_GOAL := help
