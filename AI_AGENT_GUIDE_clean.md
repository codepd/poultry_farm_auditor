# Poultry Farm Management System - AI Agent Guide

**Purpose**: This guide helps AI coding assistants (Cursor, GitHub Copilot, etc.) understand the project architecture, features, components, and remaining work.

**Last Updated**: December 2024

---

## Table of Contents

1. [System Architecture Overview](#system-architecture-overview)
2. [Technology Stack](#technology-stack)
3. [Database Schema](#database-schema)
4. [Python Backend](#python-backend)
5. [Go REST API](#go-rest-api)
6. [React Frontend](#react-frontend)
7. [Data Flow](#data-flow)
8. [CLI Tools Reference](#cli-tools-reference)
9. [Key Features Implemented](#key-features-implemented)
10. [TODO: Remaining Work](#todo-remaining-work)
11. [Development Guidelines](#development-guidelines)

---

## System Architecture Overview

**Three-Tier Architecture:**

```
┌─────────────────┐
│  React Frontend │  (TypeScript/TSX, Recharts, React Router)
│   Port: 3000     │
└────────┬────────┘
         │ HTTP/REST
         ▼
┌─────────────────┐
│   Go REST API   │  (Gorilla Mux, PostgreSQL driver)
│   Port: 8080     │
└────────┬────────┘
         │ SQL
         ▼
┌─────────────────┐
│  PostgreSQL DB  │  (Docker container)
│   Port: 5432     │
└─────────────────┘
         ▲
         │ Direct SQL
         │
┌─────────────────┐
│ Python Backend   │  (PDF parsing, data import, analytics)
│  (CLI Tools)     │
└─────────────────┘
```

**Key Design Principles:**
- **Python**: Handles PDF parsing, data extraction, and direct database population
- **Go**: Provides REST APIs for frontend consumption
- **React**: Full-featured web application for visualization and data management
- **PostgreSQL**: Single source of truth for all data

---

## Technology Stack

### Python Backend
- **Python 3.7+**
- **Libraries**:
  - `psycopg2-binary`: PostgreSQL connection
  - `pdfplumber`: PDF text extraction
  - `pandas`: Data processing
  - `openpyxl`: Excel file handling
  - `matplotlib` & `seaborn`: Data visualization
  - `pytesseract` & `Pillow`: OCR for image-based price imports

### Go API
- **Go 1.19+**
- **Frameworks**:
  - `gorilla/mux`: HTTP router
  - `lib/pq`: PostgreSQL driver
  - `database/sql`: Standard database interface

### React Frontend
- **React 19+** with **TypeScript**
- **Libraries**:
  - `react-router-dom`: Navigation
  - `axios`: HTTP client
  - `recharts`: Chart visualization
  - `react-scripts`: Build tooling

### Database
- **PostgreSQL 14+** (Docker container)
- Connection pooling
- Database triggers for data integrity

---

## Database Schema

### Core Tables

#### `tenants`
Multi-tenant support.
- `id` (SERIAL PRIMARY KEY)
- `name` (VARCHAR)
- `created_at` (TIMESTAMP)

#### `transactions`
All financial transactions (sales, purchases, payments, etc.).
- `id` (SERIAL PRIMARY KEY)
- `tenant_id` (INTEGER, FK to tenants)
- `transaction_date` (DATE)
- `transaction_type` (ENUM: 'SALE', 'PURCHASE', 'PAYMENT', 'TDS', 'DISCOUNT')
- `category` (ENUM: 'EGG', 'FEED', 'MEDICINE', 'OTHER')
- `item_name` (VARCHAR) - e.g., "LARGE EGG", "LAYER MASH"
- `quantity` (DECIMAL)
- `unit` (VARCHAR) - e.g., "NOS", "KGS"
- `rate` (DECIMAL)
- `amount` (DECIMAL)
- `notes` (TEXT)
- `created_at`, `updated_at` (TIMESTAMP)

**Indexes**: `(tenant_id, transaction_date)`, `(tenant_id, category, transaction_date)`

#### `price_history`
Historical price tracking for feed and eggs.
- `id` (SERIAL PRIMARY KEY)
- `tenant_id` (INTEGER, FK to tenants)
- `price_date` (DATE)
- `price_type` (ENUM: 'EGG', 'FEED')
- `item_name` (VARCHAR) - e.g., "LAYER MASH", "LARGE EGG"
- `price` (DECIMAL)
- `created_at` (TIMESTAMP)
- **Unique constraint**: `(tenant_id, price_date, price_type, item_name)`

**Indexes**: `(tenant_id, price_type, price_date)`

#### `ledger_parses`
Tracks PDF parsing history and monthly summaries.
- `id` (SERIAL PRIMARY KEY)
- `tenant_id` (INTEGER, FK to tenants)
- `pdf_filename` (VARCHAR)
- `parse_date` (DATE)
- `month` (INTEGER, 1-12)
- `year` (INTEGER)
- `opening_balance`, `closing_balance` (DECIMAL)
- `total_eggs`, `total_feeds`, `total_medicines` (DECIMAL)
- `net_profit` (DECIMAL)
- `eggs_large_qty`, `eggs_medium_qty`, `eggs_small_qty` (DECIMAL) - Quantity summaries
- `feeds_total_kg` (DECIMAL) - Total feed quantity in kg
- `parsed_at` (TIMESTAMP)
- **Unique constraint**: `(tenant_id, year, month, pdf_filename)`

**Indexes**: `(tenant_id, year, month)`

#### `ledger_breakdowns`
Detailed quantity breakdowns by type (eggs: large/medium/small, feeds: layer/grower/pre-layer/chick).
- `id` (SERIAL PRIMARY KEY)
- `ledger_parse_id` (INTEGER, FK to ledger_parses)
- `breakdown_type` (VARCHAR) - e.g., "EGG_LARGE", "EGG_MEDIUM", "FEED_LAYER_MASH"
- `quantity` (DECIMAL)
- `created_at`, `updated_at` (TIMESTAMP)
- **Unique constraint**: `(ledger_parse_id, breakdown_type)`

**Indexes**: `(ledger_parse_id)`, `(breakdown_type)`

### Database Triggers

**Automatic Updates**: When transactions are inserted/updated/deleted, triggers automatically:
1. Recalculate `ledger_parses` totals (total_eggs, total_feeds, etc.)
2. Update quantity summaries (eggs_large_qty, eggs_medium_qty, etc.)
3. Update `ledger_breakdowns` with detailed breakdowns

**Function**: `update_ledger_parses_from_transactions()` - Triggered on transaction changes.

---

## Python Backend

### Directory Structure

```
python_backend/
├── database/
│   ├── connection.py      # Connection pooling, database initialization
│   └── schema.py          # SQL DDL for all tables, enums, triggers
├── parsers/
│   ├── pdf_parser.py     # PDF text extraction and parsing
│   ├── db_importer.py    # Import parsed data to database
│   ├── excel_importer.py # Import validated Excel files
│   └── alternative_excel_importer.py  # Import Numbers-exported Excel files
├── utils/
│   ├── egg_normalizer.py  # Normalize egg item names
│   ├── feed_normalizer.py # Normalize feed item names
│   └── validators.py      # Data validation utilities
└── cli/
    ├── parse_and_import.py              # Parse PDF → Excel → (optional) DB
    ├── import_excel.py                  # Import validated Excel to DB
    ├── batch_import_historical.py       # Batch import PDFs/Excel files
    ├── import_price_history.py          # Import price history from Excel
    ├── import_feed_prices_from_images.py # OCR-based price import from images
    ├── visualize_data.py                # Generate charts from database
    ├── analyze_feed_price_changes.py    # Compare feed prices between periods
    ├── query_price_history.py           # Query and display price history
    ├── normalize_egg_names.py           # Normalize existing egg names in DB
    ├── normalize_feed_names.py         # Normalize existing feed names in DB
    ├── update_quantities.py             # Recalculate quantity summaries
    └── remove_duplicates.py            # Remove duplicate transactions
```

### Key Components

#### PDF Parser (`parsers/pdf_parser.py`)
- Extracts text from PDF ledger statements using `pdfplumber`
- Identifies date headers, item lines, payments, TDS, discounts
- Categorizes items (EGG, FEED, MEDICINE, OTHER)
- Handles various date formats (DD-MMM-YY, DD-MMM-YYYY)
- Returns structured data dictionary

#### Database Importer (`parsers/db_importer.py`)
- Inserts transactions into `transactions` table
- Creates `ledger_parses` records
- Normalizes item names before insertion
- Handles duplicates (checks `ledger_parses` before importing)
- Uses transactions for atomicity

#### Excel Importer (`parsers/excel_importer.py`)
- Reads validated Excel files (from PDF parsing)
- Imports transactions, payments, TDS, discounts
- Populates `ledger_breakdowns` with detailed quantities
- Updates `ledger_parses` with summary quantities
- Prevents duplicate imports

#### Alternative Excel Importer (`parsers/alternative_excel_importer.py`)
- Handles Excel files exported from Apple Numbers
- Different sheet structure than PDF parser output
- Extracts from "Egg Summary", "Feed", "Balance Sheet" sheets
- Used for importing historical data from Numbers files

### Data Normalization

**Egg Names**: "CORRECT EGG", "EXPORT EGG", "CORRECT SIZE" → "LARGE EGG"

**Feed Names**: 
- "LAYER MASH BULK" → "LAYER MASH"
- "GROWER MASH BULK" → "GROWER MASH"
- "PRE LAYER MASH BULK" → "PRE-LAYER MASH"

**Feed Types Tracked**:
- LAYER MASH
- GROWER MASH
- PRE-LAYER MASH
- CHICK MASH (newly added)

---

## Go REST API

### Directory Structure

```
go_api/
├── main.go              # Application entry point, route setup
├── config/
│   └── config.go        # Configuration (DB connection, server port)
├── database/
│   └── postgres.go     # Database connection and query helpers
├── models/
│   ├── transaction.go  # Transaction struct and methods
│   ├── price_history.go # Price history struct
│   ├── analytics.go     # Analytics response structs
│   └── tenant.go       # Tenant struct
├── handlers/
│   ├── transactions.go # Transaction CRUD handlers
│   ├── prices.go        # Price CRUD handlers
│   └── analytics.go    # Analytics endpoint handlers
└── middleware/
    ├── cors.go         # CORS middleware
    └── logging.go      # Request logging middleware
```

### API Endpoints

#### Analytics Endpoints
- `GET /api/analytics/monthly-summary?tenant_id={id}&year={year}` - Monthly aggregated data
- `GET /api/analytics/yearly-summary?tenant_id={id}&year={year}` - Yearly summary
- `GET /api/analytics/price-trends?tenant_id={id}&price_type={egg|feed}&start_date={date}&end_date={date}` - Price trends
- `GET /api/analytics/category-breakdown?tenant_id={id}&year={year}&category={egg|feed}` - Breakdown by item type
- `GET /api/analytics/profit-loss?tenant_id={id}&year={year}` - Profit/loss analysis

#### Transaction CRUD
- `GET /api/transactions?tenant_id={id}&start_date={date}&end_date={date}&category={category}` - List transactions
- `POST /api/transactions` - Create transaction (eggs sold, feed purchased)
- `PUT /api/transactions/{id}` - Update transaction
- `DELETE /api/transactions/{id}` - Delete transaction

#### Price CRUD
- `GET /api/prices?tenant_id={id}&price_type={egg|feed}&start_date={date}&end_date={date}` - List prices
- `POST /api/prices` - Add/update price (feed price, egg price)
- `PUT /api/prices/{id}` - Update price
- `DELETE /api/prices/{id}` - Delete price

#### Health Check
- `GET /health` - Server health check

### Response Format

```json
{
  "success": true,
  "data": {...},
  "error": null
}
```

---

## React Frontend

### Directory Structure

```
react_frontend/
├── src/
│   ├── components/
│   │   ├── Dashboard/
│   │   │   ├── MonthlySummary.tsx      # Monthly metrics cards
│   │   │   └── Charts/
│   │   │       ├── MonthlyBarChart.tsx # Monthly bar charts
│   │   │       └── PriceTrendChart.tsx  # Price trend line charts
│   │   ├── DataEntry/
│   │   │   ├── AddTransaction.tsx      # Form to add eggs sold, feed purchased
│   │   │   └── AddPrice.tsx            # Form to add/update feed/egg prices
│   │   └── Layout/
│   │       └── Layout.tsx              # Main layout with navigation
│   ├── pages/
│   │   ├── DashboardPage.tsx           # Main dashboard
│   │   ├── TransactionsPage.tsx        # Transaction management
│   │   ├── PricesPage.tsx               # Price management
│   │   └── DataEntryPage.tsx           # Data entry forms
│   ├── services/
│   │   └── api.ts                      # Axios instance and API calls
│   └── hooks/
│       ├── useAnalytics.ts             # Custom hook for analytics data
│       └── useTransactions.ts          # Custom hook for transactions
└── package.json
```

### Features

#### Dashboard
- Monthly/yearly summary cards (total eggs sold, feed purchased, profit)
- Interactive charts (bar, line, stacked)
- Price trend visualizations
- Category breakdowns

#### Data Entry
- Add eggs sold (quantity, unit, rate, date)
- Add feed purchased (item name, quantity, rate, date)
- Add/update feed prices (LAYER MASH, GROWER MASH, PRE-LAYER MASH, CHICK MASH)
- Add/update egg prices (LARGE, MEDIUM, SMALL)

#### Management
- View all transactions with filters
- Edit/delete transactions
- View price history
- Upload PDF for parsing (future: via API)

---

## Data Flow

### PDF → Excel → Database Workflow

1. **Parse PDF** (`parse_and_import.py`):
   - Extract text from PDF using `pdfplumber`
   - Parse transactions, payments, TDS, discounts
   - Export to Excel for validation

2. **Validate Excel** (Manual):
   - Review `raw_rows` sheet
   - Check `ambiguous_rows` for parsing issues
   - Correct errors in Excel

3. **Import to Database** (`import_excel.py`):
   - Read validated Excel
   - Insert transactions
   - Create `ledger_parses` record
   - Populate `ledger_breakdowns`

### Price History Import

1. **From Excel** (`import_price_history.py`):
   - Read price history Excel file
   - Extract feed/egg prices by date
   - Import to `price_history` table

2. **From Images** (`import_feed_prices_from_images.py`):
   - Use OCR (Tesseract) to extract text from images
   - Parse dates and prices from image text
   - Import to `price_history` table

### Database → API → Frontend

1. **Go API** queries PostgreSQL
2. **React Frontend** calls Go API endpoints
3. **Charts** render using Recharts

---

## CLI Tools Reference

### Data Import Tools

#### `parse_and_import.py`
Parse PDF and export to Excel (optionally import to DB).

```bash
python_backend/venv/bin/python3 python_backend/cli/parse_and_import.py <pdf_path> --tenant-id 1 [--output-dir <dir>] [--direct-import]
```

**Options**:
- `--tenant-id`: Tenant ID (required)
- `--output-dir`: Directory for Excel output (default: same as PDF)
- `--direct-import`: Also import to database (skips validation step)

#### `import_excel.py`
Import validated Excel file to database.

```bash
python_backend/venv/bin/python3 python_backend/cli/import_excel.py <excel_path> --tenant-id 1
```

#### `batch_import_historical.py`
Batch import historical data (prioritizes PDFs, falls back to Excel).

```bash
python_backend/venv/bin/python3 python_backend/cli/batch_import_historical.py --ledger-dir <dir> --excel-dir <dir> --tenant-id 1
```

### Price Management Tools

#### `import_price_history.py`
Import price history from Excel file.

```bash
python_backend/venv/bin/python3 python_backend/cli/import_price_history.py <excel_file> --tenant-id 1
```

#### `import_feed_prices_from_images.py`
Extract prices from images using OCR.

```bash
python_backend/venv/bin/python3 python_backend/cli/import_feed_prices_from_images.py [image_path|--folder <dir>] --tenant-id 1 [--verbose]
```

#### `analyze_feed_price_changes.py`
Compare feed prices between two periods.

```bash
python_backend/venv/bin/python3 python_backend/cli/analyze_feed_price_changes.py --tenant-id 1 --start-year 2024 --start-month 6 --end-year 2024 --end-month 11
```

#### `query_price_history.py`
Query and display price history.

```bash
python_backend/venv/bin/python3 python_backend/cli/query_price_history.py --tenant-id 1 [--price-type FEED|EGG] [--item-name "LAYER MASH"] [--start-date YYYY-MM-DD] [--end-date YYYY-MM-DD] [--format table|csv|summary]
```

### Data Maintenance Tools

#### `normalize_egg_names.py`
Normalize egg item names in existing transactions.

```bash
python_backend/venv/bin/python3 python_backend/cli/normalize_egg_names.py --tenant-id 1
```

#### `normalize_feed_names.py`
Normalize feed item names in existing transactions.

```bash
python_backend/venv/bin/python3 python_backend/cli/normalize_feed_names.py --tenant-id 1
```

#### `update_quantities.py`
Recalculate quantity summaries in `ledger_parses`.

```bash
python_backend/venv/bin/python3 python_backend/cli/update_quantities.py --tenant-id 1
```

#### `remove_duplicates.py`
Remove duplicate transactions.

```bash
python_backend/venv/bin/python3 python_backend/cli/remove_duplicates.py --tenant-id 1
```

### Visualization Tools

#### `visualize_data.py`
Generate charts from database data.

```bash
python_backend/venv/bin/python3 python_backend/cli/visualize_data.py --tenant-id 1 [--start-year YYYY] [--end-year YYYY] [--output-dir <dir>]
```

**Output**: Creates PNG charts:
- `monthly_trends.png`: Monthly trends for eggs, feeds, profit
- `egg_breakdown.png`: Egg quantities by type
- `feed_breakdown.png`: Feed purchases by type
- `summary_dashboard.png`: Comprehensive dashboard

---

## Key Features Implemented

### ✅ Completed Features

1. **PDF Parsing**
   - Extract transactions from PDF ledger statements
   - Categorize items (EGG, FEED, MEDICINE, OTHER)
   - Parse payments, TDS, discounts
   - Export to Excel for validation

2. **Database Import**
   - Import from PDF (via Excel validation step)
   - Import from Excel files
   - Import from Numbers-exported Excel files
   - Batch import historical data

3. **Price History Management**
   - Import from Excel files
   - Import from images using OCR
   - Track feed prices (LAYER, GROWER, PRE-LAYER, CHICK MASH)
   - Track egg prices (LARGE, MEDIUM, SMALL)

4. **Data Normalization**
   - Normalize egg names (CORRECT EGG → LARGE EGG)
   - Normalize feed names (LAYER MASH BULK → LAYER MASH)
   - Automatic normalization on import

5. **Quantity Tracking**
   - Track egg quantities by type (large, medium, small)
   - Track feed quantities by type (layer, grower, pre-layer, chick)
   - Automatic updates via database triggers

6. **Data Integrity**
   - Database triggers auto-update summaries on transaction changes
   - Duplicate prevention
   - Transaction rollback on errors

7. **Go REST API**
   - Analytics endpoints (monthly/yearly summaries, price trends, breakdowns)
   - Transaction CRUD operations
   - Price CRUD operations
   - CORS and logging middleware

8. **React Frontend**
   - Dashboard with charts
   - Data entry forms
   - Transaction management
   - Price management

9. **Visualization**
   - Python-based chart generation
   - Monthly trends, breakdowns, summaries

10. **Historical Data Import**
    - Import from PDFs (2022-2025)
    - Import from Numbers files (converted to Excel)
    - Prioritize PDFs over Excel for duplicates

---

## TODO: Remaining Work

### High Priority

#### 1. Go API Enhancements
- [ ] **Update analytics handlers to use `ledger_breakdowns` table**
  - Currently analytics may not use detailed breakdowns
  - Should query `ledger_breakdowns` for egg type and feed type breakdowns
  - Update `handlers/analytics.go` to include breakdown data

- [ ] **Add endpoint for ledger history**
  - `GET /api/ledgers?tenant_id={id}&year={year}&month={month}`
  - Return `ledger_parses` data with breakdowns
  - Include parsing metadata (pdf_filename, parse_date)

- [ ] **Add PDF upload endpoint**
  - `POST /api/upload-pdf` - Upload PDF file
  - Call Python backend to parse (via subprocess or HTTP)
  - Return parsed data or Excel file
  - **Alternative**: Keep Python CLI separate, document integration

- [ ] **Add authentication/authorization**
  - JWT-based authentication
  - Multi-tenant isolation
  - User roles (admin, viewer, etc.)

- [ ] **Add error handling and validation**
  - Input validation for all endpoints
  - Proper error responses
  - Request/response logging

#### 2. React Frontend Enhancements
- [ ] **Complete dashboard implementation**
  - Ensure all charts are connected to API
  - Add date range filters
  - Add export functionality (CSV, PDF)

- [ ] **Add ledger history page**
  - Display parsed ledger history
  - Show breakdowns by type
  - Allow re-import or correction

- [ ] **Add PDF upload component**
  - Upload PDF files
  - Show parsing progress
  - Display parsed results for validation
  - Option to import to database

- [ ] **Add image upload for feed prices**
  - Upload feed price images
  - Show OCR extraction results
  - Confirm before importing

- [ ] **Add advanced analytics page**
  - Year-over-year comparisons
  - Profit margin analysis
  - Feed cost per egg analysis
  - Price forecasting (optional)

- [ ] **Add data export functionality**
  - Export transactions to CSV/Excel
  - Export charts as images
  - Generate PDF reports

- [ ] **Improve error handling and user feedback**
  - Loading states
  - Error messages
  - Success notifications
  - Form validation

#### 3. Python Backend Enhancements
- [ ] **Add egg price import from images**
  - Extend `import_feed_prices_from_images.py` to handle egg prices
  - Or create separate `import_egg_prices_from_images.py`

- [ ] **Add batch price import from folder**
  - Process multiple price images in one command
  - Support both feed and egg prices

- [ ] **Add data validation and reporting**
  - Validate price history completeness
  - Report missing price data
  - Suggest corrections

- [ ] **Add automated testing**
  - Unit tests for parsers
  - Integration tests for database operations
  - Test data fixtures

### Medium Priority

#### 4. Database Enhancements
- [ ] **Add indexes for performance**
  - Review query performance
  - Add indexes as needed
  - Monitor slow queries

- [ ] **Add database migrations**
  - Use migration tool (golang-migrate, Alembic, etc.)
  - Version control schema changes
  - Rollback support

- [ ] **Add data backup/restore scripts**
  - Automated backups
  - Export/import utilities
  - Data archival

#### 5. Documentation
- [ ] **API documentation**
  - OpenAPI/Swagger specification
  - Endpoint documentation
  - Request/response examples

- [ ] **User guide**
  - How to use the system
  - Common workflows
  - Troubleshooting guide

- [ ] **Developer guide**
  - Setup instructions
  - Development workflow
  - Contributing guidelines

#### 6. Integration & Deployment
- [ ] **Dockerize Python backend**
  - Create Dockerfile for Python tools
  - Docker Compose for full stack
  - Environment configuration

- [ ] **CI/CD pipeline**
  - Automated testing
  - Build and deploy
  - Code quality checks

- [ ] **Production deployment guide**
  - Server setup
  - Database configuration
  - Security best practices

### Low Priority / Future Enhancements

#### 7. Advanced Features
- [ ] **Real-time updates**
  - WebSocket support for live data
  - Push notifications for new data

- [ ] **Mobile app**
  - React Native app
  - Quick data entry
  - Price alerts

- [ ] **Email reports**
  - Scheduled monthly/yearly reports
  - Price change alerts
  - Profit/loss summaries

- [ ] **Price forecasting**
  - ML-based price predictions
  - Trend analysis
  - Seasonal patterns

- [ ] **Multi-currency support**
  - Support for different currencies
  - Exchange rate tracking

- [ ] **Inventory management**
  - Track feed inventory
  - Low stock alerts
  - Purchase planning

---

## Development Guidelines

### Code Style

**Python**:
- Follow PEP 8
- Use type hints where possible
- Document functions with docstrings
- Use absolute imports from `python_backend`

**Go**:
- Follow Go conventions
- Use `gofmt` for formatting
- Document exported functions
- Handle errors explicitly

**TypeScript/React**:
- Use TypeScript strict mode
- Follow React best practices
- Use functional components with hooks
- Type all props and state

### Database Changes

1. **Schema Changes**:
   - Update `python_backend/database/schema.py`
   - Update `init.sql` if using Docker
   - Test migrations on development database first

2. **Trigger Changes**:
   - Update `CREATE_UPDATE_LEDGER_FUNCTION` in `schema.py`
   - Test trigger behavior with sample data
   - Document trigger logic

### Adding New Features

1. **Python CLI Tool**:
   - Create in `python_backend/cli/`
   - Use `get_db_cursor()` for database access
   - Add to `requirements.txt` if new dependencies
   - Document in this guide

2. **Go API Endpoint**:
   - Add handler in `go_api/handlers/`
   - Add route in `go_api/main.go`
   - Add model if needed in `go_api/models/`
   - Update API documentation

3. **React Component**:
   - Create in `react_frontend/src/components/`
   - Add API call in `react_frontend/src/services/api.ts`
   - Add route in `App.tsx`
   - Update navigation

### Testing

- **Unit Tests**: Test individual functions/components
- **Integration Tests**: Test database operations
- **E2E Tests**: Test full workflows (optional)

### Error Handling

- **Python**: Use try/except, log errors, rollback transactions
- **Go**: Return proper HTTP status codes, log errors
- **React**: Display user-friendly error messages

### Data Validation

- Validate all user inputs
- Check data types and ranges
- Sanitize inputs to prevent SQL injection
- Use parameterized queries

---

## Quick Reference

### Environment Variables

```bash
# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=poultry_farm
DB_USER=postgres
DB_PASSWORD=postgres

# Go API
API_PORT=8080

# React
REACT_APP_API_URL=http://localhost:8080
```

### Common Commands

```bash
# Start database
docker-compose up -d

# Parse PDF to Excel
python_backend/venv/bin/python3 python_backend/cli/parse_and_import.py ledger.pdf --tenant-id 1

# Import Excel to database
python_backend/venv/bin/python3 python_backend/cli/import_excel.py parsed_ledger.xlsx --tenant-id 1

# Import price history from Excel
python_backend/venv/bin/python3 python_backend/cli/import_price_history.py prices.xlsx --tenant-id 1

# Import prices from images
python_backend/venv/bin/python3 python_backend/cli/import_feed_prices_from_images.py --folder feed_price_history --tenant-id 1

# Generate visualizations
python_backend/venv/bin/python3 python_backend/cli/visualize_data.py --tenant-id 1

# Start Go API
cd go_api && go run main.go

# Start React frontend
cd react_frontend && npm start
```

---

## Notes for AI Agents

1. **Always check existing code** before creating new functionality
2. **Follow the established patterns** in each component
3. **Update this guide** when adding new features
4. **Test database operations** with sample data first
5. **Use connection pooling** for database access (Python)
6. **Handle errors gracefully** with proper logging
7. **Maintain data integrity** using database constraints and triggers
8. **Document new CLI tools** in this guide
9. **Keep dependencies minimal** - only add what's necessary
10. **Consider multi-tenant isolation** in all database queries

---

**End of Guide**

For questions or clarifications, refer to:
- `WORKFLOW.md` - Data import workflow
- `QUICK_START.md` - Quick setup guide
- `OCR_SETUP.md` - OCR setup for image imports
- Individual component READMEs


