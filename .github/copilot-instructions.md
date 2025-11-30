# Copilot instructions for 'automate_using_python' project

This repository contains a small Python utility to parse poultry ledger PDF statements and export a structured Excel workbook. The instructions below are focused, actionable guidance for coding agents working on this codebase.

Quick facts
- Single primary script: `parse_poultry_statement.py` (executable script, CLI: `python parse_poultry_statement.py path/to/statement.pdf`).
- Uses `pdfplumber` for PDF extraction and `pandas` + `openpyxl` for Excel output.
- Input format: free-form ledger PDFs where each page is `pdfplumber`-extractable plain text lines.
- Output: `parsed_<originalname>.xlsx` written next to the input PDF.

Big-picture architecture (why and how)
- One-shot CLI utility: the project is intentionally script-like, not a package or service.
- Data flow:
  1. PDF text extraction (page-by-page) -> list of lines (`all_lines`).
  2. Line-wise scanning: detect date-header lines with a `DATE_HDR_RE` regex.
  3. For each header, consume following lines until the next date header; parse item lines (kg, nos, rate/qty formats), payments, TDS and discounts.
  4. Collected rows are normalized into pandas DataFrames and aggregated by `item_name` and `unit`.
  5. Exports multiple sheets to an Excel workbook (raw rows, eggs, feeds, medicines, other items, payments, tds, discounts, summary).
- Design rationale visible in code: heuristic parsing is used because ledger PDFs are irregular; domain-specific keyword lists (EGG_KEYWORDS, FEED_KEYWORDS, MEDICINES_KNOWN) are used for classification.

Project-specific conventions and patterns
- Uppercase normalization: item names are normalized to upper-case when classifying (see `classify_row`).
- Date detection: `DATE_HDR_RE` expects formats like `25-Jul-25` or `25-Jul-2025`; parsed with `strptime` attempts for `%d-%b-%y` then `%d-%b-%Y`.
- Numeric parsing: `NUM_RE` and `clean_number()` handle comma thousand separators and pick first numeric token from noisy strings.
- Item parsing: `parse_item_line()` is the canonical place to modify parsing heuristics. It:
  - finds the last numeric token as the line amount
  - looks for rate tokens containing `/` (e.g., `90/KGS`) to extract rate and qty
  - recognizes units `kgs, kg, nos, no, nos.`
- Aggregation: `agg()` groups by `item_name` and `unit` and sums `qty` and `amount`.
- Grand total formula is implemented in `parse_pdf_statement()` and intentionally uses: `Eggs - Feed - Medicine - Payments - TDS + Discount`.

Important files to reference
- `parse_poultry_statement.py` — single authoritative source. All edits that change behavior should update this file.

Common developer workflows (how to run & debug)
- Install dependencies (assumed): `pdfplumber`, `pandas`, `openpyxl`.
  - No lockfile present. If adding dependencies, update a `requirements.txt`.
- Run on a PDF: `python parse_poultry_statement.py path/to/PRADEEP\ PF\ JULY\ 25.pdf`
- Output file location: same directory as input PDF, named `parsed_<originalname>.xlsx`.
- Quick debug approach: run script against a sample PDF, inspect `raw_rows` sheet to see how lines were tokenized and classified. Add temporary `print()` statements or logging in `parse_pdf_statement()` or `parse_item_line()` for specific lines.

Project-specific tests and changes to make (discoverable patterns)
- When adding parsing rules, add a small sample input PDF (or an equivalent `.txt` extraction sample) under `tests/samples/` and write a short unit test that calls `parse_pdf_statement()` with a mocked or saved text extraction.
- Prefer to change keyword lists (`EGG_KEYWORDS`, `FEED_KEYWORDS`, `MEDICINES_KNOWN`) over rewriting classification logic unless necessary.

Integration and extension points
- PDF extraction is done with `pdfplumber.open(pdf_path)`. If PDFs have images/scans, OCR preprocessing would be required (not present). Agents should not add network calls or secret use.
- Excel writing uses `pandas.ExcelWriter(..., engine='openpyxl')` — keep this when modifying output format.

Editing guidance for AI agents (concise rules)
1. Preserve CLI behavior unless explicitly asked to package as an importable module.
2. When improving parsing heuristics, add small, focused unit tests using representative text snippets (store them under `tests/`), and keep changes isolated in `parse_item_line()` or `classify_row()`.
3. Avoid adding heavy dependencies; prefer small, widely-used libraries and update `requirements.txt` if added.
4. When changing output sheets or column names, update `export_to_excel()` and ensure `summary` keys remain stable.
5. For date formatting, prefer `format_date_obj()` for sheet-ready strings.

Examples from codebase (useful snippets to reference)
- Date header detection: `DATE_HDR_RE = re.compile(r'^(\d{1,2}-[A-Za-z]{3}-\d{2,4})\b')`
- Item parsing entry point: `parse_item_line(line)` — change here for new item formats.
- Classification: `df_items['category'] = df_items['item_name'].apply(classify_row)`

What not to change (explicit)
- Do not replace `pdfplumber` with an OCR solution without adding a clear opt-in mechanism; many ledger PDFs are text-extractable.
- Do not change the output file naming convention `parsed_<originalname>.xlsx` unless the user asks.

If something's unclear, ask these targeted questions
- Provide a sample PDF or the `raw_rows` output for the variant you want handled.
- Do you want the script packaged as a library, or kept as a simple CLI script?

Please review and tell me any missing examples or workflows to add.