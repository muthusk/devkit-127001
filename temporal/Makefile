# =============================================================================
# Temporal Local Dev Environment — Makefile
# =============================================================================
#
# Usage:
#   make up          Start everything
#   make seed        Trigger all example workflows
#   make down        Stop and remove containers + volumes
#
# =============================================================================

.DEFAULT_GOAL := help
SHELL := /bin/bash

# Load .env
include .env
export

COMPOSE := docker compose
ADMIN_TOOLS := $(COMPOSE) exec temporal-admin-tools
TEMPORAL_CLI := $(ADMIN_TOOLS) temporal

# =============================================================================
# Environment
# =============================================================================

.PHONY: up
up: ## Start all services (builds workers if needed)
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

.PHONY: down
down: ## Stop all services and remove volumes (clean slate)
	$(COMPOSE) down -v
	@echo "✅ Environment torn down."

.PHONY: restart
restart: ## Full restart (down + up)
	$(MAKE) down
	$(MAKE) up

.PHONY: stop
stop: ## Stop services without removing volumes (preserves data)
	$(COMPOSE) stop

.PHONY: start
start: ## Start previously stopped services
	$(COMPOSE) start

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

.PHONY: seed-go
seed-go: ## Trigger all Go workflows
	@./scripts/seed/seed-go.sh

.PHONY: seed-ts
seed-ts: ## Trigger all TS workflows
	@./scripts/seed/seed-ts.sh

# --- Individual Go workflows ---

.PHONY: seed-event-go
seed-event-go: ## Trigger Go event-driven workflow
	$(TEMPORAL_CLI) workflow start \
		--task-queue $(GO_TASK_QUEUE) \
		--type EventProcessingWorkflow \
		--workflow-id event-go-$$(date +%s) \
		--input '{"type":"order.created","id":"order-123","timestamp":"$(shell date -u +%FT%TZ)"}'

.PHONY: seed-wait-go
seed-wait-go: ## Trigger Go dummy wait workflow (30s)
	$(TEMPORAL_CLI) workflow start \
		--task-queue $(GO_TASK_QUEUE) \
		--type DummyWaitWorkflow \
		--workflow-id wait-go-$$(date +%s) \
		--input '"30s"'

.PHONY: seed-approval-go
seed-approval-go: ## Trigger Go human approval workflow
	$(TEMPORAL_CLI) workflow start \
		--task-queue $(GO_TASK_QUEUE) \
		--type HumanApprovalWorkflow \
		--workflow-id approval-go-$$(date +%s) \
		--input '{"requester":"alice","item":"production-deploy-v2.1","timeout":"5m"}'

.PHONY: seed-saga-go
seed-saga-go: ## Trigger Go saga workflow
	$(TEMPORAL_CLI) workflow start \
		--task-queue $(GO_TASK_QUEUE) \
		--type SagaWorkflow \
		--workflow-id saga-go-$$(date +%s) \
		--input '{"orderId":"order-456","amount":99.99,"item":"Widget Pro"}'

.PHONY: seed-child-go
seed-child-go: ## Trigger Go child fanout workflow
	$(TEMPORAL_CLI) workflow start \
		--task-queue $(GO_TASK_QUEUE) \
		--type ChildFanoutWorkflow \
		--workflow-id fanout-go-$$(date +%s) \
		--input '{"items":["item-a","item-b","item-c","item-d","item-e"]}'

# --- Individual TS workflows ---

.PHONY: seed-event-ts
seed-event-ts: ## Trigger TS event-driven workflow
	$(TEMPORAL_CLI) workflow start \
		--task-queue $(TS_TASK_QUEUE) \
		--type eventProcessingWorkflow \
		--workflow-id event-ts-$$(date +%s) \
		--input '{"type":"user.signup","id":"user-789","timestamp":"$(shell date -u +%FT%TZ)"}'

.PHONY: seed-wait-ts
seed-wait-ts: ## Trigger TS dummy wait workflow (30s)
	$(TEMPORAL_CLI) workflow start \
		--task-queue $(TS_TASK_QUEUE) \
		--type dummyWaitWorkflow \
		--workflow-id wait-ts-$$(date +%s) \
		--input '"30s"'

.PHONY: seed-approval-ts
seed-approval-ts: ## Trigger TS human approval workflow
	$(TEMPORAL_CLI) workflow start \
		--task-queue $(TS_TASK_QUEUE) \
		--type humanApprovalWorkflow \
		--workflow-id approval-ts-$$(date +%s) \
		--input '{"requester":"bob","item":"staging-db-migration","timeout":"5m"}'

.PHONY: seed-saga-ts
seed-saga-ts: ## Trigger TS saga workflow
	$(TEMPORAL_CLI) workflow start \
		--task-queue $(TS_TASK_QUEUE) \
		--type sagaWorkflow \
		--workflow-id saga-ts-$$(date +%s) \
		--input '{"orderId":"order-789","amount":149.99,"item":"Gadget Ultra"}'

.PHONY: seed-child-ts
seed-child-ts: ## Trigger TS child fanout workflow
	$(TEMPORAL_CLI) workflow start \
		--task-queue $(TS_TASK_QUEUE) \
		--type childFanoutWorkflow \
		--workflow-id fanout-ts-$$(date +%s) \
		--input '{"items":["task-1","task-2","task-3","task-4","task-5"],"concurrencyLimit":3}'

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
clean: ## Full cleanup: remove containers, volumes, and images
	$(COMPOSE) down -v --rmi local
	@echo "✅ Full cleanup complete."

# =============================================================================
# Help
# =============================================================================

.PHONY: help
help: ## Show this help
	@echo "Temporal Local Dev Environment"
	@echo "=============================="
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}'
	@echo ""
