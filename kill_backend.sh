#!/bin/bash

# Script to kill Go backend processes

echo "Killing Go backend processes..."

# Kill processes matching "go run main.go"
pkill -f "go run main.go" 2>/dev/null

# Kill processes matching "main.go"
pkill -f "main.go" 2>/dev/null

# Kill any delve debugger processes
pkill -f "dlv" 2>/dev/null

# Kill any processes listening on port 8080
lsof -ti:8080 | xargs kill -9 2>/dev/null

sleep 1

# Verify
if ps aux | grep -E "go run|main.go" | grep -v grep > /dev/null; then
    echo "⚠️  Some Go processes may still be running"
    ps aux | grep -E "go run|main.go" | grep -v grep
else
    echo "✅ All Go backend processes killed"
fi

# Check if port 8080 is still in use
if lsof -ti:8080 > /dev/null 2>&1; then
    echo "⚠️  Port 8080 is still in use"
    lsof -ti:8080
else
    echo "✅ Port 8080 is free"
fi

