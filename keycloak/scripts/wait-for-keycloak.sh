#!/usr/bin/env bash
# =============================================================================
# wait-for-keycloak.sh — Polls Keycloak until it's ready
# =============================================================================

set -euo pipefail

MAX_RETRIES=60
RETRY_INTERVAL=3
KEYCLOAK_URL="http://localhost:${KEYCLOAK_HEALTH_PORT:-9100}/health/ready"

echo "Waiting for Keycloak at ${KEYCLOAK_URL}..."

for i in $(seq 1 $MAX_RETRIES); do
    if curl -sf --max-time 3 "$KEYCLOAK_URL" >/dev/null 2>&1; then
        echo "Keycloak is ready! (attempt ${i}/${MAX_RETRIES})"
        exit 0
    fi
    echo "  Attempt ${i}/${MAX_RETRIES} — not ready yet, retrying in ${RETRY_INTERVAL}s..."
    sleep $RETRY_INTERVAL
done

echo "❌ Keycloak did not become ready after $((MAX_RETRIES * RETRY_INTERVAL))s"
exit 1
