#!/usr/bin/env python3

import os, sys
sys.path.insert(0, os.path.dirname(os.path.dirname(__file__)))
from parse_poultry_statement import parse_item_line, classify_item_by_name, find_last_number_token, find_last_amount_token

SAMPLES = [
    # Straightforward egg
    "LARGE EGG 10 NOS 100 1000",
    # Egg but no 'EGG' word, only 'LARGE' (should still be recognized?)
    "LARGE 10 NOS 100 1000",
    # Egg with trailing bill number
    "LARGE EGG 10 NOS 100 1000 BILL NO 1234",
    # Feed straightforward
    "LAYER MASH 50 KGS 90/KGS 4500",
    # Feed with trailing bill number
    "LAYER MASH 50 KGS 90/KGS 4500 BILL NO 102",
    # Medicine straightforward
    "VETMULIN 2 KGS 500/KGS 1000",
    # Mixed: amount vs bill number at end
    "CORRECT EGG 10 NOS 110 1100 BILL NO 1000",
    "PRE LAYER MASH 60 KGS 85/KGS 5100 2000 BILL NO 105",
    # often format: qty unit rate amount
    "SMALL EGG 100 NOS 4.5 450",
    "OXYCYCLINE 1 NOS 1200 1200"
]

for s in SAMPLES:
    parsed = parse_item_line(s)
    tokens = s.strip().split()
    ln_idx, ln_tok = find_last_number_token(tokens)
    la_idx, la_tok = find_last_amount_token(tokens)
    category = classify_item_by_name(parsed.get('item_name') if parsed else None) if parsed else None
    print('\nINPUT:', s)
    print('PARSED:', parsed)
    print('tokens:', tokens)
    print('last-number (raw):', ln_idx, ln_tok)
    print('last-amount (heuristic):', la_idx, la_tok)
    print('CATEGORY:', category)
    # validation against expected
    EXPECTED = {
        SAMPLES[0]: ('egg', 1000.0),
        SAMPLES[1]: ('other', 1000.0),
        SAMPLES[2]: ('egg', 1000.0),
        SAMPLES[3]: ('feed', 4500.0),
        SAMPLES[4]: ('feed', 4500.0),
        SAMPLES[5]: ('medicine', 1000.0),
        SAMPLES[6]: ('egg', 1100.0),
        SAMPLES[7]: ('feed', 5100.0),
        SAMPLES[8]: ('egg', 450.0),
        SAMPLES[9]: ('medicine', 1200.0),
    }
    exp_cat, exp_amt = EXPECTED[s]
    parsed_amt = parsed.get('amount') if parsed else None
    ok = (category == exp_cat) and (parsed_amt == exp_amt)
    print('TEST PASS:' if ok else 'TEST FAIL:', f"expected=({exp_cat},{exp_amt}), got=({category},{parsed_amt})")

print('\nDone testing.')
