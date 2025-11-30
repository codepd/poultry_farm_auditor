import os, sys
sys.path.insert(0, os.path.dirname(os.path.dirname(__file__)))
import unittest
from parse_poultry_statement import parse_pdf_statement

class ParsePdfJulyTests(unittest.TestCase):
    def test_july_pdf_parsing(self):
        parsed = parse_pdf_statement('JULY_25.pdf')
        summary = parsed['summary']
        # Expected values derived from ledger
        expected_net = 1791511.20
        expected_tds = 4669.00
        expected_discounts = 50600.00
        expected_opening = 1070815.35
        self.assertAlmostEqual(summary['net_profit'], expected_net, places=2)
        self.assertAlmostEqual(summary['total_tds'], expected_tds, places=2)
        self.assertAlmostEqual(summary['total_discounts'], expected_discounts, places=2)
        self.assertAlmostEqual(summary['opening_balance'], expected_opening, places=2)

if __name__ == '__main__':
    unittest.main()
