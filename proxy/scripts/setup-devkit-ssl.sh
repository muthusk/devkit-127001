#!/usr/bin/env bash
# =============================================================================
# setup-devkit-ssl.sh — Create a local CA + wildcard cert for *.dev.kit
#
# Usage:
#   ./setup-devkit-ssl.sh
#
# What it does:
#   1. Creates a local root CA (if not already present)
#   2. Generates a wildcard cert for *.dev.kit signed by that CA
#   3. Installs the CA into macOS system keychain (so browsers trust it)
#   4. Creates a combined CA bundle for terminal tools (curl, python, etc.)
#   5. Optionally adds NODE_EXTRA_CA_CERTS + SSL_CERT_FILE exports to ~/.zshrc
#
# Output files in ~/.devkit-ssl/:
#   ca.pem / ca-key.pem         — Root CA (keep ca-key.pem safe!)
#   combined-ca.pem             — System CAs + DevKit CA (for SSL_CERT_FILE)
# =============================================================================

set -euo pipefail

CA_DIR="$HOME/.devkit-ssl"
CA_CERT="$CA_DIR/ca.pem"
CA_KEY="$CA_DIR/ca-key.pem"
SSL_DIR="./cert"
SERVER_CERT="$SSL_DIR/_wildcard.dev.kit-cert.pem"
SERVER_KEY="$SSL_DIR/_wildcard.dev.kit-key.pem"
SERVER_CSR="$SSL_DIR/dev.kit.csr"
CNF_FILE="$SSL_DIR/dev.kit.cnf"
CA_NAME="DevKit Local CA"

echo "============================================"
echo "  DevKit SSL Setup"
echo "  Output: $SSL_DIR"
echo "============================================"
echo ""


mkdir -p "$CA_DIR"
mkdir -p "$SSL_DIR"


# ---------------------------------------------------------------------------
# Step 1: Root CA
# ---------------------------------------------------------------------------
echo "[1/6] Root CA"

if [ -f "$CA_CERT" ] && [ -f "$CA_KEY" ]; then
    echo "       EXISTS — using existing CA"
    echo "       $CA_CERT"
else
    echo "       CREATING root CA..."
    openssl req -new -x509 -nodes -days 3650 \
        -keyout "$CA_KEY" \
        -out "$CA_CERT" \
        -subj "/CN=${CA_NAME}/O=DevKit"
    chmod 600 "$CA_KEY"
    echo "       CREATED — valid for 10 years"
fi
echo ""

# ---------------------------------------------------------------------------
# Step 2: OpenSSL config + CSR
# ---------------------------------------------------------------------------
echo "[2/6] Generating server certificate for *.dev.kit"

cat > "$CNF_FILE" << 'EOF'
[req]
default_bits = 2048
prompt = no
distinguished_name = dn
req_extensions = v3_req

[dn]
CN = dev.kit
O = DevKit Local
OU = Development

[v3_req]
subjectAltName = @alt_names
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth

[alt_names]
DNS.1 = *.dev.kit
DNS.2 = dev.kit
DNS.3 = localhost
IP.1 = 127.0.0.1
IP.2 = ::1
EOF

openssl req -new -newkey rsa:2048 -nodes \
    -keyout "$SERVER_KEY" \
    -out "$SERVER_CSR" \
    -config "$CNF_FILE"

chmod 600 "$SERVER_KEY"
echo "       CSR created"
echo ""

# ---------------------------------------------------------------------------
# Step 3: Sign with CA
# ---------------------------------------------------------------------------
echo "[3/6] Signing with root CA"

openssl x509 -req \
    -in "$SERVER_CSR" \
    -CA "$CA_CERT" \
    -CAkey "$CA_KEY" \
    -CAcreateserial \
    -CAserial "$CA_DIR/ca.srl" \
    -out "$SERVER_CERT" \
    -days 825 \
    -extfile "$CNF_FILE" \
    -extensions v3_req

echo "       SIGNED — valid for 825 days"
echo ""

# ---------------------------------------------------------------------------
# Cleanup temp files
# ---------------------------------------------------------------------------
rm -f "$SERVER_CSR" "$CA_DIR/ca.srl"

