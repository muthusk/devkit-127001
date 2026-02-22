.DEFAULT_GOAL := help
SHELL := /bin/bash

# Load devkit .env
include .env
export

.PHONY: up down restart logs logs-magic-link status login init dev deploy \
        registry-login clean upgrade health worker-token help

COMPOSE_DIR   := infra/hosting/docker
COMPOSE_CMD   := docker compose \
  -f $(COMPOSE_DIR)/webapp/docker-compose.yml \
  -f $(COMPOSE_DIR)/worker/docker-compose.yml
FIND_PORT     := ../shared/scripts/find-port.sh
TRIGGER_URL   := http://localhost:$(TRIGGER_PORT)
REGISTRY_HOST := localhost:5000
REGISTRY_USER := registry-user
VERSION_FILE  := .trigger-version
CLI_VERSION   = $(shell cat $(VERSION_FILE) 2>/dev/null || echo "v4.3.3")

help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ── Infrastructure ───────────────────────────

up: ## Start everything (first run: clones infra + installs deps)
	@echo "🔧 Ensuring infrastructure prerequisites..."
	@./scripts/setup.sh
	@echo ""
ifndef SKIP_UPDATE_CHECK
	@./scripts/check-update.sh || true
endif
	@# Best-effort port detection
	@if [ -x "$(FIND_PORT)" ]; then \
		TRIGGER_PORT=$$($(FIND_PORT) $(TRIGGER_PORT)); \
		export TRIGGER_PORT; \
	fi
	@echo "🚀 Starting Trigger.dev infrastructure..."
	@$(COMPOSE_CMD) up -d || { \
	  if [ -f .trigger-version.bak ]; then \
	    echo ""; \
	    echo "❌ Startup failed. Rolling back to $$(cat .trigger-version.bak)..."; \
	    ./scripts/rollback.sh; \
	    echo "Rolled back. Run 'make up' again."; \
	  fi; \
	  exit 1; \
	}
	@rm -f .trigger-version.bak
	@echo ""
	@./scripts/wait-for-webapp.sh $(TRIGGER_URL)/healthcheck 90
	@echo ""
	@echo "┌──────────────────────────────────────────────────┐"
	@echo "│  ✅ Trigger.dev is running                       │"
	@echo "├──────────────────────────────────────────────────┤"
	@echo "│  Dashboard:  $(TRIGGER_URL)            │"
	@echo "│  Registry:   $(REGISTRY_HOST)                 │"
	@echo "│  MinIO:      http://localhost:9001              │"
	@echo "├──────────────────────────────────────────────────┤"
	@echo "│  Registry login:  $(REGISTRY_USER) / very-secure-indeed │"
	@echo "│  MinIO login:     admin / very-safe-password    │"
	@echo "├──────────────────────────────────────────────────┤"
	@echo "│  Next: run 'make logs-magic-link' for login URL │"
	@echo "└──────────────────────────────────────────────────┘"
	@# Best-effort proxy registration
	@curl -sf -X POST https://home.dev.kit/proxy/register \
		-d '{"name":"trigger-dev","port":$(TRIGGER_PORT)}' >/dev/null 2>&1 \
		&& echo "   Proxy: https://trigger-dev.dev.kit" \
		|| echo "   Proxy not available. Register later with:" && \
		   echo "   curl -X POST https://home.dev.kit/proxy/register -d '{\"name\":\"trigger-dev\",\"port\":$(TRIGGER_PORT)}'"

down: ## Stop all infrastructure
	@# Best-effort proxy deregistration
	@curl -sf -X POST https://home.dev.kit/proxy/deregister \
		-d '{"name":"trigger-dev"}' >/dev/null 2>&1 || true
	$(COMPOSE_CMD) down

restart: ## Restart all infrastructure
	$(COMPOSE_CMD) down
	@$(MAKE) up

logs: ## Tail container logs (all, or S="webapp redis" for specific)
ifdef S
	$(COMPOSE_CMD) logs -f $(S)
else
	$(COMPOSE_CMD) logs -f
endif

logs-magic-link: ## Watch for magic login links
	@echo "👀 Watching for magic link (Ctrl+C to stop)..."
	$(COMPOSE_CMD) logs -f webapp 2>&1 | grep -i --line-buffered "magic\|login\|sign"

