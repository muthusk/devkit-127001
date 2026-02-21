#!/usr/bin/env bash
# Idempotent setup: clone infra, create .env, install deps.
# Called by `make up` before starting infrastructure.
# Safe to run repeatedly — skips steps that are already done.
set -euo pipefail

TRIGGER_VERSION="$(cat .trigger-version 2>/dev/null || echo "v4.3.3")"

# ── Clone infra (if needed) ──────────────────
# Compose files live on main and aren't tagged per-release.
# Version pinning is done via TRIGGER_IMAGE_TAG in .env.
if [ ! -d "infra/.git" ]; then
  echo "📦 Cloning Trigger.dev infrastructure (compose files from main)..."
  git clone --depth=1 --filter=blob:none --sparse \
    https://github.com/triggerdotdev/trigger.dev infra
  cd infra
  git sparse-checkout set hosting/docker
  cd ..
else
  echo "✅ infra/ already present, skipping clone."
fi

# ── Create .env from example (if needed) ─────
if [ ! -f "infra/hosting/docker/.env" ]; then
  if [ ! -f "infra/hosting/docker/.env.example" ]; then
    echo "❌ infra/hosting/docker/.env.example not found."
    echo "   Try removing infra/ and running 'make up' again."
    exit 1
  fi
  cp infra/hosting/docker/.env.example infra/hosting/docker/.env
  if grep -q "TRIGGER_IMAGE_TAG" infra/hosting/docker/.env; then
    sed -i.bak "s|^TRIGGER_IMAGE_TAG=.*|TRIGGER_IMAGE_TAG=${TRIGGER_VERSION}|" infra/hosting/docker/.env
    rm -f infra/hosting/docker/.env.bak
  else
    echo "" >> infra/hosting/docker/.env
    echo "TRIGGER_IMAGE_TAG=${TRIGGER_VERSION}" >> infra/hosting/docker/.env
  fi
  echo "📝 Created infra/hosting/docker/.env (pinned to ${TRIGGER_VERSION})"
else
  echo "✅ .env already present, skipping."
fi

# ── Install npm deps (if needed) ─────────────
if [ ! -d "node_modules" ]; then
  echo "📦 Installing npm dependencies..."
  npm install
else
  echo "✅ node_modules/ already present, skipping install."
fi