#!/bin/bash

# Kill existing processes
echo "Stopping existing services..."
pkill -f "go run main.go" 2>/dev/null
pkill -f "react-scripts" 2>/dev/null
pkill -f "node.*react" 2>/dev/null
sleep 2

# Start Go backend
echo "Starting Go backend..."
cd go_api
source ../.env 2>/dev/null
export $(cat ../.env 2>/dev/null | grep -v '^#' | xargs) 2>/dev/null
nohup go run main.go > /tmp/go_api.log 2>&1 &
echo "Go backend started (PID: $!)"
cd ..

# Start React frontend
echo "Starting React frontend..."
cd react_frontend
nohup npm start > /tmp/react_frontend.log 2>&1 &
echo "React frontend started (PID: $!)"
cd ..

# Wait a bit for services to start
echo "Waiting for services to start..."
sleep 5

# Check if services are running
if pgrep -f "go run main.go" > /dev/null; then
    echo "✓ Go backend is running"
else
    echo "✗ Go backend failed to start. Check /tmp/go_api.log"
fi

if pgrep -f "react-scripts" > /dev/null; then
    echo "✓ React frontend is running"
else
    echo "✗ React frontend failed to start. Check /tmp/react_frontend.log"
fi

echo ""
echo "Backend logs (last 10 lines):"
tail -10 /tmp/go_api.log 2>/dev/null || echo "No backend logs yet"

echo ""
echo "Frontend logs (last 10 lines):"
tail -10 /tmp/react_frontend.log 2>/dev/null || echo "No frontend logs yet"

echo ""
echo "Opening browser in 3 seconds..."
sleep 3
open http://localhost:4300




