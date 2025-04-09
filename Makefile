# --- CONFIG ---

LOCAL_BIN     := $(CURDIR)/bin

GOLANGCI_VERSION      := v1.64.5

RUN_ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
$(eval $(RUN_ARGS):;@:)

define install_tool
	GOBIN=$(LOCAL_BIN) go install $(1)@$(2)
endef

# --- INSTALL TOOLS ---

.PHONY: install
install:
	mkdir -p $(LOCAL_BIN)
	go mod tidy
	$(call install_tool,github.com/golangci/golangci-lint/cmd/golangci-lint,$(GOLANGCI_VERSION))

# --- LINT ---

.PHONY: lint
lint:
	$(LOCAL_BIN)/golangci-lint run --fix
