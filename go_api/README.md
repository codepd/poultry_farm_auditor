# Go API - Poultry Farm Management System

## Setup

1. **Install Dependencies**:
   ```bash
   cd go_api
   go mod tidy
   ```

2. **Set Environment Variables**:
   ```bash
   export DB_HOST=localhost
   export DB_PORT=5432
   export DB_NAME=poultry_farm
   export DB_USER=postgres
   export DB_PASSWORD=postgres
   export API_PORT=8080
   export JWT_SECRET=your-secret-key-here
   export UPLOAD_PATH=./uploads
   ```

3. **Run Server**:
   ```bash
   go run main.go
   ```

## API Endpoints

### Authentication
- `POST /api/auth/login` - User login

### Tenants
- `GET /api/tenants` - List tenants (with hierarchy)
- `GET /api/tenants/{id}` - Get tenant
- `POST /api/tenants` - Create child tenant
- `PUT /api/tenants/{id}` - Update tenant

### Transactions
- `GET /api/transactions` - List transactions (with filters)
- `GET /api/transactions/{id}` - Get transaction
- `POST /api/transactions` - Create transaction
- `PUT /api/transactions/{id}` - Update transaction
- `DELETE /api/transactions/{id}` - Delete transaction
- `POST /api/transactions/{id}/submit` - Submit for approval
- `POST /api/transactions/{id}/approve` - Approve transaction
- `POST /api/transactions/{id}/reject` - Reject transaction

### Hen Batches
- `GET /api/hen-batches` - List hen batches
- `GET /api/hen-batches/{id}` - Get hen batch
- `POST /api/hen-batches` - Create hen batch
- `PUT /api/hen-batches/{id}` - Update hen batch
- `POST /api/hen-batches/mortality` - Record mortality

### Employees
- `GET /api/employees` - List employees
- `GET /api/employees/{id}` - Get employee
- `POST /api/employees` - Create employee
- `PUT /api/employees/{id}` - Update employee

### User Management
- `GET /api/users` - List users for tenant
- `POST /api/users/invite` - Invite user
- `GET /api/users/invitations` - List invitations
- `POST /api/users/accept-invite` - Accept invitation (public)

### Analytics
- `GET /api/analytics/enhanced-monthly-summary?year={year}&month={month}` - Monthly summary
- `GET /api/analytics/all-years-summary` - Yearly summaries

### Sensitive Data Config
- `GET /api/sensitive-data-config` - Get config
- `PUT /api/sensitive-data-config` - Update config

### Receipts
- `GET /api/transactions/{transaction_id}/receipts` - List receipts
- `POST /api/transactions/{transaction_id}/receipts` - Upload receipt

## Authentication

All endpoints except `/api/auth/login` and `/api/users/accept-invite` require authentication.

Include JWT token in Authorization header:
```
Authorization: Bearer <token>
```

## Response Format

Success:
```json
{
  "success": true,
  "data": {...}
}
```

Error:
```json
{
  "error": "Error message"
}
```


