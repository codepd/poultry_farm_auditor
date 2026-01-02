# Feed Price Image Processing

This script processes feed price images (like Everest Feeds rate sheets) and loads the prices into the database.

## Prerequisites

1. **Install Tesseract OCR:**
   ```bash
   # macOS
   brew install tesseract
   
   # Ubuntu/Debian
   sudo apt-get install tesseract-ocr
   
   # Windows
   # Download from: https://github.com/UB-Mannheim/tesseract/wiki
   ```

2. **Install Python dependencies:**
   ```bash
   cd python_backend
   source venv/bin/activate  # or venv\Scripts\activate on Windows
   pip install pytesseract pillow
   ```

## Usage

### Basic Usage
```bash
python python_backend/cli/process_feed_price_images.py <image_path>
```

### With Tenant Name
```bash
python python_backend/cli/process_feed_price_images.py <image_path> --tenant-name "Pradeep Farm"
```

### With Tenant ID
```bash
python python_backend/cli/process_feed_price_images.py <image_path> --tenant-id "8d7939f7-b716-4eb0-98d4-544c18c8dfb8"
```

## Image Format

The script expects images with feed price sheets containing:
- **Date**: In format DD.MM.YY or DD-MM-YY (e.g., "08.12.25", "22-12-25")
- **Feed Prices**: 
  - Layer Mash
  - Pre-Layer Mash (or Pre-Layer Mash, PLM)
  - Grower Mash
  - Chick Mash

Example format:
```
EVEREST FEEDS, NAMAKKAL
DATE: 08.12.25
LAYER FEED RATE
Layer Mash: 2060/-
Pre-Layer Mash: 2070/-
Grower Mash: 2160/-
Chick Mash: 2480/-
```

## How It Works

1. **OCR Extraction**: Uses Tesseract OCR to extract text from the image
2. **Date Parsing**: Extracts date from the image (supports multiple formats)
3. **Price Extraction**: Finds feed names and their corresponding prices
4. **Database Insert**: Inserts/updates prices in `price_history` table

## Feed Name Normalization

The script normalizes feed names to match database values:
- "Layer Mash" → "LAYER MASH"
- "Pre-Layer Mash", "Pre Layer Mash", "PLM" → "PRE-LAYER MASH"
- "Grower Mash" → "GROWER MASH"
- "Chick Mash", "Chick" → "CHICK MASH"

## Database Schema

Prices are stored in the `price_history` table:
- `tenant_id`: UUID of the tenant
- `price_date`: Date of the price (DATE)
- `price_type`: 'FEED'
- `item_name`: Feed name (e.g., "LAYER MASH")
- `price`: Price value (DECIMAL)

## Troubleshooting

### OCR Not Working
- Ensure Tesseract is installed and in PATH
- Check image quality (should be clear, high resolution)
- Try preprocessing the image (increase contrast, remove noise)

### Date Not Found
- Check if date format matches expected patterns (DD.MM.YY or DD-MM-YY)
- Verify date is clearly visible in the image

### Prices Not Extracted
- Ensure feed names match expected patterns (Layer Mash, Pre-Layer Mash, etc.)
- Check if prices are in format like "2060/-" or "2060"
- Verify image text extraction is working (check printed OCR output)

## Example

```bash
# Process a single image
python python_backend/cli/process_feed_price_images.py feed_price_08_12_25.png

# Process with specific tenant
python python_backend/cli/process_feed_price_images.py feed_price_08_12_25.png --tenant-name "Pradeep Farm"
```

## Batch Processing

To process multiple images, use a simple loop:

```bash
for img in feed_prices/*.png; do
    python python_backend/cli/process_feed_price_images.py "$img"
done
```


