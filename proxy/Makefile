# =============================================================================
# devkit Reverse Proxy — Makefile
# =============================================================================

.DEFAULT_GOAL := help
SHELL := /bin/bash

include .env
export

BINARY := devkit-proxy
PID_FILE := $(BINARY).pid

# =============================================================================
# Build
# =============================================================================

.PHONY: build
build: ## Compile the proxy binary
	go build -o $(BINARY) .
	@echo "✅ Built $(BINARY)"

# =============================================================================
# Lifecycle
# =============================================================================

.PHONY: up
up: build ## Start the proxy in the background
	@if [ -f $(PID_FILE) ] && kill -0 $$(cat $(PID_FILE)) 2>/dev/null; then \
		echo "⚠️  Proxy is already running (PID $$(cat $(PID_FILE)))"; \
	else \
		./$(BINARY) & echo $$! > $(PID_FILE); \
		sleep 1; \
		if kill -0 $$(cat $(PID_FILE)) 2>/dev/null; then \
			echo ""; \
			echo "✅ devkit proxy is running"; \
			echo "   Dashboard: http://localhost:$(PROXY_PORT)"; \
			echo "   PID: $$(cat $(PID_FILE))"; \
		else \
			echo "❌ Proxy failed to start"; \
			rm -f $(PID_FILE); \
			exit 1; \
		fi; \
	fi

.PHONY: down
down: ## Stop the proxy
	@if [ -f $(PID_FILE) ]; then \
		kill $$(cat $(PID_FILE)) 2>/dev/null && echo "✅ Proxy stopped" || echo "⚠️  Process already gone"; \
		rm -f $(PID_FILE); \
	else \
		echo "⚠️  No PID file found (proxy not running?)"; \
	fi

.PHONY: restart
restart: ## Restart the proxy
	$(MAKE) down
	$(MAKE) up

.PHONY: status
status: ## Check if the proxy is running and show route table
	@if [ -f $(PID_FILE) ] && kill -0 $$(cat $(PID_FILE)) 2>/dev/null; then \
		echo "🟢 Proxy is running (PID $$(cat $(PID_FILE)))"; \
		echo ""; \
		curl -sf http://localhost:$(PROXY_PORT)/ | grep -o '<td>[^<]*</td>' | paste - - - - || true; \
	else \
		echo "🔴 Proxy is not running"; \
	fi

# =============================================================================
# Test
# =============================================================================

.PHONY: test
test: ## Run Go tests
	go test -v ./...

# =============================================================================
# Cleanup
# =============================================================================

.PHONY: clean
clean: down ## Stop proxy and remove binary + state
	rm -f $(BINARY) state.json
	@echo "✅ Cleaned up"

# =============================================================================
# Help
# =============================================================================

.PHONY: help
help: ## Show this help
	@echo "devkit Reverse Proxy"
	@echo "===================="
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(firstword $(MAKEFILE_LIST)) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}'
	@echo ""
