.PHONY: setup up down restart logs logs-magic-link status init dev deploy \
        registry-login clean nuke upgrade health worker-token help

COMPOSE_DIR   := infra/hosting/docker
COMPOSE_CMD   := docker compose \
  -f $(COMPOSE_DIR)/webapp/docker-compose.yml \
  -f $(COMPOSE_DIR)/worker/docker-compose.yml
TRIGGER_URL   := http://localhost:8030
REGISTRY_HOST := localhost:5000
REGISTRY_USER := registry-user
VERSION_FILE  := .trigger-version
CLI_VERSION   = $(shell cat $(VERSION_FILE) 2>/dev/null || echo "v4.3.3")

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ── Setup ────────────────────────────────────

setup: ## One-time: clone infra + install deps
	@./scripts/setup.sh

# ── Infrastructure ───────────────────────────

up: ## Start all infrastructure
ifndef SKIP_UPDATE_CHECK
	@./scripts/check-update.sh || true
endif
	@echo "🚀 Starting Trigger.dev infrastructure..."
	@$(COMPOSE_CMD) up -d || { \
	  if [ -f .trigger-version.bak ]; then \
	    echo ""; \
	    echo "❌ Startup failed. Rolling back to $(cat .trigger-version.bak)..."; \
	    ./scripts/rollback.sh; \
	    echo "Rolled back. Run 'make up' again."; \
	  fi; \
	  exit 1; \
	}
	@rm -f .trigger-version.bak
	@echo ""
	@echo "✅ Infrastructure starting. Dashboard: $(TRIGGER_URL)"
	@echo "   Run 'make logs-magic-link' to get your login link (wait ~30-60s)"

down: ## Stop all infrastructure
	$(COMPOSE_CMD) down

restart: ## Restart all infrastructure
	$(COMPOSE_CMD) down
	$(COMPOSE_CMD) up -d

logs: ## Tail all container logs
	$(COMPOSE_CMD) logs -f

logs-magic-link: ## Watch for magic login links
	@echo "👀 Watching for magic link (Ctrl+C to stop)..."
	$(COMPOSE_CMD) logs -f webapp 2>&1 | grep -i --line-buffered "magic\|login\|sign"

status: ## Show container status
	$(COMPOSE_CMD) ps

health: ## Check webapp health
	@curl -sf $(TRIGGER_URL)/health && echo " ✅ Webapp is healthy" || echo " ❌ Webapp not responding"

worker-token: ## Show worker token from logs
	@echo "🔑 Looking for worker token..."
	$(COMPOSE_CMD) logs webapp 2>&1 | grep -A5 "Worker Token" || echo "Token not found yet."

# ── Task project ─────────────────────────────

init: ## Link task project (usage: make init PROJECT=proj_xxxxx)
ifndef PROJECT
	@echo "❌ Usage: make init PROJECT=proj_xxxxx"
	@echo "   Get your project ref from the dashboard at $(TRIGGER_URL)"
	@exit 1
endif
	npx trigger.dev@$(CLI_VERSION) init -p $(PROJECT) -a $(TRIGGER_URL)

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

clean: ## Stop infra, remove volumes (⚠️ destroys data)
	@echo "⚠️  This will destroy ALL data (database, registry, object storage)."
	@read -p "Are you sure? [y/N] " confirm && [ "$confirm" = "y" ] || exit 1
	$(COMPOSE_CMD) down -v
	@echo "🧹 Cleaned up."

nuke: ## Full wipe: containers, volumes, infra, caches. Optionally images. (☢️ nuclear)
	@echo "☢️  This will destroy containers, volumes, infra, and caches."
	@read -p "Are you absolutely sure? [y/N] " confirm && [ "$$confirm" = "y" ] || exit 1
	@echo ""
	@echo "  🛑 Stopping containers..."
	@$(COMPOSE_CMD) down -v --remove-orphans 2>/dev/null \
	  && echo "  ✅ Containers and volumes removed." \
	  || echo "  ⚠️  Compose down failed (may already be stopped). Continuing..."
	@echo ""
	@echo "  🗑️  Remove trigger Docker images as well?"
	@echo "     This frees disk but requires re-pulling on next 'make up'."
	@read -p "  Type 'delete' to remove images, or press Enter to skip: " del_img; \
	  if [ "$$del_img" = "delete" ]; then \
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
	@echo "  ☢️  Nuked. Run 'make setup && make up' to rebuild from scratch."
