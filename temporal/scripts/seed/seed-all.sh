#!/usr/bin/env bash
# =============================================================================
# seed-all.sh — Triggers all example workflows across both Go and TS workers
# =============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "🌱 Seeding all example workflows..."
echo ""

echo "=== Go Worker Workflows ==="
"${SCRIPT_DIR}/seed-go.sh"
echo ""

echo "=== TypeScript Worker Workflows ==="
"${SCRIPT_DIR}/seed-ts.sh"
echo ""

echo "✅ All workflows seeded! Open http://localhost:${TEMPORAL_UI_PORT:-8080} to view them."
