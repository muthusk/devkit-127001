#!/usr/bin/env bash
# =============================================================================
# disable-ssl.sh — Disable SSL requirement on master realm for local HTTP access
# =============================================================================

set -euo pipefail

echo "🔓 Disabling SSL requirement on master realm..."

docker compose exec -T keycloak /opt/keycloak/bin/kcadm.sh config credentials \
    --server http://localhost:8080${KC_CONTEXT_PATH:-/keycloak} \
    --realm master \
    --user "${KC_BOOTSTRAP_ADMIN_USERNAME:-admin}" \
    --password "${KC_BOOTSTRAP_ADMIN_PASSWORD:-admin}" 2>/dev/null

docker compose exec -T keycloak /opt/keycloak/bin/kcadm.sh update realms/master \
    -s sslRequired=NONE 2>/dev/null

echo "   Done."
