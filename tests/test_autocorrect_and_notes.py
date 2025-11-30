import os, sys
sys.path.insert(0, os.path.dirname(os.path.dirname(__file__)))
import unittest
from parse_poultry_statement import parse_item_line, determine_category, parse_pdf_statement

class AutoCorrectNotesTests(unittest.TestCase):
    def test_parse_item_auto_correct_and_notes(self):
        s = 'LARGE 10 NOS 100 1000'
        parsed = parse_item_line(s)
        # Without auto-correct, determine_category returns other
        cat, note = determine_category(parsed['item_name'], parsed['unit'], auto_correct=False)
        self.assertEqual(cat, 'other')
        # With auto-correct, it's egg
        cat2, note2 = determine_category(parsed['item_name'], parsed['unit'], auto_correct=True)
        self.assertEqual(cat2, 'egg')
        self.assertEqual(note2, 'auto_classified_to_egg')

    def test_parse_pdf_has_parsing_notes(self):
        parsed = parse_pdf_statement('JULY_25.pdf', auto_correct=True, add_parsing_notes=True)
        raw = parsed['raw_items']
        self.assertIn('parsing_notes', raw.columns)
        # There should be at least one parsing note marked (tds/disc/auto)`
        has_notes = raw['parsing_notes'].notna().sum() > 0
        self.assertTrue(has_notes)
    def test_generic_tds_capture(self):
        # Create a sample line that mentions TDS not in 'To TDS' form
        s = 'TDS DEDUCTED BILL NO 1213 143.00'
        # Simulate parsing loop capture: tokens and find_last_amount_token
        tokens = s.split()
        from parse_poultry_statement import find_last_amount_token, clean_number
        idx, tok = find_last_amount_token(tokens)
        amt = clean_number(tok)
        self.assertEqual(amt, 143.0)

if __name__ == '__main__':
    unittest.main()
