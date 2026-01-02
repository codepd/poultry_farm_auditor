# Postman Collection Setup Guide

## Files Created

1. **Poultry_Farm_API.postman_collection.json** - Main API collection
2. **local_Pradeep_Farm.postman_environment.json** - Environment configuration

## Setup Instructions

### Step 1: Import Collection and Environment

1. Open Postman
2. Click **Import** button (top left)
3. Import both files:
   - `Poultry_Farm_API.postman_collection.json`
   - `local_Pradeep_Farm.postman_environment.json`

### Step 2: Select Environment

1. In the top-right corner of Postman, click the environment dropdown
2. Select **"local+Pradeep Farm"** environment

### Step 3: Verify Environment Variables

The environment should have these variables pre-configured:

- `base_url`: `http://localhost:8080/api`
- `tenant_id`: `8d7939f7-b716-4eb0-98d4-544c18c8dfb8`
- `tenant_name`: `Pradeep Farm`
- `user_email`: `test.employee1@gmail.com`
- `user_password`: `test123` (marked as secret)
- `jwt_token`: (empty, will be auto-filled after login)
- `user_id`: (empty, will be auto-filled after login)
- `last_transaction_id`: (empty, will be auto-filled after creating transactions)
- `last_hen_batch_id`: (empty, will be auto-filled after creating batches)

## Usage Flow

### 1. Login First

1. Navigate to **Authentication > Login**
2. Click **Send**
3. The JWT token will be automatically saved to the environment variable `jwt_token`
4. All subsequent requests will use this token for authentication

### 2. Test Transactions

1. **Create Transaction** - Creates a transaction and saves the ID to `last_transaction_id`
2. **Get Transaction by ID** - Uses the saved `last_transaction_id`
3. **Update Transaction** - Updates the transaction using `last_transaction_id`
4. **Delete Transaction** - Deletes the transaction using `last_transaction_id`

### 3. Test Hen Batches

1. **Create Hen Batch** - Creates a batch and saves the ID to `last_hen_batch_id`
2. **Get Hen Batch by ID** - Uses the saved `last_hen_batch_id`
3. **Update Hen Batch** - Updates the batch using `last_hen_batch_id`
4. **Delete Hen Batch** - Deletes the batch using `last_hen_batch_id`

## Automatic Variable Management

The collection includes **Test Scripts** that automatically:

- ✅ Save JWT token after login
- ✅ Save user ID after login
- ✅ Save transaction ID after creating a transaction
- ✅ Save hen batch ID after creating a batch

## Environment Variables Reference

| Variable | Description | Auto-Updated |
|----------|-------------|--------------|
| `base_url` | API base URL | No |
| `tenant_id` | Tenant UUID | No |
| `tenant_name` | Tenant name | No |
| `jwt_token` | JWT authentication token | ✅ Yes (Login) |
| `user_id` | Current user ID | ✅ Yes (Login) |
| `user_email` | User email for login | No |
| `user_password` | User password for login | No |
| `last_transaction_id` | Last created transaction ID | ✅ Yes (Create Transaction) |
| `last_hen_batch_id` | Last created hen batch ID | ✅ Yes (Create Hen Batch) |

## Testing Tips

1. **Always login first** - The JWT token is required for most endpoints
2. **Check console** - Postman console shows variable updates (View > Show Postman Console)
3. **Use environment variables** - All requests use `{{variable_name}}` syntax
4. **Update tenant info** - Edit environment if you need to test with different tenant

## Troubleshooting

### Token Not Saving
- Check Postman console for errors
- Verify login response contains `data.token`
- Ensure environment is selected

### 401 Unauthorized
- Make sure you've run Login request first
- Check that `jwt_token` environment variable is set
- Verify token hasn't expired

### 500 Internal Server Error
- Check backend logs
- Verify database connection
- Check request parameters match API expectations

## Creating Additional Environments

To create environments for other tenants:

1. Duplicate the `local_Pradeep_Farm` environment
2. Update `tenant_id` and `tenant_name`
3. Update `user_email` and `user_password` if needed
4. Name it as `local+<TenantName>`

