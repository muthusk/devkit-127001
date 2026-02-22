# =============================================================================
# Keycloak Local Dev Environment — Makefile
# =============================================================================
#
# Usage:
#   make up          Start Keycloak + Postgres
#   make seed        Create sample realm, client, and users
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

# =============================================================================
# Lifecycle
# =============================================================================

.PHONY: up
up: ## Start Keycloak and Postgres
	@# Best-effort port detection
	@if [ -x "$(FIND_PORT)" ]; then \
		PORTS=$$($(FIND_PORT) $(KEYCLOAK_PORT) $(KEYCLOAK_POSTGRES_PORT) $(KEYCLOAK_HEALTH_PORT)); \
		KEYCLOAK_PORT=$$(echo "$$PORTS" | sed -n '1p'); \
		KEYCLOAK_POSTGRES_PORT=$$(echo "$$PORTS" | sed -n '2p'); \
		KEYCLOAK_HEALTH_PORT=$$(echo "$$PORTS" | sed -n '3p'); \
		export KEYCLOAK_PORT KEYCLOAK_POSTGRES_PORT KEYCLOAK_HEALTH_PORT; \
	fi; \
	$(COMPOSE) up -d --build
	@echo ""
	@echo "⏳ Waiting for Keycloak to be ready..."
	@./scripts/wait-for-keycloak.sh
	@./scripts/disable-ssl.sh
	@echo ""
	@echo "✅ Keycloak is ready!"
	@echo "   Admin Console: http://localhost:$(KEYCLOAK_PORT)"
	@echo "   Username:      $(KC_BOOTSTRAP_ADMIN_USERNAME)"
	@echo "   Password:      $(KC_BOOTSTRAP_ADMIN_PASSWORD)"
	@echo ""
	@echo "Run 'make seed' to create sample realm, client, and users."
	@# Best-effort proxy registration
	@curl -sf -X POST https://home.dev.kit/proxy/register \
		-d '{"name":"keycloak","port":$(KEYCLOAK_PORT)}' >/dev/null 2>&1 \
		&& echo "   Proxy: https://keycloak.dev.kit" \
		|| echo "   Proxy not available. Register later with:" && \
		   echo "   curl -X POST https://home.dev.kit/proxy/register -d '{\"name\":\"keycloak\",\"port\":$(KEYCLOAK_PORT)}'"

.PHONY: down
down: ## Graceful shutdown (preserves volumes)
	@# Best-effort proxy deregistration
	@curl -sf -X POST https://home.dev.kit/proxy/deregister \
		-d '{"name":"keycloak"}' >/dev/null 2>&1 || true
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
	@echo "--- Keycloak Health ---"
	@curl -sf http://localhost:$(KEYCLOAK_HEALTH_PORT)/health/ready >/dev/null 2>&1 \
		&& echo "✅ Keycloak is healthy" \
		|| echo "❌ Keycloak is unhealthy or not running"

.PHONY: logs
logs: ## Tail logs for all containers
	$(COMPOSE) logs -f

# =============================================================================
# Data
# =============================================================================

.PHONY: seed
seed: ## Create sample realm, client, and users
	@./scripts/seed.sh

.PHONY: shell
shell: ## Open a shell in the Keycloak container
	$(COMPOSE) exec keycloak bash

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
	@echo "Keycloak Local Dev Environment"
	@echo "=============================="
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(firstword $(MAKEFILE_LIST)) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}'
	@echo ""