status: ## Show container status + health check
	@$(COMPOSE_CMD) ps
	@echo ""
	@curl -sf $(TRIGGER_URL)/healthcheck > /dev/null 2>&1 \
	  && echo "🟢 Webapp: healthy" \
	  || echo "🔴 Webapp: not responding"

worker-token: ## Show worker token from logs
	@echo "🔑 Looking for worker token..."
	$(COMPOSE_CMD) logs webapp 2>&1 | grep -A5 "Worker Token" || echo "Token not found yet."

# ── Task project ─────────────────────────────

login: ## Login CLI to local Trigger.dev instance
	npx trigger.dev@$(CLI_VERSION) login -a $(TRIGGER_URL)

init: ## Set project ref in trigger.config.ts (usage: make init PROJECT=proj_xxxxx)
ifndef PROJECT
	@echo "❌ Usage: make init PROJECT=proj_xxxxx"
	@echo "   Get your project ref from the dashboard at $(TRIGGER_URL)"
	@exit 1
endif
	@if ! grep -q '<project-ref>' trigger.config.ts 2>/dev/null; then \
	  echo "⚠️  trigger.config.ts already has a project ref set."; \
	  echo "   Current value:"; \
	  grep 'project:' trigger.config.ts; \
	  read -p "   Overwrite? [y/N] " confirm && [ "$$confirm" = "y" ] || exit 0; \
	fi
	@sed -i.bak 's|project: ".*"|project: "$(PROJECT)"|' trigger.config.ts && rm -f trigger.config.ts.bak
	@echo "✅ Project ref set to $(PROJECT) in trigger.config.ts"

dev: ## Run trigger dev watcher
	npx trigger.dev@$(CLI_VERSION) dev -a $(TRIGGER_URL)

deploy: ## Deploy tasks to local instance
	npx trigger.dev@$(CLI_VERSION) deploy -a $(TRIGGER_URL)

registry-login: ## Login to local Docker registry
	@echo "🔐 Logging into local registry (password: very-secure-indeed)..."
	docker login -u $(REGISTRY_USER) $(REGISTRY_HOST)

# ── Upgrade / Cleanup ────────────────────────

upgrade: ## Upgrade to latest Trigger.dev release
	@FORCE_UPGRADE=1 ./scripts/check-update.sh

clean: ## Remove containers, volumes, and local caches (optionally images)
	@echo "⚠️  This will destroy ALL data (database, registry, object storage, caches)."
	@read -p "Are you sure? [y/N] " confirm && [ "$$confirm" = "y" ] || exit 1
	@echo ""
	@echo "  🛑 Stopping containers..."
	@$(COMPOSE_CMD) down -v --remove-orphans 2>/dev/null \
	  && echo "  ✅ Containers and volumes removed." \
	  || echo "  ⚠️  Compose down failed (may already be stopped). Continuing..."
	@echo ""
	@read -p "  Also remove Docker images? [y/N] " del_img; \
	  if [ "$$del_img" = "y" ]; then \
	    images=$$(docker images --format '{{.Repository}}:{{.Tag}}' 2>/dev/null | grep -i 'trigger' || true); \
	    if [ -n "$$images" ]; then \
	      echo "$$images" | while read img; do \
	        printf "     Removing $$img... "; \
	        docker rmi -f "$$img" 2>/dev/null && echo "done." || echo "failed (may be in use)."; \
	      done; \
	    else \
	      echo "     No trigger images found."; \
	    fi; \
	  else \
	    echo "  ⏭️  Skipping image removal."; \
	  fi
	@echo ""
	@echo "  🧹 Removing local files..."
	@for dir in infra node_modules .trigger dist; do \
	  if [ -e "$$dir" ]; then \
	    printf "     Removing $$dir/... "; \
	    rm -rf "$$dir" && echo "done." || echo "failed."; \
	  fi; \
	done
	@for f in .trigger-update-cache .trigger-version.bak; do \
	  if [ -e "$$f" ]; then \
	    printf "     Removing $$f... "; \
	    rm -f "$$f" && echo "done." || echo "failed."; \
	  fi; \
	done
	@echo ""
	@echo "✅ Cleaned up. Run 'make up' to rebuild from scratch."