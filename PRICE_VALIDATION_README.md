# Price Validation Setup

## Step 1: Convert Numbers File to Excel

You need to convert `Poultry_Farm_Price_History_Details_2025.numbers` to Excel format first.

### Option A: Manual Export (Recommended)
1. Open `Poultry_Farm_Price_History_Details_2025.numbers` in Numbers app
2. Go to **File > Export To > Excel**
3. Save as `Poultry_Farm_Price_History_Details_2025.xlsx` in the same directory

### Option B: Using Python Script
```bash
python3 convert_numbers_to_excel.py
```

Note: This requires `numbers-parser` library which may have installation issues. Manual export is more reliable.

## Step 2: Excel File Structure

The Excel file should have at least these sheets:

### FEED PRICE Sheet
Should contain columns like:
- **Date** (or similar date column name)
- **LAYER MASH** (or LAYER MASH BULK)
- **GROWER MASH** (or GROWER MASH BULK)
- **PRE LAYER MASH** or **PLM** (or PRE LAYER MASH BULK)

Example:
| Date | LAYER MASH | GROWER MASH | PRE LAYER MASH |
|------|------------|-------------|----------------|
| 1-Jul-25 | 45.50 | 44.00 | 46.00 |
| 8-Jul-25 | 46.00 | 44.50 | 46.50 |

### EGG PRICE Sheet
Should contain columns like:
- **Date** (or similar date column name)
- **LARGE EGG**
- **MEDIUM EGG**
- **SMALL EGG**

Example:
| Date | LARGE EGG | MEDIUM EGG | SMALL EGG |
|------|-----------|------------|-----------|
| 1-Jul-25 | 7.50 | 7.00 | 6.50 |
| 8-Jul-25 | 7.75 | 7.25 | 6.75 |

## Step 3: Run Price Validation

After converting the Numbers file to Excel, run:

```bash
# Validate prices for July
python3 validate_prices.py parsed_JULY_25.xlsx

# Validate prices for August
python3 validate_prices.py parsed_AUG_25.xlsx

# Validate prices for September
python3 validate_prices.py parsed_SEP_25.xlsx
```

## How It Works

The validation script:
1. Loads price history from the Excel file
2. For each item in the parsed PDF:
   - Identifies the item type (feed/egg) and subtype (LAYER/GROWER/PLM or LARGE/MEDIUM/SMALL)
   - Looks up the expected price for that date
   - Compares with the rate in the PDF
3. Reports any mismatches (differences > 0.01 or > 1%)
4. Saves mismatches to a new Excel file: `parsed_<month>_price_validation.xlsx`

## Price Matching Logic

- **Feed Prices**: Matches based on feed type (LAYER MASH, GROWER MASH, PRE LAYER MASH) and uses the most recent price on or before the transaction date
- **Egg Prices**: Matches based on egg size (LARGE, MEDIUM, SMALL) and uses the most recent price on or before the transaction date
- **Tolerance**: Differences of 0.01 or less, or 1% or less, are considered acceptable (rounding differences)

## Output

If mismatches are found, you'll see:
- A table showing all mismatches with:
  - Date
  - Item name
  - Rate in PDF vs Expected rate
  - Difference amount and percentage
  - Transaction amount

The mismatches are also saved to an Excel file for further analysis.


