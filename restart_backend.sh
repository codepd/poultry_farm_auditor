#!/bin/bash

# Script to restart the Go backend API

echo "Restarting Go Backend API..."
echo ""

# Stop existing processes
echo "Stopping existing Go processes..."
pkill -f "go run main.go" 2>/dev/null
pkill -f "main.go" 2>/dev/null
sleep 2

# Navigate to go_api directory
cd "$(dirname "$0")/go_api"

# Load environment variables
if [ -f ../.env ]; then
    source ../.env
    export $(cat ../.env | grep -v '^#' | xargs)
fi

# Start the server
echo "Starting Go API server on port ${API_PORT:-8080}..."
echo ""

go run main.go

