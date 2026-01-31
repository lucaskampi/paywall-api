#!/bin/bash

# Load environment variables from .env file
# Only export lines with KEY=value format (skips comments and empty lines)
if [ -f .env ]; then
    while IFS='=' read -r key value; do
        # Skip if key is empty or starts with # or contains spaces
        if [[ -n "$key" && ! "$key" =~ ^# && ! "$key" =~ [[:space:]] ]]; then
            export "$key=$value"
        fi
    done < <(grep -E '^[A-Z_][A-Z0-9_]*=' .env)
fi

# If port 8080 is in use, kill the process(es) listening on it to free the port.
# Be cautious: this forcefully kills processes that may be serving on 8080.
if ss -ltnp 2>/dev/null | grep -q ':8080'; then
    echo "Port 8080 is in use — killing process(es) listening on :8080"
    ss -ltnp 2>/dev/null | grep ':8080' | awk '{print $6}' | grep -oP 'pid=\K[0-9]+' | xargs -r kill -9 || true
    sleep 0.5
fi

# Start the server
go run .
