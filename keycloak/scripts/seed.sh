#!/usr/bin/env bash
# =============================================================================
# seed.sh — Create sample realm, client, and users in Keycloak
# =============================================================================

#set -euo pipefail

KEYCLOAK_URL="http://localhost:${KEYCLOAK_PORT:-8180}"
REALM="devkit-sample"
CLIENT_ID="devkit-app"

# ---------------------------------------------------------------------------
# Get admin access token
# ---------------------------------------------------------------------------
echo "🔑 Getting admin access token..."
TOKEN=$(curl -sf -X POST "${KEYCLOAK_URL}/realms/master/protocol/openid-connect/token" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -d "username=${KC_BOOTSTRAP_ADMIN_USERNAME:-admin}" \
    -d "password=${KC_BOOTSTRAP_ADMIN_PASSWORD:-admin}" \
    -d "grant_type=password" \
    -d "client_id=admin-cli" | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])")

if [ -z "$TOKEN" ]; then
    echo "❌ Failed to get admin token. Is Keycloak running?"
    exit 1
fi

AUTH="Authorization: Bearer ${TOKEN}"

# ---------------------------------------------------------------------------
# Create realm
# ---------------------------------------------------------------------------
echo "🏰 Creating realm: ${REALM}..."
EXISTING=$(curl -sf -o /dev/null -w "%{http_code}" \
    -H "$AUTH" "${KEYCLOAK_URL}/admin/realms/${REALM}")

if [ "$EXISTING" = "200" ]; then
    echo "   Realm already exists, skipping."
else
    curl -sf -X POST "${KEYCLOAK_URL}/admin/realms" \
        -H "$AUTH" \
        -H "Content-Type: application/json" \
        -d "{
            \"realm\": \"${REALM}\",
            \"enabled\": true,
            \"registrationAllowed\": true,
            \"sslRequired\": \"NONE\"
        }"
    echo "   Created."
fi

# ---------------------------------------------------------------------------
# Create OIDC client
# ---------------------------------------------------------------------------
echo "📱 Creating client: ${CLIENT_ID}..."
CLIENT_EXISTS=$(curl -sf -H "$AUTH" \
    "${KEYCLOAK_URL}/admin/realms/${REALM}/clients?clientId=${CLIENT_ID}" \
    | python3 -c "import sys,json; data=json.load(sys.stdin); print('yes' if data else 'no')")

if [ "$CLIENT_EXISTS" = "yes" ]; then
    echo "   Client already exists, skipping."
else
    curl -sf -X POST "${KEYCLOAK_URL}/admin/realms/${REALM}/clients" \
        -H "$AUTH" \
        -H "Content-Type: application/json" \
        -d "{
            \"clientId\": \"${CLIENT_ID}\",
            \"enabled\": true,
            \"publicClient\": true,
            \"standardFlowEnabled\": true,
            \"directAccessGrantsEnabled\": true,
            \"redirectUris\": [\"http://localhost:*\"],
            \"webOrigins\": [\"http://localhost:*\"]
        }"
    echo "   Created."
fi

# ---------------------------------------------------------------------------
# Create users
# ---------------------------------------------------------------------------
create_user() {
    local username=$1
    local password=$2

    echo "👤 Creating user: ${username}..."
    USER_EXISTS=$(curl -sf -H "$AUTH" \
        "${KEYCLOAK_URL}/admin/realms/${REALM}/users?username=${username}&exact=true" \
        | python3 -c "import sys,json; data=json.load(sys.stdin); print('yes' if data else 'no')")

    if [ "$USER_EXISTS" = "yes" ]; then
        echo "   User already exists, skipping."
        return
    fi

    # Create user
    curl -sf -X POST "${KEYCLOAK_URL}/admin/realms/${REALM}/users" \
        -H "$AUTH" \
        -H "Content-Type: application/json" \
        -d "{
            \"username\": \"${username}\",
            \"enabled\": true,
            \"emailVerified\": true,
            \"email\": \"${username}@example.com\",
            \"firstName\": \"$(echo ${username} | sed 's/./\U&/')\"
        }"

    # Get user ID
    USER_ID=$(curl -sf -H "$AUTH" \
        "${KEYCLOAK_URL}/admin/realms/${REALM}/users?username=${username}&exact=true" \
        | python3 -c "import sys,json; print(json.load(sys.stdin)[0]['id'])")

    # Set password
    curl -sf -X PUT "${KEYCLOAK_URL}/admin/realms/${REALM}/users/${USER_ID}/reset-password" \
        -H "$AUTH" \
        -H "Content-Type: application/json" \
        -d "{
            \"type\": \"password\",
            \"value\": \"${password}\",
            \"temporary\": false
        }"

    echo "   Created (password: ${password})."
}

create_user "alice" "alice123"
create_user "bob" "bob123"

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
echo ""
echo "✅ Seed complete!"
echo ""
echo "   Realm:    ${REALM}"
echo "   Client:   ${CLIENT_ID} (public, OIDC)"
echo "   Users:    alice / alice123"
echo "             bob   / bob123"
echo ""
echo "   OIDC Discovery: ${KEYCLOAK_URL}/realms/${REALM}/.well-known/openid-configuration"
