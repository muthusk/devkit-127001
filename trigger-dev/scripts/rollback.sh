#!/usr/bin/env bash
set -uo pipefail

VERSION_FILE=".trigger-version"
ENV_FILE="infra/hosting/docker/.env"
BACKUP="$VERSION_FILE.bak"

if [ ! -f "$BACKUP" ]; then
  echo "  ⚠️  No backup version found. Nothing to roll back."
  exit 1
fi

OLD_VERSION="$(cat "$BACKUP")"
echo "  ↩️  Rolling back to $OLD_VERSION..."

# Restore version file
echo "$OLD_VERSION" > "$VERSION_FILE"

# Restore Docker image tag in .env
if [ -f "$ENV_FILE" ]; then
  if grep -q "^TRIGGER_IMAGE_TAG=" "$ENV_FILE"; then
    sed -i.bak "s|^TRIGGER_IMAGE_TAG=.*|TRIGGER_IMAGE_TAG=${OLD_VERSION}|" "$ENV_FILE"
    rm -f "$ENV_FILE.bak"
  fi
fi

# Restore SDK version in package.json
version_no_v="${OLD_VERSION#v}"
npm pkg set "dependencies.@trigger.dev/sdk=^${version_no_v}" 2>/dev/null
npm install --silent 2>/dev/null

rm -f "$BACKUP"
rm -f .trigger-update-cache
echo "  ✅ Rolled back to $OLD_VERSION"
