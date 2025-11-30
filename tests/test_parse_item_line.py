import os, sys
sys.path.insert(0, os.path.dirname(os.path.dirname(__file__)))
import unittest
from parse_poultry_statement import parse_item_line, classify_item_by_name
from parse_poultry_statement import determine_category

class ParseItemLineTests(unittest.TestCase):
    def test_samples(self):
        samples = [
            ("LARGE EGG 10 NOS 100 1000", 'egg', 1000.0),
            ("LARGE 10 NOS 100 1000", 'other', 1000.0),
            ("LARGE EGG 10 NOS 100 1000 BILL NO 1234", 'egg', 1000.0),
            ("LAYER MASH 50 KGS 90/KGS 4500", 'feed', 4500.0),
            ("LAYER MASH 50 KGS 90/KGS 4500 BILL NO 102", 'feed', 4500.0),
            ("VETMULIN 2 KGS 500/KGS 1000", 'medicine', 1000.0),
            ("CORRECT EGG 10 NOS 110 1100 BILL NO 1000", 'egg', 1100.0),
            ("PRE LAYER MASH 60 KGS 85/KGS 5100 2000 BILL NO 105", 'feed', 5100.0),
            ("SMALL EGG 100 NOS 4.5 450", 'egg', 450.0),
            ("OXYCYCLINE 1 NOS 1200 1200", 'medicine', 1200.0),
        ]
        for s, expected_cat, expected_amount in samples:
            parsed = parse_item_line(s)
            self.assertIsNotNone(parsed, msg=f"Parse failed for: {s}")
            cat = classify_item_by_name(parsed['item_name'])
            self.assertEqual(cat, expected_cat, msg=f"Wrong category for {s}: got {cat}")
            self.assertAlmostEqual(float(parsed['amount']), float(expected_amount), places=2, msg=f"Wrong amount for {s}: got {parsed['amount']}")

    def test_qty_rate_amount_source(self):
        s = "PRE LAYER MASH 60 KGS 85/KGS 5100 2000"
        parsed = parse_item_line(s)
        self.assertEqual(parsed['amount_source'], 'qty_rate')

    def test_auto_classification(self):
        # Without auto-correct, 'LARGE' alone is 'other'
        cat, note = determine_category('LARGE', unit='NOS', auto_correct=False)
        self.assertEqual(cat, 'other')
        # With auto-correct, it becomes 'egg'
        cat, note = determine_category('LARGE', unit='NOS', auto_correct=True)
        self.assertEqual(cat, 'egg')
        self.assertEqual(note, 'auto_classified_to_egg')

if __name__ == '__main__':
    unittest.main()
