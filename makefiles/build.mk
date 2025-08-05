# Build and installation management makefile
# This file contains all build and installation-related targets

# Hook definitions (shared by build, install, and uninstall targets)
HOOKS := bash-block:cmd/bash-block file-format:cmd/file-format

##@ Build

.PHONY: build
build: ## Build all hook binaries
	@printf "$(YELLOW)Building all hooks...$(NC)\n"
	@mkdir -p $(BUILD_DIR)
	@$(foreach hook,$(HOOKS), \
		printf "$(YELLOW)Building $(HOOK_PREFIX)$(word 1,$(subst :, ,$(hook)))...$(NC)\n"; \
		go build -ldflags="-w -s" -o $(BUILD_DIR)/$(HOOK_PREFIX)$(word 1,$(subst :, ,$(hook))) ./$(word 2,$(subst :, ,$(hook))) || exit 1; \
		printf "$(GREEN)✓ Built $(BUILD_DIR)/$(HOOK_PREFIX)$(word 1,$(subst :, ,$(hook)))$(NC)\n"; \
	)
	@printf "$(GREEN)✓ All hooks built$(NC)\n"

# Template for individual hook build targets
define hook-build-template
.PHONY: build-$(1)
build-$(1): ## Build $(1) hook
	@printf "$$(YELLOW)Building $$(HOOK_PREFIX)$(1)...$$(NC)\n"
	@mkdir -p $$(BUILD_DIR)
	@go build -ldflags="-w -s" -o $$(BUILD_DIR)/$$(HOOK_PREFIX)$(1) ./$(2)
	@printf "$$(GREEN)✓ Built $$(BUILD_DIR)/$$(HOOK_PREFIX)$(1)$$(NC)\n"
endef

# Generate individual hook build targets
# NOTE: When adding a new hook, add it to HOOKS above AND add an eval line below
$(eval $(call hook-build-template,bash-block,cmd/bash-block))
$(eval $(call hook-build-template,file-format,cmd/file-format))

##@ Installation

.PHONY: install
install: build ## Install all hook binaries to project .claude/hooks/ directory
	@printf "$(YELLOW)Installing all hooks to project...$(NC)\n"
	@mkdir -p $(HOOK_DIR)
	@$(foreach hook,$(HOOKS), \
		cp $(BUILD_DIR)/$(HOOK_PREFIX)$(word 1,$(subst :, ,$(hook))) $(HOOK_DIR)/$(HOOK_PREFIX)$(word 1,$(subst :, ,$(hook))); \
		chmod +x $(HOOK_DIR)/$(HOOK_PREFIX)$(word 1,$(subst :, ,$(hook))); \
		printf "$(GREEN)✓ Installed $(HOOK_PREFIX)$(word 1,$(subst :, ,$(hook))) to $(HOOK_DIR)/$(NC)\n"; \
	)

.PHONY: install-user
install-user: build ## Install all hook binaries to user ~/.claude/hooks/ directory
	@printf "$(YELLOW)Installing all hooks to user config...$(NC)\n"
	@mkdir -p $(USER_HOOK_DIR)
	@$(foreach hook,$(HOOKS), \
		cp $(BUILD_DIR)/$(HOOK_PREFIX)$(word 1,$(subst :, ,$(hook))) $(USER_HOOK_DIR)/$(HOOK_PREFIX)$(word 1,$(subst :, ,$(hook))); \
		chmod +x $(USER_HOOK_DIR)/$(HOOK_PREFIX)$(word 1,$(subst :, ,$(hook))); \
		printf "$(GREEN)✓ Installed $(HOOK_PREFIX)$(word 1,$(subst :, ,$(hook))) to $(USER_HOOK_DIR)/$(NC)\n"; \
	)

