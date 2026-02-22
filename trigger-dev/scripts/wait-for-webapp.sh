#!/usr/bin/env bash
URL="${1:-http://localhost:8030/healthcheck}"
MAX_WAIT="${2:-120}"

echo "⏳ Waiting for webapp at $URL (max ${MAX_WAIT}s)..."
elapsed=0
while [ $elapsed -lt $MAX_WAIT ]; do
  if curl -sf "$URL" > /dev/null 2>&1; then
    echo "✅ Webapp is ready! (${elapsed}s)"
    exit 0
  fi
  sleep 2
  elapsed=$((elapsed + 2))
  printf "."
done
echo ""
echo "❌ Webapp did not become healthy within ${MAX_WAIT}s"
exit 1