# ---------------------------------------------------------------------------
# Verify
# ---------------------------------------------------------------------------
echo "--- Verification ---"
echo ""
echo "Subject:"
openssl x509 -in "$SERVER_CERT" -noout -subject
echo ""
echo "SANs:"
openssl x509 -in "$SERVER_CERT" -noout -text | grep -A1 "Subject Alternative Name"
echo ""
echo "Issuer:"
openssl x509 -in "$SERVER_CERT" -noout -issuer
echo ""
echo "Expiry:"
openssl x509 -in "$SERVER_CERT" -noout -dates
echo ""

# ---------------------------------------------------------------------------
# Step 4: Trust CA in system keychain
# ---------------------------------------------------------------------------
echo "[4/6] Trust CA in macOS keychain"

if security find-certificate -c "$CA_NAME" /Library/Keychains/System.keychain > /dev/null 2>&1; then
    echo "       EXISTS — CA already trusted"
else
    echo "       INSTALLING into system keychain (requires sudo)..."
    sudo security add-trusted-cert -d -r trustRoot \
        -k /Library/Keychains/System.keychain "$CA_CERT"
    echo "       INSTALLED — browsers will trust certs signed by this CA"
fi
echo ""

# ---------------------------------------------------------------------------
# Step 5: Combined CA bundle for terminal tools
# ---------------------------------------------------------------------------
COMBINED_CA="$CA_DIR/combined-ca.pem"
echo "[5/6] Combined CA bundle"

cp /etc/ssl/cert.pem "$COMBINED_CA"
cat "$CA_CERT" >> "$COMBINED_CA"
echo "       CREATED — $COMBINED_CA"
echo "       (system CAs + DevKit CA)"
echo ""

# ---------------------------------------------------------------------------
# Step 6: Shell exports for NODE_EXTRA_CA_CERTS + SSL_CERT_FILE
# ---------------------------------------------------------------------------
SHELL_RC="$HOME/.zshrc"
NODE_EXPORT='export NODE_EXTRA_CA_CERTS="$HOME/.devkit-ssl/ca.pem"'
SSL_EXPORT='export SSL_CERT_FILE="$HOME/.devkit-ssl/combined-ca.pem"'

echo "[6/6] Shell exports"

NEEDS_NODE=true
NEEDS_SSL=true
if [ -f "$SHELL_RC" ]; then
    grep -qF 'NODE_EXTRA_CA_CERTS="$HOME/.devkit-ssl/ca.pem"' "$SHELL_RC" 2>/dev/null && NEEDS_NODE=false
    grep -qF 'SSL_CERT_FILE="$HOME/.devkit-ssl/combined-ca.pem"' "$SHELL_RC" 2>/dev/null && NEEDS_SSL=false
fi

if [ "$NEEDS_NODE" = false ] && [ "$NEEDS_SSL" = false ]; then
    echo "       EXISTS — exports already in $SHELL_RC"
else
    echo ""
    echo "       The following exports are needed in $SHELL_RC:"
    $NEEDS_NODE && echo "         $NODE_EXPORT"
    $NEEDS_SSL && echo "         $SSL_EXPORT"
    echo ""
    read -p "       Add to $SHELL_RC? [y/N] " add_exports
    if [ "$add_exports" = "y" ] || [ "$add_exports" = "Y" ]; then
        echo "" >> "$SHELL_RC"
        echo "# DevKit SSL — trust local CA in terminal tools" >> "$SHELL_RC"
        $NEEDS_NODE && echo "$NODE_EXPORT" >> "$SHELL_RC"
        $NEEDS_SSL && echo "$SSL_EXPORT" >> "$SHELL_RC"
        echo "       ADDED to $SHELL_RC"
        echo "       Run: source $SHELL_RC"
    else
        echo ""
        echo "       Add these lines to your shell rc manually:"
        echo "         $NODE_EXPORT"
        echo "         $SSL_EXPORT"
    fi
fi
echo ""

echo "============================================"
echo "  Done! Files in $CA_DIR:"
echo ""
echo "  CA:          $CA_CERT"
echo "  CA key:      $CA_KEY  (keep safe!)"
echo "  Combined:    $COMBINED_CA"
echo ""
echo "  Server cert: $SERVER_CERT"
echo "  Server key:  $SERVER_KEY"
echo ""
echo "  To uninstall:"
echo "    sudo security delete-certificate -c \"$CA_NAME\" /Library/Keychains/System.keychain"
echo "    rm -rf $CA_DIR $SSL_DIR"
echo "============================================"