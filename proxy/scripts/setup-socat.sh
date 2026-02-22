#!/usr/bin/env bash
# =============================================================================
# setup-socat.sh — Forward port 443 → 7443 via socat + launchd
#
# Usage:
#   sudo ./setup-socat.sh              Install LaunchDaemon (survives reboots)
#   sudo ./setup-socat.sh --remove     Uninstall LaunchDaemon
#
# Port 443 is privileged, so the LaunchDaemon runs as root.
# This only needs sudo once — launchd manages the process from then on.
# =============================================================================

set -euo pipefail

LABEL="dev.kit.port-forward"
PLIST="/Library/LaunchDaemons/${LABEL}.plist"
SOCAT_BIN="$(command -v socat 2>/dev/null || true)"

# ---------------------------------------------------------------------------
# --remove: tear down
# ---------------------------------------------------------------------------
if [ "${1:-}" = "--remove" ]; then
    echo "============================================"
    echo "  devkit socat teardown"
    echo "============================================"
    echo ""

    if launchctl list "$LABEL" &>/dev/null; then
        echo "[1/2] Unloading LaunchDaemon..."
        launchctl unload "$PLIST" 2>/dev/null || true
        echo "       DONE"
    else
        echo "[1/2] LaunchDaemon not loaded — skipping"
    fi
    echo ""

    if [ -f "$PLIST" ]; then
        echo "[2/2] Removing plist: ${PLIST}"
        rm -f "$PLIST"
        echo "       DONE"
    else
        echo "[2/2] No plist found — skipping"
    fi
    echo ""

    echo "============================================"
    echo "  Removed. Port 443 is no longer forwarded."
    echo "============================================"
    exit 0
fi

# ---------------------------------------------------------------------------
# Install
# ---------------------------------------------------------------------------
echo "============================================"
echo "  devkit socat setup — port 443 → 7443"
echo "============================================"
echo ""

# ---------------------------------------------------------------------------
# Step 1: Check socat
# ---------------------------------------------------------------------------
echo "[1/3] Checking for socat..."

if [ -z "$SOCAT_BIN" ]; then
    echo "       socat not found."
    echo ""
    echo "       Install with:  brew install socat"
    echo ""
    exit 1
fi
echo "       Found: ${SOCAT_BIN}"
echo ""

# ---------------------------------------------------------------------------
# Step 2: Write LaunchDaemon plist
# ---------------------------------------------------------------------------
echo "[2/3] LaunchDaemon: ${PLIST}"

if [ -f "$PLIST" ]; then
    echo "       EXISTS — plist already installed"
else
    cat > "$PLIST" <<PLIST_EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>${LABEL}</string>
    <key>ProgramArguments</key>
    <array>
        <string>${SOCAT_BIN}</string>
        <string>TCP-LISTEN:443,fork,reuseaddr</string>
        <string>TCP:127.0.0.1:7443</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/dev.kit.port-forward.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/dev.kit.port-forward.log</string>
</dict>
</plist>
PLIST_EOF
    echo "       CREATED"
fi
echo ""

# ---------------------------------------------------------------------------
# Step 3: Load the daemon
# ---------------------------------------------------------------------------
echo "[3/3] Loading LaunchDaemon..."

# Unload first in case it's already loaded with stale config
launchctl unload "$PLIST" 2>/dev/null || true
launchctl load "$PLIST"
echo "       LOADED"
echo ""

# ---------------------------------------------------------------------------
# Verify
# ---------------------------------------------------------------------------
echo "--- Verification ---"
echo ""

if launchctl list "$LABEL" &>/dev/null; then
    echo "  LaunchDaemon: running"
else
    echo "  LaunchDaemon: NOT running (check /tmp/dev.kit.port-forward.log)"
fi

sleep 1
if curl -sk --max-time 2 "https://localhost/" -o /dev/null 2>/dev/null; then
    echo "  Port 443:     responding (proxy is running)"
else
    echo "  Port 443:     not responding (proxy may not be running yet — that's OK)"
fi
echo ""

echo "============================================"
echo "  Done! Port 443 → 7443 (survives reboots)"
echo ""
echo "  Remove: sudo ./scripts/setup-socat.sh --remove"
echo "  Logs:   /tmp/dev.kit.port-forward.log"
echo "============================================"