# Template for individual hook install targets
define hook-install-template
.PHONY: install-$(1)
install-$(1): build-$(1) ## Install $(1) hook to project
	@printf "$$(YELLOW)Installing $$(HOOK_PREFIX)$(1) to project...$$(NC)\n"
	@mkdir -p $$(HOOK_DIR)
	@cp $$(BUILD_DIR)/$$(HOOK_PREFIX)$(1) $$(HOOK_DIR)/$$(HOOK_PREFIX)$(1)
	@chmod +x $$(HOOK_DIR)/$$(HOOK_PREFIX)$(1)
	@printf "$$(GREEN)✓ Installed $$(HOOK_PREFIX)$(1) to $$(HOOK_DIR)/$$(NC)\n"

.PHONY: install-user-$(1)
install-user-$(1): build-$(1) ## Install $(1) hook to user config
	@printf "$$(YELLOW)Installing $$(HOOK_PREFIX)$(1) to user config...$$(NC)\n"
	@mkdir -p $$(USER_HOOK_DIR)
	@cp $$(BUILD_DIR)/$$(HOOK_PREFIX)$(1) $$(USER_HOOK_DIR)/$$(HOOK_PREFIX)$(1)
	@chmod +x $$(USER_HOOK_DIR)/$$(HOOK_PREFIX)$(1)
	@printf "$$(GREEN)✓ Installed $$(HOOK_PREFIX)$(1) to $$(USER_HOOK_DIR)/$$(NC)\n"
endef

##@ Uninstallation

.PHONY: uninstall
uninstall: ## Remove all installed project hooks
	@printf "$(YELLOW)Removing all project hooks...$(NC)\n"
	@$(foreach hook,$(HOOKS), \
		rm -f $(HOOK_DIR)/$(HOOK_PREFIX)$(word 1,$(subst :, ,$(hook))); \
		printf "$(GREEN)✓ Removed $(HOOK_PREFIX)$(word 1,$(subst :, ,$(hook))) from $(HOOK_DIR)/$(NC)\n"; \
	)

.PHONY: uninstall-user
uninstall-user: ## Remove all installed user hooks
	@printf "$(YELLOW)Removing all user hooks...$(NC)\n"
	@$(foreach hook,$(HOOKS), \
		rm -f $(USER_HOOK_DIR)/$(HOOK_PREFIX)$(word 1,$(subst :, ,$(hook))); \
		printf "$(GREEN)✓ Removed $(HOOK_PREFIX)$(word 1,$(subst :, ,$(hook))) from $(USER_HOOK_DIR)/$(NC)\n"; \
	)

.PHONY: uninstall-all
uninstall-all: uninstall uninstall-user ## Remove all installed hooks (project and user)
	@printf "$(GREEN)✓ All hooks removed$(NC)\n"

# Template for individual hook uninstall targets
define hook-uninstall-template
.PHONY: uninstall-$(1)
uninstall-$(1): ## Remove $(1) hook from project
	@printf "$$(YELLOW)Removing $$(HOOK_PREFIX)$(1) from project...$$(NC)\n"
	@rm -f $$(HOOK_DIR)/$$(HOOK_PREFIX)$(1)
	@printf "$$(GREEN)✓ Removed $$(HOOK_PREFIX)$(1)$$(NC)\n"

.PHONY: uninstall-user-$(1)
uninstall-user-$(1): ## Remove $(1) hook from user config
	@printf "$$(YELLOW)Removing $$(HOOK_PREFIX)$(1) from user config...$$(NC)\n"
	@rm -f $$(USER_HOOK_DIR)/$$(HOOK_PREFIX)$(1)
	@printf "$$(GREEN)✓ Removed $$(HOOK_PREFIX)$(1)$$(NC)\n"
endef

# Generate individual hook install and uninstall targets
# NOTE: When adding a new hook, add it to HOOKS above AND add eval lines below
$(eval $(call hook-install-template,bash-block))
$(eval $(call hook-install-template,file-format))

$(eval $(call hook-uninstall-template,bash-block))
$(eval $(call hook-uninstall-template,file-format))
