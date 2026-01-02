#!/bin/bash

# Script to run Go backend with Delve debugger
# Usage: ./debug_backend.sh [port]
# Default debug port: 2345

DEBUG_PORT=${1:-2345}

echo "Starting Go API with Delve debugger..."
echo "Debug port: $DEBUG_PORT"
echo "Connect your debugger to: localhost:$DEBUG_PORT"
echo ""

cd "$(dirname "$0")/go_api"

# Load environment variables
if [ -f ../.env ]; then
    source ../.env
    export $(cat ../.env | grep -v '^#' | xargs)
fi

# Check if delve is installed
if ! command -v dlv &> /dev/null; then
    echo "❌ Delve (dlv) is not installed!"
    echo ""
    echo "Install it with:"
    echo "  go install github.com/go-delve/delve/cmd/dlv@latest"
    echo ""
    exit 1
fi

# Run with delve
dlv debug --listen=:${DEBUG_PORT} --headless=true --api-version=2 --accept-multiclient main.go

