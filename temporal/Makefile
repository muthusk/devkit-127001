# =============================================================================
# Temporal Local Dev Environment — Makefile
# =============================================================================
#
# Usage:
#   make up          Start everything
#   make seed        Trigger all example workflows
#   make down        Graceful shutdown (preserves volumes)
#
# =============================================================================

.DEFAULT_GOAL := help
SHELL := /bin/bash

# Load .env
include .env
export

COMPOSE := docker compose
FIND_PORT := ../shared/scripts/find-port.sh
ADMIN_TOOLS := $(COMPOSE) exec temporal-admin-tools
TEMPORAL_CLI := $(ADMIN_TOOLS) temporal

# =============================================================================
# Environment
# =============================================================================

.PHONY: up
up: ## Start all services (builds workers if needed)
	@# Best-effort port detection
	@if [ -x "$(FIND_PORT)" ]; then \
		PORTS=$$($(FIND_PORT) $(TEMPORAL_UI_PORT) $(TEMPORAL_GRPC_PORT) $(TEMPORAL_HTTP_PORT) $(POSTGRES_PORT)); \
		TEMPORAL_UI_PORT=$$(echo "$$PORTS" | sed -n '1p'); \
		TEMPORAL_GRPC_PORT=$$(echo "$$PORTS" | sed -n '2p'); \
		TEMPORAL_HTTP_PORT=$$(echo "$$PORTS" | sed -n '3p'); \
		POSTGRES_PORT=$$(echo "$$PORTS" | sed -n '4p'); \
		export TEMPORAL_UI_PORT TEMPORAL_GRPC_PORT TEMPORAL_HTTP_PORT POSTGRES_PORT; \
	fi; \
	$(COMPOSE) up -d --build
	@echo ""
	@echo "⏳ Waiting for Temporal server to be ready..."
	@./scripts/wait-for-temporal.sh
	@echo ""
	@echo "✅ Temporal is ready!"
	@echo "   UI:   http://localhost:$(TEMPORAL_UI_PORT)"
	@echo "   gRPC: localhost:$(TEMPORAL_GRPC_PORT)"
	@echo ""
	@echo "Run 'make seed' to trigger example workflows."
	@# Best-effort proxy registration
	@curl -sf -X POST https://home.dev.kit/proxy/register \
		-d '{"name":"temporal","port":$(TEMPORAL_UI_PORT)}' >/dev/null 2>&1 \
		&& echo "   Proxy: https://temporal.dev.kit" \
		|| echo "   Proxy not available. Register later with:" && \
		   echo "   curl -X POST https://home.dev.kit/proxy/register -d '{\"name\":\"temporal\",\"port\":$(TEMPORAL_UI_PORT)}'"

.PHONY: down
down: ## Graceful shutdown (preserves volumes)
	@# Best-effort proxy deregistration
	@curl -sf -X POST https://home.dev.kit/proxy/deregister \
		-d '{"name":"temporal"}' >/dev/null 2>&1 || true
	$(COMPOSE) down
	@echo "✅ Environment stopped."

.PHONY: restart
restart: ## Full restart (down + up)
	$(MAKE) down
	$(MAKE) up

.PHONY: status
status: ## Check health of all services
	@$(COMPOSE) ps
	@echo ""
	@echo "--- Temporal Server Health ---"
	@$(TEMPORAL_CLI) operator cluster health 2>/dev/null && echo "✅ Healthy" || echo "❌ Unhealthy"

# =============================================================================
# Logs
# =============================================================================

.PHONY: logs
logs: ## Tail logs for all containers
	$(COMPOSE) logs -f

.PHONY: logs-go
logs-go: ## Tail Go worker logs
	$(COMPOSE) logs -f worker-go

.PHONY: logs-ts
logs-ts: ## Tail TypeScript worker logs
	$(COMPOSE) logs -f worker-ts

.PHONY: logs-server
logs-server: ## Tail Temporal server logs
	$(COMPOSE) logs -f temporal-server

# =============================================================================
# Seeding — Trigger example workflows
# =============================================================================

.PHONY: seed
seed: ## Trigger all example workflows (both Go and TS)
	@./scripts/seed/seed-all.sh


# =============================================================================
# Interactions — Signals and Queries
# =============================================================================

.PHONY: signal-approve
signal-approve: ## Send approval signal (usage: make signal-approve WID=approval-go-xxx)
	@if [ -z "$(WID)" ]; then echo "❌ Usage: make signal-approve WID=<workflow-id>"; exit 1; fi
	$(TEMPORAL_CLI) workflow signal \
		--workflow-id $(WID) \
		--name approval \
		--input '{"approved":true,"approver":"admin","reason":"Looks good"}'
	@echo "✅ Approval signal sent to $(WID)"

.PHONY: signal-reject
signal-reject: ## Send rejection signal (usage: make signal-reject WID=approval-xxx)
	@if [ -z "$(WID)" ]; then echo "❌ Usage: make signal-reject WID=<workflow-id>"; exit 1; fi
	$(TEMPORAL_CLI) workflow signal \
		--workflow-id $(WID) \
		--name approval \
		--input '{"approved":false,"approver":"admin","reason":"Not ready yet"}'
	@echo "✅ Rejection signal sent to $(WID)"

.PHONY: query-approval-ts
query-approval-ts: ## Query TS approval workflow status (usage: make query-approval-ts WID=approval-ts-xxx)
	@if [ -z "$(WID)" ]; then echo "❌ Usage: make query-approval-ts WID=<workflow-id>"; exit 1; fi
	$(TEMPORAL_CLI) workflow query \
		--workflow-id $(WID) \
		--name approvalStatus

# =============================================================================
# Development
# =============================================================================

.PHONY: rebuild-go
rebuild-go: ## Rebuild and restart Go worker only
	$(COMPOSE) up -d --build --force-recreate worker-go
	@echo "✅ Go worker rebuilt and restarted."

.PHONY: rebuild-ts
rebuild-ts: ## Rebuild and restart TS worker only
	$(COMPOSE) up -d --build --force-recreate worker-ts
	@echo "✅ TS worker rebuilt and restarted."

.PHONY: shell
shell: ## Open a shell in the admin-tools container
	$(ADMIN_TOOLS) bash

# =============================================================================
# Cleanup
# =============================================================================

.PHONY: clean
clean: ## Remove containers and volumes (optionally images)
	@read -p "Also remove built images? [y/N] " img; \
	if [ "$$img" = "y" ]; then \
		$(COMPOSE) down -v --rmi local; \
	else \
		$(COMPOSE) down -v; \
	fi
	@echo "✅ Cleaned up."

# =============================================================================
# Help
# =============================================================================

.PHONY: help
help: ## Show this help
	@echo "Temporal Local Dev Environment"
	@echo "=============================="
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(firstword $(MAKEFILE_LIST)) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}'
	@echo ""
