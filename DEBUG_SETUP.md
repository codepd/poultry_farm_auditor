# Go Backend Debugging Setup

This guide explains how to debug the Go backend API.

## Prerequisites

1. **Install Delve (Go debugger)**:
   ```bash
   go install github.com/go-delve/delve/cmd/dlv@latest
   ```

2. **Verify installation**:
   ```bash
   dlv version
   ```

## Debugging Methods

### Method 1: VS Code Debugger (Recommended)

1. **Open the project in VS Code**
2. **Install the Go extension** (if not already installed)
   - Extension ID: `golang.go`
3. **Set breakpoints** in your Go code by clicking in the gutter
4. **Start debugging**:
   - Press `F5` or
   - Go to Run and Debug (Ctrl+Shift+D / Cmd+Shift+D)
   - Select "Launch Go API" from the dropdown
   - Click the green play button

The debugger will:
- Start the API server
- Stop at breakpoints
- Allow you to inspect variables
- Step through code

### Method 2: Command Line with Delve

1. **Run the debug script**:
   ```bash
   ./debug_backend.sh
   ```

2. **In another terminal, attach to the debugger**:
   ```bash
   dlv connect localhost:2345
   ```

3. **Use Delve commands**:
   - `break main.go:50` - Set breakpoint at line 50
   - `continue` or `c` - Continue execution
   - `next` or `n` - Step to next line
   - `step` or `s` - Step into function
   - `print variable` - Print variable value
   - `locals` - Show local variables
   - `args` - Show function arguments
   - `exit` or `quit` - Exit debugger

### Method 3: Attach to Running Process

1. **Start the API normally**:
   ```bash
   cd go_api
   go run main.go
   ```

2. **In VS Code**:
   - Go to Run and Debug
   - Select "Attach to Go API"
   - Enter the process ID when prompted

## Debug Configuration Files

- **`.vscode/launch.json`** - VS Code debug configurations
- **`go_api/.vscode/launch.json`** - Go API specific configuration
- **`debug_backend.sh`** - Script to run with Delve

## Environment Variables

The debug configurations load environment variables from `.env` file. Make sure it exists:
```bash
DB_HOST=localhost
DB_PORT=5432
DB_NAME=poultry_farm
DB_USER=postgres
DB_PASSWORD=postgres
API_PORT=8080
JWT_SECRET=change-this-secret-key-in-production
```

## Common Debugging Scenarios

### Debug Login Flow

1. Set breakpoint in `go_api/handlers/auth.go` at line 39 (Login function)
2. Set breakpoint at line 73 (password verification)
3. Start debugger
4. Try to login from frontend
5. Step through the authentication flow

### Debug CORS Issues

1. Set breakpoint in `go_api/middleware/cors.go` at line 8
2. Inspect the `origin` header
3. Check which CORS headers are being set

### Debug Database Queries

1. Set breakpoint in any handler function
2. Inspect database query results
3. Check for SQL errors

## Tips

- **Conditional breakpoints**: Right-click on a breakpoint to add conditions
- **Watch expressions**: Add variables to watch panel
- **Call stack**: View the call stack to understand execution flow
- **Debug console**: Execute Go expressions in the debug console

## Troubleshooting

### "dlv: command not found"
Install Delve:
```bash
go install github.com/go-delve/delve/cmd/dlv@latest
```

### "Cannot find package"
Make sure you're in the correct directory and dependencies are installed:
```bash
cd go_api
go mod download
```

### Debugger not attaching
- Check if the port (2345) is available
- Make sure the API is running with debug flags
- Verify firewall settings

## Resources

- [Delve Documentation](https://github.com/go-delve/delve)
- [VS Code Go Extension](https://marketplace.visualstudio.com/items?itemName=golang.go)

