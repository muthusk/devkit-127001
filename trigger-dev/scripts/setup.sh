#!/usr/bin/env bash
set -euo pipefail

TRIGGER_VERSION="$(cat .trigger-version 2>/dev/null || echo "v4.3.3")"

echo "🔧 Setting up Trigger.dev local dev (${TRIGGER_VERSION})..."

# ── Clone infra (if needed) ──────────────────
# Note: compose files live on main and aren't tagged per-release.
# Version pinning is done via TRIGGER_IMAGE_TAG in .env.
if [ ! -d "infra/.git" ]; then
  echo "📦 Cloning Trigger.dev infrastructure (compose files from main)..."
  git clone --depth=1 --filter=blob:none --sparse \
    https://github.com/triggerdotdev/trigger.dev infra
  cd infra
  git sparse-checkout set hosting/docker
  cd ..
else
  echo "📦 infra/ already exists, skipping clone."
  echo "   To re-clone: rm -rf infra && make setup"
fi

# ── Create .env from example ─────────────────
if [ ! -f "infra/hosting/docker/.env" ]; then
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
  echo "📝 .env already exists, skipping."
fi

# ── Install npm deps ─────────────────────────
echo "📦 Installing npm dependencies..."
npm install

echo ""
echo "✅ Setup complete! Next: make up"
