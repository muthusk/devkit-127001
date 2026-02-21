#!/usr/bin/env bash
# =============================================================================
# wait-for-temporal.sh — Polls Temporal server until it's ready
# =============================================================================

set -euo pipefail

MAX_RETRIES=60
RETRY_INTERVAL=3
TEMPORAL_HOST="${TEMPORAL_ADDRESS:-temporal-server:7233}"

echo "Waiting for Temporal server at ${TEMPORAL_HOST}..."

for i in $(seq 1 $MAX_RETRIES); do
    if docker compose exec temporal-admin-tools temporal operator cluster health 2>/dev/null | grep -q "SERVING"; then
        echo "Temporal server is ready! (attempt ${i}/${MAX_RETRIES})"
        exit 0
    fi
    echo "  Attempt ${i}/${MAX_RETRIES} — not ready yet, retrying in ${RETRY_INTERVAL}s..."
    sleep $RETRY_INTERVAL
done

echo "❌ Temporal server did not become ready after $((MAX_RETRIES * RETRY_INTERVAL))s"
exit 1
