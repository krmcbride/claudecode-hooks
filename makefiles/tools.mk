# Tool management makefile
# This file contains all tool-related targets and configurations

## Tool binaries location
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	@mkdir -p $(LOCALBIN)

## Define the go-install-tool function
define go-install-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)@$(3)" ;\
GOBIN=$(LOCALBIN) go install $(2)@$(3) ;\
rm -rf $$TMP_DIR ;\
}
endef

## Tool Binaries
GOIMPORTS_REVISER = $(LOCALBIN)/goimports-reviser
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint
GOTESTSUM = $(LOCALBIN)/gotestsum

## Tool Versions
GOIMPORTS_REVISER_VERSION = v3.8.2
GOLANGCI_LINT_VERSION = v2.3.1
GOTESTSUM_VERSION = v1.12.3

##@ Tools

.PHONY: tools
tools: $(GOIMPORTS_REVISER) $(GOLANGCI_LINT) $(GOTESTSUM) ## Install all development tools

.PHONY: goimports-reviser
goimports-reviser: $(GOIMPORTS_REVISER) ## Install goimports-reviser
$(GOIMPORTS_REVISER): $(LOCALBIN)
	$(call go-install-tool,$(GOIMPORTS_REVISER),github.com/incu6us/goimports-reviser/v3,$(GOIMPORTS_REVISER_VERSION))

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Install golangci-lint
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

.PHONY: gotestsum
gotestsum: $(GOTESTSUM) ## Install gotestsum
$(GOTESTSUM): $(LOCALBIN)
	$(call go-install-tool,$(GOTESTSUM),gotest.tools/gotestsum,$(GOTESTSUM_VERSION))

.PHONY: clean-tools
clean-tools: ## Remove all installed tools
	@printf "$(YELLOW)Removing tools...$(NC)\n"
	@rm -rf $(LOCALBIN)
	@printf "$(GREEN)✓ Tools removed$(NC)\n"
