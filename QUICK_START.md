# Quick Start Instructions

## Services Status

✅ **Frontend is running** - Browser opened to http://localhost:4300

## To Start Backend (if not running):

Open a terminal and run:

```bash
cd /Users/pradeep/Documents/poultry/automate_using_python/go_api
source ../.env
export $(cat ../.env | grep -v '^#' | xargs)
go run main.go
```

Or use the restart script:

```bash
cd /Users/pradeep/Documents/poultry/automate_using_python
./restart_backend.sh
```

## Verify Services

- **Frontend**: http://localhost:4300 (should show login page)
- **Backend**: http://localhost:8080/health (should return "OK")

## Login Credentials

Use your existing credentials:
- Email: ppradeep0610@gmail.com
- Password: P@sswd123!

## What's New

After logging in, you'll see:

1. **New Header**: Farm name (Pradeep Farm) on left, profile icon on right
2. **Navigation Bar**: Home, Expenses, Hen Batches
3. **Hen Age Display**: Shows all batches with age and head count on home page
4. **Expenses Page**: View and add miscellaneous expenses
5. **Hen Batches Page**: View and add hen batches (with permission checks)

## Troubleshooting

If services aren't running:

1. Check if ports are in use:
   ```bash
   lsof -i :8080  # Backend
   lsof -i :4300  # Frontend
   ```

2. Kill existing processes:
   ```bash
   pkill -f "go run main.go"
   pkill -f "react-scripts"
   ```

3. Restart using the scripts provided
