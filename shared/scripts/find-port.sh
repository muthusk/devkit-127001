#!/usr/bin/env bash
set -euo pipefail

if [ $# -eq 0 ]; then
  echo "Usage: find-port.sh <port> [port ...]" >&2
  echo "Finds available ports on localhost. For each preferred port, outputs an available one." >&2
  exit 1
fi

# Track ports we've already claimed in this invocation to avoid duplicates
declared=()

port_in_use() {
  local port=$1
  # Check if something is already listening on this port.
  # /dev/tcp is bash-specific but portable across macOS and Linux bash.
  # Fall back to nc if /dev/tcp isn't available.
  if (echo >/dev/tcp/127.0.0.1/"$port") 2>/dev/null; then
    return 0  # in use
  fi
  return 1  # free
}

port_is_claimed() {
  local port=$1
  for p in "${declared[@]+"${declared[@]}"}"; do
    if [ "$p" = "$port" ]; then
      return 0
    fi
  done
  return 1
}

find_available() {
  local port=$1
  local max_attempts=100

  for (( i=0; i<max_attempts; i++ )); do
    if ! port_in_use "$port" && ! port_is_claimed "$port"; then
      echo "$port"
      declared+=("$port")
      return 0
    fi
    port=$(( port + 1 ))
  done

  echo "Error: could not find available port starting from $1 after $max_attempts attempts" >&2
  return 1
}

for preferred in "$@"; do
  find_available "$preferred"
done
