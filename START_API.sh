#!/bin/bash

# Script to start the Go API server
# Usage: ./START_API.sh

cd "$(dirname "$0")/go_api"

echo "Starting Go API server..."
echo ""

# Load environment variables
if [ -f ../.env ]; then
    source ../.env
    export $(cat ../.env | grep -v '^#' | xargs)
fi

# Download Go dependencies if needed
echo "Checking Go dependencies..."
if [ ! -f go.sum ] || ! go mod download 2>/dev/null; then
    echo "Downloading Go dependencies..."
    go mod download
    go mod tidy
    echo "✅ Dependencies downloaded"
    echo ""
fi

# Start the server
echo "Starting server on port ${API_PORT:-8080}..."
echo ""
go run main.go

