#!/usr/bin/env bash
# Checks for newer Trigger.dev releases and optionally upgrades.
# Called by `make up` (interactive) and `make upgrade` (FORCE_UPGRADE=1).
#
# Skip: SKIP_UPDATE_CHECK=1 make up
# Permanent: echo "UPDATE_CHECK=false" >> .trigger-local

set -uo pipefail

CACHE_FILE=".trigger-update-cache"
CACHE_TTL=14400  # 4 hours
VERSION_FILE=".trigger-version"
ENV_FILE="infra/hosting/docker/.env"
FORCE="${FORCE_UPGRADE:-0}"

# --- Bail early (unless forced) ---
if [ "$FORCE" != "1" ]; then
  [ "${CI:-}" = "true" ] && exit 0
  [ "${SKIP_UPDATE_CHECK:-}" = "1" ] && exit 0
  if [ -f ".trigger-local" ]; then
    grep -qi "UPDATE_CHECK=false" .trigger-local 2>/dev/null && exit 0
  fi
fi

CURRENT="$(cat "$VERSION_FILE" 2>/dev/null || echo "unknown")"

# --- Helpers ---
major_version() { echo "$1" | sed 's/^v//' | cut -d. -f1; }

image_size() {
  docker images --format '{{.Repository}}:{{.Tag}} {{.Size}}' 2>/dev/null \
    | grep "triggerdotdev/trigger.dev:$1" | awk '{print $2}' | head -1
}

# --- Resolve latest version (cache or fetch) ---
latest=""
if [ "$FORCE" != "1" ] && [ -f "$CACHE_FILE" ]; then
  cached_time=$(head -1 "$CACHE_FILE")
  now=$(date +%s)
  if [ $((now - cached_time)) -lt $CACHE_TTL ]; then
    latest=$(tail -1 "$CACHE_FILE")
  fi
fi

if [ -z "$latest" ]; then
  # GitHub releases use monorepo tags like "@trigger.dev/core@4.3.3"
  # We need to extract the version and convert to Docker image tag format "v4.3.3"
  response=$(curl -sf --max-time 3 \
    "https://api.github.com/repos/triggerdotdev/trigger.dev/releases/latest" 2>/dev/null) || {
    [ "$FORCE" = "1" ] && echo "❌ Could not fetch latest version." && exit 1
    exit 0
  }
  raw_tag=$(echo "$response" | grep '"tag_name"' | head -1 | sed 's/.*: *"//;s/".*//')
  [ -z "$raw_tag" ] && exit 0
  # Extract version: "trigger.dev@4.3.3" or "@trigger.dev/core@4.3.3" → "v4.3.3"
  version_num=$(echo "$raw_tag" | sed 's/.*@//')
  latest="v${version_num}"
  echo "$(date +%s)" > "$CACHE_FILE"
  echo "$latest" >> "$CACHE_FILE"
fi

# --- Already current ---
if [ "$latest" = "$CURRENT" ]; then
  [ "$FORCE" = "1" ] && echo "✅ Already on the latest version ($CURRENT)."
  exit 0
fi

# --- Major version jump: warn only ---
current_major=$(major_version "$CURRENT")
latest_major=$(major_version "$latest")

if [ "$current_major" != "$latest_major" ]; then
  echo ""
  echo "  ⚠️  Trigger.dev $latest is available (you're on $CURRENT)"
  echo "     This is a MAJOR version change — review release notes before upgrading:"
  echo "     https://github.com/triggerdotdev/trigger.dev/releases/tag/$latest"
  echo "     The compose files may have breaking changes. Manual review recommended."
  echo ""
  exit 0
fi

# --- Minor/patch: offer interactive upgrade ---
echo ""
echo "  ⬆️  Trigger.dev $latest is available (you're on $CURRENT)"
read -p "  Upgrade now? [y/N] " confirm
if [ "$confirm" != "y" ] && [ "$confirm" != "Y" ]; then
  echo "  Skipped. Run 'make upgrade' anytime, or SKIP_UPDATE_CHECK=1 make up to silence."
  echo ""
  exit 0
fi

# --- Perform upgrade ---
echo "  📦 Upgrading $CURRENT → $latest..."

# Save old version for rollback
echo "$CURRENT" > "$VERSION_FILE.bak"

echo "$latest" > "$VERSION_FILE"

if [ -f "$ENV_FILE" ]; then
  if grep -q "^TRIGGER_IMAGE_TAG=" "$ENV_FILE"; then
    sed -i.bak "s|^TRIGGER_IMAGE_TAG=.*|TRIGGER_IMAGE_TAG=${latest}|" "$ENV_FILE"
    rm -f "$ENV_FILE.bak"
  else
    echo "TRIGGER_IMAGE_TAG=${latest}" >> "$ENV_FILE"
  fi
fi

version_no_v="${latest#v}"
npm pkg set "dependencies.@trigger.dev/sdk=^${version_no_v}" 2>/dev/null
npm install --silent 2>/dev/null

rm -f "$CACHE_FILE"
echo "  ✅ Upgraded to $latest"

# --- Offer to clean up old image ---
old_size=$(image_size "$CURRENT")
if [ -n "$old_size" ]; then
  in_use=$(docker ps --format '{{.Image}}' 2>/dev/null | grep "triggerdotdev/trigger.dev:$CURRENT" || true)
  if [ -z "$in_use" ]; then
    echo ""
    echo "  🗑️  Old image $CURRENT (~$old_size) is not in use by running containers."
    read -p "  Delete it to free space? [y/N] " del_confirm
    if [ "$del_confirm" = "y" ] || [ "$del_confirm" = "Y" ]; then
      docker rmi "ghcr.io/triggerdotdev/trigger.dev:$CURRENT" 2>/dev/null && \
        echo "  ✅ Deleted old image ($old_size freed)." || \
        echo "  ⚠️  Could not delete. Run 'make nuke' for full cleanup."
    fi
  else
    echo "  ℹ️  Old image $CURRENT still in use. Stop containers first or 'make nuke'."
  fi
fi
echo ""
