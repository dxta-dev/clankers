#!/usr/bin/env bash
set -euo pipefail

TEST_DIR=$(mktemp -d)
export CLANKERS_SOCKET_PATH="$TEST_DIR/clankers.sock"
export CLANKERS_DB_PATH="$TEST_DIR/clankers.db"

cleanup() {
    echo "Cleaning up..."
    if [ -n "${DAEMON_PID:-}" ]; then
        kill "$DAEMON_PID" 2>/dev/null || true
        wait "$DAEMON_PID" 2>/dev/null || true
    fi
    rm -rf "$TEST_DIR"
}
trap cleanup EXIT

echo "Starting daemon..."
echo "  Socket: $CLANKERS_SOCKET_PATH"
echo "  DB: $CLANKERS_DB_PATH"

if [ -x "./result-daemon/bin/clankers-daemon" ]; then
    DAEMON_BIN="./result-daemon/bin/clankers-daemon"
elif command -v clankers-daemon &>/dev/null; then
    DAEMON_BIN="clankers-daemon"
else
    echo "ERROR: clankers-daemon not found"
    exit 1
fi

"$DAEMON_BIN" &
DAEMON_PID=$!

for i in $(seq 1 30); do
    if [ -S "$CLANKERS_SOCKET_PATH" ]; then
        echo "Daemon ready after $i attempts"
        break
    fi
    sleep 0.1
done

if [ ! -S "$CLANKERS_SOCKET_PATH" ]; then
    echo "ERROR: Daemon failed to start (socket not found)"
    exit 1
fi

echo ""
echo "Running integration tests..."
pnpm exec tsx tests/integration.ts
