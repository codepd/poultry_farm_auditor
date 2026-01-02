# Starting Services

Due to shell environment issues, please run these commands manually in your terminal:

## 1. Stop Existing Services

```bash
pkill -f "go run main.go"
pkill -f "react-scripts"
```

## 2. Start Go Backend

```bash
cd go_api
source ../.env
export $(cat ../.env | grep -v '^#' | xargs)
go run main.go
```

Or run in background:
```bash
cd go_api
source ../.env
export $(cat ../.env | grep -v '^#' | xargs)
nohup go run main.go > /tmp/go_api.log 2>&1 &
```

## 3. Start React Frontend

In a new terminal window:

```bash
cd react_frontend
PORT=4300 npm start
```

Or run in background:
```bash
cd react_frontend
PORT=4300 nohup npm start > /tmp/react_frontend.log 2>&1 &
```

## 4. Open Browser

Once both services are running, open:
http://localhost:4300

## Quick Start Script

Alternatively, you can use the `restart_all.sh` script:

```bash
chmod +x restart_all.sh
./restart_all.sh
```

This will:
- Kill existing processes
- Start Go backend in background
- Start React frontend in background
- Open browser automatically



